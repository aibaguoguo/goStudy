package main

import (
	"fmt"
	v3 "step1/mq/v3"
	"sync"
	"time"
)

func main() {
	//safeCounter.TestSafeCounter()
	//testChan()
	//v1.Test()
	//v2.Test()
	v3.Test()
}

func testChan() {
	ch := make(chan int, 5)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ch)
		for i := 0; i < 10; i++ {
			ch <- i
			fmt.Println("生产者写入数据", i)
			time.Sleep(time.Second)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 10)
		for i := 0; i < 10; i++ {
			r := <-ch
			fmt.Println("消费者消费数据", r)
		}
	}()
	wg.Wait()
}
