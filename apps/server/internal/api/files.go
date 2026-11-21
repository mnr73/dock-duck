package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"dockduck/internal/files"
	"dockduck/internal/projects"
)

// projectRefs resolves the project and its editable files. Paths come from the
// server's own discovery, never from the request.
func (h *dockerHandlers) projectRefs(w http.ResponseWriter, r *http.Request) (ctxName string, p projects.Project, refs []files.Ref, ok bool) {
	ctxName, p, ok = h.project(w, r)
	if !ok {
		return
	}
	return ctxName, p, files.Refs(p.Dir, p.Files, p.EnvFile), true
}

func fileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, files.ErrBadID), errors.Is(err, files.ErrNotFound):
		WriteError(w, http.StatusNotFound, "file_not_found", err.Error())
	case errors.Is(err, files.ErrConflict):
		WriteError(w, http.StatusConflict, "conflict", "the file changed on disk since you opened it")
	case errors.Is(err, files.ErrTooLarge):
		WriteError(w, http.StatusRequestEntityTooLarge, "too_large", err.Error())
	case errors.Is(err, files.ErrBinary):
		WriteError(w, http.StatusUnprocessableEntity, "not_text", err.Error())
	default:
		WriteError(w, http.StatusInternalServerError, "file_error", err.Error())
	}
}

func (h *dockerHandlers) listFiles(w http.ResponseWriter, r *http.Request) {
	_, _, refs, ok := h.projectRefs(w, r)
	if !ok {
		return
	}
	WriteJSON(w, http.StatusOK, refs)
}

// readFile returns the text. For the .env file, secret values are masked unless
// ?reveal=1 (the raw text is the user's own file, so it is only sent on reveal).
func (h *dockerHandlers) readFile(w http.ResponseWriter, r *http.Request) {
	_, _, refs, ok := h.projectRefs(w, r)
	if !ok {
		return
	}
	ref, err := files.Find(refs, r.PathValue("fid"))
	if err != nil {
		fileError(w, err)
		return
	}
	c, err := files.Read(ref)
	if err != nil {
		fileError(w, err)
		return
	}
	if ref.Kind == files.KindEnv {
		if r.URL.Query().Get("reveal") != "1" {
			c.Content = maskEnvText(c.Content)
			w.Header().Set("X-Content-Masked", "1")
		} else {
			slog.Info("env file revealed", "file", ref.Path, "remote", r.RemoteAddr)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, http.StatusOK, c)
}

// maskEnvText blanks the value of secret keys in raw .env text, line by line,
// keeping every other character (comments, order, quoting) as it was.
func maskEnvText(text string) string {
	lines := strings.Split(text, "\n")
	for i, raw := range lines {
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		body := strings.TrimPrefix(t, "export ")
		k, v, ok := strings.Cut(body, "=")
		if !ok || !files.IsSecretKey(strings.TrimSpace(k)) || strings.TrimSpace(v) == "" {
			continue
		}
		prefix := raw[:strings.Index(raw, "=")+1]
		lines[i] = prefix + "••••••••"
	}
	return strings.Join(lines, "\n")
}

type writeBody struct {
	Content string `json:"content"`
}

// writeFile saves with optimistic concurrency: `If-Match: <etag>` is required
// (empty means "create"; only valid for a not-yet-existing .env).
func (h *dockerHandlers) writeFile(w http.ResponseWriter, r *http.Request) {
	_, p, refs, ok := h.projectRefs(w, r)
	if !ok {
		return
	}
	ref, err := files.Find(refs, r.PathValue("fid"))
	if err != nil {
		fileError(w, err)
		return
	}
	var body writeBody
	r.Body = http.MaxBytesReader(w, r.Body, files.MaxSize+4096)
	if !decode(w, r, &body) {
		return
	}
	// a masked placeholder must never be saved over a real secret
	if ref.Kind == files.KindEnv && strings.Contains(body.Content, "••••••••") {
		WriteError(w, http.StatusBadRequest, "masked_value", "the content still contains masked placeholders; reveal the values before saving")
		return
	}
	etag, err := files.Write(ref, body.Content, strings.TrimSpace(r.Header.Get("If-Match")))
	if err != nil {
		fileError(w, err)
		return
	}
	slog.Info("file saved", "project", p.Name, "file", ref.Path, "bytes", len(body.Content), "remote", r.RemoteAddr)
	WriteJSON(w, http.StatusOK, map[string]any{"etag": etag, "saved_at": time.Now().UTC().Format(time.RFC3339), "backup": ref.Path + ".bak"})
}

// validateFile checks proposed content (not yet saved) with `compose config -q`.
func (h *dockerHandlers) validateFile(w http.ResponseWriter, r *http.Request) {
	ctxName, p, refs, ok := h.projectRefs(w, r)
	if !ok {
		return
	}
	ref, err := files.Find(refs, r.PathValue("fid"))
	if err != nil {
		fileError(w, err)
		return
	}
	if ref.Kind != files.KindCompose {
		WriteError(w, http.StatusBadRequest, "not_compose", "only Compose files can be validated")
		return
	}
	var body writeBody
	r.Body = http.MaxBytesReader(w, r.Body, files.MaxSize+4096)
	if !decode(w, r, &body) {
		return
	}
	at := -1
	for i, f := range p.Files {
		if f == ref.Path {
			at = i
		}
	}
	if at < 0 {
		WriteError(w, http.StatusNotFound, "file_not_found", "not a Compose file of this project")
		return
	}
	v, err := files.ValidateCompose(r.Context(), h.Stdin, ctxName, p.Dir, p.Files, at, body.Content)
	if err != nil {
		WriteError(w, http.StatusBadGateway, "docker_error", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, v)
}

// readEnv is the structured view of the project's .env for the table editor:
// parsed lines (secrets masked unless ?reveal=1) plus who uses the file.
func (h *dockerHandlers) readEnv(w http.ResponseWriter, r *http.Request) {
	ctxName, p, refs, ok := h.projectRefs(w, r)
	if !ok {
		return
	}
	ref, _ := files.Find(refs, "env")
	c, err := files.Read(ref)
	if err != nil {
		fileError(w, err)
		return
	}
	lines := files.ParseEnv(c.Content)
	reveal := r.URL.Query().Get("reveal") == "1"
	if reveal {
		slog.Info("env values revealed", "file", ref.Path, "remote", r.RemoteAddr)
	} else {
		lines = files.Mask(lines)
	}
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, http.StatusOK, map[string]any{
		"file":     c.Ref,
		"etag":     c.ETag,
		"mode":     c.Mode,
		"lines":    lines,
		"revealed": reveal,
		"example":  h.envExample(p),
		"usage":    h.envUsage(r, ctxName, p),
	})
}
