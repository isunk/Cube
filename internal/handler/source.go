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
	"time"

	. "cube/internal"
	"cube/internal/model"
	"cube/internal/util"

	"github.com/dop251/goja"
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
		err = handleSourcePut(r)
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

	// 校验类型
	if ok, _ := regexp.MatchString("^(module|controller|daemon|crontab|template|resource)$", source.Type); !ok {
		return errors.New("type must be module, controller, daemon, crontab, template or resource")
	}
	// 校验名称
	if source.Type == "module" {
		if ok, _ := regexp.MatchString("^(node_modules/)?\\w{2,32}$", source.Name); !ok {
			return errors.New("name is required, it must be a string that matches /(node_modules/)?[A-Za-z0-9_]{2,32}/")
		}
	} else {
		if ok, _ := regexp.MatchString("^\\w{2,32}$", source.Name); !ok {
			return errors.New("name is required, it must be a string that matches /[A-Za-z0-9_]{2,32}/")
		}
	}
	// 校验 active 必须为 false，不支持在创建过程中直接激活
	if source.Active {
		return errors.New("active must be false")
	}
	// 校验 url 不能重复
	if source.Type == "controller" || source.Type == "resource" {
		var count int
		if err := Db.QueryRow("select count(1) from source where type = ? and url = ? and name != ?", source.Type, source.Url, source.Name).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("url already existed")
		}
	}
	// 校验 cron 表达式
	if source.Type == "crontab" {
		if _, err := ParseCron(source.Cron); err != nil {
			return err
		}
	}
	// 校验 name 和 type 不能重复
	{
		var count int
		if Db.QueryRow("select count(1) from source where name = ? and type = ?", source.Name, source.Type).Scan(&count); count > 0 {
			return errors.New("source already existed")
		}
	}

	// 新增
	if _, err := Db.Exec("insert into source (name, type, lang, content, compiled, active, method, url, cron, tag, last_modified_date) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now', 'localtime'))", source.Name, source.Type, source.Lang, source.Content, source.Compiled, source.Active, source.Method, source.Url, source.Cron, source.Tag); err != nil {
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
		return errors.New("nothing added or modified")
	}

	// 批量新增或修改
	stmt, err := Db.Prepare("insert or replace into source (rowid, name, type, lang, content, compiled, active, method, url, cron, tag, last_modified_date) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
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

	Cache.InitRoutes()
	// 批量导入后，需要清空 module 缓存以重建
	Cache.Modules = make(map[string]*goja.Program)
	// 启动守护任务
	RunDaemons("")
	// 启动定时任务
	RunCrontabs("")

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

	res, err := Db.Exec("delete from source where name = ? and type = ?", name, stype)
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return errors.New("source does not existed")
	}

	// 删除路由
	if stype == "controller" {
		delete(Cache.Routes, name)
	}

	return nil
}

func handleSourcePut(r *http.Request) error {
	// 获取 source 对象
	var record map[string]interface{}
	if err := util.UnmarshalWithIoReader(r.Body, &record); err != nil {
		return err
	}

	// 校验类型和名称
	name, stype, url, cron, status := record["name"], record["type"], record["url"], record["cron"], record["status"]
	if name == nil {
		return errors.New("name is required")
	}
	if stype == nil {
		return errors.New("type is required")
	}
	// 校验 url 不能重复
	if url != nil && (stype == "controller" || stype == "resource") {
		var count int
		if err := Db.QueryRow("select count(1) from source where type = ? and url = ? and active = true and name != ?", stype, url, name).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("url already existed")
		}
	}
	// 校验 cron 表达式
	if cron != nil && stype == "crontab" {
		if _, err := ParseCron(cron.(string)); err != nil {
			return err
		}
	}

	// 修改
	setsen, params := "", []interface{}{}
	for _, c := range []string{"content", "compiled", "active", "method", "url", "cron", "tag"} {
		if v, ok := record[c]; ok {
			setsen += ", " + c + " = ?"
			params = append(params, v)
		}
	}
	res, err := Db.Exec("update source set last_modified_date = datetime('now', 'localtime')"+setsen+" where name = ? and type = ?", append(params, []interface{}{name, stype}...)...)
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return errors.New("source does not existed")
	}

	// 查询更新后的记录
	var source model.Source
	if err := Db.QueryRow("select name, type, lang, active, method, url, cron, tag from source where name = ? and type = ?", name, stype).Scan(&source.Name, &source.Type, &source.Lang, &source.Active, &source.Method, &source.Url, &source.Cron, &source.Tag); err != nil {
		return err
	}

	switch source.Type {
	case "module":
		if strings.HasPrefix(source.Name, "node_modules/") {
			delete(Cache.Modules, source.Name[13:]) // 删除缓存
		} else {
			delete(Cache.Modules, "./"+source.Name)
		}

	case "controller":
		if source.Active {
			Cache.SetRoute(source.Name, source.Url) // 更新路由
		} else {
			delete(Cache.Routes, source.Name) // 删除路由
		}
		delete(Cache.Controllers, source.Name) // 删除缓存
		delete(Cache.Modules, "./controller/"+source.Name)
	case "crontab":
		id, ok := Cache.Crontabs[source.Name]
		if !ok && source.Active {
			RunCrontabs(source.Name) // 启动 crontab
		}
		if ok && !source.Active {
			Crontab.Remove(id)                  // // 停止 crontab
			delete(Cache.Crontabs, source.Name) // 删除缓存
		}
		delete(Cache.Modules, "./crontab/"+source.Name)
	case "daemon":
		if source.Active {
			if Cache.Daemons[source.Name] == nil && status == "true" {
				RunDaemons(source.Name) // 启动
			}
			if Cache.Daemons[source.Name] != nil && status == "false" {
				Cache.Daemons[source.Name].Interrupt("Daemon stopped") // 停止，停止后会自动清理缓存，见 RunDaemons 方法的 defer 实现
			}
		}
		delete(Cache.Modules, "./daemon/"+source.Name)
	}

	return nil
}

