package internal

import (
	"cube/internal/cache"
	"cube/internal/log"
)

func RunDaemons(name string) {
	if name == "" {
		name = "%"
	}

	rows, err := Db.Query("select name from source where name like ? and type = 'daemon' and active = true", name)
	if err != nil {
		log.Error(0, "failed to query daemons:", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var n string

		if err := rows.Scan(&n); err != nil {
			continue
		}

		go func(n string) {
			worker := <-WorkerPool.Channels
			// 原子 check-and-add，防止并发 RunDaemons 重复启动同一 daemon
			if _, created := cache.Daemon.GetOrAdd(n, worker); !created {
				worker.Reset()
				WorkerPool.Channels <- worker
				return
			}

			defer func() {
				if x := recover(); x != nil { // 防止脚本或原生模块 panic 导致进程崩溃
					log.Error(worker.Id(), "daemon panicked:", x)
				}
				worker.Reset()
				WorkerPool.Channels <- worker
				cache.Daemon.Remove(n)
			}()

			_, err := worker.Run(worker.Runtime().ToValue("./daemon/" + n))
			if err != nil {
				log.Error(worker.Id(), err)
			}
		}(n)
	}
}
