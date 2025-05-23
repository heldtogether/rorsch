package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCommandWatcherTriggersOnFileChange(t *testing.T) {
	// Create temp directory + file
	dir := t.TempDir()
	file := filepath.Join(dir, "watched.txt")
	if err := os.WriteFile(file, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Set up command and capture output
	cmd := &Command{
		Name: "testcmd",
		Glob: filepath.ToSlash(filepath.Join(dir, "*.txt")),
	}
	updates := make(chan string, 10)

	cw := NewCommandWatcher(cmd, func(c *Command, msg string) {
		updates <- msg
	})

	// Speed up test: replace ticker with fast one
	cw.ticker.Stop()
	cw.ticker = time.NewTicker(100 * time.Millisecond)
	cw.debouncer = NewDebouncer(100 * time.Millisecond)

	go cw.Start()

	// Wait for watcher to register (and ticker to populate watched dirs)
	time.Sleep(200 * time.Millisecond)

	// Touch the file to trigger change
	if err := os.WriteFile(file, []byte(time.Now().String()), 0644); err != nil {
		t.Fatalf("failed to update file: %v", err)
	}

	// Wait for debounce and callback
	select {
	case msg := <-updates:
		if !strings.Contains(msg, "trigger:") {
			if !strings.Contains(msg, "watched.txt") {
				t.Logf("ignoring unrelated update: %s", msg)
				return
			}
			t.Errorf("unexpected update message: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive expected update")
	}
}
