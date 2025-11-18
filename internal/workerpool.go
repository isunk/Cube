package internal

import (
	"cube/internal/config"
)

var WorkerPool struct {
	Channels chan *Worker
	Workers  []*Worker
}

func InitWorkerPool() {
	WorkerPool.Workers = make([]*Worker, config.Count) // 创建 goja 实例池
	WorkerPool.Channels = make(chan *Worker, config.Count)

	// 编译源码
	program := CreateProgram()

	for i := 0; i < config.Count; i++ {
		worker := CreateWorker(program, i) // 创建 goja 运行时

		WorkerPool.Workers[i] = worker
		WorkerPool.Channels <- worker
	}
}
