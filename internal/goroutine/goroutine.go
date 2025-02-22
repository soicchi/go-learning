package goroutine

import (
	"fmt"
	"sync"
	"time"
)



func Goroutine() {
	withNoBuffer()
}

func withNoBuffer() {
	// with no buffer
	ch := make(chan int)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ch <- 1
		time.Sleep(5 * time.Second)
		fmt.Println("completed sending")
	}()

	fmt.Println(<-ch)

	wg.Wait()
}
