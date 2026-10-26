package proxy

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	xproxy "golang.org/x/net/proxy"
)

// SOCKSTransport forwards hostnames unchanged to an upstream SOCKS server.
// No local destination DNS resolution or alternative direct route is used.
type SOCKSTransport struct {
	Address         string
	Username        string
	Password        string
	Timeout         time.Duration
	RequireHostname bool
}

// x/net's SOCKS wrapper does not expose TCP half-close. Retain the underlying
// socket so EOF in one direction does not truncate the other direction.
type socketDialer struct {
	net.Dialer
	conn net.Conn
}

func (d *socketDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	c, err := d.Dialer.DialContext(ctx, network, address)
	d.conn = c
	return c, err
}
func (d *socketDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (s SOCKSTransport) Dial(network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, errors.New("only TCP is supported")
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 || host == "" {
		return nil, errors.New("invalid destination")
	}
	if s.RequireHostname && net.ParseIP(host) != nil {
		return nil, errors.New("hostname required for remote DNS")
	}
	if _, _, err := net.SplitHostPort(s.Address); err != nil {
		return nil, errors.New("invalid SOCKS upstream")
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	var auth *xproxy.Auth
	if s.Username != "" || s.Password != "" {
		auth = &xproxy.Auth{User: s.Username, Password: s.Password}
	}
	base := &socketDialer{Dialer: net.Dialer{Timeout: timeout}}
	dialer, err := xproxy.SOCKS5("tcp", s.Address, auth, base)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err = dialer.(xproxy.ContextDialer).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return base.conn, nil
}
