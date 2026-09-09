package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"cube/internal"
)

//#region Eval

// eval 在独立的 worker 上执行脚本，返回逐帧解析结果。
func eval(t *testing.T, script string) []map[string]any {
	t.Helper()
	internal.WorkerPool.Channels = make(chan *internal.Worker, 1)
	internal.WorkerPool.Channels <- internal.NewWorker(internal.NewProgram(), 0)

	req := httptest.NewRequest("EVAL", "/source", strings.NewReader(script))
	rec := httptest.NewRecorder()
	handleSourceEval(rec, req)

	frames := []map[string]any{}
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var f map[string]any
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			t.Fatalf("failed to parse frame %q: %v", line, err)
		}
		frames = append(frames, f)
	}
	return frames
}

// logs 从帧中提取日志帧（同时含 level 与 message 字段）。
func logs(frames []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(frames))
	for _, f := range frames {
		if _, ok := f["level"]; ok {
			if _, ok := f["message"]; ok {
				out = append(out, f)
			}
		}
	}
	return out
}

func TestEvalStreaming(t *testing.T) {
	// 响应头为 text/event-stream
	{
		internal.WorkerPool.Channels = make(chan *internal.Worker, 1)
		internal.WorkerPool.Channels <- internal.NewWorker(internal.NewProgram(), 0)

		req := httptest.NewRequest("EVAL", "/source", strings.NewReader("1"))
		rec := httptest.NewRecorder()
		handleSourceEval(rec, req)

		if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
			t.Fatalf("Content-Type: want text/event-stream, got %q", got)
		}
	}

	// 仅推送日志帧
	{
		frames := eval(t, `console.log("x")`)
		if len(frames) != 1 {
			t.Fatalf("expected 1 log frame, got %d: %v", len(frames), frames)
		}
		if frames[0]["message"] != "x" {
			t.Fatalf("message: want x, got %v", frames[0]["message"])
		}
	}
}

func TestEvalConsole(t *testing.T) {
	// 五级日志依次输出
	{
		got := logs(eval(t, `console.debug("d"); console.info("i"); console.log("l"); console.warn("w"); console.error("e")`))
		if len(got) != 5 {
			t.Fatalf("expected 5 log frames, got %d: %v", len(got), got)
		}
		levels := []string{"debug", "info", "log", "warn", "error"}
		msgs := []string{"d", "i", "l", "w", "e"}
		for i, f := range got {
			if f["level"] != levels[i] {
				t.Fatalf("frame %d level: want %s, got %v", i, levels[i], f["level"])
			}
			if f["message"] != msgs[i] {
				t.Fatalf("frame %d message: want %s, got %v", i, msgs[i], f["message"])
			}
		}
	}

	// 参数类型序列化：字符串原样、undefined 保留语义、其余 JSON
	{
		got := logs(eval(t, `console.log("s", 1, true, null, undefined, {a:1})`))
		if len(got) != 1 {
			t.Fatalf("expected 1 log frame, got %d", len(got))
		}
		msg := got[0]["message"].(string)
		if !strings.Contains(msg, "s 1 true null undefined") {
			t.Fatalf("primitives mismatch: %s", msg)
		}
		if !strings.Contains(msg, `{"a":1}`) {
			t.Fatalf("object json mismatch: %s", msg)
		}
	}
}

