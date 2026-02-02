package safeCounter

import "sync"

type SafeCounter struct {
	Count int
	lock  sync.Mutex
}

func (s *SafeCounter) Incr() {
	s.lock.Lock()
	s.Count++
	defer s.lock.Unlock()
}
