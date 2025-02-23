package goroutine

import (
	"fmt"
	"sync"
	"time"
)

func Goroutine() {
	// withNoBuffer()
	// withBuffer()
	deadlock()
}

func withNoBuffer() {
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

func withBuffer() {
	ch := make(chan int, 3)
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := 0; i < 3; i++ {
			ch <- 1
			fmt.Println("sent")
			time.Sleep(1 * time.Second)
		}

		close(ch)
	}()

	for v := range ch {
		fmt.Println(v)
		fmt.Println("received")
	}

	wg.Wait()
}

func deadlock() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		v := 1
		ch1 <- v
		v2 := <-ch2
		fmt.Println(v, v2)
	}()

	v := 2
	ch2 <- v
	v1 := <-ch1
	fmt.Println(v, v1)
}
