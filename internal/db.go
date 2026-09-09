package internal

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

var Db *sql.DB

func InitDb() {
	var err error

	Db, err = sql.Open("sqlite", "./cube.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)") // WAL 读写并发、NORMAL 提升写入性能、busy_timeout 5000ms 锁等待，连接建立时对每条连接生效
	if err != nil {
		panic(err)
	}

	_, err = Db.Exec(`
		create table if not exists source (
			name varchar(64) not null,
			type varchar(16) not null,
			lang varchar(16) not null,
			content text not null default '',
			compiled text not null default '',
			active boolean not null default false,
			method varchar(8) not null default '',
			url varchar(64) not null default '',
			cron varchar(16) not null default '',
			tag text not null default '',
			last_modified_date datetime default (datetime('now', 'localtime')),
			primary key(name, type)
		);
	`)
	if err != nil {
		panic(err)
	}
}
