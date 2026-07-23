<template>
    <div ref="container" :style="{ width: width, height: height }"></div>
</template>

<script>
export default {
    name: "MonacoEditor",
    props: {
        modelValue: { type: String, default: "" },
        width: { type: String, default: "100%" },
        height: { type: String, default: "180px" },
        language: { type: String, default: "typescript" },
        readOnly: { type: Boolean, default: false },
    },
    emits: ["update:modelValue", "CreateEditor"],
    setup(props, { emit }) {
        const container = Vue.ref()

        const monaco = await $import("monaco")
        const editor = monaco.editor.create(container.value, {
            language: props.language,
            value: props.modelValue,
            automaticLayout: true,
        })
        editor.onDidChangeModelContent(() => {
            emit("update:modelValue", editor.getValue())
        })
        editor.updateOptions({ readOnly: props.readOnly ?? false })
        Vue.watch(() => props.modelValue, (newValue, oldValue) => {
            if (newValue !== oldValue && newValue !== editor.getValue()) {
                editor.setValue(newValue)
            }
        })
        emit("CreateEditor", editor)

        return { container }
    },
}
</script>
