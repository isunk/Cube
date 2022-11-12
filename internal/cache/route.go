package cache

import (
	"database/sql"
	"regexp"
)

type RouteCache struct {
	routes map[string]*regexp.Regexp
	db     *sql.DB
}

func (c *RouteCache) Init() error {
	if c.routes == nil || len(c.routes) > 0 {
		c.routes = make(map[string]*regexp.Regexp)
	}
	rows, err := c.db.Query("select name, url from source where type = 'controller' order by rowid desc")
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
	for k, v := range c.routes {
		values := v.FindAllStringSubmatch(path, -1)
		if len(values) == 0 {
			continue
		}

		groups := v.SubexpNames()
		m := make(map[string]string)
		for i, name := range groups {
			if i == 0 {
				continue
			}
			m[name] = values[0][i]
		}
		return k, m
	}
	return "", nil
}

func (c *RouteCache) Set(name, path string) {
	c.routes[name] = regexp.MustCompile("^" + regexp.MustCompile("{(.*?)}").ReplaceAllString(path, "(?P<$1>.*?)") + "$")
}

func (c *RouteCache) Remove(name string) {
	delete(c.routes, name)
}
