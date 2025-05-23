package internal

import (
	"sync"
	"testing"
	"time"
)

func TestDebouncerSingleCall(t *testing.T) {
	var called int
	d := NewDebouncer(100 * time.Millisecond)

	done := make(chan struct{})
	d.Do("task1", 50*time.Millisecond, func() {
		called++
		close(done)
	})

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("function was not called within expected time")
	}

	if called != 1 {
		t.Errorf("expected function to be called once, got %d", called)
	}
}

func TestDebouncerBouncesRapidCalls(t *testing.T) {
	var called int
	d := NewDebouncer(100 * time.Millisecond)

	done := make(chan struct{})
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		d.Do("task1", 50*time.Millisecond, func() {
			mu.Lock()
			called++
			mu.Unlock()
			close(done)
		})
		time.Sleep(10 * time.Millisecond) // Rapid re-trigger
	}

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("function was not called within expected time")
	}

	time.Sleep(100 * time.Millisecond) // Ensure no additional calls

	mu.Lock()
	defer mu.Unlock()
	if called != 1 {
		t.Errorf("expected function to be called once due to debouncing, got %d", called)
	}
}

func TestDebouncerCooldownPreventsImmediateRepeat(t *testing.T) {
	var called int
	d := NewDebouncer(200 * time.Millisecond)

	done := make(chan struct{}, 2)
	var mu sync.Mutex

	callFn := func() {
		mu.Lock()
		called++
		mu.Unlock()
		done <- struct{}{}
	}

	// Trigger first call
	d.Do("task1", 10*time.Millisecond, callFn)
	<-done

	// Immediate second call should be skipped due to cooldown
	d.Do("task1", 10*time.Millisecond, callFn)

	select {
	case <-done:
		t.Error("cooldown did not prevent second call")
	case <-time.After(150 * time.Millisecond):
		// Expected timeout, no call should happen
	}

	time.Sleep(100 * time.Millisecond)

	// Now after cooldown, it should be allowed again
	d.Do("task1", 10*time.Millisecond, callFn)
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected second call after cooldown")
	}

	mu.Lock()
	defer mu.Unlock()
	if called != 2 {
		t.Errorf("expected function to be called twice, got %d", called)
	}
}
