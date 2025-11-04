# Http server that supports jpg/png resize, mp4 http-range, zip preview

1. Create a http server.
    ```typescript
    //?name=httpd&type=controller&method=GET&url=httpd/{name}&tag=http
    export default (app => app.run.bind(app))(new class {
        private filec = $native("file")

        private imagec = $native("image")

        private zipc = $native("zip")

        public run(ctx: ServiceContext) {
            return this.parse(decodeURIComponent(ctx.getPathVariables().name).split("!/"), ctx)
        }

        private parse([name, ...subnames]: string[], ctx: ServiceContext) {
            const fileType = this.toFileType(name)
            switch (fileType) {
                case "":
                    return this.filec.list(name)
                case "jpeg": case "png":
                    return this.toImage(name, ctx, fileType)
                case "mp4":
                    return this.toVideo(name, ctx)
                case "zip":
                    if (subnames.length) {
                        return this.toZip(name, subnames, ctx)
                    }
                default:
                    return this.filec.read(name)
            }
        }

        private toFileType(name) {
            if (name === "" || name.at(-1) === "/") {
                return ""
            }
            let fileType = name.match(/\.([^.]+)$/)?.[1]?.toLowerCase()
            if (!fileType) {
                // 根据文件的前 8 个字节来判断文件的类型
                const magic = this.filec.readRange(name, 0, 8).map(i => i.toString(16).padStart(2, "0")).join("").toUpperCase()
                fileType = Object.entries({
                    "FFD8FF__________": "jpeg",
                    "89504E47________": "png",
                    "47494638________": "gif",
                    "504B0304________": "zip",
                    "0000____66747970": "mp4",
                }).find(([p]) => new RegExp("^" + p.replace(/_+/g, s => `[A-F0-9]{${s.length}}`) + "$").test(magic))?.[1]
            }
            return fileType
        }

        private toImage(name: string, ctx: ServiceContext, fileType) {
            const params = ctx.getURL().params
            if (params.width) {
                const output = this.imagec.parse(this.filec.read(name)).resize(
                    Number(params.width.pop() || 1280)
                )
                return fileType === "png" ? output.toPNG() : output.toJPG()
            }
            return this.filec.read(name)
        }

        private toVideo(name: string, ctx: ServiceContext) {
            const fileSize = this.filec.stat(name).size()
            if (fileSize <= 1024 * 1024) {
                return this.filec.read(name)
            }
            // 如果文件大于 1MB，触发范围请求
            const range = ctx.getHeader().Range
            if (!range) {
                return new ServiceResponse(200, { "Accept-Ranges": "bytes", "Content-Length": fileSize + "", "Content-Type": "video/mp4" })
            }
            const ranges = range.substring(6).split("-"),
                start = Number(ranges[0]), end = Math.min(Number(ranges[1]) || (start + 1024 * 1024 - 1), fileSize - 1)
            return new ServiceResponse(206, { "Content-Range": `bytes ${start}-${end}/${fileSize}`, "Content-Length": end - start + 1 + "", "Content-Type": "video/mp4" }, this.filec.readRange(name, start, end - start + 1))
        }

        private toZip(name: string, subnames: string[], ctx: ServiceContext) {
            const entries = this.zipc.read(this.filec.read(name)).getEntries()
            this.filec = {
                list: () => entries.map(i => i.name),
                read: name => entries.find(i => i.name === name)?.getData(),
                //@ts-ignores
                readRange: (name, index, size) => new Uint8Array(entries.find(i => i.name === name)?.getData()?.slice(index, index + size)),
                //@ts-ignores
                stat: name => { return { size: () => entries.find(i => i.name === name)?.getData()?.length }}
            }
            return this.parse(subnames, ctx)
        }
    })
    ```

2. Create an image explorer.
    ```html
    //?name=ImageExplorer&type=resource&lang=html&url=image
    <!DOCTYPE html>
    <html>

    <head>
        <meta charset="UTF-8">
        <title></title>
        <style>
            body {
                margin: 0;
            }
            #app {
                position: relative;
                width: 100%;
            }
            .item {
                position: absolute;
                text-align: center;
            }
        </style>
    </head>

    <body>
        <div id="app"></div>
        <script>
            ({
                path: window.location.hash.substring(1),
                files: [], // 图片队列
                options: null, // 配置：边框大小，图片最大宽度
                columns: null, // 每列图片的高度
                observer: null, // 观察器
                render: function() {
                    const that = this;
                    const name = that.files.pop();
                    if (!name) {
                        return;
                    }
                    const e = document.createElement("img");
                    e.src = `/service/httpd/${that.path}/${name}?width=${that.options.width}`;
                    e.setAttribute("class", "item");
                    e.onload = function() {
                        const minHeight = Math.min(...that.columns), // 找到对应列最小高度的值
                            minHeightIndex = that.columns.indexOf(minHeight); // 找到对应列最小高度的下标
                        e.style.transform = `translate(${minHeightIndex * 100}%, ${minHeight}px)`; // 根据下标进行变换，变换宽度为偏移多少个下标，上下为该下标所有高度
                        document.getElementById("app").appendChild(e);
                        that.columns[minHeightIndex] += e.offsetHeight; // 对应下标增加高度
                        that.observer.observe(e); // 观察该元素用作懒加载
                    };
                    e.onclick = function() {
                        window.open(`/service/httpd/${that.path}/${name}`);
                    };
                },
                mount: function(options) {
                    const that = this;
                    // 保存配置
                    that.options = {
                        width: 140, // 默认每张图片最大宽度 140 px
                        ...options,
                    };
                    // 创建列缓存，每行展示 ${columns.length} 张图片
                    that.columns = [...new Array(Math.floor(window.innerWidth / that.options.width))].map(_ => 0);
                    // 创建观察器
                    that.observer = new IntersectionObserver(function(entries) {
                        const entry = entries[0];
                        if(entry.isIntersecting) { // 如果已进入视图，停止监听，并且生成新的元素
                            this.unobserve(entry.target); // 这里用 this 以指向观察器自己
                            that.render();
                        }
                    });
                    fetch(`/service/httpd/${that.path}`).then(r => r.json()).then(r => {
                        that.files = r.data.filter(i => /\.(jpe?g|png)$/.test(i)).reverse();
                        that.render();
                    });
                }
            }).mount({
                width: Math.floor(window.innerWidth / 2), // 每行展示 2 张图片
            });
        </script>
    </body>

    </html>
    ```

3. You can preview at [`/service/httpd/`](/service/httpd/) and [`/resource/image#/`](/resource/image#/) in browser.
    ```
    # read a.jpg with resized(width = 720)
    /service/httpd/a.jpg?width=720

    # read a.mp4 (size > 1 mb) with http range
    /service/httpd/a.mp4

    # read b.jpg in a.zip
    /service/httpd/a.zip!/b.jpg

    # read all images in a.zip
    /resource/image#a.zip!/
    ```
