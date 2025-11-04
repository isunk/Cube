# Video-on-Demand server based on [MAC CMS](ttps://github.com/magicblack/maccms10/blob/master/%E8%AF%B4%E6%98%8E%E6%96%87%E6%A1%A3/API%E6%8E%A5%E5%8F%A3%E8%AF%B4%E6%98%8E.txt), crawler

1. Create a controller with url `/service/vod` and method `Get`.
    ```typescript
    //?name=vodd&type=controller&url=vod&method=GET&tag=vod
    interface Type {
        id: number
        name: string
    }
    
    interface Media {
        id: string
        name?: string
        description?: string
        picture?: string
        uris?: [string, string][]
    }
    
    interface Pageable<T> {
        pageCount: number
        data: T[]
    }
    
    interface VODSourceInterface {
        /**
         * 查询分组
         */
        types(): Type[]
    
        /**
         * 查询媒体列表
         * 
         * @params wd 搜索关键词
         * @params t 类型 ID
         * @params pg 页数，从 1 开始
         */
        medias(wd: string, t: string, pg: number): Pageable<Media>
    
        /**
         * 查询媒体
         * @params id ID
         */
        media(id: string): Media
    }
    
    export class MacCms8VODSource implements VODSourceInterface {
        private endpoint: string
    
        private table = "vod" // vod（视频）、art（文章）、actor（演员）、role（角色）、website（网站）
    
        constructor(endpoint: string) {
            this.endpoint = endpoint
        }
    
        types(): Type[] {
            return $native("http")().request("GET", this.endpoint + "/provide/" + this.table).data.toJson().class.map(i => {
                return {
                    id: i.type_id,
                    name: i.type_name,
                }
            })
        }
    
        medias(wd: string, t: string, pg: number): Pageable<Media> {
            const { pagecount: pageCount, list } = $native("http")().request("GET", this.endpoint + "/provide/" + this.table + "?" + [
                wd && "wd=" + encodeURIComponent(wd), // wd 搜索关键词
                t && "t=" + t, // t 类型 ID
                pg && "pg=" + pg, // pg 页数，从 1 开始
                "", // h 更新时间在最近多少小时内
                "", // ids 影片 ID 集合
                "ac=videolist", // ac 模式：videolist（列表，默认）、detail（详细信息）
                "at=json", // at 输出格式：xml、json（默认）
            ].filter(i => i).join("&")).data.toJson()
            return {
                pageCount,
                data: list.map(i => {
                    return {
                        id: i.vod_id,
                        name: i.vod_name,
                        description: i.vod_content,
                        picture: i.vod_pic,
                        uris: i.vod_play_url.split(/(?:#|\$\$\$)/).map(i => i.split("$")),
                    }
                }),
            }
        }
    
        media(id: string): Media {
            const i = $native("http")().request("GET", `${this.endpoint}/provide/${this.table}?ids=${id}&ac=detail&at=json`).data.toJson().list[0]
            return {
                id: i.vod_id,
                name: i.vod_name,
                description: i.vod_content,
                picture: i.vod_pic,
                uris: i.vod_play_url.split(/(?:#|\$\$\$)/).map(i => i.split("$")),
            }
        }
    }
    
    export class CrawlerVODSource implements VODSourceInterface {
        private endpoint: string
    
        constructor(endpoint: string) {
            this.endpoint = endpoint
        }
    
        types(): Type[] {
            return []
        }
    
        medias(wd: string, t: string, pg: number): Pageable<Media> {
            const raw = this.fetch(`${this.endpoint}/vodtype/2${pg > 1 ? "-" + pg : ""}.html`)
            try {
                return {
                    pageCount: Number(raw.match(/<span class="num">\d+\/(\d+)<\/span>/)[1]),
                    data: [...raw.matchAll(/<a class="stui-vodlist__thumb lazyload" href="([^"]+)" title="([^"]+)" data-original="([^"]+)">/g)].map(i => {
                        return {
                            id: i[1].match(/\/vodplay\/([\d\-]+)\.html/)[1],
                            name: i[2],
                            description: "",
                            picture: i[3],
                        }
                    }),
                }
            } catch(e) {
                return <Pageable<Media>><unknown>raw
            }
        }
    
        media(id: string): Media {
            const raw = this.fetch(`${this.endpoint}/vodplay/${id}.html`)
            try {
                return {
                    id,
                    uris: [
                        ["默认", raw.match(/"(https:[^""]+\.m3u8)"/)[1].replaceAll("\\/", "/")]
                    ],
                }
            } catch (e) {
                return <Media><unknown>raw
            }
        }
    
        fetch(url) {
            return $native("http")().request("GET", url, {
                "User-Agent": "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; Trident/5.0;",
            }).data.toString()
        }
    }
    
    export default (app => app.run.bind(app))(new class {
        public run(ctx: ServiceContext) {
            const params = new Proxy(ctx.getURL().params, {
                get(target: any, property: string) {
                    return target[property]?.[0]
                }
            })
            const channel = this.dispatch(Number(params.ch))
            if (params.pg) {
                return channel.medias(params.wd, params.t, params.pg)
            }
            if (params.id) {
                return channel.media(params.id)
            }
            return channel.types()
        }
    
        private dispatch(id: number) {
            const channels = [
                () => new MacCms8VODSource("https://api.xinlangapi.com/xinlangapi.php"),
                // () => new CrawlerVODSource($native("http")({ isNotFollowRedirect: true }).request("GET", "http://*/").header["Location"]),
            ]
            return (channels[id] ?? channels[0])?.()
        }
    })
    ```

