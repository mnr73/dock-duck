package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"

	"dockduck/internal/dockercli"
	"dockduck/internal/projects"
	"dockduck/internal/terminal"
)

// Terminal WebSocket protocol (one session per socket, many sockets over time):
//
//	server -> client  TEXT    {"type":"attached","replay":N} first; then, if N > 0, ONE binary frame of N bytes
//	server -> client  BINARY  raw terminal output
//	server -> client  TEXT    {"type":"exit","code":N} | {"type":"taken"} | {"type":"slow"}
//	client -> server  BINARY  raw keyboard input
//	client -> server  TEXT    {"type":"resize","cols":N,"rows":N}
//
// Output and input are binary on purpose: a UTF-8 text round trip would corrupt
// partial multi-byte characters and escape sequences split across reads.

type openBody struct {
	terminal.Request
}

// openTerminal starts a session in a running container of the project.
func (h *dockerHandlers) openTerminal(w http.ResponseWriter, r *http.Request) {
	ctxName, p, ok := h.project(w, r)
	if !ok {
		return
	}
	var body openBody
	if !decode(w, r, &body) {
		return
	}
	if body.Service == "" || !dockercli.ValidName(body.Service) {
		WriteError(w, http.StatusBadRequest, "bad_request", "service is required")
		return
	}
	if err := body.Request.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	container, ok := h.runningContainer(w, r, ctxName, p, body.Service, body.Container)
	if !ok {
		return
	}

	meta := terminal.Meta{Project: p.ID, ProjectName: p.Name, Service: body.Service, Container: container,
		User: body.User, Shell: body.Shell, Command: body.Command}
	s, err := h.Terminals.OpenTagged(meta, func(id string) ([]string, string, func(), error) {
		args, display, err := terminal.BuildExecArgs(ctxName, container, id, body.Request)
		if err != nil {
			return nil, "", nil, err
		}
		argv := append([]string{h.dockerBin()}, args...)
		cleanup := func() {
			cargs, err := terminal.CleanupArgs(ctxName, container, id, body.User)
			if err != nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := h.Runner.Output(ctx, cargs...); err != nil {
				slog.Warn("terminal cleanup failed", "container", container, "err", firstLine(err.Error()))
			}
		}
		return argv, display, cleanup, nil
	}, body.Cols, body.Rows)
	switch {
	case errors.Is(err, terminal.ErrTooMany):
		WriteError(w, http.StatusTooManyRequests, "too_many", err.Error())
		return
	case err != nil:
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	slog.Info("terminal opened", "id", s.ID(), "project", p.Name, "service", body.Service, "container", container, "user", body.User, "remote", r.RemoteAddr)
	WriteJSON(w, http.StatusCreated, s.Info())
}

func (h *dockerHandlers) listTerminals(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, h.Terminals.List())
}

func (h *dockerHandlers) closeTerminal(w http.ResponseWriter, r *http.Request) {
	if !h.Terminals.Close(r.PathValue("id")) {
		WriteError(w, http.StatusNotFound, "not_found", "no such terminal session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// terminalStream attaches the socket to a session.
func (h *dockerHandlers) terminalStream(w http.ResponseWriter, r *http.Request) {
	s, ok := h.Terminals.Get(r.PathValue("id"))
	if !ok {
		WriteError(w, http.StatusNotFound, "not_found", "no such terminal session")
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: sameOriginPatterns(r)})
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(1 << 20)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	replay, att := s.Attach()
	defer s.Detach(att)

	// the viewer's size wins as soon as it reports one (resize is its first message)
	write := func(typ websocket.MessageType, b []byte) bool {
		wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
		defer wcancel()
		return c.Write(wctx, typ, b) == nil
	}
	ctl := func(v map[string]any) bool {
		b, _ := json.Marshal(v)
		return write(websocket.MessageText, b)
	}
	// announce the replay first: the client must not answer terminal queries
	// (cursor position, device attributes) that were printed long ago and are
	// part of the replayed bytes, or the answers would be typed into the shell
	if !ctl(map[string]any{"type": "attached", "replay": len(replay)}) {
		return
	}
	if len(replay) > 0 && !write(websocket.MessageBinary, replay) {
		return
	}

	// keyboard -> process
	readErr := make(chan error, 1)
	go func() {
		for {
			typ, data, err := c.Read(ctx)
			if err != nil {
				readErr <- err
				return
			}
			switch typ {
			case websocket.MessageBinary:
				if err := s.Write(data); err != nil {
					readErr <- err
					return
				}
			case websocket.MessageText:
				var m struct {
					Type string `json:"type"`
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				if json.Unmarshal(data, &m) == nil && m.Type == "resize" {
					_ = s.Resize(m.Cols, m.Rows)
				}
			}
		}
	}()

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case b, ok := <-att.C:
			if !ok {
				switch att.Reason() {
				case "taken", "slow":
					ctl(map[string]any{"type": att.Reason()})
				default:
					<-s.Done()
					if h.Terminals.ShuttingDown() {
						// the process was killed because the server stops, not because the shell ended
						ctl(map[string]any{"type": "gone"})
					} else {
						ctl(map[string]any{"type": "exit", "code": s.Exit()})
					}
				}
				_ = c.Close(websocket.StatusNormalClosure, "")
				return
			}
			if !write(websocket.MessageBinary, b) {
				return
			}
		case <-ping.C:
			pctx, pcancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.Ping(pctx)
			pcancel()
			if err != nil {
				return
			}
		case <-readErr:
			return // the browser went away; the shell keeps running for a re-attach
		case <-ctx.Done():
			return
		}
	}
}

// dockerBin is the docker client binary used for exec sessions.
func (h *dockerHandlers) dockerBin() string {
	if h.DockerBin != "" {
		return h.DockerBin
	}
	return "docker"
}

// runningContainer picks the running container of `service` from the project's own
// state (never from the request, except to choose one replica), or answers 409.
func (h *dockerHandlers) runningContainer(w http.ResponseWriter, r *http.Request, ctxName string, p projects.Project, service, replica string) (string, bool) {
	d, err := h.Projects.Detail(r.Context(), ctxName, p)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_error", err.Error())
		return "", false
	}
	for _, s := range d.Services {
		if s.Name != service {
			continue
		}
		for _, c := range s.Containers {
			if c.State == "running" && (replica == "" || replica == c.Name) {
				return c.Name, true
			}
		}
	}
	WriteError(w, http.StatusConflict, "not_running", "the service has no running container to attach to")
	return "", false
}

