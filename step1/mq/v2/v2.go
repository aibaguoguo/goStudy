package v2

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// 进阶生产者：支持优雅退出，非阻塞写入
func advancedProducer(ctx context.Context, wg *sync.WaitGroup, dataChan chan<- string, producerName string) {
	defer wg.Done()

	i := 1
	for {
		// 优先判断退出信号，收到则直接退出
		select {
		case <-ctx.Done():
			fmt.Printf("[生产] %s 收到退出信号，停止生产\n", producerName)
			return
		default:
			data := fmt.Sprintf("%s-高级数据-%d", producerName, i)
			// 非阻塞写入：尝试写入通道，同时监听退出信号
			select {
			case dataChan <- data:
				fmt.Printf("[生产] %s 已写入：%s\n", producerName, data)
				i++
				time.Sleep(200 * time.Millisecond)
			case <-ctx.Done():
				fmt.Printf("[生产] %s 写入时收到退出信号，停止生产\n", producerName)
				return
			}
		}

		// 模拟生产10条数据后自动停止
		if i > 10 {
			fmt.Printf("[生产] %s 已完成10条数据生产，停止\n", producerName)
			return
		}
	}
}

// 进阶消费者：支持处理重试，优雅退出
func advancedConsumer(ctx context.Context, wg *sync.WaitGroup, dataChan <-chan string, consumerName string) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[消费] %s 收到退出信号，停止消费\n", consumerName)
			return
		case data, ok := <-dataChan: // 手动判断通道是否关闭
			if !ok {
				fmt.Printf("[消费] %s 通道已关闭，停止消费\n", consumerName)
				return
			}

			// 模拟最多3次重试
			processSuccess := false
			for retry := 1; retry <= 3; retry++ {
				fmt.Printf("[消费] %s 第%d次处理：%s\n", consumerName, retry, data)
				time.Sleep(300 * time.Millisecond)

				// 模拟处理成功（实际业务中替换为真实逻辑）
				processSuccess = true
				break
			}

			if processSuccess {
				fmt.Printf("[消费] %s 处理成功：%s\n", consumerName, data)
			} else {
				fmt.Printf("[消费] %s 处理失败：%s（已重试3次）\n", consumerName, data)
			}
		}
	}
}

func Test() {
	// 1. 创建有缓冲通道，容量15
	dataChan := make(chan string, 15)

	// 2. 创建上下文，5秒后自动退出（支持优雅关闭）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 确保上下文最终关闭

	// 3. 定义WaitGroup
	var producerWg sync.WaitGroup
	var consumerWg sync.WaitGroup

	// 4. 启动2个进阶生产者
	producerCount := 2
	producerWg.Add(producerCount)
	for i := 1; i <= producerCount; i++ {
		go advancedProducer(ctx, &producerWg, dataChan, fmt.Sprintf("生产者-%d", i))
	}

	// 5. 启动3个进阶消费者
	consumerCount := 3
	consumerWg.Add(consumerCount)
	for i := 1; i <= consumerCount; i++ {
		go advancedConsumer(ctx, &consumerWg, dataChan, fmt.Sprintf("消费者-%d", i))
	}

	// 6. 等待生产者完成，关闭通道
	go func() {
		producerWg.Wait()
		close(dataChan)
		fmt.Println("=== 所有高级生产者已完成，通道已关闭 ===")
	}()

	// 7. 等待消费者完成
	consumerWg.Wait()
	fmt.Println("=== 所有高级消费者已完成，程序退出 ===")
}
