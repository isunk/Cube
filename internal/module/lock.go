package module

import (
	"errors"
	"sync"
	"time"
)

func init() {
	register("lock", func(ctx Context) interface{} {
		return func(name string) *LockClient {
			locks.Lock()
			defer locks.Unlock()
			if locks.clients == nil {
				locks.clients = make(map[string]*LockClient)
			}
			client := locks.clients[name]
			if client == nil {
				client = &LockClient{
					name: name,
					ch:   make(chan struct{}, 1),
				}
				locks.clients[name] = client
			}
			ctx.Worker.AddDefer(func() {
				client.Unlock()
			})
			return client
		}
	})
}

var locks struct {
	sync.Mutex
	clients map[string]*LockClient
}

type LockClient struct {
	name string
	ch   chan struct{}
}

func (l *LockClient) Lock(timeout int) error {
	// timeout <= 0 表示非阻塞尝试，获取不到立即返回失败；timeout > 0 表示最多等待 timeout 毫秒
	if timeout <= 0 {
		select {
		case l.ch <- struct{}{}:
			return nil
		default:
			return errors.New("acquire lock " + l.name + " timed out")
		}
	}
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()
	select {
	case l.ch <- struct{}{}:
		return nil
	case <-timer.C:
		// 超时表示始终未成功获取锁，不能调用 Unlock（会错误释放其他 goroutine 持有的锁）
		return errors.New("acquire lock " + l.name + " timed out")
	}
}

func (l *LockClient) Unlock() {
	select {
	case <-l.ch:
	default:
	}
}
