package internal

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

type ExecerCallback func(c *Command, line string, err error, done bool)

type Execer struct {
	Command  *Command
	Callback ExecerCallback

	mu   sync.Mutex
	proc *exec.Cmd
}

func NewExecer(c *Command, cb ExecerCallback) *Execer {
	return &Execer{
		Command:  c,
		Callback: cb,
	}
}

func (e *Execer) Start() {
	e.Stop()

	parts := strings.Fields(e.Command.Exec)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = e.Command.Cwd
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.Callback(e.Command, "", err, true)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		e.Callback(e.Command, "", err, true)
		return
	}

	if err := cmd.Start(); err != nil {
		e.Callback(e.Command, "", err, true)
		return
	}

	slog.Info("Starting command", "cmd", strings.Join(cmd.Args, " "), "PID", cmd.Process.Pid)

	e.mu.Lock()
	e.proc = cmd
	e.mu.Unlock()

	go func() {
		r := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(r)

		for scanner.Scan() {
			e.mu.Lock()
			e.Callback(e.Command, fmt.Sprintf("%s", scanner.Text()), nil, false)
			e.mu.Unlock()
		}
	}()
	err = cmd.Wait()
	e.mu.Lock()
	e.Callback(e.Command, "", err, true)
	e.mu.Unlock()
}

func (e *Execer) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.proc != nil && e.proc.Process != nil {
		slog.Debug("killing proc", "PID", e.proc.Process.Pid)
		pgid, err := syscall.Getpgid(e.proc.Process.Pid)
		if err == nil {
			err = syscall.Kill(-pgid, syscall.SIGKILL)
			slog.Debug("killed process group", "PID", e.proc.Process.Pid, "error", err)
		} else {
			err = e.proc.Process.Kill()
			slog.Debug("killed process", "PID", e.proc.Process.Pid, "error", err)
		}
		e.proc = nil
		return err
	}

	return nil
}
