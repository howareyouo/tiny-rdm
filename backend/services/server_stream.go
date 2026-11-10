package services

import (
	"context"
	"sync"
	"time"
)

// serverStream batches streamed items and emits them as a single event
// shared by the monitor and pubsub modules
type serverStream struct {
	ctx       context.Context
	eventName string
	mutex     sync.Mutex
	cache     []any
	limit     int      // emit once the cache reaches this size
	interval  time.Duration
	closeCh   chan struct{}
	closeOnce sync.Once
}

func newServerStream(ctx context.Context, eventName string, limit int, interval time.Duration) *serverStream {
	s := &serverStream{
		ctx:       ctx,
		eventName: eventName,
		cache:     make([]any, 0, limit),
		limit:     limit,
		interval:  interval,
		closeCh:   make(chan struct{}),
	}
	go s.loop()
	return s
}

// push appends an item and emits the cache at once when it is full
func (s *serverStream) push(item any) {
	s.mutex.Lock()
	s.cache = append(s.cache, item)
	var out []any
	if len(s.cache) >= s.limit {
		out = s.takeLocked()
	}
	s.mutex.Unlock()
	if out != nil {
		EventsEmit(s.ctx, s.eventName, out)
	}
}

// take swaps out all cached items
func (s *serverStream) take() []any {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.takeLocked()
}

func (s *serverStream) takeLocked() []any {
	cache := s.cache
	// never reuse the swapped out slice, it is emitted outside the lock
	s.cache = make([]any, 0, s.limit)
	return cache
}

func (s *serverStream) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if out := s.take(); len(out) > 0 {
				EventsEmit(s.ctx, s.eventName, out)
			}
		case <-s.closeCh:
			return
		}
	}
}

// stop stops the stream loop, redisStop closes the underlying redis subscription
func (s *serverStream) stop(redisStop func()) {
	s.closeOnce.Do(func() {
		if redisStop != nil {
			redisStop()
		}
		close(s.closeCh)
	})
}
