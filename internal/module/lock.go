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
				var mutex sync.Mutex
				client = &LockClient{
					name:   &name,
					mutex:  &mutex,
					locked: new(bool),
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
	name   *string
	mutex  *sync.Mutex
	locked *bool
}

func (l *LockClient) tryLock() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if *l.locked {
		return false
	}
	*l.locked = true
	return true
}

func (l *LockClient) Lock(timeout int) error {
	for i := 0; i < timeout; i++ {
		if l.tryLock() {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	l.Unlock()
	return errors.New("acquire lock " + *l.name + " timeout")
}

func (l *LockClient) Unlock() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	*l.locked = false
}
