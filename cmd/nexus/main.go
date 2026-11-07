// Command nexus runs the NEXUS IA v6.0 backend: dual SQLite graph stores,
// event-driven exploration engine, and JSON HTTP API consumed by the
// shadcn/ui frontend.
//
// Absorbido de github.com/mhtsec/ARTEX (cmd/artex/main.go) — Licencia AGPL-3.0
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	agent "github.com/nexus-ia/nexus/agents/red"
	config "github.com/nexus-ia/nexus/config/app"
	"github.com/nexus-ia/nexus/orchestrator/server"
	selfupdate "github.com/nexus-ia/nexus/selfimprove/update"
)

// version is the build version, injected at release time via
// -ldflags "-X main.version=<tag>". Defaults to "dev" for local builds.
var version = "dev"

const banner = `
    _   _ _____  __ ____  ___
   | \ | | ____| \ \/ /  \/  |
   |  \| |  _|   \  /| |\/| |
   | |\  | |___  /  \| |  | |
   |_| \_|_____|/_/\_\_|  |_|

   NEXUS IA v6.0 — Sistema de Investigación Adversarial Autónomo
   "Simula · Predice · Aprende · Defiende · Gobierna · Nunca se detiene"

   ■ SOLO PARA LABORATORIO CONTROLADO ■
`

// printBanner writes the startup banner + version/runtime info to stdout.
func printBanner(addr string) {
	fmt.Print(banner)
	fmt.Printf("\n  Versión %s  ·  %s/%s  ·  %s  ·  Escuchando en %s\n\n",
		version, runtime.GOOS, runtime.GOARCH, runtime.Version(), addr)
}

// main only maps run's result onto the process exit code.
func main() {
	// The runner re-execs this binary as runner-init/runner-exec, and the
	// tool-runner container runs it as runner-serve. Handle those before flags.
	maybeRunnerSubcommand()
	os.Exit(run())
}

func run() int {
	var (
		addr    = flag.String("addr", ":8787", "HTTP listen address")
		dataDir = flag.String("data", filepath.Join(config.BaseDir(), "data"), "data directory for SQLite stores")
		proxy   = flag.String("proxy", "127.0.0.1:8788", "traffic recording proxy address (empty to disable)")
	)
	flag.Parse()

	// hand the build version to the server package so GET /api/health can report it
	server.BuildVersion = version

	// Route agent shell commands through the isolated runner when configured.
	// Must happen before the engine starts any task.
	wireLauncher()

	printBanner(*addr)

	// capture backend logs into the in-memory sink (still to stderr) so the /logs
	// page can show a live log stream. Do this first, to catch startup logs too.
	server.StartLogCapture()

	// Self-update bootstrap
	action, upState := selfupdate.Bootstrap()
	server.SetBootUpdateState(upState)
	if action == selfupdate.Restart {
		return selfupdate.ExitRestart
	}

	// surface which config file the binary reads
	cfgPath := config.Path()
	if abs, e := filepath.Abs(cfgPath); e == nil {
		cfgPath = abs
	}
	if _, e := os.Stat(cfgPath); e == nil {
		log.Printf("[config] archivo de configuración: %s", cfgPath)
	} else {
		log.Printf("[config] archivo de configuración: %s (no existe — usando solo variables de entorno NEXUS_PG_DSN)", cfgPath)
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, shutdown := shutdownContext(sigCtx)
	defer shutdown(agent.AbortShutdown)

	mgr, err := server.NewManager(*dataDir, *proxy)
	if err != nil {
		log.Fatalf("open stores: %v", err)
	}
	defer mgr.Close()

	// Surviving this long means a freshly swapped-in build actually works
	settle := time.AfterFunc(selfupdate.SettleDelay, selfupdate.Settle)
	defer settle.Stop()

	skillDir := config.SkillDir()
	if abs, err := filepath.Abs(skillDir); err == nil {
		skillDir = abs
	}
	log.Printf("[config] skill dir: %s", skillDir)
	srv := server.New(ctx, mgr, skillDir, *dataDir, keyDir())
	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		// Reap idle keep-alive connections so slow/abandoned clients can't pile up
		// and exhaust file descriptors. No WriteTimeout on purpose: it would cut
		// off long-lived SSE streams and large uploads.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MiB
	}

	go func() {
		log.Printf("NEXUS IA %s backend listening on %s (data=%s, workers=%d)", version, *addr, *dataDir, mgr.Workers())
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	// Two ways out: signal (normal stop) or restart requested (staged update)
	code := 0
	select {
	case <-ctx.Done():
	case <-server.RestartRequested():
		code = selfupdate.ExitRestart
		shutdown(agent.AbortShutdown)
	}

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	return code
}

// shutdownContext deliberately does not derive from signalCtx.
func shutdownContext(signalCtx context.Context) (context.Context, context.CancelCauseFunc) {
	ctx, shutdown := context.WithCancelCause(context.Background())
	go func() {
		select {
		case <-signalCtx.Done():
			shutdown(agent.AbortShutdown)
		case <-ctx.Done():
		}
	}()
	return ctx, shutdown
}

// keyDir is where the JWT signing key lives. The same key also derives the key
// that encrypts 2FA and SSO secrets, so losing it signs everyone out and locks
// out every user with 2FA. By default it sits next to the binary; NEXUS_KEY_DIR
// points it at persistent storage (in Docker the binary's directory is thrown
// away with the container). Kept apart from the data dir, which is browsable.
func keyDir() string {
	base := config.BaseDir()
	dir := strings.TrimSpace(os.Getenv("NEXUS_KEY_DIR"))
	if dir == "" || dir == base {
		return base
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Fatalf("[auth] NEXUS_KEY_DIR %s: %v", dir, err)
	}
	// Carry over a key created next to the binary before NEXUS_KEY_DIR was set.
	dst := filepath.Join(dir, "jwt.key")
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		if data, rerr := os.ReadFile(filepath.Join(base, "jwt.key")); rerr == nil {
			if werr := os.WriteFile(dst, data, 0o600); werr != nil {
				log.Fatalf("[auth] copy JWT key to %s: %v", dst, werr)
			}
			log.Printf("[auth] JWT key copied to %s", dst)
		}
	}
	return dir
}
