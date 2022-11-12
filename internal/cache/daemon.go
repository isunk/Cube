package cache

import "github.com/dop251/goja"

// Worker 需要从 internal 包导入，但由于循环依赖问题，这里使用 interface
// 实际使用时，需要确保传入的是正确的 Worker 类型
type Worker interface {
	Interrupt(reason string)
	Reset()
	Id() int
	Runtime() *goja.Runtime
	Run(params ...goja.Value) (goja.Value, error)
}

type DaemonCache struct {
	daemons map[string]Worker
}

func (c *DaemonCache) Add(name string, worker Worker) {
	c.daemons[name] = worker
}

func (c *DaemonCache) Get(name string) (Worker, bool) {
	worker, exists := c.daemons[name]
	return worker, exists
}

func (c *DaemonCache) Remove(name string) {
	delete(c.daemons, name)
}
