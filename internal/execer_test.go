package internal

import (
	"strings"
	"testing"
	"time"
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

func TestExecerStopsOldProcessBeforeStart(t *testing.T) {
	cmd := &Command{Exec: "sleep 10"}

	var callbackCalls int
	var doneCalled bool

	execer := NewExecer(cmd, func(c *Command, line string, err error, done bool) {
		callbackCalls++
		if done {
			doneCalled = true
		}
	})

	// First long-running command
	go execer.Start()

	// Give the process a moment to start
	time.Sleep(100 * time.Millisecond)

	// Check that the process is running
	execer.mu.Lock()
	firstProc := execer.proc
	execer.mu.Unlock()

	if firstProc == nil || firstProc.Process == nil {
		t.Fatal("expected first process to be started")
	}

	// Trigger Start again, which should call Stop()
	go execer.Start()

	// Wait for the stop to take effect
	time.Sleep(100 * time.Millisecond)

	execer.mu.Lock()
	secondProc := execer.proc
	execer.mu.Unlock()

	if secondProc == nil || secondProc.Process == nil {
		t.Fatal("expected second process to be started")
	}

	if firstProc.Process == secondProc.Process {
		t.Error("expected second process to replace the first one")
	}

	if !doneCalled {
		t.Error("expected first callback to call done = true after stop")
	}

	if callbackCalls != 1 {
		t.Errorf("expected callback to be called once, got %d", callbackCalls)
	}
}

