package cache

import (
	"database/sql"
	"errors"
	"sync"
)

type DBCache struct {
	sync.RWMutex
	connections map[string]*sql.DB
}

func (c *DBCache) Get(dbType, connection string) (db *sql.DB, err error) {
	c.RLock()
	db = c.connections[connection]
	c.RUnlock()

	if db == nil {
		switch dbType {
		case "sqlite":
			db, err = sql.Open("sqlite", connection)
		case "mysql":
			db, err = sql.Open("mysql", connection)
		default:
			err = errors.New("invalid database type: only 'sqlite' and 'mysql' are supported")
		}
		if err != nil {
			return
		}
		c.Lock()
		c.connections[connection] = db
		c.Unlock()
	}
	if err = db.Ping(); err != nil {
		c.Lock()
		delete(c.connections, connection)
		c.Unlock()
		return
	}
	return
}
