package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"cube/internal"
	"cube/internal/cache"
	"cube/internal/model"
	"cube/internal/util"

	"github.com/dop251/goja"
)

var (
	REGEX_SOURCE_TYPE_ENUM  = regexp.MustCompile(`^(module|controller|daemon|crontab|template|resource)$`) // 源码类型枚举值
	REGEX_MODULE_NAME       = regexp.MustCompile(`^(node_modules/)?\w{2,32}$`)                             // 模块名（可选 node_modules/ 前缀，2-32 位 word）
	REGEX_SOURCE_PLAIN_NAME = regexp.MustCompile(`^\w{2,32}$`)                                             // 非 module 类型源码的普通名称（2-32 位 word）
	REGEX_SOURCE_SORT       = regexp.MustCompile(`^(rowid|name|last_modified_date) (asc|desc)$`)           // 源码排序条件
)

func HandleSource(w http.ResponseWriter, r *http.Request) {
	var (
		data       interface{}
		returnless bool
		err        error
	)
	switch r.Method {
	case http.MethodPost:
		if _, bulk := r.URL.Query()["bulk"]; !bulk {
			err = handleSourcePost(r)
		} else {
			err = handleSourceBulkPost(r)
		}
	case http.MethodDelete:
		err = handleSourceDelete(r)
	case http.MethodPut:
		data, err = handleSourcePut(r)
	case http.MethodGet:
		data, returnless, err = handleSourceGet(w, r)
	case "EVAL":
		handleSourceEval(w, r)
		returnless = true
	default:
		Error(w, http.StatusMethodNotAllowed)
		return
	}
	if err != nil {
		Error(w, err)
		return
	}
	if !returnless {
		Success(w, data)
	}
}

func handleSourcePost(r *http.Request) error {
	// 获取 source 对象
	var source model.Source
	if err := util.UnmarshalWithIoReader(r.Body, &source); err != nil {
		return err
	}

	// 校验类型与名称
	if err := util.ValidateString(source.Type, REGEX_SOURCE_TYPE_ENUM, "type must be module, controller, daemon, crontab, template or resource"); err != nil {
		return err
	}
	if source.Type == "module" {
		if err := util.ValidateString(source.Name, REGEX_MODULE_NAME, "name is required, it must be a string that matches /(node_modules/)?[A-Za-z0-9_]{2,32}/"); err != nil {
			return err
		}
	} else {
		if err := util.ValidateString(source.Name, REGEX_SOURCE_PLAIN_NAME, "name is required, it must be a string that matches /[A-Za-z0-9_]{2,32}/"); err != nil {
			return err
		}
	}
	// 校验 active 必须为 false，不支持在创建过程中直接激活
	if source.Active {
		return errors.New("active must be false")
	}
	// 校验 url 不能重复
	if source.Type == "controller" || source.Type == "resource" {
		var count int
		if err := internal.Db.QueryRow("select count(1) from source where type = ? and url = ? and name != ?", source.Type, source.Url, source.Name).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("url already exists")
		}
	}
	// 校验 cron 表达式
	if source.Type == "crontab" {
		if _, err := util.ParseCron(source.Cron); err != nil {
			return err
		}
	}
	// 校验 name 和 type 不能重复
	{
		var count int
		if err := internal.Db.QueryRow("select count(1) from source where name = ? and type = ?", source.Name, source.Type).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("source already exists")
		}
	}

	// 新增
	if _, err := internal.Db.Exec("insert into source (name, type, lang, content, compiled, active, method, url, cron, tag, last_modified_date) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now', 'localtime'))", source.Name, source.Type, source.Lang, source.Content, source.Compiled, source.Active, source.Method, source.Url, source.Cron, source.Tag); err != nil {
		return err
	}

	return nil
}

func handleSourceBulkPost(r *http.Request) error {
	// 将请求入参转换为 source 对象数组
	var sources []model.Source
	if err := util.UnmarshalWithIoReader(r.Body, &sources); err != nil {
		return err
	}
	if len(sources) == 0 {
		return errors.New("nothing was added or modified")
	}

	// 批量新增或修改
	stmt, err := internal.Db.Prepare("insert or replace into source (rowid, name, type, lang, content, compiled, active, method, url, cron, tag, last_modified_date) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, source := range sources {
		if source.Name == "" || source.Type == "" {
			continue
		}
		if _, err = stmt.Exec(source.Id, source.Name, source.Type, source.Lang, source.Content, source.Compiled, source.Active, source.Method, source.Url, source.Cron, source.Tag, source.LastModifiedDate.String()); err != nil {
			return err
		}
	}

	cache.Route.Init()
	// 批量导入后，需要清空 module 缓存以重建
	cache.Module.Clear()
	// 启动守护任务
	internal.RunDaemons("")
	// 启动定时任务
	internal.RunCrontabs("")

	return nil
}

