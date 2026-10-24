package internal

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

//#region 基础能力

func TestWorkerNew(t *testing.T) {
	w := NewWorker(NewProgram(), 7)
	if w.Id() != 7 {
		t.Fatalf("new failed: %d", w.Id())
	}
	if w.Runtime() == nil {
		t.Fatal("runtime failed")
	}
	if w.EventLoop() == nil {
		t.Fatal("event loop failed")
	}
}

func TestWorkerRuntime(t *testing.T) {
	w := NewWorker(NewProgram(), 0)
	rt := w.Runtime()

	t.Run("求值基本类型", func(t *testing.T) {
		if v, err := rt.RunString("1 + 2"); err != nil || v.ToInteger() != 3 {
			t.Fatalf("evaluate failed: %v", v)
		}
		if v, err := rt.RunString("'hi'"); err != nil || v.String() != "hi" {
			t.Fatalf("evaluate failed: %v", v)
		}
	})

	t.Run("注入全局对象", func(t *testing.T) {
		for _, c := range []struct{ name, want string }{
			{"exports", "object"},
			{"require", "function"},
			{"$native", "function"},
		} {
			v, err := rt.RunString("typeof " + c.name)
			if err != nil {
				t.Fatalf("typeof failed: %v", err)
			}
			if got := v.String(); got != c.want {
				t.Errorf("typeof failed: %q", got)
			}
		}
	})
}

func TestWorkerDefer(t *testing.T) {
	w := NewWorker(NewProgram(), 0)
	var order []int
	for i := 0; i < 3; i++ {
		w.AddDefer(func() { order = append(order, i) })
	}
	w.CleanDefers()
	if len(order) != 3 {
		t.Fatalf("defer count failed: %d", len(order))
	}
	for i, got := range order {
		if got != i {
			t.Errorf("defer order failed: %d", got)
		}
	}
	w.CleanDefers()
}

func TestWorkerInterruptReset(t *testing.T) {
	t.Run("中断清理并报错", func(t *testing.T) {
		w := NewWorker(NewProgram(), 0)
		var cleaned bool
		w.AddDefer(func() { cleaned = true })
		w.Interrupt("boom")
		if !cleaned {
			t.Fatal("interrupt clean failed")
		}
		if w.err == nil || w.err.Error() != "boom" {
			t.Fatalf("interrupt error failed: %v", w.err)
		}
	})

	t.Run("重置清空状态", func(t *testing.T) {
		w := NewWorker(NewProgram(), 0)
		w.Interrupt("boom")
		w.SetLogger(func(level string, args ...goja.Value) {})
		if w.Logger() == nil {
			t.Fatal("set logger failed")
		}
		w.Reset()
		if w.err != nil {
			t.Fatalf("reset error failed: %v", w.err)
		}
		if w.Logger() != nil {
			t.Fatal("reset logger failed")
		}
	})
}

func TestWorkerRegexp(t *testing.T) {
	type tc struct {
		note string
		expr string
		want string
	}
	w := NewWorker(NewProgram(), 0)
	eval := func(expr string) string {
		t.Helper()
		v, err := w.Runtime().RunString(expr)
		if err != nil {
			t.Fatalf("regexp eval failed: %v", err)
		}
		return v.String()
	}
	run := func(t *testing.T, cases []tc) {
		t.Helper()
		for _, c := range cases {
			if got := eval(c.expr); got != c.want {
				t.Errorf("[%s] regexp failed: %q", c.note, got)
			}
		}
	}

	t.Run("正则标志位", func(t *testing.T) {
		run(t, []tc{
			{"i 忽略大小写", `/a/i.test("A")`, "true"},
			{"默认区分大小写", `/a/.test("A")`, "false"},
			{"m 多行", `/^b/m.test("a\nb")`, "true"},
			{"无 m 不跨行", `/^b/.test("a\nb")`, "false"},
			{"s dotAll", `/a.b/s.test("a\nb")`, "true"},
			{"无 s 点不匹配换行", `/a.b/.test("a\nb")`, "false"},
			{"y 粘滞", `/a/y.test("a")`, "true"},
			{"y 非起始不匹配", `/a/y.test("ba")`, "false"},
			{"u 代理对", `/😀/u.test("😀")`, "true"},
			{"u 码点转义", `/\u{1F600}/u.test("😀")`, "true"},
			{"g 收集全部匹配", `"aba".match(/a/g).length`, "2"},
		})
	})

	t.Run("环视与反向引用", func(t *testing.T) {
		run(t, []tc{
			{"正向先行", `/a(?=b)/.test("ab")`, "true"},
			{"负向先行", `/a(?!b)/.test("ac")`, "true"},
			{"正向后行", `/(?<=a)b/.test("ab")`, "true"},
			{"负向后行", `/(?<!x)b/.test("ab")`, "true"},
			{"反向引用", `/(a)\1/.test("aa")`, "true"},
		})
	})

	t.Run("Unicode 属性转义", func(t *testing.T) {
		run(t, []tc{
			{"lookahead 后 p{L} 生效", `/a(?!b)[\p{L}]/u.test("ac")`, "true"},
			{"lookahead 后 p{N} 生效", `/x(?!y)[\p{N}]/u.test("x5")`, "true"},
			{"无 lookahead 时 p{L} 退化", `/\p{L}/u.test("a")`, "false"},
			{"字符类 p{L} 当作字面量 p", `/[\p{L}_$]/gu.test("p")`, "true"},
		})
	})
}

