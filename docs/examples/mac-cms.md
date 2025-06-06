# Video-on-Demand website based on [MAC CMS](ttps://github.com/magicblack/maccms10/blob/master/%E8%AF%B4%E6%98%8E%E6%96%87%E6%A1%A3/API%E6%8E%A5%E5%8F%A3%E8%AF%B4%E6%98%8E.txt)

1. Create a controller with url `/service/api/mac/cms` and method `Get`.
    ```typescript
    //?name=MacCms8Api&type=controller&url=api/mac/cms&method=GET&tag=maccms
    export const api = new class MacCms8Api {
        private endpoint: string

        private table: "vod" | "art" | "actor" | "role" | "website"

        /**
        * 苹果 CMS 8 API
        * 
        * @params endpoint 资源地址
        * @params type 资源类型：vod（视频）、art（文章）、actor（演员）、role（角色）、website（网站）
        */
        constructor(endpoint: string = "https://api.xinlangapi.com/xinlangapi.php") {
            this.endpoint = endpoint
            this.table = "vod"
        }

        /**
        * 查询
        * 
        * @params wd 搜索关键词
        * @params t 类型 ID
        * @params pg 页数，从 1 开始
        * @params h 更新时间在最近多少小时内
        * @params ids 影片 ID 集合
        * @params ac 模式：videolist（列表，默认）、detail（详细信息）
        * @params at 输出格式：xml、json（默认）
        */
        public query(wd?: string, t?: string, pg?: number, h?: number, ids?: string[], ac?: "videolist" | "detail", at?: "xml") {
            return $native("http")().request("GET", this.endpoint + "/provide/" + this.table + "?" + [
                wd && "wd=" + encodeURIComponent(wd),
                t && "t=" + t,
                pg && "pg=" + pg,
                ids && "ids=" + ids.join(","),
                ac && "ac=" + ac,
                at && "at=" + at,
            ].filter(i => i).join("&"))
        }
    }(
        "https://api.xinlangapi.com/xinlangapi.php", // 新浪资源
        // "https://api.apibdzy.com/api.php", // 百度云资源
        // "https://api.wujinapi.com/api.php", // 无尽资源
        // "https://www.hongniuzy2.com/api.php", // 红牛
    )

    export default (app => app.run.bind(app))(new class {
        public run(ctx: ServiceContext) {
            const params = ctx.getURL().params
            //@ts-ignore
            return api.query(params.wd?.[0], params.t?.[0], params.pg?.[0], params.h?.[0], params.ids, params.ac?.[0], params.at?.[0]).data
        }
    })
    ```