func handleSourceDelete(r *http.Request) error {
	r.ParseForm()
	name, stype := r.Form.Get("name"), r.Form.Get("type")
	if name == "" {
		return errors.New("name is required")
	}
	if stype == "" {
		return errors.New("type is required")
	}

	res, err := internal.Db.Exec("delete from source where name = ? and type = ?", name, stype)
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return errors.New("source does not exist")
	}

	// 删除路由
	if stype == "controller" {
		cache.Route.Remove(name) // 直接删除该 controller 的路由
	}

	return nil
}

func handleSourcePut(r *http.Request) (interface{}, error) {
	// 获取 source 对象
	var record map[string]interface{}
	if err := util.UnmarshalWithIoReader(r.Body, &record); err != nil {
		return nil, err
	}

	name, stype, url, cron, status, mdate := record["name"], record["type"], record["url"], record["cron"], record["status"], record["last_modified_date"]
	// 校验类型和名称
	if name == nil {
		return nil, errors.New("name is required")
	}
	if stype == nil {
		return nil, errors.New("type is required")
	}
	// 校验 url 不能重复
	if url != nil && (stype == "controller" || stype == "resource") {
		var count int
		if err := internal.Db.QueryRow("select count(1) from source where type = ? and url = ? and active = true and name != ?", stype, url, name).Scan(&count); err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("url already exists")
		}
	}
	// 校验 cron 表达式
	if cron != nil && stype == "crontab" {
		c, ok := cron.(string)
		if !ok {
			return nil, errors.New("cron must be a string")
		}
		if _, err := util.ParseCron(c); err != nil {
			return nil, err
		}
	}
	// 校验最后修改时间（版本号）
	if mdate != nil {
		var rdate string
		if err := internal.Db.QueryRow("select last_modified_date from source where name = ? and type = ?", name, stype).Scan(&rdate); err != nil {
			return nil, err
		}
		if rdate == "" {
			return nil, errors.New("source does not exist")
		}
		if mdate != strings.Replace(strings.Replace(rdate, "T", " ", 1), "Z", "", 1) {
			return nil, errors.New("source version conflict")
		}
	}

	// 初始化修改字段
	sets, params := "", []interface{}{}
	for _, c := range []string{"content", "compiled", "active", "method", "url", "cron", "tag"} {
		if v, ok := record[c]; ok {
			sets += ", " + c + " = ?"
			params = append(params, v)
		}
	}
	// 执行修改
	res, err := internal.Db.Exec("update source set last_modified_date = datetime('now', 'localtime')"+sets+" where name = ? and type = ?", append(params, []interface{}{name, stype}...)...)
	if err != nil {
		return nil, err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return nil, errors.New("source does not existed")
	}

	// 查询更新后的记录
	var source model.Source
	if err := internal.Db.QueryRow("select name, type, lang, active, method, url, cron, tag, last_modified_date from source where name = ? and type = ?", name, stype).Scan(&source.Name, &source.Type, &source.Lang, &source.Active, &source.Method, &source.Url, &source.Cron, &source.Tag, &source.LastModifiedDate); err != nil {
		return nil, err
	}

	switch source.Type {
	case "module":
		if strings.HasPrefix(source.Name, "node_modules/") {
			cache.Module.Remove(source.Name[13:]) // 删除缓存
		} else {
			cache.Module.Remove("./" + source.Name)
		}
	case "controller":
		if source.Active {
			// 更新路由
			cache.Route.Set(source.Name, source.Url) // 更新路由
		} else {
			// 删除路由
			cache.Route.Remove(source.Name)
		}
		cache.Controller.Remove(source.Name) // 删除缓存
		cache.Module.Remove("./controller/" + source.Name)
	case "crontab":
		id, ok := cache.Crontab.Get(source.Name)
		if !ok && source.Active {
			internal.RunCrontabs(source.Name) // 启动 crontab
		}
		if ok && !source.Active {
			internal.Crontab.Remove(id)
			cache.Crontab.Remove(source.Name) // 删除缓存
		}
		cache.Module.Remove("./crontab/" + source.Name)
	case "daemon":
		if source.Active {
			if _, exists := cache.Daemon.Get(source.Name); !exists && status == "true" {
				internal.RunDaemons(source.Name) // 启动
			}
			if worker, exists := cache.Daemon.Get(source.Name); exists && status == "false" {
				worker.Interrupt("Daemon stopped") // 停止，停止后会自动清理缓存，见 RunDaemons 方法的 defer 实现
			}
		}
		cache.Module.Remove("./daemon/" + source.Name)
	}

	return map[string]interface{}{
		"last_modified_date": source.LastModifiedDate,
	}, nil
}

