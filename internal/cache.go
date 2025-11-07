package internal

import (
	"cube/internal/model"
	"database/sql"
	"errors"
	"github.com/dop251/goja"
	"github.com/robfig/cron/v3"
	"regexp"
)

var Cache *CacheClient

func InitCache() {
	Cache = &CacheClient{
		Controllers: make(map[string]*model.Source),
		Crontabs:    make(map[string]cron.EntryID),
		Daemons:     make(map[string]*Worker),
		Modules:     make(map[string]*goja.Program),
		DbSources:   make(map[string]*sql.DB),
	}
	Cache.InitRoutes()
}

type CacheClient struct {
	Routes      map[string]*regexp.Regexp
	Controllers map[string]*model.Source
	Crontabs    map[string]cron.EntryID
	Daemons     map[string]*Worker
	Modules     map[string]*goja.Program
	DbSources   map[string]*sql.DB
}

func (s *CacheClient) GetController(name string) *model.Source {
	source := s.Controllers[name]
	if source == nil {
		source = &model.Source{}
		if err := Db.QueryRow("select name, method from source where name = ? and type = 'controller' and active = true", name).Scan(&source.Name, &source.Method); err != nil {
			return nil
		}
		s.Controllers[name] = source
	}
	return source
}

func (s *CacheClient) InitRoutes() {
	s.Routes = make(map[string]*regexp.Regexp)
	rows, err := Db.Query("select name, url from source where type = 'controller' order by rowid desc")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name, path string
		rows.Scan(&name, &path)
		s.SetRoute(name, path)
	}
}

func (s *CacheClient) SetRoute(name string, path string) {
	s.Routes[name] = regexp.MustCompile("^" + regexp.MustCompile("{(.*?)}").ReplaceAllString(path, "(?P<$1>.*?)") + "$")
}

func (s *CacheClient) GetRoute(path string) (string, map[string]string) {
	for k, v := range s.Routes {
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

func (s *CacheClient) GetDbSource(dbType string, connection string) (db *sql.DB, err error) {
	if db = s.DbSources[connection]; db == nil {
		switch dbType {
		case "sqlite3":
			db, err = sql.Open("sqlite3", connection)
		case "mysql":
			db, err = sql.Open("mysql", connection)
		default:
			err = errors.New("invalid database type: only 'sqlite' and 'mysql' are supported")
		}
		if err != nil {
			return
		}
		s.DbSources[connection] = db
	}
	if err = db.Ping(); err != nil {
		delete(s.DbSources, connection)
		return
	}
	return
}
