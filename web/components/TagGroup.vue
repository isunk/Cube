<template>
    <div style="display: inline-flex; flex-flow: wrap; align-items: center; gap: 4px;">
        <el-tag v-if="dataset.length" v-for="e in dataset.slice(0, count)" :closable="closable" @close="onRemoveTag(e)">{{ e }}</el-tag>
        <el-popover v-if="dataset.length > count" placement="bottom-start" :width="0" trigger="hover">
            <template #reference>
                <el-tag>+{{ dataset.length - count }}</el-tag>
            </template>
            <div style="display: flex; flex-wrap: wrap; gap: 4px; max-width: 400px;">
                <el-tag v-for="e in dataset.slice(count)" :closable="closable" @close="onRemoveTag(e)">{{ e }}</el-tag>
            </div>
        </el-popover>
        <div v-if="newable" style="display: flex;">
            <el-input v-if="input.visible" ref="InputRef" v-model="input.value" size="small" @keyup.enter="onAddTag" @blur="onAddTag"></el-input>
            <el-button v-else size="small" @click="onNewTag">+ New Tag</el-button>
        </div>
    </div>
</template>

<script>
export default {
    props: {
        modelValue: { type: String, default: "", },
        count: { type: Number, default: 99, },
        closable: { type: Boolean, default: false, },
        newable: { type: Boolean, default: false, },
    },
    emits: ["update:modelValue"],
    setup() {
        return {
            InputRef: Vue.ref(),
        }
    },
    computed: {
        dataset: {
            get() {
                return this.modelValue.split(",")?.filter(i => i)
            },
            set(v) {
                this.$emit("update:modelValue", v.join(","))
            },
        },
    },
    data() {
        return {
            input: {
                value: "",
                visible: false,
            },
        }
    },
    methods: {
        onRemoveTag(tag) {
            this.dataset = this.dataset.filter(i => i !== tag)
        },
        onAddTag() {
            if (this.input.value && !~this.dataset.indexOf(this.input.value)) {
                this.dataset = [...this.dataset, this.input.value]
            }
            this.input.visible = false
            this.input.value = ""
        },
        onNewTag() {
            this.input.visible = true
            Vue.nextTick(() => {
                this.InputRef.input.focus()
            })
        },
    },
}
</script>

<style scoped>
    .el-tag {
        max-width: 160px;
    }
    .el-tag .el-tag__content {
        overflow: hidden;
        text-overflow: ellipsis;
        line-height: 1rem;
    }
</style>