2. Create a resource with url `/resource/mac/cms`.
    ```html
    //?name=MacCms8View&type=resource&lang=html&url=mac/cms&tag=maccms
    <!DOCTYPE html>
    <html>

    <head>
        <meta charset="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0, viewport-fit=cover" />
        <title>My Vod</title>
        <style>
            body, p, h3 {
                margin: 0;
            }
            body {
                margin: 12px 8px 8px 8px;
            }
            .search-area {
                display: flex;
            }
            .search-area > input {
                flex-grow: 1;
                border: 1px #d9d9d9 solid;
                border-right: none;
                outline-style: none;
                padding: 0 8px;
            }
            .search-area > button {
                padding: 0 1rem;
                height: 2rem;
                background-color: #fff;
                border: 1px #d9d9d9 solid;
                cursor: pointer;
            }
            .search-area > button:hover {
                border: 1px #000000 solid;
            }
            .search-area > button:active {
                background-color: #eaeaea;
            }
            .types-area {
                display: flex;
                flex-wrap: wrap;
                column-gap: 8px; row-gap: 4px;
                margin-top: 12px;
            }
            .types-area > a {
                cursor: pointer;
            }
            .types-area > a.active {
                font-weight: bold;
            }
            .medias-area {
                margin-top: 12px;
                display: grid;
                grid-template-columns: repeat(auto-fill, 120px);
                justify-content: space-around;
                row-gap: 4px;
            }
            .medias-area > .media {
                width: 100%;
                cursor: pointer;
            }
            .medias-area > .media > img {
                width: 100%;
                aspect-ratio: 0.75;
                object-fit: cover;
                height: -webkit-fill-available;
            }
            .medias-area > .media > p {
                margin: 0;
                text-align: center;
                white-space: nowrap;
                overflow: hidden; text-overflow: ellipsis;
            }
            .loading-area {
                width: 100%;
                text-align: center;
                color: #ccc;
                margin: 12px 0;
            }
            .detail-area {
                position: fixed; top: 0; left: 0;
                width: 100%; height: 100vh;
                background: white;
                display: flex; flex-direction: column;
            }
            .back-area {
                margin: 0 8px;
                padding: 12px 12px 12px 0;
                width: fit-content;
                display: flex; align-items: center;
                cursor: pointer;
                font-weight: bold;
            }
            .back-area > svg {
                width: 1em;
                height: 1em;
                vertical-align: middle;
                fill: currentColor;
                overflow: hidden;
            }
            .info-area {
                margin: 0 8px;
                height: 320px;
                display: flex;
            }
            .info-area > img {
                height: 100%;
                aspect-ratio: 0.75;
            }
            .info-area > div {
                margin: 0 12px;
                flex-grow: 1; width: 0;
            }
            .info-area > div > p {
                word-break: break-all;
            }
            .videos-area {
                margin-top: 8px;
                padding: 0 8px 8px 8px;
                flex-grow: 1;
                overflow-y: auto;
            }
            .videos-area > div + div {
                margin-top: 8px;
            }
        </style>
    </head>

    <body v-clock>
        <div class="search-area">
            <input type="text" v-model="keyword"></input>
            <button @click="fetch(true)">Search</button>
        </div>
        <div class="types-area">
            <a v-for="type in types" @click="() => { typeId = type.id; pageIndex = 1; fetch(true); }" :class="type.id === typeId ? 'active' : ''">
                {{ type.name }}
            </a>
        </div>
        <div class="medias-area">
            <div class="media" v-for="media in medias" @click="detail(media)">
                <img :src="media.pic" :title="media.desc" />
                <p :title="media.name">{{ media.name }}</p>
            </div>
        </div>
        <div class="loading-area" ref="loading" v-show="fetching || !ending"></div>
        <div class="detail-area" v-show="detailing">
            <div class="back-area" @click="close()">
                <svg class="icon" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" p-id="1754"><path d="M538.288 198.624l-11.312-11.312a16 16 0 0 0-22.64 0L187.312 504.336a16 16 0 0 0 0 22.64L504.336 844a16 16 0 0 0 22.64 0l11.312-11.312a16 16 0 0 0 0-22.624l-294.4-294.4 294.4-294.4a16 16 0 0 0 0-22.64z" fill="#000000" p-id="1755"></path></svg>
                Back
            </div>
            <div class="info-area">
                <img :src="media.pic" />
                <div>
                    <h3>{{ media.name }}</h3>
                    <p style="margin-top: 12px;">{{ media.desc }}</p>
                </div>
            </div>
            <div class="videos-area">
                <div v-for="uri in media.uris">
                    <p>{{ uri[0] }}</p>
                    <p><a :href="uri[1]" target="_blank">{{ uri[1] }}</a></p>
                </div>
            </div>
        </div>
        <script src="https://unpkg.com/vue@3.4.6/dist/vue.global.prod.js"></script>
        <script>
            const app = Vue.createApp({
                data() {
                    return {
                        observer: undefined,
                        types: [], medias: [],
                        fetching: false, ending: false,
                        pageIndex: 0, typeId: "", keyword: "",
                        detailing: false, media: {},
                    }
                },
                methods: {
                    fetch(reset = false) {
                        if (this.fetching) {
                            return
                        }
                        if (reset) {
                            this.pageIndex = 0
                            this.medias = []
                            this.$refs.loading.innerText = "Loading..."
                        }
                        this.fetching = true
                        fetch(`/service/api/mac/cms?t=${this.typeId}&ac=videolist&pg=${this.pageIndex + 1}&wd=${this.keyword}`).then(i => i.json()).then(data => {
                            this.pageIndex = Number(data.page)
                            this.medias.push(...data.list.map(i => {
                                return {
                                    name: i.vod_name,
                                    desc: i.vod_content,
                                    pic: i.vod_pic,
                                    uris: i.vod_play_url.split(/(?:#|\$\$\$)/).map(i => i.split("$")),
                                }
                            }))
                            if (this.pageIndex < data.pagecount) {
                                // this.$nextTick(() => this.observer.observe(document.querySelector(".medias > .media:last-child"))) // 监听最后一个元素
                                this.observer.observe(this.$refs.loading) // 监听底部 loading 元素
                                this.ending = false
                                return
                            }
                            this.ending = true
                        }).catch(e => {
                            this.$refs.loading.innerText = e.message
                        }).finally(() => {
                            this.fetching = false
                        })
                    },
                    detail(media) {
                        this.detailing = true
                        this.media = media
                        document.body.style.overflow = "hidden"
                    },
                    close() {
                        this.detailing = false
                        document.body.style.overflow = "auto"
                    },
                },
                mounted() {
                    const that = this
                    this.observer = new IntersectionObserver(function(entries) {
                        const entry = entries[0]
                        if(entry.isIntersecting) { // 如果已进入视图，停止监听，并且生成新的元素
                            this.unobserve(entry.target)
                            that.fetch(false)
                        }
                    })
                    fetch("/service/api/mac/cms").then(i => i.json()).then(data => {
                        that.types = data.class.map(i => {
                            return {
                                id: i.type_id,
                                name: i.type_name,
                            }
                        })
                        that.typeId = that.types[0].id
                        that.fetch(true)
                    })
                    
                },
            })
            app.mount(document.body)
        </script>
    </body>

    </html>
    ```
