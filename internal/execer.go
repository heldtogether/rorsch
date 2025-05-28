package internal

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"syscall"

	"github.com/creack/pty"
)

type ExecerCallback func(c *Command, line string, err error, done bool)

type Execer struct {
	Command  *Command
	Callback ExecerCallback
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

	ptmx, err := pty.Start(cmd)
	if err != nil {
		e.Callback(e.Command, "", err, true)
		return
	}

	slog.Info("Starting command", "cmd", strings.Join(cmd.Args, " "), "PID", cmd.Process.Pid)

	e.Command.mu.Lock()
	e.Command.proc = cmd
	e.Command.mu.Unlock()

	go func() {
		r := io.MultiReader(ptmx)
		scanner := bufio.NewScanner(r)

		for scanner.Scan() {
			e.Command.mu.Lock()
			e.Callback(e.Command, fmt.Sprintf("%s", scanner.Text()), nil, false)
			e.Command.mu.Unlock()
		}
	}()
	err = cmd.Wait()
	e.Command.mu.Lock()
	e.Callback(e.Command, "", err, true)
	e.Command.mu.Unlock()
}

func (e *Execer) Stop() error {
	e.Command.mu.Lock()
	defer e.Command.mu.Unlock()

	if e.Command.proc != nil && e.Command.proc.Process != nil {
		slog.Debug("killing proc", "PID", e.Command.proc.Process.Pid)
		pgid, err := syscall.Getpgid(e.Command.proc.Process.Pid)
		if err == nil {
			err = syscall.Kill(-pgid, syscall.SIGKILL)
			slog.Debug("killed process group", "PID", e.Command.proc.Process.Pid, "error", err)
		} else {
			err = e.Command.proc.Process.Kill()
			slog.Debug("killed process", "PID", e.Command.proc.Process.Pid, "error", err)
		}
		e.Command.proc = nil
		return err
	}

	return nil
}
