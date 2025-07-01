# Makefile 中命令需要转义字符 `$` -> `$$`、`'` -> `'\''`

.ONESHELL: # target 中的每行命令使用同一个 shell，用于支持多行命令

# 运行
run: config # 从代码中运行
	@go run .

watch: # 监听当前目录下的相关文件变动，实时编译、运行
	@gowatch -o ./cube

kill:
	@ps -ef | grep -P "/cube|/gowatch" | grep -v "grep" | awk '{print $$2}' | xargs kill -9

test: # 执行（非基准）单元测试用例
	@go test -v ./... | grep -v 'no test files'

bench: # 执行（基准）单元测试用例
	@go test -v -run=^$ -benchmem -bench=. ./... | grep -v 'no test files'

# 编译
build: clean config # 默认不使用 CDN 资源、不使用 UPX 压缩，即 make build CDN=0 UPX=0
	@
	# 编译（删除符号、调试信息）
	go build -ldflags "-s -w" .
	# 是否使用 UPX 压缩
	if [ "$(UPX)" = "1" ]; then
		if [ "$(shell uname)" = "Linux" ]; then
			upx -9 -q -o cubemin cube
		else
			upx -9 -q -o cubemin.exe cube.exe
		fi
	fi

config:
	@
	if [ "$(CDN)" = "1" ]; then # 使用 CDN 资源
	    sed -i "s#window.location.origin + \"/libs/monaco-editor/$$version/min/vs\"#\"/libs/monaco-editor/$$version/min/vs\"#g" web/editor.html # 由于这里的 URL 需要在 Service Worker 中动态获取，因此需要补充完整的域名
		sed -i 's#"/libs/#"https://cdn.bootcdn.net/ajax/libs/#g' web/*.html
	else # 构建一个不依赖于 CDN 的版本，依赖的 js、css 等库文件将下载至本地 web/libs 目录下
		# 下载除 monaco-editor 外所有 css、js 资源文件
		grep -hor "/libs/[^\"'\'''\'']*" ./web | grep -v "monaco-editor" | sort | uniq | while read uri
		do
			name=$${uri#/libs/}
			if [ -f "web/libs/$$name" ]; then
				continue
			fi
			wget --no-check-certificate "https://cdn.bootcdn.net/ajax/libs/$$name" -P "web/libs/$$(dirname $$name)"
		done
		# 下载 monaco-editor 资源
		export LANG=C.UTF-8
		export version=`grep -horP "monaco-editor/[\d\.]+" ./web | uniq | cut -d "/" -f 2`
		if [ ! -d "./web/libs/monaco-editor/$$version/" ]; then
			mkdir -p "./web/libs/monaco-editor/$$version/"
			wget --no-check-certificate "https://registry.npm.taobao.org/monaco-editor/-/monaco-editor-$$version.tgz"
			tar -zxf "monaco-editor-$$version.tgz" -C "./web/libs/monaco-editor/$$version/" --strip-components 1 "package/min"
			rm monaco-editor-$$version.tgz
		fi
	fi

clean:
	@rm -f cube cubemin cube.exe cubemin.exe

# 开发
tidy: # 安装依赖、删除 go.mod、go.sum 中的无用依赖
	@go mod tidy

update: # 更新依赖
	@go get -u .

wrk: # 性能压测
	@wrk -t8 -c256 -R 20000 -d5s http://127.0.0.1:8090/service/greeting

fmt: # 格式化代码
	@
	# 格式化 .go 文件
	if command -v gofumpt >/dev/null 2>&1; then
		gofumpt -l -w . # 优先使用 gofumpt 格式化代码，安装：go install mvdan.cc/gofumpt@latest
	else
		find ./ -name "*.go" | xargs -I {} go fmt {}
	fi
	# 格式化 .md 文件（去除由空格组成的空行）
	find -name "*.md" | xargs sed -i "s/^[[:space:]]*$$//g"

vet: # 静态代码检查
	@go vet ./...

crt: # 创建 CA 证书和服务端证书
	@
	ls | grep -P 'ca\.(key|crt)' > /dev/null \
		&& echo 'The ca.key or ca.crt already existed, skip.' \
		|| openssl req -new -days 3650 -x509 -nodes -subj "/C=CN/ST=BJ/L=BJ/O=Sunke, Inc./CN=Sunke Root CA" -keyout ca.key -out ca.crt
	bash -c '
		openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=CN/ST=BJ/L=BJ/O=Sunke, Inc./CN=localhost" -out server.csr \
			&& openssl x509 -sha256 -req -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt
	'

ccrt: # 创建客户端证书
	@
	openssl req -newkey rsa:2048 -nodes -keyout client.key -subj "/C=CN/ST=BJ/L=BJ/O=/CN=" -out client.csr \
		&& openssl x509 -sha256 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt
