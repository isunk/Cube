package cache

import (
	"database/sql"
	"regexp"
	"sync"
)

var REGEX_ROUTE_PARAM = regexp.MustCompile(`{(.*?)}`) // 路由路径参数正则

type RouteEntry struct {
	name  string
	regex *regexp.Regexp
}

type RouteCache struct {
	sync.RWMutex
	routes []RouteEntry
	db     *sql.DB
}

func (c *RouteCache) Init() error {
	c.Lock()
	c.routes = make([]RouteEntry, 0)
	c.Unlock()

	// 按 rowid asc 查询后经 Set 头插，最终顺序为 rowid 大的（新路由）在前、优先匹配，与运行时 Set 头插语义一致
	rows, err := c.db.Query("select name, url from source where type = 'controller' order by rowid asc")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, path string
		if err := rows.Scan(&name, &path); err != nil {
			continue
		}
		c.Set(name, path)
	}
	return nil
}

func (c *RouteCache) Get(path string) (string, map[string]string) {
	c.RLock()
	defer c.RUnlock()

	for _, entry := range c.routes {
		values := entry.regex.FindAllStringSubmatch(path, -1)
		if len(values) == 0 {
			continue
		}

		groups := entry.regex.SubexpNames()
		m := make(map[string]string)
		for i, name := range groups {
			if i == 0 {
				continue
			}
			m[name] = values[0][i]
		}
		return entry.name, m
	}
	return "", nil
}

func (c *RouteCache) Set(name, path string) {
	c.Lock()
	defer c.Unlock()
	// 保持按 rowid desc 的插入顺序：先移除已有项，再插入到头部，使新路由优先匹配
	for i, e := range c.routes {
		if e.name == name {
			c.routes = append(c.routes[:i], c.routes[i+1:]...)
			break
		}
	}
	c.routes = append([]RouteEntry{{name: name, regex: regexp.MustCompile("^" + REGEX_ROUTE_PARAM.ReplaceAllString(path, "(?P<$1>.*?)") + "$")}}, c.routes...)
}

func (c *RouteCache) Remove(name string) {
	c.Lock()
	defer c.Unlock()
	for i, e := range c.routes {
		if e.name == name {
			c.routes = append(c.routes[:i], c.routes[i+1:]...)
			break
		}
	}
}
