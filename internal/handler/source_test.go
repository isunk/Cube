package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"cube/internal"
)

//#region Eval

func TestEval(t *testing.T) {
	eval := func(t *testing.T, script string) []map[string]any {
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
				t.Fatalf("parse frame %q: %v", line, err)
			}
			frames = append(frames, f)
		}
		return frames
	}

	logs := func(frames []map[string]any) []map[string]any {
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

	t.Run("响应头为 text/event-stream", func(t *testing.T) {
		internal.WorkerPool.Channels = make(chan *internal.Worker, 1)
		internal.WorkerPool.Channels <- internal.NewWorker(internal.NewProgram(), 0)
		req := httptest.NewRequest("EVAL", "/source", strings.NewReader("1"))
		rec := httptest.NewRecorder()
		handleSourceEval(rec, req)
		if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
			t.Fatalf("header failed: %q", got)
		}
	})

	t.Run("仅推送日志帧", func(t *testing.T) {
		frames := eval(t, `console.log("x")`)
		if len(frames) != 1 {
			t.Fatalf("eval count failed: %d", len(frames))
		}
		if frames[0]["message"] != "x" {
			t.Fatalf("eval message failed: %v", frames[0]["message"])
		}
	})

	t.Run("五级日志依次输出", func(t *testing.T) {
		got := logs(eval(t, `console.debug("d"); console.info("i"); console.log("l"); console.warn("w"); console.error("e")`))
		wantLevels := []string{"debug", "info", "log", "warn", "error"}
		wantMsgs := []string{"d", "i", "l", "w", "e"}
		if len(got) != 5 {
			t.Fatalf("eval count failed: %d", len(got))
		}
		for i, f := range got {
			if f["level"] != wantLevels[i] {
				t.Errorf("eval level failed: %v", f["level"])
			}
			if f["message"] != wantMsgs[i] {
				t.Errorf("eval message failed: %v", f["message"])
			}
		}
	})

	t.Run("参数类型序列化：字符串原样、undefined 保留语义、其余 JSON", func(t *testing.T) {
		got := logs(eval(t, `console.log("s", 1, true, null, undefined, {a:1})`))
		if len(got) != 1 {
			t.Fatalf("eval count failed: %d", len(got))
		}
		msg := got[0]["message"].(string)
		if !strings.Contains(msg, "s 1 true null undefined") {
			t.Errorf("eval message failed: %q", msg)
		}
		if !strings.Contains(msg, `{"a":1}`) {
			t.Errorf("eval message failed: %q", msg)
		}
	})

	t.Run("同步 throw 推送 error 级日志", func(t *testing.T) {
		got := logs(eval(t, `throw new Error("boom")`))
		if len(got) != 1 {
			t.Fatalf("eval count failed: %d", len(got))
		}
		if got[0]["level"] != "error" {
			t.Fatalf("eval level failed: %v", got[0]["level"])
		}
		if !strings.Contains(got[0]["message"].(string), "boom") {
			t.Errorf("eval message failed: %v", got[0]["message"])
		}
	})
}

//#endregion
