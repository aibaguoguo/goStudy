package v3

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// 1. 消息结构体：封装消息内容和所属Topic
type Message struct {
	Topic string      // 消息所属主题
	Data  interface{} // 消息内容（用interface{}支持任意类型数据）
}

// 2. 主题管理器：管理所有Topic和对应的消息通道，处理订阅/发送逻辑
type TopicManager struct {
	mu         sync.RWMutex            // 读写锁，保证并发安全（读多写少场景高效）
	topicChans map[string]chan Message // Topic -> 对应消息通道的映射
	bufferSize int                     // 每个Topic通道的缓冲区大小
	isClosed   bool                    // 管理器是否已关闭
}

// 新建主题管理器（指定每个Topic通道的默认缓冲区大小）
func NewTopicManager(bufferSize int) *TopicManager {
	return &TopicManager{
		topicChans: make(map[string]chan Message),
		bufferSize: bufferSize,
		isClosed:   false,
	}
}

// 内部方法：获取或创建某个Topic对应的通道（保证每个Topic只有一个通道）
func (tm *TopicManager) getOrCreateTopicChan(topic string, msg string) chan Message {
	// 先加读锁，查询是否存在该Topic的通道
	tm.mu.RLock()
	ch, exists := tm.topicChans[topic]
	tm.mu.RUnlock()

	// 若存在，直接返回
	if exists {
		return ch
	}

	// 若不存在，加写锁，创建新通道（双重检查，避免重复创建）
	tm.mu.Lock()
	defer tm.mu.Unlock()
	ch, exists = tm.topicChans[topic]
	if !exists {
		ch = make(chan Message, tm.bufferSize)
		tm.topicChans[topic] = ch
		fmt.Printf("【Topic管理器】[%s]创建新Topic：%s，通道缓冲区大小：%d\n", msg, topic, tm.bufferSize)
	}
	return ch
}

// 发送消息到指定Topic（生产者调用）
func (tm *TopicManager) SendMessage(topic string, data interface{}) error {
	// 检查管理器是否已关闭  会导致并发死锁问题
	//tm.mu.RLock()
	//defer tm.mu.RUnlock()
	//if tm.isClosed {
	//	return fmt.Errorf("Topic管理器已关闭，无法发送消息到Topic：%s", topic)
	//}

	tm.mu.RLock()
	isClosed := tm.isClosed // 拷贝到局部变量，避免后续再次加锁
	tm.mu.RUnlock()         // 立即释放读锁，不持有到后续操作

	//检查是否关闭
	if isClosed {
		return fmt.Errorf("Topic管理器已关闭，无法发送消息到Topic：%s", topic)
	}

	// 获取或创建Topic通道，写入消息
	ch := tm.getOrCreateTopicChan(topic, "生产者")
	msg := Message{
		Topic: topic,
		Data:  data,
	}

	// 写入通道（缓冲区满时阻塞，若需非阻塞可使用select）
	ch <- msg
	fmt.Printf("【发送消息】Topic：%s，内容：%v\n", topic, data)
	return nil

	//select {
	//case ch <- msg:
	//	fmt.Printf("【发送消息】Topic：%s，内容：%v\n", topic, data)
	//default:
	//	return fmt.Errorf("Topic：%s 通道缓冲区已满，无法发送消息", topic)
	//}
	//return nil
}

// 订阅指定Topic（消费者调用），返回只读通道和取消订阅函数
func (tm *TopicManager) Subscribe(topic string) (<-chan Message, error) {
	// 检查管理器是否已关闭
	//tm.mu.RLock()
	//defer tm.mu.RUnlock()
	//if tm.isClosed {
	//	return nil, fmt.Errorf("Topic管理器已关闭，无法订阅Topic：%s", topic)
	//}

	tm.mu.RLock()
	isClosed := tm.isClosed // 拷贝到局部变量，避免后续再次加锁
	tm.mu.RUnlock()         // 立即释放读锁，不持有到后续操作

	//检查是否关闭
	if isClosed {
		return nil, fmt.Errorf("Topic管理器已关闭，无法订阅Topic：%s", topic)
	}

	// 获取或创建Topic通道，返回只读副本（避免消费者误写通道）
	ch := tm.getOrCreateTopicChan(topic, "消费者")
	fmt.Printf("【订阅成功】消费者订阅Topic：%s\n", topic)
	return ch, nil
}

// 关闭主题管理器，关闭所有Topic通道，禁止后续发送/订阅
func (tm *TopicManager) Close() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.isClosed {
		return
	}

	// 标记管理器为关闭状态
	tm.isClosed = true
	fmt.Println("【Topic管理器】开始关闭所有Topic通道...")

	// 关闭所有Topic的通道
	for topic, ch := range tm.topicChans {
		close(ch)
		fmt.Printf("【Topic管理器】关闭Topic：%s 的通道\n", topic)
	}

	// 清空Topic映射
	tm.topicChans = make(map[string]chan Message)
	fmt.Println("【Topic管理器】所有资源已清理完成")
}

