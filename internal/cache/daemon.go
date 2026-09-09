package cache

import (
	"sync"

	"github.com/dop251/goja"
)

// Worker 需要从 internal 包导入，但由于循环依赖问题，这里使用 interface
type Worker interface {
	Interrupt(reason string)
	Reset()
	Id() int
	Runtime() *goja.Runtime
	Run(params ...goja.Value) (goja.Value, error)
}

type DaemonCache struct {
	sync.RWMutex
	daemons map[string]Worker
}

func (c *DaemonCache) Add(name string, worker Worker) {
	c.Lock()
	defer c.Unlock()
	c.daemons[name] = worker
}

// GetOrAdd 原子地检查并添加：已存在返回已存在的 worker 与 false，否则添加并返回该 worker 与 true，消除 check-then-add 的 TOCTOU 竞态。
func (c *DaemonCache) GetOrAdd(name string, worker Worker) (Worker, bool) {
	c.Lock()
	defer c.Unlock()
	if existing, exists := c.daemons[name]; exists {
		return existing, false
	}
	c.daemons[name] = worker
	return worker, true
}

func (c *DaemonCache) Get(name string) (Worker, bool) {
	c.RLock()
	defer c.RUnlock()
	worker, exists := c.daemons[name]
	return worker, exists
}

func (c *DaemonCache) Remove(name string) {
	c.Lock()
	defer c.Unlock()
	delete(c.daemons, name)
}
