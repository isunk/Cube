package builtin

import (
	"database/sql"

	"github.com/dop251/goja"
)

var Builtins = make([]func(ctx Context), 0)

type Worker interface {
	AddDefer(d func())
	Id() int
	Runtime() *goja.Runtime
	EventLoop() *EventLoop
	Interrupt(reason string)
}

type Cache interface {
	GetDbSource(dbType, connection string) (*sql.DB, error)
}

type Context struct {
	Worker Worker
	Db     *sql.DB
	Cache  Cache
}