// 3. 消费者封装：简化多Topic订阅和消息处理
// 消费者函数类型：定义消费者处理消息的逻辑
type ConsumerHandler func(msg Message)

// 启动消费者，订阅多个Topic，使用指定的处理函数处理消息
func StartConsumer(ctx context.Context, wg *sync.WaitGroup, tm *TopicManager, topics []string, handler ConsumerHandler) {
	defer wg.Done()
	// 为每个订阅的Topic启动独立的读取goroutine
	var topicWg sync.WaitGroup
	for _, topic := range topics {
		topicWg.Add(1)
		go func(t string) {
			defer topicWg.Done()
			// 订阅该Topic
			ch, err := tm.Subscribe(t)
			if err != nil {
				fmt.Printf("【消费者错误】无法订阅Topic：%s，错误：%v\n", t, err)
				return
			}

			// 循环读取消息，直到通道关闭或收到退出信号
			for {
				select {
				case <-ctx.Done():
					fmt.Printf("【消费者】收到退出信号，停止消费Topic：%s\n", t)
					return
				case msg, ok := <-ch:
					if !ok {
						fmt.Printf("【消费者】Topic：%s 的通道已关闭，停止消费\n", t)
						return
					}
					// 调用处理函数处理消息
					handler(msg)
				}
			}
		}(topic)
	}

	// 等待所有Topic的消费goroutine完成
	topicWg.Wait()
	fmt.Println("【消费者】所有订阅的Topic已停止消费，退出")
}

// 4. 测试主函数
func Test() {
	// 1. 创建Topic管理器，每个Topic通道缓冲区大小为10
	tm := NewTopicManager(10)
	defer tm.Close() // 程序退出时关闭管理器

	// 2. 创建上下文，支持8秒后自动退出（优雅关闭）
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// 3. 定义WaitGroup，等待所有消费者完成
	var wg sync.WaitGroup

	// 4. 启动消费者1：订阅「topic1」（处理数字消息）
	wg.Add(1)
	consumer1Handler := func(msg Message) {
		data := msg.Data.(int)
		fmt.Printf("【消费者1处理】Topic：%s，内容：%d，处理结果：%d（×2）\n", msg.Topic, data, data*2)
		time.Sleep(300 * time.Millisecond) // 模拟处理耗时
	}
	go StartConsumer(ctx, &wg, tm, []string{"topic1"}, consumer1Handler)

	// 5. 启动消费者2：订阅「topic2」（处理字符串消息）
	wg.Add(1)
	consumer2Handler := func(msg Message) {
		data := msg.Data.(string)
		fmt.Printf("【消费者2处理】Topic：%s，内容：%s，处理结果：%s（已归档）\n", msg.Topic, data, data)
		time.Sleep(500 * time.Millisecond) // 模拟处理耗时
	}
	go StartConsumer(ctx, &wg, tm, []string{"topic2"}, consumer2Handler)

	// 6. 启动消费者3：订阅「topic1」和「topic2」（广播消费，接收两个Topic的消息）
	wg.Add(1)
	consumer3Handler := func(msg Message) {
		fmt.Printf("【消费者3（广播）处理】Topic：%s，内容：%v，处理结果：已记录日志\n", msg.Topic, msg.Data)
		time.Sleep(200 * time.Millisecond) // 模拟处理耗时
	}
	go StartConsumer(ctx, &wg, tm, []string{"topic1", "topic2"}, consumer3Handler)

	// 7. 生产者1：向「topic1」发送数字消息（1-10）
	go func() {
		for i := 1; i <= 10; i++ {
			err := tm.SendMessage("topic1", i)
			if err != nil {
				fmt.Printf("【生产者1错误】无法发送消息，错误：%v\n", err)
				return
			}
			time.Sleep(200 * time.Millisecond) // 模拟生产耗时
		}
		fmt.Println("【生产者1】已发送所有topic1消息")
	}()

	// 8. 生产者2：向「topic2」发送字符串消息（hello1-hello5）
	go func() {
		for i := 1; i <= 5; i++ {
			data := fmt.Sprintf("hello%d", i)
			err := tm.SendMessage("topic2", data)
			if err != nil {
				fmt.Printf("【生产者2错误】无法发送消息，错误：%v\n", err)
				return
			}
			time.Sleep(400 * time.Millisecond) // 模拟生产耗时
		}
		fmt.Println("【生产者2】已发送所有topic2消息")
	}()

	// 9. 等待所有消费者完成
	wg.Wait()
	fmt.Println("【主程序】所有消费者已退出，程序结束")
}
