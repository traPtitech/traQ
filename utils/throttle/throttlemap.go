package throttle

import (
	"sync"
	"time"

	"github.com/boz/go-throttle"
)

type Map[K comparable] struct {
	throttles map[K]*throttleEntry
	interval  time.Duration
	ttl       time.Duration
	mu        sync.Mutex
	callback  func(K)
}

type throttleEntry struct {
	driver        throttle.ThrottleDriver
	lastTriggered time.Time
}

// NewThrottleMap creates a new ThrottleMap.
// The callback function will be called with the key when the throttle is triggered.
// The interval is the time period in which the callback can be triggered.
// The ttl is the time to live for each key in the map, after which it will be removed.
func NewThrottleMap[K comparable](interval, ttl time.Duration, callback func(K)) *Map[K] {
	t := &Map[K]{
		throttles: make(map[K]*throttleEntry),
		interval:  interval,
		ttl:       ttl,
		callback:  callback,
	}
	go t.gcLoop()
	return t
}

// Trigger triggers the throttle for the given key.
// The callback will be scheduled to be called after the interval.
func (tm *Map[K]) Trigger(key K) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	entry, exists := tm.throttles[key]
	if !exists {
		entry = &throttleEntry{
			driver: throttle.ThrottleFunc(tm.interval, true, func() {
				tm.callback(key)
			}),
		}
		tm.throttles[key] = entry
	}
	entry.lastTriggered = time.Now()
	entry.driver.Trigger()
}

func (tm *Map[K]) Stop(key K) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if entry, exists := tm.throttles[key]; exists {
		entry.driver.Stop()
		delete(tm.throttles, key)
	}
}

func (tm *Map[K]) gcLoop() {
	for range time.Tick(tm.ttl / 2) {
		tm.mu.Lock()
		now := time.Now()
		for key, entry := range tm.throttles {
			if now.Sub(entry.lastTriggered) > tm.ttl {
				entry.driver.Stop()
				delete(tm.throttles, key)
			}
		}
		tm.mu.Unlock()
	}
}
