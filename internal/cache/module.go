package cache

import "github.com/dop251/goja"

type ModuleCache struct {
	modules map[string]*goja.Program
}

func (c *ModuleCache) Add(name string, program *goja.Program) {
	c.modules[name] = program
}

func (c *ModuleCache) Get(name string) (*goja.Program, bool) {
	program, exists := c.modules[name]
	return program, exists
}

func (c *ModuleCache) Remove(name string) {
	delete(c.modules, name)
}

func (c *ModuleCache) Clear() {
	c.modules = make(map[string]*goja.Program)
}
