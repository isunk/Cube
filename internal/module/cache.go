package module

import (
	"sync"
	"time"
)

func init() {
	register("cache", func(ctx Context) interface{} {
		return &cache
	})
}

var cache = NewMemoryCache()

type MemoryCache struct {
	sync.Map
	sync.Mutex // 保护 timers map，避免并发访问
	timers     map[interface{}]*time.Timer
}

func (c *MemoryCache) Set(key interface{}, value interface{}, timeout int) {
	c.Lock()
	defer c.Unlock()
	// 删除旧定时器
	if t, ok := c.timers[key]; ok {
		t.Stop()
		delete(c.timers, key)
	}
	// 如果值为 nil 或失效时间小于等于 0，则立即删除缓存并返回
	if value == nil || timeout <= 0 {
		c.Delete(key)
		return
	}
	// 设置缓存和新的失效定时器
	c.Store(key, value)
	c.timers[key] = time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		c.Lock()
		defer c.Unlock()
		c.Delete(key)
		delete(c.timers, key)
	})
}

func (c *MemoryCache) Get(key interface{}) interface{} {
	if value, ok := c.Load(key); ok {
		return value
	}
	return nil
}

func (c *MemoryCache) Has(key interface{}) bool {
	_, ok := c.Load(key)
	return ok
}

func (c *MemoryCache) Expire(key interface{}, timeout int) {
	c.Lock()
	defer c.Unlock()
	// 查询旧定时器，如果存在则停止并删除
	if t, ok := c.timers[key]; ok {
		t.Stop()
		delete(c.timers, key)
	} else {
		return // 定时器不存在
	}

	// 如果新的失效时间小于等于 0，则立即删除缓存
	if timeout <= 0 {
		c.Delete(key)
		return
	}

	// 设置新的失效时间
	c.timers[key] = time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		c.Lock()
		defer c.Unlock()
		c.Delete(key)
		delete(c.timers, key)
	})
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		timers: make(map[interface{}]*time.Timer),
	}
}
