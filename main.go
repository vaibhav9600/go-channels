package main

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Channel[M any] struct {
	queue    *list.List
	cond     *sync.Cond
	closed   bool
	capacity int
}

func NewChannel[M any](capacity int) *Channel[M] {
	return &Channel[M]{
		queue:    list.New(),
		capacity: capacity,
		closed:   false,
		cond:     sync.NewCond(&sync.Mutex{}),
	}
}

func (c *Channel[M]) Send(message M) error {
	// Acquire exclusive access to channel
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	// If channel is closed, return an error
	if c.closed {
		return errors.New("send on closed channel")
	}
	// If channel is Full, wait for the channel to have space
	for c.queue.Len() == c.capacity {
		c.cond.Wait()
	}
	// Add the data to the queue
	c.queue.PushBack(message)
	// Notify all other goroutines to wake up
	c.cond.Broadcast()
	return nil
}

func (c *Channel[M]) Receive() (M, bool) {
	// only one go routine can access the channel at a time
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	// if channel is closed we return default value of channel and bool
	if c.closed {
		var zero M
		return zero, false
	}

	// Increase the capacity of the channel by 1
	// Notify all other go routines to wake up
	c.capacity++
	c.cond.Broadcast()
	// if channel is empty we need to make receiver wait
	for c.queue.Len() == 0 {
		c.cond.Wait()
	}
	// if channel is not empty, we can successfully receive data, remove data from queue
	message := c.queue.Remove(c.queue.Front()).(M)
	// Decrease the capacity of the channel
	c.capacity--
	c.cond.Broadcast()
	// After receiving data broadcast other go routines to wake up
	return message, true
}

func (c *Channel[M]) Close() error {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	if c.closed {
		return errors.New("channel already closed")
	}

	c.closed = true
	c.cond.Broadcast()
	return nil
}

func main() {
	ch := NewChannel[int](0)

	go func() {
		for i := 1; i <= 5; i++ {
			err := ch.Send(i)
			if err != nil {
				fmt.Printf("send error: %v\n", err)
				return
			}

			fmt.Printf("Send: %d\n", i)
		}
		// time.Sleep(2 * time.Second)
		ch.Close()
	}()

	for {
		val, ok := ch.Receive()
		if !ok {
			fmt.Println("Channel Closed")
			break
		}
		fmt.Printf("Received: %d\n", val)
	}

	time.Sleep(5 * time.Second)
}