func TestEvalAsync(t *testing.T) {
	// setTimeout 异步回调日志到达
	{
		got := logs(eval(t, `console.log("start"); setTimeout(() => console.log("async"), 5)`))
		if len(got) != 2 {
			t.Fatalf("expected 2 log frames, got %d: %v", len(got), got)
		}
		if got[0]["message"] != "start" {
			t.Fatalf("frame 0 message: want start, got %v", got[0]["message"])
		}
		if got[1]["message"] != "async" {
			t.Fatalf("frame 1 message: want async, got %v", got[1]["message"])
		}
	}

	// setTimeout 回调内 throw 被捕获为 error 日志
	{
		got := logs(eval(t, `console.log("before"); setTimeout(() => { throw new Error("async boom") }, 5); console.log("after")`))
		if len(got) != 3 {
			t.Fatalf("expected 3 log frames, got %d: %v", len(got), got)
		}
		if got[0]["message"] != "before" || got[1]["message"] != "after" {
			t.Fatalf("ordered logs mismatch: %v %v", got[0]["message"], got[1]["message"])
		}
		if got[2]["level"] != "error" {
			t.Fatalf("error level: want error, got %v", got[2]["level"])
		}
		if !strings.Contains(got[2]["message"].(string), "async boom") {
			t.Fatalf("error message should contain 'async boom', got %v", got[2]["message"])
		}
	}

	// setInterval 三次 tick 后 clearInterval 正常终止
	{
		got := logs(eval(t, `let i=0; const t=setInterval(()=>{console.log("tick", i); if(++i>=3) clearInterval(t)}, 1)`))
		if len(got) != 3 {
			t.Fatalf("expected 3 tick log frames, got %d: %v", len(got), got)
		}
		for i, f := range got {
			if f["message"] != "tick "+string(rune('0'+i)) {
				t.Fatalf("frame %d message: want tick %d, got %v", i, i, f["message"])
			}
		}
	}
}

func TestEvalError(t *testing.T) {
	// 同步 throw 推送 error 级日志
	{
		got := logs(eval(t, `console.log("before"); throw new Error("boom")`))
		if len(got) != 2 {
			t.Fatalf("expected 2 log frames (before + error), got %d: %v", len(got), got)
		}
		if got[1]["level"] != "error" {
			t.Fatalf("level: want error, got %v", got[1]["level"])
		}
		if !strings.Contains(got[1]["message"].(string), "boom") {
			t.Fatalf("message should contain 'boom', got %v", got[1]["message"])
		}
	}

	// 语法错误推送含 SyntaxError 的 error 日志
	{
		got := logs(eval(t, `function ( {`))
		if len(got) != 1 {
			t.Fatalf("expected 1 error log frame, got %d: %v", len(got), got)
		}
		if got[0]["level"] != "error" {
			t.Fatalf("level: want error, got %v", got[0]["level"])
		}
		if !strings.Contains(got[0]["message"].(string), "SyntaxError") {
			t.Fatalf("message should contain 'SyntaxError', got %v", got[0]["message"])
		}
	}
}

// 验证 Promise 链中未被 .catch 处理的 rejection 会被 EventLoop 捕获并作为 error 日志推送（时序：function 返回后 goja leave() 排空 jobQueue，then 回调 throw 触发 tracker(Reject)，由 Run 转为 err）
func TestEvalPromise(t *testing.T) {
	// unhandled rejection → error 日志
	{
		got := logs(eval(t, `console.log("before"); Promise.resolve().then(() => { throw new Error("async boom") })`))
		if len(got) != 2 {
			t.Fatalf("expected 2 log frames (before + unhandled rejection), got %d: %v", len(got), got)
		}
		if got[0]["message"] != "before" {
			t.Fatalf("frame 0 message: want before, got %v", got[0]["message"])
		}
		if got[1]["level"] != "error" {
			t.Fatalf("rejection level: want error, got %v", got[1]["level"])
		}
		if !strings.Contains(got[1]["message"].(string), "async boom") {
			t.Fatalf("rejection message should contain 'async boom', got %v", got[1]["message"])
		}
	}

	// 带 .catch 的链：rejection 被捕获后不再触发 unhandled rejection，仅一条 warn 日志
	{
		got := logs(eval(t, `Promise.resolve().then(() => { throw new Error("x") }).catch(e => console.warn("caught:", e.message))`))
		if len(got) != 1 {
			t.Fatalf("expected 1 warn log frame (caught), got %d: %v", len(got), got)
		}
		if got[0]["level"] != "warn" {
			t.Fatalf("level: want warn, got %v", got[0]["level"])
		}
		msg := got[0]["message"].(string)
		if !strings.Contains(msg, "caught") || !strings.Contains(msg, "x") {
			t.Fatalf("warn message should contain 'caught' and 'x', got %s", msg)
		}
	}
}

//#endregion
