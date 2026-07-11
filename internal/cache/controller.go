package cache

import (
	"database/sql"

	"cube/internal/model"
)

type ControllerCache struct {
	controllers map[string]*model.Source
	db          *sql.DB
}

func (c *ControllerCache) Get(name string) *model.Source {
	if source, exists := c.controllers[name]; exists {
		return source
	}

	source := &model.Source{}
	if err := c.db.QueryRow("select name, method from source where name = ? and type = 'controller' and active = true", name).Scan(&source.Name, &source.Method); err != nil {
		return nil
	}

	c.controllers[name] = source
	return source
}

func (c *ControllerCache) Remove(name string) {
	delete(c.controllers, name)
}
