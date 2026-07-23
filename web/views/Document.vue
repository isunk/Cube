<template>
    <div class="doc-content" v-cloak>
        <div v-html="htmlContent" ref="ContentRef"></div>
    </div>
</template>

<script>
const MonacoEditor = $import("/components/MonacoEditor.vue")
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
            document.title = this.toTitle(markdown)

            const marked = await $import("marked")
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
            this.htmlContent = marked.parse(markdown)

            await Vue.nextTick()

            await this.processCodeBlocks()
            this.processMermaid()
        },
        async processCodeBlocks() {
            const els = this.$refs.ContentRef.querySelectorAll("pre code")
            for (const e of els) {
                const [metadata] = e.innerText.match(/^\/\/\?name=[\w\/]+&type=\w+[^\n]*\n/) || [""]
                const content = e.innerText.substring(metadata.length)
                const lang = e.className.replace(/^language-/, "") || "typescript"
                const lineCount = content.split("\n").length
                const height = Math.min(Math.max(lineCount * 20 + 10, 60), 400) + "px"

                e.parentNode.style.display = "none"

                const wrapper = document.createElement("div")
                wrapper.style.position = "relative"
                wrapper.style.width = "100%"
                wrapper.style.height = height
                wrapper.style.border = "1px solid #dcdfe6"
                e.parentNode.parentNode.insertBefore(wrapper, e.parentNode)

                const that = this
                const app = Vue.createApp({
                    template: `<div style="position: relative; width: 100%; height: 100%;">
                        <span v-if="showAdd" class="btn-add" @click="onInstall"></span>
                        <monaco-editor :modelValue="code" :language="lang" :height="height" readOnly />
                    </div>`,
                    data() {
                        return {
                            code: content,
                            lang,
                            height,
                            showAdd: !!metadata,
                        }
                    },
                    methods: {
                        onInstall() {
                            fetch("/source", {
                                method: "POST",
                                body: JSON.stringify({
                                    name: that.params.get("name"),
                                    type: that.params.get("type"),
                                    lang: that.params.get("lang") || "typescript",
                                    content,
                                    compiled: "",
                                    method: that.params.get("method") || "",
                                    url: that.params.get("url") || "",
                                    tag: that.params.get("tag") || "",
                                }),
                            }).then(r => r.json()).then(r => {
                                if (r.code === "0") {
                                    ElementPlus.ElMessage.success("Install successfully")
                                } else {
                                    ElementPlus.ElMessage.error(r.message)
                                }
                            })
                        },
                    },
                    components: {
                        "monaco-editor": MonacoEditor,
                    },
                })
                if (metadata) {
                    that.params = new URL("http://0.0.0.0/?" + metadata.substring(3)).searchParams
                }
                app.mount(wrapper)
            }
        },
        async processMermaid() {
            const mermaid = await $import("mermaid")
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
span.btn-add {
    position: absolute;
    top: 4px;
    right: 18px;
    z-index: 10;
    width: 24px;
    height: 24px;
    cursor: pointer;
    background: #f1f1f1 url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24"><path d="M20 17H4V5h8V3H2v16h6v2h8v-2h6v-5h-2z" fill="currentColor"></path><path d="M17 14l5-5l-1.41-1.41L18 10.17V3h-2v7.17l-2.59-2.58L12 9z" fill="currentColor"></path></svg>') center/ 18px 18px no-repeat;
    border: 1px solid #dcdfe6;
    border-radius: 2px;
}
:not(pre) > code {
    border: 1px dashed #dcdfe6;
    padding: 0 3px;
}
[v-cloak] {
    display: none;
}
</style>
