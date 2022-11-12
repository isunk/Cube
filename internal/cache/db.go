package cache

import (
	"database/sql"
	"errors"
)

type DBCache struct {
	connections map[string]*sql.DB
}

func (c *DBCache) Get(dbType, connection string) (db *sql.DB, err error) {
	if db = c.connections[connection]; db == nil {
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
		c.connections[connection] = db
	}
	if err = db.Ping(); err != nil {
		delete(c.connections, connection)
		return
	}
	return
}
