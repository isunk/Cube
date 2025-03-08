# WebRTC

1. Create a html resource.
    ```html
    //?name=Webchat&type=resource&lang=html&url=webchat
    <html>
    
    <head>
        <meta charset="utf-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0, user-scalable=no">
        <style>
            body {
                display: flex; justify-content: center; align-items: center;
            }
            .container {
                width: 720px; height: 600px;
                padding: 0 12px;
                border-radius: 6px; box-shadow: 0 4px 30px rgba(0, 0, 0, 0.1);
                display: flex; flex-direction: column;
                background-color: #EDEDED;
            }
            .container > .title {
                display: flex; justify-content: center;
                color: black;
                font-weight: 600;
                padding: 12px 0 8px 0;
                border-bottom: solid 0.1px #E9E9E9;
            }
            .container > .title > .success {
                color: lightgreen;
            }
            .container > .messages {
                padding: 4px 0;
                display: flex; flex-direction: column; flex-grow: 1;
                overflow-y: auto;
            }
            .container > .messages::-webkit-scrollbar {
                width: 2px;
            }
            .container > .messages::-webkit-scrollbar-thumb {
                background-color: #CCC;
            }
            .container > .messages > div {
                width: fit-content; max-width: calc(100% - 32px);
                border-radius: 8px;
                background-color: white;
                margin: 4px 12px;
            }
            .container > .messages > div.sent {
                align-self: end;
            }
            .container > .messages > div.text {
                padding: 8px;
                word-wrap: break-word;
                white-space: pre-wrap;
            }
            .container > .messages > div.image > img {
                max-height: 96px; min-width: 96px;
                border-radius: 8px;
                object-fit: contain;
            }
            .container > .messages > div.file {
                padding: 8px;
            }
            .container > .messages > div.file > a {
                color: #409EFF;
                text-decoration: none;
            }
            .container > .messages > div.file > span {
                color: grey;
                margin-left: 4px;
            }
            .container > .messages > div.sent.text {
                background-color: #95EC69;
            }
            .container > .sender {
                display: flex; align-items: end;
                border-top: solid 0.1px #F0F0F0;
                padding: 8px;
                background-color: #F7F7F7;
            }
            .container > .sender > textarea {
                border: 0;
                outline: none;
                flex-grow: 1;
                background-color: white;
                border-radius: 2px;
                height: 32px;
                padding: 4px 8px;
                resize: none;
                overflow: hidden;
                line-height: 1.5;
                font-size: 16px;
                margin-right: 6px;
            }
            .container > .sender > div {
                width: 28px; height: 28px;
                padding: 2px;
            }
            .masking {
                position: fixed; top: 0; left: 0; right: 0; bottom: 0;
                background-color: #22222290;
                display: flex; justify-content: center; align-items: center;
            }
            .masking img {
                max-height: 100vh; max-width: 100vw;
            }
            @media screen and (max-width: 768px) {
                body {
                    margin: 0;
                }
                .container {
                    width: 100%;
                    height: -webkit-fill-available;
                    padding: 0;
                    box-shadow: unset;
                    border-radius: 0;
                }
            }
            @media (prefers-color-scheme: dark) {
                .container {
                    background-color: #111111;
                }
                .container > .title {
                    border-bottom: solid 0.1px #101010;
                }
                .container > .messages > div {
                    background-color: #242424;
                }
                .container > .messages > div.sent.text {
                    background-color: #3EB575;
                }
                .container > .sender {
                    background-color: #1E1E1E;
                }
                .container > .sender > textarea {
                    background-color: #292929;
                }
            }
        </style>
    </head>
    
    <body>
        <div id="app" class="container" v-cloak>
            <div class="title">
                <span :class="connected ? 'success' : ''">{{ id }} - {{ conn?.peer || "?" }}</span>
            </div>
            <div class="messages" ref="messages">
                <template v-for="msg in messages" >
                    <div :class="{ 'sent': msg.sent, 'text': true }" v-if="msg.type === 'text'">
                        {{ msg.value }}
                    </div>
                    <div :class="{ 'sent': msg.sent, 'image': true }" v-else-if="/^image/.test(msg.type)" @click="preview(msg.value)">
                        <img :src="msg.value" />
                    </div>
                    <div :class="{ 'sent': msg.sent, 'file': true }" v-else-if="msg.meta">
                        <a :href="msg.value" :download="msg.meta.name">{{ msg.meta.name }}</a>
                        <span>({{ msg.meta.size }}B)</span>
                    </div>
                </template>
            </div>
            <div class="sender">
                <textarea v-model="text" ref="text"></textarea>
                <div @click="submit" v-if="text">
                    <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 20 20"><g fill="none"><path d="M2.721 2.051l15.355 7.566a.5.5 0 0 1 0 .897L2.72 18.08a.5.5 0 0 1-.704-.576l1.969-7.434l-1.97-7.442a.5.5 0 0 1 .705-.577zm.543 1.383l1.61 6.082l.062-.012L5 9.5h7a.5.5 0 0 1 .09.992L12 10.5H5a.506.506 0 0 1-.092-.008l-1.643 6.206l13.458-6.632L3.264 3.434z" fill="currentColor"></path></g></svg>
                </div>
                <div @click="() => $refs.file.click()" v-else>
                    <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24"><path d="M16.5 6v11.5c0 2.21-1.79 4-4 4s-4-1.79-4-4V5a2.5 2.5 0 0 1 5 0v10.5c0 .55-.45 1-1 1s-1-.45-1-1V6H10v9.5a2.5 2.5 0 0 0 5 0V5c0-2.21-1.79-4-4-4S7 2.79 7 5v12.5c0 3.04 2.46 5.5 5.5 5.5s5.5-2.46 5.5-5.5V6h-1.5z" fill="currentColor"></path></svg>
                    <input type="file" style="display: none;" ref="file" @change="upload" />
                </div>
            </div>
            <div v-if="masking.visiable" class="masking" @click="() => masking.visiable = false">
                <img :src="masking.img" />
            </div>
        </div>
        <script src="https://cdnjs.cloudflare.com/ajax/libs/vue/3.5.13/vue.global.min.js"></script>
        <script src="https://cdnjs.cloudflare.com/ajax/libs/peerjs/1.5.4/peerjs.min.js"></script>
        <script>
            Vue.createApp({
                data() {
                    return {
                        id: "",
                        peer: undefined,
                        conn: undefined,
                        connected: false,
                        text: "",
                        messages: [{
                            sent: false,
                            type: "text",
                            value: 'Please input peer ID to connect...',
                        }],
                        masking: {
                            img: undefined,
                            visiable: false,
                        },
                    }
                },
                watch: {
                    conn: {
                        handler(conn) {
                            if (!conn) {
                                return
                            }
                            conn.on("open", () => {
                                this.connected = true
                            }),
                            conn.on("data", data => {
                                this.messages.push(data)
                            })
                            conn.on("close", () => {
                                this.connected = false
                            })
                            conn.on("error", err => {
                                this.connected = false
                                this.messages.push({
                                    sent: false,
                                    type: "text",
                                    value: err.message,
                                })
                            })
                        },
                    },
                    text: {
                        handler(value) {
                            this.$refs.text.style.height = value.split("").filter(i => i === "\n").length * 24 + 32 + "px"
                        }
                    },
                },
                methods: {
                    submit() {
                        if (!this.text) {
                            return
                        }
                        this.send(!this.conn ? "connect" : "text", this.text)
                        this.text = ""
                    },
                    upload({ target }) {
                        const file = target.files[0]
                        if (file) {
                            const reader = new FileReader()
                            reader.onload = e => {
                                this.send(file.type, e.target.result, {
                                    name: file.name,
                                    size: file.size,
                                })
                            }
                            reader.readAsDataURL(file)
                        }
                        this.$refs.file.value = ""
                    },
                    send(type, value, meta) {
                        const message = {
                            type,
                            value,
                            meta,
                        }
                        if (!this.conn) {
                            if (type === "connect") {
                                this.conn = this.peer.connect(this.text) // 主动发起连接
                            }
                            return
                        }
                        this.conn.send(message)
                        this.messages.push({
                            ...message,
                            sent: true,
                        })
                        this.$nextTick(() => {
                            this.$refs.messages.scrollTop = this.$refs.messages.scrollHeight
                        })
                    },
                    preview(img) {
                        this.masking.visiable = true
                        this.masking.img = img
                    },
                },
                mounted() {
                    this.id = (Math.random() * 100_000 + "").substring(0, 4)
                    this.peer = new Peer(
                        this.id,
                        // { host: "127.0.0.1", port: 9000, path: "/", }
                    )
                    this.peer.on("connection", conn => { // 等待被连接
                        this.conn = conn
                    })
                }
            }).mount("#app")
        </script>
    </body>
    
    </html>
    ```