//#endregion

//#region 异步与错误传播

type logEntry struct {
	level   string
	message string
}

func TestWorkerScript(t *testing.T) {
	run := func(script string) ([]logEntry, error) {
		w := NewWorker(NewProgram(), 0)
		var entries []logEntry
		w.SetLogger(func(level string, args ...goja.Value) {
			parts := make([]string, 0, len(args))
			for _, a := range args {
				if a == goja.Undefined() {
					parts = append(parts, "undefined")
				} else {
					parts = append(parts, a.String())
				}
			}
			entries = append(entries, logEntry{level, strings.Join(parts, " ")})
		})
		_, err := w.EventLoop().Run(func() (goja.Value, error) {
			return w.Runtime().RunString(script)
		})
		w.Reset()
		return entries, err
	}

	t.Run("setTimeout 异步回调日志到达", func(t *testing.T) {
		got, err := run(`console.log("start"); setTimeout(() => console.log("async"), 5)`)
		if err != nil {
			t.Fatalf("setTimeout failed: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("setTimeout log failed: %d", len(got))
		}
		if got[0].message != "start" || got[1].message != "async" {
			t.Errorf("setTimeout message failed: %q/%q", got[0].message, got[1].message)
		}
	})

	t.Run("setTimeout 回调内 throw 被捕获为 error 日志", func(t *testing.T) {
		got, err := run(`console.log("before"); setTimeout(() => { throw new Error("async boom") }, 5); console.log("after")`)
		if len(got) != 2 {
			t.Fatalf("setTimeout log failed: %d", len(got))
		}
		if got[0].message != "before" || got[1].message != "after" {
			t.Errorf("setTimeout message failed: %q/%q", got[0].message, got[1].message)
		}
		if err == nil || !strings.Contains(err.Error(), "async boom") {
			t.Fatalf("setTimeout error failed: %v", err)
		}
	})

	t.Run("setInterval 三次 tick 后 clearInterval 正常终止", func(t *testing.T) {
		got, err := run(`let i=0; const t=setInterval(()=>{console.log("tick", i); if(++i>=3) clearInterval(t)}, 1)`)
		if err != nil {
			t.Fatalf("setInterval failed: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("setInterval log failed: %d", len(got))
		}
		for i, e := range got {
			want := "tick " + string(rune('0'+i))
			if e.message != want {
				t.Errorf("setInterval message failed: %q", e.message)
			}
		}
	})

	t.Run("unhandled rejection → error 日志", func(t *testing.T) {
		got, err := run(`console.log("before"); Promise.resolve().then(() => { throw new Error("async boom") })`)
		if len(got) != 1 {
			t.Fatalf("rejection log failed: %d", len(got))
		}
		if got[0].message != "before" {
			t.Errorf("rejection message failed: %q", got[0].message)
		}
		if err == nil || !strings.Contains(err.Error(), "async boom") {
			t.Fatalf("rejection error failed: %v", err)
		}
	})

	t.Run("带 .catch 的链：rejection 被捕获后不再触发 unhandled rejection，仅一条 warn 日志", func(t *testing.T) {
		got, err := run(`Promise.resolve().then(() => { throw new Error("x") }).catch(e => console.warn("caught:", e.message))`)
		if err != nil {
			t.Fatalf("catch failed: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("catch log failed: %d", len(got))
		}
		if got[0].level != "warn" {
			t.Errorf("catch level failed: %q", got[0].level)
		}
		if !strings.Contains(got[0].message, "caught") || !strings.Contains(got[0].message, "x") {
			t.Errorf("catch message failed: %q", got[0].message)
		}
	})

	t.Run("同步 throw 推送 error 级日志", func(t *testing.T) {
		got, err := run(`console.log("before"); throw new Error("boom")`)
		if len(got) != 1 {
			t.Fatalf("throw log failed: %d", len(got))
		}
		if got[0].message != "before" {
			t.Errorf("throw message failed: %q", got[0].message)
		}
		if err == nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("throw error failed: %v", err)
		}
	})

	t.Run("语法错误推送含 SyntaxError 的 error 日志", func(t *testing.T) {
		got, err := run(`function ( {`)
		if len(got) != 0 {
			t.Fatalf("syntax log failed: %d", len(got))
		}
		if err == nil || !strings.Contains(err.Error(), "SyntaxError") {
			t.Fatalf("syntax error failed: %v", err)
		}
	})
}

//#endregion
