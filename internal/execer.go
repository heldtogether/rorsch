package internal

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
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
	parts := strings.Fields(e.Command.Exec)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = e.Command.Cwd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.Callback(e.Command, "", err, false)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		e.Callback(e.Command, "", err, false)
		return
	}

	if err := cmd.Start(); err != nil {
		e.Callback(e.Command, "", err, false)
		return
	}

	r := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		e.Callback(e.Command, scanner.Text(), nil, false)
	}
	err = cmd.Wait()
	e.Callback(e.Command, "", err, true)
}
