package module

import (
	"sync"

	"cube/internal/builtin"

	"github.com/dop251/goja"
)

func init() {
	register("event", func(ctx Context) interface{} {
		return &EventClient{ctx}
	})
}

//#region 事件订阅者

type EventSubscriber struct {
	trigger *builtin.EventTaskTrigger
	data    chan interface{} // 用于接收生产者（即 EventBus）发送的数据
	stop    chan struct{}    // 用于向生产者（即 EventBus）发送关闭通知
}

func (s *EventSubscriber) Next() interface{} {
	return <-s.data
}

func (s *EventSubscriber) Cancel() {
	if s.trigger.Cancel() {
		close(s.stop) // 广播通知 EventBus，不再推送数据
		close(s.data)
	}
}

//#endregion

//#region 事件总线

var bus EventBus

type EventBus struct {
	sync.Mutex
	subscribers map[string][]*EventSubscriber
}

func (b *EventBus) subscribe(topic string, s *EventSubscriber) {
	b.Lock()
	defer b.Unlock()
	if b.subscribers == nil {
		b.subscribers = make(map[string][]*EventSubscriber)
	}
	b.subscribers[topic] = append(b.subscribers[topic], s)
}

func (b *EventBus) emit(topic string, data interface{}) {
	b.Lock()
	subscribers, found := b.subscribers[topic]
	if !found {
		b.Unlock()
		return
	}
	i := 0
	for _, s := range subscribers {
		if s.trigger.IsCancelled() {
			continue
		}
		select {
		case <-s.stop:
			continue
		default:
			subscribers[i] = s
			i++
		}
	}
	b.subscribers[topic] = subscribers[:i]                        // 通过位移法删除已关闭的通道
	active := append([]*EventSubscriber(nil), subscribers[:i]...) // 快照活跃订阅者
	b.Unlock()

	// 锁外阻塞发送，保证事件送达，同时避免慢消费者阻塞整个 bus（其他 topic 的订阅/取消不受影响）
	for _, s := range active {
		select {
		case <-s.stop: // 消费者在发送前已关闭，跳过
		case s.data <- data: // 发送数据
		}
	}
}

//#endregion

type EventClient struct {
	ctx Context
}

func (c *EventClient) Emit(topic string, data interface{}) {
	bus.emit(topic, data)
}

func (c *EventClient) CreateSubscriber(topics ...string) *EventSubscriber {
	s := &EventSubscriber{c.ctx.Worker.EventLoop().NewEventTaskTrigger(), make(chan interface{}), make(chan struct{}, 1)}

	c.ctx.Worker.AddDefer(s.Cancel)

	for _, topic := range topics {
		bus.subscribe(topic, s)
	}

	return s
}

func (c *EventClient) On(call goja.FunctionCall) goja.Value { // 见 goja.Runtime.ToValue 函数，这里需要传递 func(FunctionCall) Value 类型的方法
	// 主题
	topic, ok := call.Argument(0).Export().(string)
	if !ok {
		c.ctx.Worker.Interrupt("invalid argument topic, not a string")
		return nil
	}

	// 回调方法
	fn, ok := goja.AssertFunction(call.Argument(1))
	if !ok {
		c.ctx.Worker.Interrupt("invalid argument callback, not a function")
		return nil
	}

	runtime := c.ctx.Worker.Runtime()

	s := c.CreateSubscriber(topic)

	go func() {
	L:
		for {
			select {
			case <-s.stop:
				break L
			case data, received := <-s.data:
				if !received {
					s.Cancel()
					break L
				}
				s.trigger.AddTask(func() error {
					_, err := fn(nil, runtime.ToValue(data))
					return err
				})
			}
		}
	}()

	return runtime.ToValue(s)
}
