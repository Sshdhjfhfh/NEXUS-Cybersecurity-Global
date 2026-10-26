// Real SOCKS gateway: transport and process lifecycle only.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nexus-ia/nexus/phantom/proxy"
)

func main() {
	upstream := os.Getenv("PHANTOM_UPSTREAM")
	if upstream == "" {
		log.Fatal("PHANTOM_UPSTREAM is required; direct egress disabled")
	}
	var isolation [16]byte
	if _, err := rand.Read(isolation[:]); err != nil {
		log.Fatal("isolation key unavailable")
	}
	cfg := proxy.DefaultConfig()
	cfg.ListenAddr = "0.0.0.0:1080"
	cfg.Dialer = proxy.SOCKSTransport{Address: upstream, Username: hex.EncodeToString(isolation[:]), Password: "nexus", RequireHostname: true}
	p := proxy.NewProxy(cfg)
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
	log.Print("SOCKS gateway ready; remote DNS required, no direct fallback")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	_ = p.Stop()
}
