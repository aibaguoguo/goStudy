package v1

import (
	"fmt"
	"sync"
	"time"
)

// 生产者函数：向通道写入数据
// wg：标记生产者执行完成
// dataChan：数据缓冲区（只写通道，提高安全性）
// producerName：生产者名称，便于日志区分
func producer(wg *sync.WaitGroup, dataChan chan<- string, producerName string) {
	defer wg.Done() // 生产者退出时，通知WaitGroup计数-1

	// 模拟生成5条数据
	for i := 1; i <= 5; i++ {
		data := fmt.Sprintf("%s-数据-%d", producerName, i)
		dataChan <- data // 向通道写入数据，缓冲区未满时不阻塞
		fmt.Printf("[生产] %s 已写入：%s\n", producerName, data)
		time.Sleep(200 * time.Millisecond) // 模拟生产耗时
	}
}

// 消费者函数：从通道读取并处理数据
// wg：标记消费者执行完成
// dataChan：数据缓冲区（只读通道，提高安全性）
// consumerName：消费者名称，便于日志区分
func consumer(wg *sync.WaitGroup, dataChan <-chan string, consumerName string) {
	defer wg.Done() // 消费者退出时，通知WaitGroup计数-1

	// for range 遍历通道：自动读取数据，直到通道关闭且无剩余数据
	for data := range dataChan {
		fmt.Printf("[消费] %s 已处理：%s\n", consumerName, data)
		time.Sleep(500 * time.Millisecond) // 模拟处理耗时
	}
}

func Test() {
	// 1. 创建有缓冲通道，容量10（可根据产销速度调整）
	dataChan := make(chan string, 10)

	// 2. 定义两个WaitGroup，分别管理生产者和消费者
	var producerWg sync.WaitGroup
	var consumerWg sync.WaitGroup

	// 3. 启动2个生产者
	producerCount := 2
	producerWg.Add(producerCount)
	for i := 1; i <= producerCount; i++ {
		go producer(&producerWg, dataChan, fmt.Sprintf("生产者-%d", i))
	}

	// 4. 启动3个消费者
	consumerCount := 3
	consumerWg.Add(consumerCount)
	for i := 1; i <= consumerCount; i++ {
		go consumer(&consumerWg, dataChan, fmt.Sprintf("消费者-%d", i))
	}

	// 5. 单独启动goroutine，等待所有生产者完成后关闭通道（关键！）
	// 不能在主goroutine直接等待，否则会阻塞消费者启动
	go func() {
		producerWg.Wait() // 阻塞直到所有生产者完成写入
		close(dataChan)   // 关闭通道，通知消费者无新数据
		fmt.Println("=== 所有生产者已完成，通道已关闭 ===")
	}()

	// 6. 等待所有消费者完成数据处理
	consumerWg.Wait()
	fmt.Println("=== 所有消费者已完成，程序退出 ===")
}