// systemTerminalInfo says whether a native terminal window can be opened.
func (h *dockerHandlers) systemTerminalInfo(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"available": false}
	t := h.systemTerminal()
	switch {
	case t == nil:
		out["reason"] = "no terminal emulator found (set DOCKDUCK_TERMINAL, e.g. DOCKDUCK_TERMINAL=kitty)"
	case !isLoopback(r.RemoteAddr):
		out["reason"] = "only available when DockDuck is opened from the machine it runs on"
	default:
		if _, ok := terminal.DisplayEnv(os.Environ(), os.Getenv("XDG_RUNTIME_DIR"), "/tmp/.X11-unix"); !ok {
			out["reason"] = terminal.ErrNoDisplay.Error()
		} else {
			out["available"], out["name"] = true, t.Name
		}
	}
	WriteJSON(w, http.StatusOK, out)
}

func (h *dockerHandlers) systemTerminal() *terminal.SystemTerminal {
	if h.SystemTerminal != nil {
		return h.SystemTerminal()
	}
	return terminal.DetectSystemTerminal()
}

// openSystemTerminal opens the same `docker exec` in the user's own terminal
// emulator (their fonts, shortcuts, tabs, scrollback). Loopback clients only: the
// window appears on the machine DockDuck runs on.
func (h *dockerHandlers) openSystemTerminal(w http.ResponseWriter, r *http.Request) {
	if !isLoopback(r.RemoteAddr) {
		WriteError(w, http.StatusForbidden, "local_only", "a native terminal can only be opened from the machine DockDuck runs on")
		return
	}
	ctxName, p, ok := h.project(w, r)
	if !ok {
		return
	}
	var body openBody
	if !decode(w, r, &body) {
		return
	}
	if body.Service == "" || !dockercli.ValidName(body.Service) {
		WriteError(w, http.StatusBadRequest, "bad_request", "service is required")
		return
	}
	t := h.systemTerminal()
	if t == nil {
		WriteError(w, http.StatusNotImplemented, "no_terminal", "no terminal emulator found (set DOCKDUCK_TERMINAL, e.g. DOCKDUCK_TERMINAL=kitty)")
		return
	}
	container, ok := h.runningContainer(w, r, ctxName, p, body.Service, body.Container)
	if !ok {
		return
	}
	marker := terminal.NewMarker()
	args, display, err := terminal.BuildExecArgs(ctxName, container, marker, body.Request)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := t.Launch(h.dockerBin(), ctxName, container, marker, body.User, args); err != nil {
		code := http.StatusBadGateway
		if errors.Is(err, terminal.ErrNoDisplay) {
			code = http.StatusConflict
		}
		WriteError(w, code, "launch_failed", err.Error())
		return
	}
	slog.Info("system terminal opened", "terminal", t.Name, "project", p.Name, "service", body.Service, "container", container, "user", body.User)
	WriteJSON(w, http.StatusOK, map[string]any{"terminal": t.Name, "container": container, "display": display})
}

func isLoopback(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
