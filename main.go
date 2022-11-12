package main

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"fmt"
	"net/http"
	"os"

	. "cube/internal"
	"cube/internal/config"
	"cube/internal/handler"
	"cube/internal/log"

	"github.com/quic-go/quic-go/http3"
)

//go:embed web/*
var web embed.FS

func init() {
	// 初始化数据库
	InitDb()

	// 初始化日志文件
	log.Init()

	// 初始化缓存
	InitCache()

	// 初始化虚拟机池
	InitWorkerPool()

	// 初始化路由
	handler.InitHandle(&web)
}

func main() {
	// 监控当前进程的内存和 cpu 使用率
	go RunMonitor()

	// 启动守护任务
	RunDaemons("")

	// 启动定时服务
	RunCrontabs("")

	// 启动服务
	serve()
}

func serve() {
	if !config.Secure {
		// 启用 HTTP
		fmt.Println("Server has started on http://127.0.0.1:" + config.Port + " 🚀")
		http.ListenAndServe(":"+config.Port, nil)
		return
	}

	c := &tls.Config{}

	if config.ClientCertVerify {
		// 设置对服务端证书校验
		c.ClientAuth = tls.RequireAndVerifyClientCert
		b, _ := os.ReadFile("./ca.crt")
		c.ClientCAs = x509.NewCertPool()
		c.ClientCAs.AppendCertsFromPEM(b)
	}

	fmt.Println("Server has started on https://127.0.0.1:" + config.Port + " 🚀")

	if !config.Http3 {
		// 启用 HTTPS 或 HTTP/2
		server := &http.Server{
			Addr:      ":" + config.Port,
			TLSConfig: c,
		}
		server.ListenAndServeTLS(config.ServerCert, config.ServerKey)
		return
	}

	// 启用 HTTP/3
	server := &http3.Server{
		Addr:      ":" + config.Port,
		TLSConfig: c,
	}
	server.ListenAndServeTLS(config.ServerCert, config.ServerKey)
}
