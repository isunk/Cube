<template>
    <div style="display:none"></div>
</template>

<script>
let _loading = null

export default {
    methods: {
        async loadMonaco() {
            if (window.monaco) return
            if (_loading) return _loading
            _loading = new Promise((resolve) => {
                const prevDefine = window.define
                const prevRequire = window.require
                delete window.define
                delete window.require
                const script = document.createElement("script")
                script.src = "/libs/monaco-editor/0.55.1/min/vs/loader.js"
                script.onload = () => {
                    const r = window.require
                    window.define = prevDefine
                    window.require = prevRequire
                    r.config({ paths: { vs: "/libs/monaco-editor/0.55.1/min/vs" } })
                    r(["vs/editor/editor.main"], resolve)
                }
                document.head.appendChild(script)
            })
            return _loading
        },
        async colorize(el) {
            await this.loadMonaco()
            return monaco.editor.colorizeElement(el, { theme: "vs" })
        },
    },
    mounted() {
        this.loadMonaco()
    },
}
</script>
