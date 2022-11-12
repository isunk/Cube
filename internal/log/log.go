package log

import (
	"log"
	"os"
	"time"
)

func Init() {
	fd, err := os.OpenFile("./cube.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		panic(err)
	}
	log.SetOutput(fd)
	log.SetFlags(log.Lmsgprefix) // 去除日志每行开头自带的时间戳前缀
}

func Log(id int, e ...interface{}) {
	log.Println(append([]interface{}{time.Now().Format("2006-01-02 15:04:05.000"), id, "Log"}, e...)...)
}

func Debug(id int, e ...interface{}) {
	log.Println(append(append([]interface{}{"\033[1;30m" + time.Now().Format("2006-01-02 15:04:05.000"), id, "Debug"}, e...), "\033[m")...)
}

func Info(id int, e ...interface{}) {
	log.Println(append(append([]interface{}{"\033[0;34m" + time.Now().Format("2006-01-02 15:04:05.000"), id, "Info"}, e...), "\033[m")...)
}

func Warn(id int, e ...interface{}) {
	log.Println(append(append([]interface{}{"\033[0;33m" + time.Now().Format("2006-01-02 15:04:05.000"), id, "Warn"}, e...), "\033[m")...)
}

func Error(id int, e ...interface{}) {
	log.Println(append(append([]interface{}{"\033[0;31m" + time.Now().Format("2006-01-02 15:04:05.000"), id, "Error"}, e...), "\033[m")...)
}
