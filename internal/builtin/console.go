package builtin

import "cube/internal/log"

func init() {
	Factories = append(Factories, func(ctx Context) {
		ctx.Worker.Runtime().Set("console", &ConsoleClient{worker: ctx.Worker})
	})
}

type ConsoleClient struct {
	worker Worker
}

func (c *ConsoleClient) Log(e ...interface{}) {
	log.Log(c.worker.Id(), e...)
}

func (c *ConsoleClient) Debug(e ...interface{}) {
	log.Debug(c.worker.Id(), e...)
}

func (c *ConsoleClient) Info(e ...interface{}) {
	log.Info(c.worker.Id(), e...)
}

func (c *ConsoleClient) Warn(e ...interface{}) {
	log.Warn(c.worker.Id(), e...)
}

func (c *ConsoleClient) Error(e ...interface{}) {
	log.Error(c.worker.Id(), e...)
}