func handleSourceGet(w http.ResponseWriter, r *http.Request) (interface{}, bool, error) {
	// 解析 URL 入参
	p := &util.QueryParams{Values: r.URL.Query()}
	// 初始化查询参数
	name, stype := p.GetOrDefault("name", "%"), p.GetOrDefault("type", "%")
	tag := p.GetOrDefault("tag", "")
	include, exclude := p.GetOrDefault("include", ""), p.GetOrDefault("exclude", "")
	from, size := p.GetIntOrDefault("from", 0), p.GetIntOrDefault("size", 10)
	sort := p.Get("sort")

	// 初始化查询条件
	wheres, params := "name like ? and type like ?", []interface{}{name, stype}
	// 构造标签查询条件
	if tag != "" {
		wheres += " and (1 != 1"
		for _, v := range strings.Split(tag, ",") {
			wheres += " or tag like ?"
			params = append(params, "%"+v+"%")
		}
		wheres += ")"
	}
	// 构造 ID 过滤查询条件
	if include != "" {
		wheres += " and rowid in (" + strings.Repeat(",?", strings.Count(include, ",")+1)[1:] + ")"
		for _, v := range strings.Split(include, ",") {
			params = append(params, v)
		}
	}
	if exclude != "" {
		wheres += " and rowid not in (" + strings.Repeat(",?", strings.Count(exclude, ",")+1)[1:] + ")"
		for _, v := range strings.Split(exclude, ",") {
			params = append(params, v)
		}
	}
	// 初始化排序条件
	orders := "rowid desc"
	if REGEX_SOURCE_SORT.MatchString(sort) {
		orders = sort
	}

	// 初始化返回对象
	var data struct {
		Sources []model.Source `json:"sources"`
		Total   int            `json:"total"`
	}
	data.Sources = make([]model.Source, 0, size)

	// 查询总数
	if err := internal.Db.QueryRow("select count(1) from source where "+wheres, params...).Scan(&data.Total); err != nil { // 调用 QueryRow 方法后，须调用 Scan 方法，否则连接将不会被释放
		return data, false, err
	}

	// 分页查询，默认查询所有字段
	columns := "rowid, name, type, lang, content, compiled, active, method, url, cron, tag, last_modified_date"
	if p.Has("content") { // 不返回 compiled 字段，用于编辑器查询源码
		columns = strings.Replace(columns, ", compiled", ", '' compiled", 1)
	}
	if p.Has("basic") { // 不返回 content、compiled 字段，用于列表查询
		columns = strings.Replace(columns, ", content", ", '' content", 1)
		columns = strings.Replace(columns, ", compiled", ", '' compiled", 1)
	}
	rows, err := internal.Db.Query("select "+columns+" from source where "+wheres+" order by "+orders+" limit ?, ?", append(params, []interface{}{from, size}...)...)
	if err != nil {
		return data, false, err
	}
	defer rows.Close()
	for rows.Next() {
		source := model.Source{}
		if err := rows.Scan(&source.Id, &source.Name, &source.Type, &source.Lang, &source.Content, &source.Compiled, &source.Active, &source.Method, &source.Url, &source.Cron, &source.Tag, &source.LastModifiedDate); err != nil {
			continue
		}
		if source.Type == "daemon" { // 如果是 daemon，写入状态
			_, exists := cache.Daemon.Get(source.Name)
			source.Status = strconv.FormatBool(exists)
		}
		data.Sources = append(data.Sources, source)
	}

	// 是否导出为文件
	if p.Has("bulk") {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.Encode(data.Sources)
		w.Header().Set("Content-Disposition", "attachment; filename=\"sources-"+strconv.FormatInt(time.Now().UnixMilli(), 10)+".json\"")
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len())) // 部分浏览器在下载文件时依赖响应头 Content-Length，如果不返回该属性字段，则下载的文件内容为空
		w.Write(buf.Bytes())
		return nil, true, nil
	}

	return data, false, err
}

