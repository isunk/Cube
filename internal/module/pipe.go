package module

func init() {
	register("pipe", func(ctx Context) interface{} {
		return func(name string) *BlockingQueueClient {
			if PipeCache == nil {
				PipeCache = make(map[string]*BlockingQueueClient, 99)
			}
			if PipeCache[name] == nil {
				PipeCache[name] = &BlockingQueueClient{
					queue: make(chan interface{}, 99),
				}
			}
			return PipeCache[name]
		}
	})
}

var PipeCache map[string]*BlockingQueueClient
