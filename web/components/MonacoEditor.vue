<template>
    <div ref="container" :style="{ width: width, height: height }"></div>
</template>

<script>
export default {
    name: "MonacoEditor",
    emits: ["update:modelValue", "CreateEditor"],
    props: {
        modelValue: { type: String, default: "" },
        width: { type: String, default: "100%" },
        height: { type: String, default: "180px" },
        language: { type: String, default: "typescript" },
        readOnly: { type: Boolean, default: false },
    },
    watch: {
        modelValue(newValue, oldValue) {
            if (this.editor && newValue !== oldValue && newValue !== this.editor.getValue()) {
                this.editor.setValue(newValue)
            }
        },
    },
    async created() {
        const monaco = await $import("monaco")
        // this.editor 是非响应式的：Vue 会深度递归劫持 `data()` 返回的对象，而 Monaco Editor 的实例极其庞大且包含循环引用，调用 `getValue()` 会触发 Vue 劫持的 getter 陷阱，撑爆主线程 CPU 导致卡死
        this.editor = monaco.editor.create(this.$refs.container, {
            language: this.language,
            value: this.modelValue,
            automaticLayout: true,
        })
        this.editor.onDidChangeModelContent(() => {
            this.$emit("update:modelValue", this.editor.getValue())
        })
        this.editor.updateOptions({ readOnly: this.readOnly ?? false })
        this.$emit("CreateEditor", this.editor)
    },
    beforeUnmount() {
        if (this.editor) {
            this.editor.dispose()
        }
    },
}
</script>
