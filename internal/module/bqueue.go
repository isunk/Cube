package module

import (
	"context"
	"errors"
	"sync"
	"time"
)

func init() {
	register("bqueue", func(ctx Context) interface{} {
		return func(size int) *BlockingQueueClient {
			return &BlockingQueueClient{
				queue: make(chan interface{}, size),
			}
		}
	})
}

type BlockingQueueClient struct {
	queue chan interface{}
	sync.Mutex
}

func (b *BlockingQueueClient) Put(input interface{}, timeout int) error {
	b.Lock()
	defer b.Unlock()
	select {
	case b.queue <- input:
		return nil
	case <-time.After(time.Duration(timeout) * time.Millisecond): // 队列入列最大超时时间为 timeout 毫秒
		return errors.New("blocking queue is full, put timed out")
	}
}

func (b *BlockingQueueClient) Poll(timeout int) (interface{}, error) {
	b.Lock()
	defer b.Unlock()
	select {
	case output := <-b.queue:
		return output, nil
	case <-time.After(time.Duration(timeout) * time.Millisecond): // 队列出列最大超时时间为 timeout 毫秒
		return nil, errors.New("blocking queue is empty, poll timed out")
	}
}

func (b *BlockingQueueClient) Drain(size int, timeout int) (output []interface{}) {
	b.Lock()
	defer b.Unlock()
	output = make([]interface{}, 0, size) // 创建切片，初始大小为 0，最大为 size

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond) // 使用 context 实现超时控制（替代原 timer 通道方案，避免 goroutine 泄漏）
	defer cancel()
	for i := 0; i < size; i++ {
		select {
		case val, ok := <-b.queue:
			if ok {
				output = append(output, val)
			} else {
				return
			}
		case <-ctx.Done():
			return
		}
	}
	return
}