func handleSourceEval(w http.ResponseWriter, r *http.Request) {
	script, err := util.StringWithIoReader(r.Body)
	if err != nil {
		Error(w, err)
		return
	}

	// 获取 vm 实例
	var worker *internal.Worker
	select {
	case worker = <-internal.WorkerPool.Channels:
	default:
		Error(w, http.StatusServiceUnavailable)
		return
	}

	// 复用 service 正式接口的上下文（支持 Write、Flush 流式输出，与 controller 中的 ctx.flush() 一致）
	ctx := internal.NewServiceContext(r, w, nil, nil)

	// 流式输出：每行推送一个 JSON 对象（NDJSON），返回是否写入成功
	push := func(v interface{}) bool {
		data, err := json.Marshal(v)
		if err != nil {
			return false
		}
		data = append(data, '\n')
		if _, err := ctx.Write(data); err != nil {
			return false
		}
		return ctx.Flush() == nil
	}

	defer func() {
		if x := recover(); x != nil { // 从内部异常（如执行 crypto module 的原生方法时出现的 panic 异常）中恢复执行，防止服务端因异常而导致接口 pending
			push(map[string]interface{}{
				"datetime": time.Now().Format(time.RFC3339),
				"level":    "error",
				"message":  fmt.Sprint(x),
			})
		}
		worker.Reset()
		internal.WorkerPool.Channels <- worker // 归还实例
	}()

	// 允许最大执行的时间为 60 秒
	timer := time.AfterFunc(60*time.Second, func() {
		worker.Interrupt("service executed timeout")
	})
	defer timer.Stop()

	// 脚本执行完成标记
	var completed atomic.Bool

	// 监听客户端是否主动取消请求
	go func() {
		<-r.Context().Done()   // 客户端主动取消
		if !completed.Load() { // 如果脚本已执行结束，不再中断 goja 运行时，否则中断信号无法被触发和清除（需要 goja 运行时执行指令栈才会触发中断操作），导致回收再复用时直接抛出 "Client cancelled." 的异常
			worker.Interrupt("client cancelled")
		}
	}()

	// eval 接口始终为流式返回，仅推送日志帧
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no") // 禁用反向代理（nginx 等）对响应的缓冲，保证日志实时推送

	// 编译：脚本直接拼接进 IIFE，console 使用 worker 全局注册的 ConsoleClient（eval 期间经 worker.SetLogger 重定向到客户端）
	entry, err := worker.Runtime().RunString(strings.Join([]string{
		"(function () {",
		script,
		"})",
	}, "\n"))
	if err != nil {
		push(map[string]interface{}{
			"datetime": time.Now().Format(time.RFC3339),
			"level":    "error",
			"message":  err.Error(),
		})
		return
	}
	function, ok := goja.AssertFunction(entry)
	if !ok {
		push(map[string]interface{}{
			"datetime": time.Now().Format(time.RFC3339),
			"level":    "error",
			"message":  "eval script is not callable",
		})
		return
	}

	// 日志重定向目标（脚本中的 console.log 经全局 ConsoleClient 实时推送到客户端，worker.Reset 时清除）
	worker.SetLogger(func(level string, args ...goja.Value) {
		parts := make([]string, 0, len(args))
		for _, a := range args {
			if a == goja.Undefined() { // 保持 undefined 语义，避免与 null 混淆
				parts = append(parts, "undefined")
				continue
			}
			if v, e := util.ExportGojaValue(a); e == nil {
				if s, ok := v.(string); ok { // 字符串原样展示，其他类型 JSON 序列化
					parts = append(parts, s)
					continue
				}
				if b, e := json.Marshal(v); e == nil {
					parts = append(parts, string(b))
				} else {
					parts = append(parts, fmt.Sprint(v))
				}
			} else {
				parts = append(parts, a.String())
			}
		}
		push(map[string]interface{}{
			"datetime": time.Now().Format(time.RFC3339),
			"level":    level,
			"message":  strings.Join(parts, " "),
		})
	})

	// 执行
	_, err = worker.EventLoop().Run(func() (goja.Value, error) {
		return function(nil)
	})

	// 标记脚本执行完成
	completed.Store(true)

	if err != nil {
		push(map[string]interface{}{
			"datetime": time.Now().Format(time.RFC3339),
			"level":    "error",
			"message":  err.Error(),
		})
		return
	}
}
