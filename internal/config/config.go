package config

import "flag"

var (
	Count            int
	Port             string
	Secure           bool
	Http3            bool
	ServerKey        string
	ServerCert       string
	ClientCertVerify bool
	IdeAuthorization string
)

func init() {
	// 获取启动参数
	flag.IntVar(&Count, "n", 16, "count of virtual machines") // 定义命令行参数 n，表示虚拟机的个数，返回 Int 类型指针，默认值为 16，其值在 Parse 后会被修改为命令参数指定的值
	flag.StringVar(&Port, "p", "8090", "port to listen")
	flag.BoolVar(&Secure, "s", false, "enable https")
	flag.BoolVar(&Http3, "3", false, "enable http3")
	flag.StringVar(&ServerKey, "k", "server.key", "SSL key")
	flag.StringVar(&ServerCert, "c", "server.crt", "SSL cert")
	flag.BoolVar(&ClientCertVerify, "v", false, "enable client cert verification")
	flag.StringVar(&IdeAuthorization, "a", "", "<username:password> for IDE authorization verification")

	// 不在此处调用 flag.Parse()：init 阶段 Parse 会抢在 testing 框架注册 -test.* flag 之前，导致 go test 报 "flag provided but not defined: -test.testlogfile" 失败；
	// 改由 main 包 init 首行调用 Parse()，生产解析时机不变；测试 binary 不含 main 包，由 testing main 调用 flag.Parse，互不冲突。
}

// Parse 解析命令行参数。必须在 main 包 init 的首行调用（早于 InitDb/InitWorkerPool 等依赖配置的初始化）。
func Parse() {
	flag.Parse()
}
