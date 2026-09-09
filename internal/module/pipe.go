package module

import "sync"

var pipes = struct {
	sync.Mutex
	m map[string]*BlockingQueueClient
}{m: make(map[string]*BlockingQueueClient, 99)}

func init() {
	register("pipe", func(ctx Context) interface{} {
		return func(name string) *BlockingQueueClient {
			pipes.Lock()
			defer pipes.Unlock()
			if pipes.m[name] == nil {
				pipes.m[name] = &BlockingQueueClient{
					queue: make(chan interface{}, 99),
				}
			}
			return pipes.m[name]
		}
	})
}
