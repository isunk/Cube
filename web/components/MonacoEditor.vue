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
    data() {
        return {
            editor: undefined,
        }
    },
    watch: {
        modelValue(newValue, oldValue) {
            if (this.editor && newValue !== oldValue && newValue !== this.editor.getValue()) {
                this.editor.setValue(newValue)
            }
        },
    },
    async mounted() {
        const monaco = await $import("monaco")
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
