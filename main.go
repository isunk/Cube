package main

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"fmt"
	"net/http"
	"os"

	"cube/internal"
	"cube/internal/cache"
	"cube/internal/config"
	"cube/internal/handler"
	"cube/internal/log"

	"github.com/quic-go/quic-go/http3"
)

//go:embed web/*
var web embed.FS

func init() {
	// 解析命令行参数：config 包的 init 只定义 flag 不 Parse（详见 config.go 注释），必须在 InitDb/InitWorkerPool 等依赖 config 字段的初始化之前完成解析
	config.Parse()

	// 初始化数据库
	internal.InitDb()

	// 初始化日志文件
	log.Init()

	// 初始化缓存
	if err := cache.Init(internal.Db); err != nil {
		panic(err)
	}

	// 初始化虚拟机池
	internal.InitWorkerPool()

	// 初始化路由
	handler.InitHandle(&web)
}

func main() {
	// 监控当前进程的内存和 cpu 使用率
	go internal.RunMonitor()

	// 启动守护任务
	internal.RunDaemons("")

	// 启动定时服务
	internal.RunCrontabs("")

	// 启动服务
	serve()
}

func serve() {
	if !config.Secure {
		// 启用 HTTP
		fmt.Println("Server has started on http://127.0.0.1:" + config.Port + " 🚀")
		if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
			log.Fatal(err)
		}
		return
	}

	c := &tls.Config{}

	if config.ClientCertVerify {
		// 设置对服务端证书校验
		c.ClientAuth = tls.RequireAndVerifyClientCert
		b, err := os.ReadFile("./ca.crt")
		if err != nil {
			log.Fatal("failed to read ca.crt: ", err)
		}
		c.ClientCAs = x509.NewCertPool()
		if !c.ClientCAs.AppendCertsFromPEM(b) {
			log.Fatal("failed to parse ca.crt: no certificates found")
		}
	}

	fmt.Println("Server has started on https://127.0.0.1:" + config.Port + " 🚀")

	if !config.Http3 {
		// 启用 HTTPS 或 HTTP/2
		server := &http.Server{
			Addr:      ":" + config.Port,
			TLSConfig: c,
		}
		if err := server.ListenAndServeTLS(config.ServerCert, config.ServerKey); err != nil {
			log.Fatal(err)
		}
		return
	}

	// 启用 HTTP/3
	server := &http3.Server{
		Addr:      ":" + config.Port,
		TLSConfig: c,
	}
	if err := server.ListenAndServeTLS(config.ServerCert, config.ServerKey); err != nil {
		log.Fatal(err)
	}
}
