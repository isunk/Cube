package internal

import (
	"errors"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"cube/internal/builtin"
	"cube/internal/cache"
	m "cube/internal/module"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
)

type Worker struct {
	id       int
	runtime  *goja.Runtime
	function goja.Callable
	defers   []func()
	loop     *builtin.EventLoop                     // 事件循环
	err      error                                  // 中断异常
	logger   func(level string, args ...goja.Value) // eval 场景的日志重定向目标，nil 时 console 写服务端日志
}

func (w *Worker) Logger() func(level string, args ...goja.Value) {
	return w.logger
}

func (w *Worker) SetLogger(fn func(level string, args ...goja.Value)) {
	w.logger = fn
}

func (w *Worker) Run(params ...goja.Value) (goja.Value, error) {
	return w.loop.Run(func() (goja.Value, error) {
		val, err := w.function(nil, params...)
		if w.err != nil { // 优先返回 interrupt 的中断信息
			return val, w.err
		}
		return val, err
	})
}

func (w *Worker) Id() int {
	return w.id
}

func (w *Worker) Runtime() *goja.Runtime {
	return w.runtime
}

func (w *Worker) EventLoop() *builtin.EventLoop {
	return w.loop
}

func (w *Worker) AddDefer(d func()) {
	w.defers = append(w.defers, d)
}

func (w *Worker) CleanDefers() {
	if len(w.defers) == 0 {
		return
	}

	for _, d := range w.defers {
		d()
	}

	w.defers = make([]func(), 0)
}

func (w *Worker) Interrupt(reason string) {
	// 中断事件循环
	w.loop.Interrupt()

	// 发送中断信号
	w.Runtime().Interrupt(reason)

	// 记录中断异常
	w.err = errors.New(reason)

	// 清理句柄
	w.CleanDefers() // 这里清理句柄，用于防止阻塞，例如监听网络连接：在此时关闭监听器，可以使得监听方法出现异常，可以避免 goja 的中断信号无法被触发问题
}

func (w *Worker) Reset() {
	// 清理句柄
	w.CleanDefers() // 用于非中断场景下的句柄清理

	// 清理中断信号
	w.Runtime().ClearInterrupt()

	// 清理中断异常
	w.err = nil

	// 重置事件循环
	w.loop.Reset()

	// 清除 eval 场景设置的日志重定向目标，恢复 console 写服务端日志
	w.logger = nil
}

func NewProgram() *goja.Program {
	// 编译源码
	program, err := goja.Compile(
		"index",
		"(function (id, ...params) { return require(id).default(...params); })", // 使用闭包，防止全局变量污染
		false, // 关闭严格模式，增加运行时的容错能力
	)
	if err != nil {
		panic(err)
	}
	return program
}

func NewWorker(program *goja.Program, id int) *Worker {
	runtime := goja.New()

	entry, err := runtime.RunProgram(program) // 这里使用 RunProgram，可复用已编译的代码，相比直接调用 RunString 更显著提升性能
	if err != nil {
		panic(err)
	}
	function, ok := goja.AssertFunction(entry)
	if !ok {
		panic("program is not a function")
	}

	worker := Worker{id, runtime, function, make([]func(), 0), builtin.NewEventLoop(), nil, nil}

	// 接入 goja 内置 Promise 的 unhandled rejection：tracker 在 leave 阶段（jobQueue 排空时）触发，收集到 EventLoop，由 Run 转为异步异常返回，使 Promise.resolve().then(()=>{throw}) 可被捕获
	loop := worker.loop
	runtime.SetPromiseRejectionTracker(func(p *goja.Promise, op goja.PromiseRejectionOperation) {
		if op == goja.PromiseRejectionReject {
			loop.TrackRejection(p, p.Result().String())
		} else {
			loop.UntrackRejection(p)
		}
	})

	runtime.Set("require", func(id string) (goja.Value, error) {
		program, exists := cache.Module.Get(id)
		if !exists { // 如果缓存不存在，则查询数据库
			// 获取名称、类型
			var name, stype string
			if strings.HasPrefix(id, "./controller/") {
				name, stype = id[13:], "controller"
			} else if strings.HasPrefix(id, "./daemon/") {
				name, stype = id[9:], "daemon"
			} else if strings.HasPrefix(id, "./crontab/") {
				name, stype = id[10:], "crontab"
			} else if strings.HasPrefix(id, "./") {
				name, stype = path.Clean(id), "module"
			} else if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
				name, stype = id, "link"
			} else { // 如果没有 "./" 前缀，则视为 node_modules
				name, stype = "node_modules/"+id, "module"
			}

			var src string
			if stype == "link" {
				// 在线请求网络源码（设置超时，避免因网络异常长时间阻塞）
				client := &http.Client{Timeout: 30 * time.Second}
				resp, err := client.Get(name)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}
				src = string(body)
			} else {
				// 根据名称查找数据库源码
				if err := Db.QueryRow("select compiled from source where name = ? and type = ? and active = true", name, stype).Scan(&src); err != nil {
					return nil, err
				}
			}

			if src == "" {
				return nil, errors.New("module not found")
			}

			// 编译
			parsed, err := goja.Parse(
				name,
				"(function(exports, require, module) {"+src+"\n})",
				parser.WithSourceMapLoader(func(p string) ([]byte, error) {
					// 项目未提供 source map 文件，统一返回 nil 避免把源码误当作 map 解析
					return nil, nil
				}),
			)
			if err != nil {
				return nil, err
			}
			program, err = goja.CompileAST(parsed, false)
			if err != nil {
				return nil, err
			}

			// 缓存当前 module 的 program
			// 这里不应该直接缓存 module，因为 module 依赖当前 vm 实例，在开启多个 vm 实例池的情况下，调用会错乱从而导致异常 "TypeError: Illegal runtime transition of an Object at ..."
			cache.Module.Add(id, program)
		}

		exports := runtime.NewObject()
		module := runtime.NewObject()
		module.Set("exports", exports)

		// 运行
		entry, err := runtime.RunProgram(program)
		if err != nil {
			return nil, err
		}
		if function, ok := goja.AssertFunction(entry); ok {
			_, err = function(
				exports,                // this
				exports,                // exports
				runtime.Get("require"), // require
				module,                 // module
			)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("entry is not a function")
		}
		return module.Get("exports"), nil
	})

	runtime.Set("exports", runtime.NewObject())

	runtime.SetFieldNameMapper(goja.UncapFieldNameMapper()) // 该转换器会将 go 对象中的属性、方法以小驼峰式命名规则映射到 js 对象中

	ctx := builtin.Context{
		Worker: &worker,
		Db:     Db,
	}

	runtime.Set("$native", func(name string) (interface{}, error) {
		// 通过 Set 方法内置的 []byte 类型的变量或方法：
		// 入参如果是 []byte 类型，可接受 js 中 string 或 Array<number> 类型的变量
		// 出参如果是 []byte 类型，将会隐式地转换为 js 的 Array<number> 类型的变量（见 goja.objectGoArrayReflect._init() 方法实现，class 为 "Array", prototype 为 ArrayPrototype）
		factory, ok := m.Factories[name]
		if ok {
			return factory(ctx), nil
		}
		return nil, errors.New("module not found: " + name)
	})

	for _, factory := range builtin.Factories {
		factory(ctx)
	}

	runtime.SetMaxCallStackSize(2048)

	return &worker
}
