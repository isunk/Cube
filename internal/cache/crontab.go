package cache

import (
	"sync"

	"github.com/robfig/cron/v3"
)

type CrontabCache struct {
	sync.RWMutex
	crontabs map[string]cron.EntryID
}

func (c *CrontabCache) Add(name string, id cron.EntryID) {
	c.Lock()
	defer c.Unlock()
	c.crontabs[name] = id
}

// GetOrAdd 原子地检查并添加：已存在返回 false，否则添加并返回 true。
func (c *CrontabCache) GetOrAdd(name string, id cron.EntryID) bool {
	c.Lock()
	defer c.Unlock()
	if _, exists := c.crontabs[name]; exists {
		return false
	}
	c.crontabs[name] = id
	return true
}

func (c *CrontabCache) Get(name string) (cron.EntryID, bool) {
	c.RLock()
	defer c.RUnlock()
	id, exists := c.crontabs[name]
	return id, exists
}

func (c *CrontabCache) Remove(name string) {
	c.Lock()
	defer c.Unlock()
	delete(c.crontabs, name)
}
