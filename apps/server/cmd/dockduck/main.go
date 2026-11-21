// Command dockduck is the DockDuck server: API + embedded web UI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dockduck/internal/api"
	"dockduck/internal/config"
	"dockduck/internal/contexts"
	"dockduck/internal/dockercli"
	"dockduck/internal/projects"
	"dockduck/internal/webui"
)

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	dev := flag.Bool("dev", false, "development mode (debug logs; UI is served by the Vite dev server)")
	cfgPath := flag.String("config", "", "config file (default: <user config dir>/dockduck/config.json)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	level := slog.LevelInfo
	if *dev {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	if err := run(*addr, *cfgPath); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(addr, cfgPath string) error {
	// Phase 8 adds auth; until then only loopback binds are allowed.
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid --addr: %w", err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("refusing to bind a non-loopback address before auth is implemented (Phase 8)")
	}

	if cfgPath == "" {
		if cfgPath, err = config.DefaultPath(); err != nil {
			return err
		}
	}
	store, err := config.Open(cfgPath)
	if err != nil {
		return fmt.Errorf("config %s: %w", cfgPath, err)
	}
	docker := dockercli.New()

	srv := &http.Server{
		Addr: addr,
		Handler: api.New(api.Deps{
			Info:     api.Info{Version: version},
			UI:       webui.Handler(),
			Store:    store,
			Contexts: &contexts.Service{Docker: docker, Store: store},
			Projects: &projects.Service{Docker: docker, Store: store},
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", "http://"+addr, "version", version)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
