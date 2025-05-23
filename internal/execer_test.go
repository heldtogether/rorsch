package internal

import (
	"strings"
	"testing"
)

func TestExecerRunsAndCapturesOutput(t *testing.T) {
	cmd := &Command{Exec: "echo hello"}
	var lines []string
	var finalErr error
	var doneCalled bool

	execer := NewExecer(cmd, func(c *Command, line string, err error, done bool) {
		if line != "" {
			lines = append(lines, line)
		}
		if err != nil {
			finalErr = err
		}
		if done {
			doneCalled = true
		}
	})

	execer.Start()

	if !doneCalled {
		t.Error("expected done callback to be called")
	}
	if finalErr != nil {
		t.Errorf("unexpected error: %v", finalErr)
	}
	if len(lines) != 1 || strings.TrimSpace(lines[0]) != "hello" {
		t.Errorf("unexpected output lines: %v", lines)
	}
}

func TestExecerHandlesInvalidCommand(t *testing.T) {
	cmd := &Command{Exec: "nonexistentcommand"}
	var callbackCalled bool
	var capturedErr error

	execer := NewExecer(cmd, func(c *Command, line string, err error, done bool) {
		callbackCalled = true
		if err != nil {
			capturedErr = err
		}
	})

	execer.Start()

	if !callbackCalled {
		t.Error("expected callback to be called on error")
	}
	if capturedErr == nil {
		t.Error("expected error from invalid command")
	}
}