func handleSourceGet(w http.ResponseWriter, r *http.Request) (interface{}, bool, error) {
	// 解析 URL 入参
	p := &util.QueryParams{Values: r.URL.Query()}
	name, stype := p.GetOrDefault("name", "%"), p.GetOrDefault("type", "%")
	tag := p.GetOrDefault("tag", "")
	from, size := p.GetIntOrDefault("from", 0), p.GetIntOrDefault("size", 10)
	sort := p.Get("sort")

	// 初始化排序字段
	orders := "rowid desc"
	if ok, _ := regexp.MatchString("^(rowid|name|last_modified_date) (asc|desc)$", sort); ok {
		orders = sort
	}

	// 初始化查询条件
	condition, params := "name like ? and type like ?", []interface{}{name, stype}

	// 构造标签查询条件
	if tag != "" {
		condition += " and (1 != 1"
		for _, v := range strings.Split(tag, ",") {
			condition += " or tag like ?"
			params = append(params, "%"+v+"%")
		}
		condition += " )"
	}

	// 初始化返回对象
	var data struct {
		Sources []model.Source `json:"sources"`
		Total   int            `json:"total"`
	}
	data.Sources = make([]model.Source, 0, size)

	// 查询总数
	if err := Db.QueryRow("select count(1) from source where "+condition, params...).Scan(&data.Total); err != nil { // 调用 QueryRow 方法后，须调用 Scan 方法，否则连接将不会被释放
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
	rows, err := Db.Query("select "+columns+" from source where "+condition+" order by "+orders+" limit ?, ?", append(params, []interface{}{from, size}...)...)
	if err != nil {
		return data, false, err
	}
	defer rows.Close()
	for rows.Next() {
		source := model.Source{}
		rows.Scan(&source.Id, &source.Name, &source.Type, &source.Lang, &source.Content, &source.Compiled, &source.Active, &source.Method, &source.Url, &source.Cron, &source.Tag, &source.LastModifiedDate)
		if source.Type == "daemon" { // 如果是 daemon，写入状态
			source.Status = fmt.Sprintf("%v", Cache.Daemons[source.Name] != nil)
		}
		data.Sources = append(data.Sources, source)
	}

	if p.Has("bulk") { // 导出为文件
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.Encode(data.Sources)
		w.Header().Set("Content-Disposition", "attachment; filename=\"sources-"+strconv.FormatInt(time.Now().UnixMilli(), 10)+".json\"")
		w.Header().Set("Content-Length", fmt.Sprint(buf.Len())) // 部分浏览器在下载文件时依赖响应头 Content-Length，如果不返回该属性字段，则下载的文件内容为空
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
	var worker *Worker
	select {
	case worker = <-WorkerPool.Channels:
	default:
		Error(w, http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if x := recover(); x != nil {
			Error(w, x)
		}
		worker.Reset()
		WorkerPool.Channels <- worker
	}()

	// 允许最大执行的时间为 60 秒
	timer := time.AfterFunc(60*time.Second, func() {
		worker.Interrupt("service executed timeout")
	})
	defer timer.Stop()

	// 脚本执行完成标记
	completed := false

	// 监听客户端是否主动取消请求
	go func() {
		<-r.Context().Done() // 客户端主动取消
		if !completed {      // 如果脚本已执行结束，不再中断 goja 运行时，否则中断信号无法被触发和清除（需要 goja 运行时执行指令栈才会触发中断操作），导致回收再复用时直接抛出 "Client cancelled." 的异常
			worker.Interrupt("client cancelled")
		}
	}()

	// 编译
	entry, _ := worker.Runtime().RunString(strings.Join([]string{
		"(function () {",
		"const console = { __logs__: [], log: function(...args) { this.__logs__.push(['log', new Date(), ...args]) }, };",
		script,
		";return { logs: console.__logs__, };",
		"})",
	}, "\n"))
	function, _ := goja.AssertFunction(entry)

	// 执行
	value, err := worker.EventLoop().Run(func() (goja.Value, error) {
		return function(nil)
	})

	// 标记脚本执行完成
	completed = true

	if err != nil {
		Error(w, err)
		return
	}

	data, err := util.ExportGojaValue(value)
	if err != nil {
		Error(w, err)
		return
	}

	Success(w, data)
}
