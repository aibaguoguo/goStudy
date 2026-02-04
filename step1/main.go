package main

import (
	"fmt"
	"reflect"
	"step1/safeCounter"
	"sync"
	"time"
)

func main() {
	//safeCounter.TestSafeCounter()
	//testChan()
	//v1.Test()
	//v2.Test()
	//v3.Test()
	person := safeCounter.Person{Name: "peter", Sex: "男"}
	t := reflect.TypeOf(person)
	v := reflect.ValueOf(person)
	for i := 0; i < t.NumField(); i++ {
		fmt.Println(t.Field(i).Type, t.Field(i).Name, v.FieldByName(t.Field(i).Name), t.Field(i).Anonymous)
	}
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
