package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"dockduck/internal/jobs"
	"dockduck/internal/projects"
)

// project resolves {id} in the request's docker context; writes the error response itself.
func (h *dockerHandlers) project(w http.ResponseWriter, r *http.Request) (ctxName string, p projects.Project, ok bool) {
	ctxName, err := h.contextFor(r)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_unavailable", err.Error())
		return
	}
	p, found, err := h.Projects.Find(r.Context(), ctxName, r.PathValue("id"))
	switch {
	case err != nil:
		WriteError(w, http.StatusInternalServerError, "list_failed", err.Error())
	case !found:
		WriteError(w, http.StatusNotFound, "project_not_found", "unknown project")
	default:
		ok = true
	}
	return
}

func (h *dockerHandlers) projectDetail(w http.ResponseWriter, r *http.Request) {
	ctxName, p, ok := h.project(w, r)
	if !ok {
		return
	}
	d, err := h.Projects.Detail(r.Context(), ctxName, p)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_error", err.Error())
		return
	}
	resp := map[string]any{"project": d.Project, "services": d.Services, "config_error": d.ConfigError}
	if j, busy := h.Jobs.Active(p.ID); busy {
		resp["active_job"] = j.Job
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *dockerHandlers) projectStats(w http.ResponseWriter, r *http.Request) {
	ctxName, p, ok := h.project(w, r)
	if !ok {
		return
	}
	d, err := h.Projects.Detail(r.Context(), ctxName, p)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_error", err.Error())
		return
	}
	var ids []string
	for _, s := range d.Services {
		for _, c := range s.Containers {
			if c.State == "running" {
				ids = append(ids, c.ID)
			}
		}
	}
	u, err := h.Projects.Stats(r.Context(), ctxName, p, ids)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, u)
}

func (h *dockerHandlers) projectAction(w http.ResponseWriter, r *http.Request) {
	ctxName, p, ok := h.project(w, r)
	if !ok {
		return
	}
	var opts projects.ActionOpts
	if r.ContentLength != 0 && !decode(w, r, &opts) {
		return
	}
	action := r.PathValue("action")
	args, err := projects.ActionArgs(ctxName, p, action, opts)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_action", err.Error())
		return
	}
	j, err := h.Jobs.Start(p.ID, action, opts.Services, args)
	if errors.Is(err, jobs.ErrBusy) {
		WriteError(w, http.StatusConflict, "busy", err.Error())
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "job_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"job_id": j.ID, "job": j.Job})
}

func (h *dockerHandlers) listJobs(w http.ResponseWriter, _ *http.Request) {
	rec := h.Jobs.Recent(30)
	out := make([]jobs.Job, 0, len(rec))
	for _, s := range rec {
		out = append(out, s.Job)
	}
	WriteJSON(w, http.StatusOK, out)
}

func (h *dockerHandlers) getJob(w http.ResponseWriter, r *http.Request) {
	s, ok := h.Jobs.Get(r.PathValue("id"))
	if !ok {
		WriteError(w, http.StatusNotFound, "job_not_found", "unknown job")
		return
	}
	WriteJSON(w, http.StatusOK, s.Job)
}

// jobStream: text frames = output lines, then {"type":"done","status":...,"error":...}.
func (h *dockerHandlers) jobStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := h.Jobs.Get(id); !ok {
		WriteError(w, http.StatusNotFound, "job_not_found", "unknown job")
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: sameOriginPatterns(r)})
	if err != nil {
		return
	}
	defer c.CloseNow()
	ctx := c.CloseRead(r.Context())

	pos := 0
	for {
		lines, next, st, wait, ok := h.Jobs.Follow(id, pos)
		if !ok {
			return
		}
		pos = next
		for _, l := range lines {
			wctx, cancel := wsTimeout(ctx)
			err := c.Write(wctx, websocket.MessageText, []byte(l))
			cancel()
			if err != nil {
				return
			}
		}
		if len(lines) > 0 {
			continue
		}
		if st.Status != jobs.Running {
			_ = wsjson.Write(ctx, c, map[string]string{"type": "done", "status": st.Status, "error": st.Error})
			_ = c.Close(websocket.StatusNormalClosure, "done")
			return
		}
		select {
		case <-wait:
		case <-ctx.Done():
			return
		}
	}
}

// eventStream pushes compose container events of the selected context.
func (h *dockerHandlers) eventStream(w http.ResponseWriter, r *http.Request) {
	ctxName, err := h.contextFor(r)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_unavailable", err.Error())
		return
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: sameOriginPatterns(r)})
	if err != nil {
		return
	}
	defer c.CloseNow()
	ctx := c.CloseRead(r.Context())

	ch, unsubscribe := h.Events.Subscribe(ctxName)
	defer unsubscribe()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case ev := <-ch:
			wctx, cancel := wsTimeout(ctx)
			err := wsjson.Write(wctx, c, ev)
			cancel()
			if err != nil {
				return
			}
		case <-ping.C:
			pctx, cancel := wsTimeout(ctx)
			err := c.Ping(pctx)
			cancel()
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// sameOriginPatterns allows only the request's own host (plus the Vite dev origin
// when the request comes from loopback), blocking cross-site WebSocket hijacking.
func sameOriginPatterns(r *http.Request) []string {
	pats := []string{r.Host}
	if h := r.Host; strings.HasPrefix(h, "127.0.0.1:") || strings.HasPrefix(h, "localhost:") {
		pats = append(pats, "127.0.0.1:5173", "localhost:5173")
	}
	return pats
}
