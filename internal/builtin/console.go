package builtin

import (
	"cube/internal/log"

	"github.com/dop251/goja"
)

func init() {
	Factories = append(Factories, func(ctx Context) {
		ctx.Worker.Runtime().Set("console", &ConsoleClient{worker: ctx.Worker})
	})
}

type ConsoleClient struct {
	worker Worker
}

// emit 输出日志：eval 场景下 worker 设置了重定向目标（Logger() 非 nil），转发给客户端；否则写服务端日志。
func (c *ConsoleClient) emit(level string, args ...goja.Value) {
	if fn := c.worker.Logger(); fn != nil {
		fn(level, args...)
		return
	}
	out := make([]interface{}, len(args))
	for i, a := range args {
		out[i] = a
	}
	switch level {
	case "debug":
		log.Debug(c.worker.Id(), out...)
	case "info":
		log.Info(c.worker.Id(), out...)
	case "warn":
		log.Warn(c.worker.Id(), out...)
	case "error":
		log.Error(c.worker.Id(), out...)
	default:
		log.Log(c.worker.Id(), out...)
	}
}

func (c *ConsoleClient) Log(args ...goja.Value) {
	c.emit("log", args...)
}

func (c *ConsoleClient) Debug(args ...goja.Value) {
	c.emit("debug", args...)
}

func (c *ConsoleClient) Info(args ...goja.Value) {
	c.emit("info", args...)
}

func (c *ConsoleClient) Warn(args ...goja.Value) {
	c.emit("warn", args...)
}

func (c *ConsoleClient) Error(args ...goja.Value) {
	c.emit("error", args...)
}
