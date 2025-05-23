package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fsnotify/fsnotify"
)

type CommandWatcher struct {
	command     *Command
	onUpdate    func(*Command, string)
	watcher     *fsnotify.Watcher
	watchedDirs map[string]bool
	debouncer   *Debouncer
	ticker      *time.Ticker
}

func NewCommandWatcher(c *Command, onUpdate func(*Command, string)) *CommandWatcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	return &CommandWatcher{
		command:     c,
		onUpdate:    onUpdate,
		watcher:     watcher,
		watchedDirs: make(map[string]bool),
		debouncer:   NewDebouncer(3 * time.Second),
		ticker:      time.NewTicker(10 * time.Second),
	}
}

func (cw *CommandWatcher) Start() {
	cw.addWatchedFiles()

	defer cw.watcher.Close()
	defer cw.ticker.Stop()

	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			if !strings.HasSuffix(event.Name, "~") {
				cw.debouncer.Do(cw.command.Name, 200*time.Millisecond, func() {
					cw.onUpdate(cw.command, fmt.Sprintf("trigger: %s %s", event.Name, event.Op))
				})
			}
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			cw.onUpdate(cw.command, fmt.Sprintf("watcher error: %s", err.Error()))
		case <-cw.ticker.C:
			cw.addWatchedFiles()
		}
	}
}

func (cw *CommandWatcher) addWatchedFiles() {
	var basepath string
	var pattern string
	basepath, pattern = doublestar.SplitPattern(cw.command.Glob)

	fsys := os.DirFS(basepath)

	files, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

	for _, file := range files {
		abs := filepath.Join(basepath, file)
		dir := filepath.Dir(abs)

		if cw.watchedDirs[dir] {
			continue
		}

		err := cw.watcher.Add(dir)
		if err != nil {
			cw.onUpdate(cw.command, fmt.Sprintf("error watching %s: %s", dir, err.Error()))
		} else {
			cw.watchedDirs[dir] = true
		}
	}
}
