package main

import (
	"fmt"
	"mpool"
	"sync"
	"sync/atomic"
	"time"
)

var sum int32

func demoFunc() {
	time.Sleep(10 * time.Millisecond)
	fmt.Println("Hello World!")
}

func myFunc(i interface{}) {
	n := i.(int32)
	atomic.AddInt32(&sum, n)
	fmt.Printf("run with %d\n", n)
}

func main() {
	// defer mpool.Release()

	runTimes := 1000

	// Use the common pool.
	var wg sync.WaitGroup
	// syncCalculateSum := func() {
	// 	demoFunc()
	// 	wg.Done()
	// }
	// for i := 0; i < runTimes; i++ {
	// 	wg.Add(1)
	// 	_ = mpool.Submit(syncCalculateSum)
	// }
	// wg.Wait()
	// fmt.Printf("running goroutines: %d\n", mpool.Running())
	// fmt.Printf("finish all tasks.\n")

	// Use the pool with a method,
	// set 10 to the capacity of goroutine pool and 1 second for expired duration.
	p, _ := mpool.NewPoolWithFunc(10, func(i interface{}) {
		myFunc(i)
		wg.Done()
	}, mpool.WithPreAlloc(true))
	defer p.Release()
	// Submit tasks one by one.
	for i := 0; i < runTimes; i++ {
		wg.Add(1)
		_ = p.Invoke(int32(i))
	}
	wg.Wait()
	fmt.Printf("running goroutines: %d\n", p.Running())
	fmt.Printf("finish all tasks, result is %d\n", sum)
	if sum != 499500 {
		panic("the final result is wrong!!!")
	}
}
