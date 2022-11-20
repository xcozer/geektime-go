package queue

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"
)

type ConcurrentBlockingQueue[T any] struct {
	mutex *sync.Mutex
	data []T
	// notFull chan struct{}
	// notEmpty chan struct{}
	maxSize int

	notEmptyCond *Cond
	notFullCond *Cond
}

func NewConcurrentBlockingQueue[T any](maxSize int) *ConcurrentBlockingQueue[T] {
	m := &sync.Mutex{}
	return &ConcurrentBlockingQueue[T]{
		data: make([]T, 0, maxSize),
		mutex: m,
		// notFull: make(chan struct{}, 1),
		// notEmpty: make(chan struct{}, 1),
		maxSize: maxSize,
		notFullCond: NewCond(m),
		notEmptyCond: NewCond(m),
	}
}


func (c *ConcurrentBlockingQueue[T]) EnQueue(ctx context.Context, data T) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	c.mutex.Lock()
	for c.isFull() {
		err := c.notFullCond.WaitWithTimeout(ctx)
		if err != nil {
			return err
		}
	}
	c.data = append(c.data, data)
	c.notEmptyCond.Broadcast()
	c.mutex.Unlock()
	// 没有人等 notEmpty 的信号，这一句就会阻塞住
	return nil
}

func (c *ConcurrentBlockingQueue[T]) DeQueue(ctx context.Context) (T, error) {
	if ctx.Err() != nil {
		var t T
		return t, ctx.Err()
	}
	c.mutex.Lock()
	for c.isEmpty() {
		err := c.notEmptyCond.WaitWithTimeout(ctx)
		if err != nil {
			var t T
			return t, err
		}
	}
	t := c.data[0]
	c.data = c.data[1:]
	c.notFullCond.Broadcast()
	c.mutex.Unlock()
	// 没有人等 notFull 的信号，这一句就会阻塞住
	return t, nil
}

func (c *ConcurrentBlockingQueue[T]) IsFull() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.isFull()
}

func (c *ConcurrentBlockingQueue[T]) isFull() bool {
	return len(c.data) == c.maxSize
}

func (c *ConcurrentBlockingQueue[T]) IsEmpty() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.isEmpty()
}


func (c *ConcurrentBlockingQueue[T]) isEmpty() bool {
	return len(c.data) == 0
}

func (c *ConcurrentBlockingQueue[T]) Len() uint64 {
	return uint64(len(c.data))
}


// func (c *ConcurrentBlockingQueue[T]) EnQueueV1(ctx context.Context, data T) error {
	// select {
	// case <- c.notFullCond.Wait():
	// 	case <- ctx.Done() :
	//
	// }

// 	c.notFullCond.Wait(timeout)
// }

// func (c *ConcurrentBlockingQueue[T]) DeQueueV1(ctx context.Context, data T) error {
// 	c.notFullCond.Signal()
// 	return nil
// }

// Conditional variable implementation that uses channels for notifications.
// Only supports .Broadcast() method, however supports timeout based Wait() calls
// unlike regular sync.Cond.
type Cond struct {
	L sync.Locker
	n unsafe.Pointer
}

func NewCond(l sync.Locker) *Cond {
	c := &Cond{L: l}
	n := make(chan struct{})
	c.n = unsafe.Pointer(&n)
	return c
}

// Waits for Broadcast calls. Similar to regular sync.Cond, this unlocks the underlying
// locker first, waits on changes and re-locks it before returning.
func (c *Cond) Wait() {
	n := c.NotifyChan()
	c.L.Unlock()
	<-n
	c.L.Lock()
}

// Same as Wait() call, but will only wait up to a given timeout.
func (c *Cond) WaitWithTimeout(ctx context.Context) error {
	n := c.NotifyChan()
	c.L.Unlock()
	select {
	case <-n:
		c.L.Lock()
		return nil
	case <- ctx.Done():
		c.L.Lock()
		return ctx.Err()
	}
}

// Returns a channel that can be used to wait for next Broadcast() call.
func (c *Cond) NotifyChan() <-chan struct{} {
	ptr := atomic.LoadPointer(&c.n)
	return *((*chan struct{})(ptr))
}

// Broadcast call notifies everyone that something has changed.
func (c *Cond) Broadcast() {
	n := make(chan struct{})
	ptrOld := atomic.SwapPointer(&c.n, unsafe.Pointer(&n))
	close(*(*chan struct{})(ptrOld))
}