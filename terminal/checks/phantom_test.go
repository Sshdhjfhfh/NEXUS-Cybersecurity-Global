// These audit probes exercise existing Phantom behavior without changing it.
package checks

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/phantom/onion"
	"github.com/nexus-ia/nexus/phantom/proxy"
)

func connectSOCKS(t *testing.T, server, target string) (net.Conn, byte) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", server, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
		t.Fatal(err)
	}
	hello := make([]byte, 2)
	if _, err := io.ReadFull(conn, hello); err != nil || !bytes.Equal(hello, []byte{5, 0}) {
		t.Fatalf("SOCKS greeting failed: %v %v", hello, err)
	}
	host, rawPort, err := net.SplitHostPort(target)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatal(err)
	}
	request := append([]byte{5, 1, 0, 1}, net.ParseIP(host).To4()...)
	request = binary.BigEndian.AppendUint16(request, uint16(port))
	if _, err := conn.Write(request); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, 10)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}
	return conn, reply[1]
}

func TestDefaultProxyRejectsMissingTransport(t *testing.T) {
	p := proxy.NewProxy(proxy.DefaultConfig())
	if err := p.Start(); err == nil {
		p.Stop()
		t.Fatal("proxy accepted missing transport")
	}
	if p.Addr() != nil {
		t.Fatal("listener opened before transport validation")
	}
}

type denyingDialer struct{}

func (denyingDialer) Dial(string, string) (net.Conn, error) {
	return nil, errors.New("audit transport denied")
}

func TestConfiguredDialerFailureDoesNotFallBackToDirect(t *testing.T) {
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	cfg := proxy.DefaultConfig()
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.Dialer = denyingDialer{}
	p := proxy.NewProxy(cfg)
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
	_, code := connectSOCKS(t, p.Addr().String(), target.Addr().String())
	if code == proxy.RepSuccess {
		t.Fatal("denied transport unexpectedly succeeded")
	}
	target.(*net.TCPListener).SetDeadline(time.Now().Add(100 * time.Millisecond))
	conn, err := target.Accept()
	if err == nil {
		conn.Close()
		t.Fatal("proxy bypassed configured transport")
	}
	if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
		t.Fatalf("unexpected listener error: %v", err)
	}
	t.Log("CONFIRMED: a supplied failing transport does not trigger direct fallback")
}

func TestOnionSendBuildsCiphertextWithoutContactingRelays(t *testing.T) {
	directory := onion.NewDirectory()
	listeners := make([]*net.TCPListener, 0, 3)
	for i := range 3 {
		ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()
		listeners = append(listeners, ln)
		directory.Add(&onion.NodeInfo{ID: onion.NodeID(fmt.Sprintf("audit-%d", i)), Endpoint: ln.Addr().String(), Active: true})
	}
	client := onion.NewClient(onion.DefaultConfig(), directory)
	defer client.Close()
	plain := []byte("isolated transport audit")
	ciphertext, circuit, err := client.Send(plain)
	if err != nil || circuit == nil || bytes.Equal(ciphertext, plain) {
		t.Fatalf("onion encryption fixture failed: %v", err)
	}
	if circuit.HopCount() != 3 || circuit.CellsSent != 1 {
		t.Fatal("unexpected in-memory circuit counters")
	}
	for _, ln := range listeners {
		ln.SetDeadline(time.Now().Add(100 * time.Millisecond))
		conn, err := ln.Accept()
		if err == nil {
			conn.Close()
			t.Fatal("onion client contacted a relay; audit conclusion must be updated")
		}
		if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
			t.Fatalf("unexpected relay listener error: %v", err)
		}
	}
	t.Log("CONFIRMED: ciphertext and sent counters exist, but none of the three relay endpoints received a connection")
}
