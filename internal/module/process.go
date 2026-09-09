package module

import (
	"os/exec"

	"cube/internal/builtin"
	"github.com/dop251/goja"
)

func init() {
	register("process", func(ctx Context) interface{} {
		return &ProcessClient{ctx}
	})
}

type ProcessClient struct {
	ctx Context
}

func (p *ProcessClient) Exec(command string, params ...string) (*builtin.Buffer, error) {
	output, err := exec.Command(command, params...).Output()
	if err != nil {
		return nil, err
	}
	return (*builtin.Buffer)(&output), nil
}

func (p *ProcessClient) Pexec(command string, params ...string) *goja.Promise {
	runtime := p.ctx.Worker.Runtime()

	promise, resolve, reject := runtime.NewPromise()

	t := p.ctx.Worker.EventLoop().NewEventTaskTrigger()

	t.AddTask(func() error {
		output, err := exec.Command(command, params...).Output()
		if err != nil {
			t.AddMicroTask(func() error {
				reject(runtime.NewGoError(err))
				t.Cancel()
				return nil
			})
			return nil
		}
		t.AddMicroTask(func() error { // resolve() must be called on the loop
			resolve(builtin.Buffer(output))
			t.Cancel()
			return nil
		})
		return nil
	})

	return promise
}
