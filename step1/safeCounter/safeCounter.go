package safeCounter

import (
	"fmt"
	"sync"
)

type SafeCounter struct {
	Count int
	lock  sync.Mutex
}

func (s *SafeCounter) Incr() {
	s.lock.Lock()
	s.Count++
	defer s.lock.Unlock()
}

// 程序计数器
func TestSafeCounter() {
	s := &SafeCounter{}
	wg := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			s.Incr()
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println(s.Count)
}
