package main

import (
	"fmt"
	"step1/safeCounter"
	"sync"
)

func main() {

}

// 程序计数器
func testSafeCounter() {
	s := &safeCounter.SafeCounter{}
	wg := sync.WaitGroup{}
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go func() {
			s.Incr()
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println(s.Count)
}
