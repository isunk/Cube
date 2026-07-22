<template>
    <div class="doc-content" v-cloak>
        <monaco-editor ref="monacoEditor" />
        <div v-html="htmlContent" ref="ContentRef"></div>
    </div>
</template>

<script>
let _markedSetup = false

export default {
    data() {
        return {
            htmlContent: "",
        }
    },
    methods: {
        toTitle(text) {
            const match = text?.match(/# ([^\n]+)/)
            return match?.[1] || "No title"
        },
        async render(markdown) {
            const { default: marked } = await import("marked")
            if (!_markedSetup) {
                marked.use({
                    renderer: {
                        link(href, title, text) {
                            if (/^[\w\/\-]+\.md/i.test(href)) {
                                href = "#/document?" + href
                            }
                            return this.__proto__.link(href, title, text)
                        },
                        code(code, type, escaped) {
                            if (type === "mermaid") {
                                return "<div class=\"mermaid\">" + code + "</div>"
                            }
                            return this.__proto__.code(code, type, escaped)
                        },
                    },
                })
                _markedSetup = true
            }
            document.title = this.toTitle(markdown)

            this.htmlContent = marked.parse(markdown)

            await Vue.nextTick()

            await this.processCodeBlocks()
            this.processMermaid()
        },
        async processCodeBlocks() {
            const editor = this.$refs.monacoEditor
            const that = this
            const els = this.$refs.ContentRef.querySelectorAll("pre code")
            for (const e of els) {
                const [metadata] = e.innerText.match(/^\/\/\?name=[\w\/]+&type=\w+[^\n]*\n/) || [""]
                const content = e.textContent = e.innerText.substring(metadata.length)
                const rows = content.split("\n").length - 1

                await editor.colorize(e)

                e.before(that.createBoxLine(rows))

                if (rows > 16) {
                    e.after(that.createBtnExpand())
                }

                if (metadata) {
                    e.after(that.createBtnAdd(content, new URL("http://0.0.0.0/?" + metadata.substring(3)).searchParams))
                }
            }
        },
        createBoxLine(count) {
            const e = document.createElement("span")
            e.className = "box-line"
            for (let i = 0; i < count; i++) {
                const l = document.createElement("span")
                l.textContent = i + 1
                e.appendChild(l)
            }
            return e
        },
        createBtnExpand() {
            const e = document.createElement("span")
            e.className = "btn-expand"
            e.onclick = function () {
                this.parentNode.style.maxHeight = "unset"
                this.className = ""
            }
            return e
        },
        createBtnAdd(content, params) {
            const that = this
            const name = params.get("name"),
                type = params.get("type"),
                lang = params.get("lang") || "typescript",
                method = params.get("method") || "",
                url = params.get("url") || "",
                tag = params.get("tag") || ""

            const e = document.createElement("span")
            e.className = "btn-add"
            e.onclick = function() {
                fetch("/source", {
                    method: "POST",
                    body: JSON.stringify({ name, type, lang, content, compiled: "", method, url, tag, }),
                }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElementPlus.ElMessage.success("Install successfully")
                    } else {
                        ElementPlus.ElMessage.error(r.message)
                    }
                })
            }
            return e
        },
        async processMermaid() {
            const { default: mermaid } = await import("mermaid")
            mermaid.initialize({
                theme: "base",
                themeVariables: {
                    primaryColor: "#FFFFFF",
                    primaryTextColor: "#333333",
                    primaryBorderColor: "#CCCCCC",
                    lineColor: "#888888",
                    tertiaryColor: "#D0E4FF",
                    tertiaryBorderColor: "#D0E4FF",
                    tertiaryTextColor: "#00A4FF",
                },
                flowchart: {
                    curve: "stepBefore",
                },
            })
            mermaid.init().then(() => this.setSvgScalable())
        },
        setSvgScalable() {
            const that = this
            that.$refs.ContentRef.querySelectorAll(".mermaid").forEach(container => {
                let scale = 1.0,
                    translateX = 0, translateY = 0,
                    dragging = false,
                    startX = 0, startY = 0,
                    svg = container.querySelector("svg"),
                    update = () => svg.style.transform = `translate(${translateX}px, ${translateY}px) scale(${scale})`

                svg.style.backgroundColor = "rgb(255, 255, 255, .95)"
                svg.style.position = "relative"
                svg.style.zIndex = 99

                container.addEventListener("wheel", event => {
                    event.preventDefault()
                    if (event.deltaY < 0) {
                        scale = Math.min(10, scale + 0.1)
                    } else {
                        scale = Math.max(1, scale - 0.1)
                        if (scale === 1) {
                            translateX = translateY = startX = startY = 0
                        }
                    }
                    update()
                }, { passive: false })

                container.onmousedown = event => {
                    if (event.button !== 1) return
                    event.preventDefault()
                    dragging = true
                    startX = event.clientX
                    startY = event.clientY
                }

                container.onmousemove = event => {
                    event.preventDefault()
                    if (!dragging) return
                    translateX += event.clientX - startX
                    translateY += event.clientY - startY
                    startX = event.clientX
                    startY = event.clientY
                    update()
                }

                container.onmouseup = container.onmouseleave = event => {
                    event.preventDefault()
                    dragging = false
                }

                update()
            })
        },
        fetchDocument(name) {
            fetch(`/document/${name}`)
                .then(r => r.text())
                .then(c => this.render(c))
                .catch(e => {
                    ElementPlus.ElMessage.error(e.message)
                })
        },
    },
    components: {
        "monaco-editor": load("/components/MonacoEditor.vue"),
    },
    watch: {
        "$route.fullPath": {
            immediate: true,
            handler(path) {
                const idx = path.indexOf("?")
                const name = idx >= 0 ? path.substring(idx + 1) : "summary.md"
                this.fetchDocument(name)
            },
        },
    },
}
</script>

<style>
.doc-content {
    padding: 20px;
    margin: 1pc auto;
    max-width: 900px;
    -ms-text-size-adjust: 100%;
    -webkit-text-size-adjust: 100%;
    line-height: 1.5;
    color: #24292e;
    font-size: 17px;
    word-wrap: break-word;
}
pre {
    display: flex;
    position: relative;
    max-height: 416px;
}
pre span.box-line {
    display: flex;
    flex-direction: column;
    padding: 1em 0.6em;
    text-align: right;
    user-select: none;
    border: 1px dashed #dcdfe6;
    border-right: none;
    min-width: fit-content;
    overflow: hidden;
}
pre span.btn-add {
    position: absolute;
    right: 0px;
    padding: 11px;
    margin: 4px 4px;
    cursor: pointer;
    background: #f1f1f1 url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24"><path d="M20 17H4V5h8V3H2v16h6v2h8v-2h6v-5h-2z" fill="currentColor"></path><path d="M17 14l5-5l-1.41-1.41L18 10.17V3h-2v7.17l-2.59-2.58L12 9z" fill="currentColor"></path></svg>') no-repeat;
    border: 3px solid #f1f1f1;
}
pre span.btn-expand {
    position: absolute;
    width: 100%;
    bottom: 0;
    height: 32px;
    cursor: pointer;
    background: linear-gradient(to top, #ffffff, rgba(0, 0, 0, 0)), url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24"><path d="M16.59 8.59L12 13.17L7.41 8.59L6 10l6 6l6-6z" fill="currentColor"></path></svg>') center/ 24px 24px no-repeat;
}
pre:has(span.btn-expand) code {
    overflow-x: hidden;
}
code {
    border: 1px dashed #dcdfe6;
    padding: 0 3px;
    overflow-y: hidden !important;
}
[v-cloak] {
    display: none;
}
</style>
