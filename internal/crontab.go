package internal

import (
	"cube/internal/cache"
	"cube/internal/log"

	"github.com/robfig/cron/v3"
)

var Crontab *cron.Cron // 定时任务

func RunCrontabs(name string) {
	if Crontab == nil { // 首次执行时，先初始化 Crontab
		Crontab = cron.New()
		Crontab.Start()
	}

	if name == "" {
		name = "%"
	}

	rows, err := Db.Query("select name, cron from source where name like ? and type = 'crontab' and active = true", name)
	if err != nil {
		log.Error(0, "failed to query crontabs:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var n, c string
		if err := rows.Scan(&n, &c); err != nil {
			continue
		}

		if _, ok := cache.Crontab.Get(n); ok { // 防止重复添加任务
			continue
		}

		id, err := Crontab.AddFunc(c, func() {
			defer func() {
				if x := recover(); x != nil { // 防止脚本或原生模块 panic 导致进程崩溃
					log.Error(0, "crontab panicked:", x)
				}
			}()

			worker := <-WorkerPool.Channels
			defer func() {
				worker.Reset()
				WorkerPool.Channels <- worker
			}()

			_, err := worker.Run(worker.Runtime().ToValue("./crontab/" + n))
			if err != nil {
				log.Error(worker.Id(), err)
			}
		})
		if err != nil {
			log.Error(0, "failed to parse crontab "+n+":", err)
			continue
		}

		if !cache.Crontab.GetOrAdd(n, id) { // 原子 check-and-add，防止并发重复注册
			Crontab.Remove(id)
		}
	}
}