2. Create a resource with url `/resource/vod`.
    ```html
    //?name=vodd&type=resource&lang=html&url=vodd&tag=vod
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
            [v-cloak] {
                display: none;
            }
            .search-area {
                display: flex;
            }
            .search-area>input {
                flex-grow: 1;
                border: 1px #d9d9d9 solid;
                border-right: none;
                outline-style: none;
                padding: 0 8px;
            }
            .search-area>button {
                padding: 0 1rem;
                height: 2rem;
                background-color: #fff;
                border: 1px #d9d9d9 solid;
                cursor: pointer;
            }
            .search-area>button:hover {
                border: 1px #000000 solid;
            }
            .search-area>button:active {
                background-color: #eaeaea;
            }
            .types-area {
                display: flex;
                flex-wrap: wrap;
                column-gap: 8px;
                row-gap: 4px;
                margin-top: 12px;
            }
            .types-area>a {
                cursor: pointer;
            }
            .types-area>a.active {
                font-weight: bold;
            }
            .medias-area {
                margin-top: 12px;
                display: grid;
                grid-template-columns: repeat(auto-fill, 120px);
                justify-content: space-around;
                row-gap: 4px;
            }
            .medias-area>.media {
                width: 100%;
                cursor: pointer;
            }
            .medias-area>.media>img {
                width: 100%;
                aspect-ratio: 0.75;
                object-fit: cover;
                height: -webkit-fill-available;
            }
            .medias-area>.media>p {
                margin: 0;
                text-align: center;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
            }
            .loading-area {
                width: 100%;
                text-align: center;
                color: #ccc;
                margin: 12px 0;
            }
            .detail-area {
                position: fixed;
                top: 0;
                left: 0;
                width: 100%;
                height: 100vh;
                background: white;
                display: flex;
                flex-direction: column;
            }
            .back-area {
                margin: 0 8px;
                padding: 12px 12px 12px 0;
                width: fit-content;
                display: flex;
                align-items: center;
                cursor: pointer;
                font-weight: bold;
            }
            .back-area>svg {
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
            .info-area>img {
                height: 100%;
                aspect-ratio: 0.75;
            }
            .info-area>div {
                margin: 0 12px;
                flex-grow: 1;
                width: 0;
            }
            .info-area>div>p {
                word-break: break-all;
            }
            .videos-area {
                margin-top: 8px;
                padding: 0 8px 8px 8px;
                flex-grow: 1;
                overflow-y: auto;
            }
            .videos-area>div+div {
                margin-top: 8px;
            }
        </style>
    </head>
    
    <body v-cloak>
        <div class="search-area">
            <input type="text" v-model="keyword"></input>
            <button @click="fetch(true)">Search</button>
        </div>
        <div class="types-area">
            <a v-for="type in types" @click="() => { typeId = type.id; pageIndex = 1; fetch(true); }"
                :class="type.id === typeId ? 'active' : ''">
                {{ type.name }}
            </a>
        </div>
        <div class="medias-area">
            <div class="media" v-for="media in medias" @click="detail(media)">
                <img :src="media.picture" :title="media.description" />
                <p :title="media.name">{{ media.name }}</p>
            </div>
        </div>
        <div class="loading-area" ref="loading" v-show="fetching || !ending"></div>
        <div class="detail-area" v-show="detailing">
            <div class="back-area" @click="() => history.back()">
                <svg class="icon" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" p-id="1754">
                    <path
                        d="M538.288 198.624l-11.312-11.312a16 16 0 0 0-22.64 0L187.312 504.336a16 16 0 0 0 0 22.64L504.336 844a16 16 0 0 0 22.64 0l11.312-11.312a16 16 0 0 0 0-22.624l-294.4-294.4 294.4-294.4a16 16 0 0 0 0-22.64z"
                        fill="#000000" p-id="1755"></path>
                </svg>
                Back
            </div>
            <div class="info-area">
                <img :src="media.picture" />
                <div>
                    <h3>{{ media.name }}</h3>
                    <p style="margin-top: 12px;">{{ media.description }}</p>
                </div>
            </div>
            <div class="videos-area">
                <div v-for="uri in media.uris">
                    <p>{{ uri[0] }}</p>
                    <p><a :href="uri[1]" target="_blank">{{ uri[1] }}</a></p>
                </div>
            </div>
        </div>
        <script src="/libs/vue/3.5.18/vue.global.prod.min.js"></script>
        <script>
            const app = Vue.createApp({
                data() {
                    return {
                        observer: undefined,
                        channel: "", types: [], medias: [],
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
                        fetch(`/service/vod?ch=${this.channel}&t=${this.typeId}&pg=${this.pageIndex + 1}&wd=${this.keyword}`).then(i => i.json()).then(({ data: { pageCount, data } }) => {
                            this.pageIndex += 1
                            this.medias.push(...data)
                            if (this.pageIndex < pageCount) {
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
                        history.pushState(null, "", "")
                        this.media = media
                        fetch(`/service/vod?ch=${this.channel}&id=${media.id}`).then(i => i.json()).then(({ data }) => {
                            this.media = {
                                ...media,
                                ...data,
                            }
                            document.body.style.overflow = "hidden"
                        })
                    },
                },
                mounted() {
                    const that = this
                    this.channel = window.location.hash.substring(1)
                    this.observer = new IntersectionObserver(function(entries) {
                        const entry = entries[0]
                        if(entry.isIntersecting) { // 如果已进入视图，停止监听，并且生成新的元素
                            this.unobserve(entry.target)
                            that.fetch(false)
                        }
                    })
                    fetch(`/service/vod?ch=${this.channel}`).then(i => i.json()).then(({ data: types }) => {
                        that.types = types
                        that.typeId = that.types[0].id
                        that.fetch(true)
                    })
                    window.addEventListener("popstate", () => this.detailing = false)
                },
            })
            app.mount(document.body)
        </script>
    </body>
    
    </html>
    ```
