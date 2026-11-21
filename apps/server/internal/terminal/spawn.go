package terminal

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type ptyProcess struct {
	cmd  *exec.Cmd
	ptmx *os.File

	once sync.Once
}

// SpawnPTY starts argv attached to a new PTY. TERM/COLORTERM are set for the
// local `docker` client (they are forwarded to the exec'd process by `-e`, see
// BuildExecArgs); the session is its own process group so Kill can hang it up.
func SpawnPTY(argv []string, cols, rows uint16) (Process, error) {
	if len(argv) == 0 {
		return nil, errors.New("empty command")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, err
	}
	return &ptyProcess{cmd: cmd, ptmx: ptmx}, nil
}

func (p *ptyProcess) Read(b []byte) (int, error)  { return p.ptmx.Read(b) }
func (p *ptyProcess) Write(b []byte) (int, error) { return p.ptmx.Write(b) }

func (p *ptyProcess) Resize(cols, rows uint16) error {
	return pty.Setsize(p.ptmx, &pty.Winsize{Cols: cols, Rows: rows})
}

func (p *ptyProcess) Wait() (int, error) {
	err := p.cmd.Wait()
	p.once.Do(func() { _ = p.ptmx.Close() })
	if p.cmd.ProcessState != nil {
		return p.cmd.ProcessState.ExitCode(), nil
	}
	return -1, err
}

// Kill closes the PTY (the child gets SIGHUP, like closing a terminal window)
// and falls back to SIGKILL on the process group if it ignores that.
func (p *ptyProcess) Kill() error {
	p.once.Do(func() { _ = p.ptmx.Close() })
	if p.cmd.Process == nil {
		return nil
	}
	_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGHUP)
	go func() {
		time.Sleep(2 * time.Second)
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	}()
	return nil
}
