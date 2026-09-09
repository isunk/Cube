package builtin

import (
	"database/sql"

	"github.com/dop251/goja"
)

var Factories = make([]func(ctx Context), 0)

type Worker interface {
	AddDefer(d func())
	Id() int
	Runtime() *goja.Runtime
	EventLoop() *EventLoop
	Interrupt(reason string)
	// Logger 返回当前日志重定向目标（eval 场景由 handler 设置，将 console 输出转发给客户端；nil 时写服务端日志）。
	Logger() func(level string, args ...goja.Value)
	// SetLogger 设置或清除（传 nil）日志重定向目标。
	SetLogger(fn func(level string, args ...goja.Value))
}

type Context struct {
	Worker Worker
	Db     *sql.DB
}
