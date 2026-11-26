// nexus-fair is the offline exhibition entry point. It never launches the
// production orchestrator or accepts external scan targets.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nexus-ia/nexus/lab/fair"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	port := flag.Int("port", 8795, "Puerto de la consola local")
	check := flag.Bool("self-check", false, "Comprobar el ciclo completo y emitir evidencias JSON")
	flag.Parse()
	if flag.NArg() != 0 || *port < 1 || *port > 65535 {
		return fmt.Errorf("puerto o argumentos no válidos")
	}
	harness, err := fair.New()
	if err != nil {
		return err
	}
	defer harness.Close()
	if *check {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		before, err := harness.Run(ctx, false)
		if err != nil {
			return err
		}
		after, err := harness.Run(ctx, true)
		if err != nil {
			return err
		}
		evaluation, err := harness.Evaluate(ctx)
		if err != nil {
			return err
		}
		if len(before.Findings) != 2 || len(after.Findings) != 0 || before.Metrics.FalseNegative != 0 || after.Metrics.FalseNegative != 0 || before.Metrics.FalsePositive != 0 || after.Metrics.FalsePositive != 0 || evaluation.Improvement.Decision != "promoted" || evaluation.Noisy.Active != "v1" {
			return fmt.Errorf("la autocomprobación no cumple el contrato de demostración")
		}
		return json.NewEncoder(os.Stdout).Encode(harness.State())
	}
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	handler, err := harness.Handler(listener.Addr().String())
	if err != nil {
		listener.Close()
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	fmt.Printf("NEXUS — consola de demostración\nAbrir http://%s\nObjetivo ficticio; datos locales; Ctrl+C para detener.\n", listener.Addr())
	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
