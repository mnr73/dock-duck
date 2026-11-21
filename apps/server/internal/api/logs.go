package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"

	"dockduck/internal/logs"
	"dockduck/internal/projects"
)

// logOptions reads ?services=a,b&tail=200&since=15m from the request.
func logOptions(r *http.Request, follow bool) logs.Options {
	q := r.URL.Query()
	o := logs.Options{Tail: 200, Follow: follow, Since: q.Get("since")}
	if v := q.Get("services"); v != "" {
		o.Services = strings.Split(v, ",")
	}
	if v := q.Get("tail"); v != "" {
		if v == "all" {
			o.Tail = -1
		} else if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			o.Tail = n
		}
	}
	return o
}

var replicaSuffix = regexp.MustCompile(`-\d+$`)

// labelMapper turns the label compose prints in front of a line (container
// name, or `service-N`) into the service name.
func labelMapper(d projects.Detail) func(label string) string {
	byName := map[string]string{}
	for _, s := range d.Services {
		for _, c := range s.Containers {
			byName[c.Name] = s.Name
		}
		byName[s.Name] = s.Name
	}
	return func(label string) string {
		if s, ok := byName[label]; ok {
			return s
		}
		if i := replicaSuffix.FindStringIndex(label); i != nil {
			if s, ok := byName[label[:i[0]]]; ok {
				return s
			}
			if s, ok := byName[strings.TrimPrefix(label[:i[0]], d.Name+"-")]; ok {
				return s
			}
		}
		return label
	}
}

// logStream: query ?services=&tail=&since=; text frames are JSON arrays of
// {s: service, t: time, m: message}, batched. A final {"type":"end"} object
// tells the client that docker's log process ended (project removed, etc.).
func (h *dockerHandlers) logStream(w http.ResponseWriter, r *http.Request) {
	ctxName, p, ok := h.project(w, r)
	if !ok {
		return
	}
	d, err := h.Projects.Detail(r.Context(), ctxName, p)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_error", err.Error())
		return
	}
	args, err := logs.Args(ctxName, p, logOptions(r, true))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: sameOriginPatterns(r)})
	if err != nil {
		return
	}
	defer c.CloseNow()
	base, stop := sessionContext(r)
	defer stop()
	ctx, cancel := context.WithCancel(c.CloseRead(base))
	defer cancel()

	lines := make(chan logs.Line, 1024)
	done := make(chan error, 1)
	name := labelMapper(d)
	go func() {
		done <- h.Streamer.Stream(ctx, args, func(raw string) {
			l := logs.Parse(raw)
			if l.Service != "" {
				l.Service = name(l.Service)
			}
			select {
			case lines <- l:
			case <-ctx.Done():
			}
		})
	}()

	const flushEvery = 60 * time.Millisecond
	const maxBatch = 500
	tick := time.NewTicker(flushEvery)
	defer tick.Stop()
	batch := make([]logs.Line, 0, maxBatch)
	flush := func() bool {
		if len(batch) == 0 {
			return true
		}
		b, _ := json.Marshal(batch)
		batch = batch[:0]
		wctx, wcancel := wsTimeout(ctx)
		defer wcancel()
		return c.Write(wctx, websocket.MessageText, b) == nil
	}
	for {
		select {
		case l := <-lines:
			batch = append(batch, l)
			if len(batch) >= maxBatch && !flush() {
				return
			}
		case <-tick.C:
			if !flush() {
				return
			}
		case err := <-done:
			// drain what is already queued, then tell the client we are done
			for len(lines) > 0 {
				batch = append(batch, <-lines)
			}
			flush()
			msg := map[string]any{"type": "end"}
			if err != nil && ctx.Err() == nil {
				msg["error"] = firstLine(err.Error())
			}
			wctx, wcancel := wsTimeout(ctx)
			_ = wsjsonWrite(wctx, c, msg)
			wcancel()
			_ = c.Close(websocket.StatusNormalClosure, "end")
			return
		case <-ctx.Done():
			return
		}
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func wsjsonWrite(ctx context.Context, c *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, b)
}

// logDownload streams a plain-text export (no follow) as an attachment.
func (h *dockerHandlers) logDownload(w http.ResponseWriter, r *http.Request) {
	ctxName, p, ok := h.project(w, r)
	if !ok {
		return
	}
	o := logOptions(r, false)
	if r.URL.Query().Get("tail") == "" {
		o.Tail = -1
	}
	args, err := logs.Args(ctxName, p, o)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	d, err := h.Projects.Detail(r.Context(), ctxName, p)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_error", err.Error())
		return
	}
	name := labelMapper(d)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-logs.txt"`, safeFile(p.Name)))
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	_ = h.Streamer.Stream(ctx, args, func(raw string) {
		l := logs.Parse(raw)
		svc := l.Service
		if svc != "" {
			svc = name(svc)
		}
		if svc == "" {
			fmt.Fprintln(w, l.Text)
			return
		}
		fmt.Fprintf(w, "%s | %s %s\n", svc, l.Time, l.Text)
	})
}

func safeFile(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, s)
}
