package module

func init() {
	register("pipe", func(ctx Context) interface{} {
		return func(name string) *BlockingQueueClient {
			if pipes == nil {
				pipes = make(map[string]*BlockingQueueClient, 99)
			}
			if pipes[name] == nil {
				pipes[name] = &BlockingQueueClient{
					queue: make(chan interface{}, 99),
				}
			}
			return pipes[name]
		}
	})
}

var pipes map[string]*BlockingQueueClient
