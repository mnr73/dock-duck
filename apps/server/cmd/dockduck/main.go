// Command dockduck is the DockDuck server: API + embedded web UI.
package main

import (
	"context"
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
	"dockduck/internal/auth"
	"dockduck/internal/config"
	"dockduck/internal/contexts"
	"dockduck/internal/dockercli"
	"dockduck/internal/events"
	"dockduck/internal/jobs"
	"dockduck/internal/ports"
	"dockduck/internal/projects"
	"dockduck/internal/registries"
	"dockduck/internal/terminal"
	"dockduck/internal/updater"
	"dockduck/internal/webui"
)

// defaultAddr is loopback on a port no common dev tool uses (not 3000, 5173, 8000, 8080...).
const defaultAddr = "127.0.0.1:9473"

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "passwd":
			os.Exit(runPasswd(os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
		case "update":
			os.Exit(runUpdate(os.Args[2:], os.Stdout, os.Stderr))
		case "version":
			fmt.Println(version)
			return
		}
	}
	addr := flag.String("addr", defaultAddr, "listen address")
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
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid --addr: %w", err)
	}
	// reachable from the network = root on this machine for whoever logs in: never without a password
	networked := !isLoopbackHost(host)

	if cfgPath == "" {
		if cfgPath, err = config.DefaultPath(); err != nil {
			return err
		}
	}
	store, err := config.Open(cfgPath)
	if err != nil {
		return fmt.Errorf("config %s: %w", cfgPath, err)
	}
	if networked && store.Get().Auth.PasswordHash == "" {
		return fmt.Errorf("refusing to listen on %s without a password: run `dockduck passwd` first (the Docker socket is root-equivalent)", addr)
	}
	authSvc := auth.NewService(store, networked)
	docker := dockercli.New()
	home, _ := os.UserHomeDir()

	terminals := terminal.NewManager(terminal.SpawnPTY)
	restart := make(chan struct{}, 1)
	srv := &http.Server{
		Addr: addr,
		Handler: api.New(api.Deps{
			Info:       api.Info{Version: version, Home: home},
			UI:         webui.Handler(),
			Store:      store,
			Contexts:   &contexts.Service{Docker: docker, Store: store},
			Projects:   &projects.Service{Docker: docker, Store: store},
			Jobs:       jobs.New(docker),
			Events:     events.New(docker),
			Ports:      &ports.Service{Docker: docker},
			Streamer:   docker,
			Stdin:      docker,
			Runner:     docker,
			Terminals:  terminals,
			Auth:       authSvc,
			Registries: &registries.Service{Docker: docker},
			Updater:    &updater.Updater{Version: version},
			Restart: func() {
				select {
				case restart <- struct{}{}:
				default:
				}
			},
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		switch {
		case store.Get().Auth.PasswordHash != "":
			slog.Info("login is required")
		case networked:
			// unreachable: refused above
		default:
			slog.Warn("no password is set: anyone on this machine can use DockDuck. Set one in the UI or with `dockduck passwd`")
		}
		slog.Info("listening", "addr", "http://"+addr, "version", version)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-restart:
		terminals.Shutdown()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return reexec()
	case <-ctx.Done():
		terminals.Shutdown() // kill shells and what they left running in containers
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// reexec starts the freshly installed binary in place of this process. Under a
// service manager (systemd sets INVOCATION_ID) it just exits: the manager's
// Restart= brings the new binary up, and exec-ing behind its back would confuse it.
func reexec() error {
	if updater.Managed() {
		slog.Info("update installed; exiting so the service manager starts the new version")
		os.Exit(0)
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("update installed, restart manually: %w", err)
	}
	slog.Info("update installed; restarting", "exe", exe)
	// the old file was replaced by rename; the path now points at the new binary
	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
		return fmt.Errorf("update installed but restart failed, start dockduck again: %w", err)
	}
	return nil
}

// isLoopbackHost: localhost or a loopback IP literal. An empty host (":9473") and
// 0.0.0.0 listen on every interface, so they are NOT loopback.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
