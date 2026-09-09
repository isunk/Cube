package cache

import (
	"sync"

	"github.com/dop251/goja"
)

type ModuleCache struct {
	sync.RWMutex
	modules map[string]*goja.Program
}

func (c *ModuleCache) Add(name string, program *goja.Program) {
	c.Lock()
	defer c.Unlock()
	c.modules[name] = program
}

func (c *ModuleCache) Get(name string) (*goja.Program, bool) {
	c.RLock()
	defer c.RUnlock()
	program, exists := c.modules[name]
	return program, exists
}

func (c *ModuleCache) Remove(name string) {
	c.Lock()
	defer c.Unlock()
	delete(c.modules, name)
}

func (c *ModuleCache) Clear() {
	c.Lock()
	defer c.Unlock()
	c.modules = make(map[string]*goja.Program)
}
