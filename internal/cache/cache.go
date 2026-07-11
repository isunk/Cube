package cache

import (
	"database/sql"
	"regexp"

	"cube/internal/model"

	"github.com/dop251/goja"
	"github.com/robfig/cron/v3"
)

var (
	Route      *RouteCache
	Controller *ControllerCache
	Crontab    *CrontabCache
	Daemon     *DaemonCache
	Module     *ModuleCache
	DB         *DBCache
)

// Init initializes all cache modules
func Init(db *sql.DB) error {
	Route = &RouteCache{
		routes: make(map[string]*regexp.Regexp),
		db:     db,
	}
	if err := Route.Init(); err != nil {
		return err
	}

	Controller = &ControllerCache{
		controllers: make(map[string]*model.Source),
		db:          db,
	}

	Crontab = &CrontabCache{
		crontabs: make(map[string]cron.EntryID),
	}

	Daemon = &DaemonCache{
		daemons: make(map[string]Worker),
	}

	Module = &ModuleCache{
		modules: make(map[string]*goja.Program),
	}

	DB = &DBCache{
		connections: make(map[string]*sql.DB),
	}

	return nil
}
