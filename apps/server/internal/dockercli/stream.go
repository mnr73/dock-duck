package dockercli

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Streamer runs a long-lived docker command and delivers its output line by line.
type Streamer interface {
	// Stream blocks until the command ends or ctx is cancelled (which kills it).
	// stdout and stderr are merged. A non-zero exit is returned as *ExitError.
	Stream(ctx context.Context, args []string, onLine func(string)) error
}

// StreamerFunc adapts a function to Streamer (tests).
type StreamerFunc func(ctx context.Context, args []string, onLine func(string)) error

// Stream implements Streamer.
func (f StreamerFunc) Stream(ctx context.Context, args []string, onLine func(string)) error {
	return f(ctx, args, onLine)
}

// Stream implements Streamer for the real binary.
func (c *Client) Stream(ctx context.Context, args []string, onLine func(string)) error {
	if err := guard(args); err != nil {
		return err
	}
	bin := c.Bin
	if bin == "" {
		bin = "docker"
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.WaitDelay = 3 * time.Second
	w := &lineWriter{emit: onLine}
	cmd.Stdout, cmd.Stderr = w, w
	err := cmd.Run()
	w.flush()
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return &ExitError{Args: args, Err: err}
	}
	return nil
}

// lineWriter splits writes into lines; safe for the shared stdout/stderr case.
type lineWriter struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	emit func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	for {
		i := bytes.IndexByte(w.buf.Bytes(), '\n')
		if i < 0 {
			break
		}
		line := string(w.buf.Next(i + 1))
		w.emit(cleanLine(line))
	}
	if w.buf.Len() > 1<<20 { // pathological line without newline
		w.emit(cleanLine(w.buf.String()))
		w.buf.Reset()
	}
	return len(p), nil
}

func (w *lineWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf.Len() > 0 {
		w.emit(cleanLine(w.buf.String()))
		w.buf.Reset()
	}
}

// cleanLine drops the newline and keeps only the last carriage-return segment.
func cleanLine(s string) string {
	s = strings.TrimRight(s, "\r\n")
	if i := strings.LastIndexByte(s, '\r'); i >= 0 {
		s = s[i+1:]
	}
	return s
}

var _ io.Writer = (*lineWriter)(nil)
