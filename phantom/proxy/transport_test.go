package proxy

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestUpstreamRemoteDNSAndBidirectionalBytes(t *testing.T) {
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer ln.Close()
	names := make(chan string, 1)
	go func() {
		c, e := ln.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(2 * time.Second))
		h := make([]byte, 2)
		if _, e = io.ReadFull(c, h); e != nil {
			return
		}
		methods := make([]byte, int(h[1]))
		io.ReadFull(c, methods)
		c.Write([]byte{5, 0})
		r := make([]byte, 5)
		if _, e = io.ReadFull(c, r); e != nil || r[3] != 3 {
			return
		}
		host := make([]byte, int(r[4]))
		io.ReadFull(c, host)
		names <- string(host)
		io.ReadFull(c, make([]byte, 2))
		c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
		payload := make([]byte, 7)
		io.ReadFull(c, payload)
		c.Write(payload)
	}()
	transport := SOCKSTransport{Address: ln.Addr().String(), RequireHostname: true, Timeout: time.Second}
	c, e := transport.Dial("tcp", "unresolvable-test.invalid:443")
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(time.Second))
	c.Write([]byte("payload"))
	out := make([]byte, 7)
	if _, e = io.ReadFull(c, out); e != nil || !bytes.Equal(out, []byte("payload")) {
		t.Fatalf("relay: %q %v", out, e)
	}
	if name := <-names; name != "unresolvable-test.invalid" {
		t.Fatal(name)
	}
	if _, e = transport.Dial("tcp", "1.1.1.1:443"); e == nil {
		t.Fatal("IP bypassed remote DNS policy")
	}
}

func TestUpstreamHandshakeHasDeadline(t *testing.T) {
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer ln.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		c, e := ln.Accept()
		if e == nil {
			defer c.Close()
			<-done
		}
	}()
	started := time.Now()
	_, e = (SOCKSTransport{Address: ln.Addr().String(), Timeout: 40 * time.Millisecond}).Dial("tcp", "example.invalid:443")
	if e == nil || time.Since(started) > time.Second {
		t.Fatalf("unbounded handshake: %v", e)
	}
}

func TestUnsupportedAuthAndCommands(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.Dialer = SOCKSTransport{Address: "127.0.0.1:1"}
	p := NewProxy(cfg)
	if e := p.Start(); e != nil {
		t.Fatal(e)
	}
	defer p.Stop()
	c, e := net.Dial("tcp", p.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(time.Second))
	c.Write([]byte{5, 1, 2})
	out := make([]byte, 2)
	io.ReadFull(c, out)
	if !bytes.Equal(out, []byte{5, 255}) {
		t.Fatalf("accepted unoffered auth: %v", out)
	}
	c2, e := net.Dial("tcp", p.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer c2.Close()
	c2.SetDeadline(time.Now().Add(time.Second))
	c2.Write([]byte{5, 1, 0})
	io.ReadFull(c2, out)
	c2.Write([]byte{5, 3, 0, 1})
	reply := make([]byte, 10)
	io.ReadFull(c2, reply)
	if reply[1] != RepCmdNotSupported {
		t.Fatal("UDP was not rejected")
	}
}

type directTestDialer struct{}

func (directTestDialer) Dial(n, a string) (net.Conn, error) {
	return net.DialTimeout(n, a, time.Second)
}

func TestRelayHalfCloseDeliversFinalResponse(t *testing.T) {
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer ln.Close()
	go func() {
		c, e := ln.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(2 * time.Second))
		b, _ := io.ReadAll(c)
		c.Write(append([]byte("received:"), b...))
	}()
	cfg := DefaultConfig()
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.Dialer = directTestDialer{}
	p := NewProxy(cfg)
	if e := p.Start(); e != nil {
		t.Fatal(e)
	}
	defer p.Stop()
	c, e := (SOCKSTransport{Address: p.Addr().String(), Timeout: time.Second}).Dial("tcp", ln.Addr().String())
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(2 * time.Second))
	c.Write([]byte("test"))
	c.(*net.TCPConn).CloseWrite()
	out, e := io.ReadAll(c)
	if e != nil || string(out) != "received:test" {
		t.Fatalf("half-close: %q %v", out, e)
	}
}
