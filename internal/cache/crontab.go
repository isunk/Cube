package cache

import "github.com/robfig/cron/v3"

type CrontabCache struct {
	crontabs map[string]cron.EntryID
}

func (c *CrontabCache) Add(name string, id cron.EntryID) {
	c.crontabs[name] = id
}

func (c *CrontabCache) Get(name string) (cron.EntryID, bool) {
	id, exists := c.crontabs[name]
	return id, exists
}

func (c *CrontabCache) Remove(name string) {
	delete(c.crontabs, name)
}
