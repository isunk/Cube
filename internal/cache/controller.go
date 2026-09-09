package cache

import (
	"database/sql"
	"sync"

	"cube/internal/model"
)

type ControllerCache struct {
	sync.RWMutex
	controllers map[string]*model.Source
	db          *sql.DB
}

func (c *ControllerCache) Get(name string) *model.Source {
	c.RLock()
	if source, exists := c.controllers[name]; exists {
		c.RUnlock()
		return source
	}
	c.RUnlock()

	source := &model.Source{}
	if err := c.db.QueryRow("select name, method from source where name = ? and type = 'controller' and active = true", name).Scan(&source.Name, &source.Method); err != nil {
		return nil
	}

	c.Lock()
	c.controllers[name] = source
	c.Unlock()
	return source
}

func (c *ControllerCache) Remove(name string) {
	c.Lock()
	defer c.Unlock()
	delete(c.controllers, name)
}
