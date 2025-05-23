package internal

import (
	"sync"
	"time"
)

type Debouncer struct {
	mu          sync.Mutex
	timers      map[string]*time.Timer
	lastTrigger map[string]time.Time
	cooldown    time.Duration
}

func NewDebouncer(cooldown time.Duration) *Debouncer {
	return &Debouncer{
		timers:      make(map[string]*time.Timer),
		lastTrigger: make(map[string]time.Time),
		cooldown:    cooldown,
	}
}

func (d *Debouncer) Do(key string, delay time.Duration, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if t, ok := d.timers[key]; ok {
		t.Stop()
	}

	d.timers[key] = time.AfterFunc(delay, func() {
		now := time.Now()

		d.mu.Lock()
		last := d.lastTrigger[key]
		if now.Sub(last) < d.cooldown {
			d.mu.Unlock()
			return
		}

		d.lastTrigger[key] = now
		delete(d.timers, key)
		d.mu.Unlock()

		fn()
	})
}
