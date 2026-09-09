<template>
    <el-row style="margin-bottom: 12px; align-items: center;">
        <el-breadcrumb separator="/">
            <el-breadcrumb-item>
                <el-link type="primary" @click="onPathUpdate('')" :underline="false" style="font-weight: bold;">~</el-link>
            </el-breadcrumb-item>
            <el-breadcrumb-item v-for="(part, idx) in parts" :key="idx">
                <el-link type="primary" @click="onPathUpdate(parts.slice(0, idx + 1).join('/'))" :underline="false">{{ part }}</el-link>
            </el-breadcrumb-item>
        </el-breadcrumb>
        <div style="margin-left: auto;">
            <el-upload :auto-upload="false" action="" :on-change="onFileUpload" :show-items="false" multiple style="display: none;">
                <el-button ref="UploadRef"></el-button>
            </el-upload>
            <el-dropdown trigger="click">
                <el-button :icon="Plus">New</el-button>
                <template #dropdown>
                    <el-dropdown-item @click="onFolderCreate">Folder</el-dropdown-item>
                    <el-dropdown-item @click="UploadClick">File</el-dropdown-item>
                </template>
            </el-dropdown>
        </div>
    </el-row>

    <el-empty v-if="!loading && !files.length" description="Empty folder" :image-size="80" />
    <div v-else v-loading="loading" class="items">
        <div v-for="(item, idx) in files" :key="idx" class="item" @click="onRowClick(item)">
            <el-icon v-if="item.folder" :size="18" style="color: var(--el-color-primary);"><FolderOpened /></el-icon>
            <el-icon v-else :size="18" style="color: var(--el-text-color-secondary);"><Document /></el-icon>
            <span :class="item.folder ? 'folder' : 'file'">{{ item.name }}</span>
            <span class="info" v-if="!item.folder">{{ toSize(item.size) }}</span>
            <span class="info">{{ toTime(item.time) }}</span>
            <el-button link type="danger" :icon="Delete" @click.stop="onFileDelete(item)" style="margin-left: 12px;" />
        </div>
    </div>

    <el-dialog v-model="dialog.visible" title="New" width="640px">
        <el-form ref="FormRef" :model="dialog.form" :rules="rules" label-position="right" label-width="96px">
            <el-form-item label="Name" prop="name">
                <el-input v-model="dialog.form.name" placeholder="Folder name" />
            </el-form-item>
            <el-form-item>
                <el-button type="primary" :loading="dialog.loading" @click="onFolderSubmit(FormRef)">Submit</el-button>
                <el-button @click="dialog.visible = false">Cancel</el-button>
            </el-form-item>
        </el-form>
    </el-dialog>
</template>

<script>
const { ElMessage, ElMessageBox } = ElementPlus
const { Delete, Document, FolderOpened, Plus } = ElementPlusIconsVue

export default {
    components: {
        FolderOpened,
        Document,
    },
    setup() {
        const { ref } = Vue
        const UploadRef = ref()
        const FormRef = ref()
        return {
            Delete,
            Document,
            FolderOpened,
            Plus,
            UploadRef,
            FormRef, // 由于这里使用的是直接传参模式（例如 `onFolderSubmit(FormRef)`），如果 setup 中未声明和返回该 ref 变量，函数接收到的入参为 `undefined`；对比其它调用模式（如 Database.vue 的 `this.$refs.XXXRef`），模板 ref 会自动挂载到实例 $refs，不依赖 setup 返回，不受此影响
            UploadClick: () => {
                UploadRef.value.ref.click()
            },
        }
    },
    data() {
        return {
            files: [],
            loading: false,
            path: "",
            dialog: {
                visible: false,
                loading: false,
                form: { name: "" },
            },
            rules: {
                name: [{ required: true, message: "Folder name is required", trigger: "blur" }],
            },
        }
    },
    computed: {
        parts() {
            return this.path ? this.path.split("/") : []
        },
    },
    mounted() {
        this.onFileFetch()
    },
    methods: {
        onFileDelete(file) {
            ElMessageBox.confirm(`"${file.name}" will be deleted permanently. Continue?`, "Warning", {
                confirmButtonText: "Confirm",
                type: "warning",
                beforeClose: (action, instance, done) => {
                    if (action === "confirm") {
                        instance.confirmButtonLoading = true
                        instance.confirmButtonText = "Deleting..."
                        fetch(`file?name=${encodeURIComponent(this.toPath(file.name))}`, { method: "DELETE" }).then(r => r.json()).then(r => {
                            if (r.code === "0") {
                                ElMessage.success("Delete succeeded")
                                this.onFileFetch()
                            } else {
                                ElMessage.error(r.message)
                            }
                        }).catch(e => {
                            ElMessage.error(e.message)
                        }).finally(() => {
                            instance.confirmButtonLoading = false
                            done()
                        })
                    } else {
                        done()
                    }
                },
            }).catch(() => { })
        },
        onFileFetch() {
            this.loading = true
            fetch("file" + (this.path ? `?path=${encodeURIComponent(this.path)}` : "")).then(r => r.json()).then(r => {
                if (r.code === "0") {
                    const list = r.data || []
                    this.files = [...list.filter(f => f.folder).sort((a, b) => a.name.localeCompare(b.name)), ...list.filter(f => !f.folder).sort((a, b) => a.name.localeCompare(b.name))]
                } else {
                    ElMessage.error(r.message)
                }
            }).catch(e => ElMessage.error(e.message)).finally(() => {
                this.loading = false
            })
        },
        onFileUpload(file) {
            const formData = new FormData()
            formData.append("file", file.raw)
            if (this.path) {
                formData.append("path", this.path)
            }
            fetch("file", { method: "POST", body: formData }).then(r => r.json()).then(r => {
                if (r.code === "0") {
                    ElMessage.success("Upload succeeded")
                    this.onFileFetch()
                } else {
                    ElMessage.error(r.message)
                }
            }).catch(e => ElMessage.error(e.message))
        },
        onFolderCreate() {
            this.dialog.form.name = ""
            this.dialog.loading = false
            this.dialog.visible = true
        },
        onFolderSubmit(formEl) {
            if (!formEl) {
                return
            }
            formEl.validate(valid => {
                if (!valid) {
                    return
                }
                const name = this.dialog.form.name.trim()
                this.dialog.loading = true
                const formData = new FormData()
                formData.append("type", "folder")
                formData.append("name", name)
                if (this.path) {
                    formData.append("path", this.path)
                }
                fetch("file", { method: "POST", body: formData }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Create succeeded")
                        this.dialog.visible = false
                        this.onFileFetch()
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => ElMessage.error(e.message)).finally(() => {
                    this.dialog.loading = false
                })
            })
        },
        onPathUpdate(path) {
            this.path = path
            this.onFileFetch()
        },
        onRowClick(row) {
            if (row.folder) {
                this.onPathUpdate(this.toPath(row.name))
            } else {
                const a = document.createElement("a")
                a.href = `file?download=${encodeURIComponent(this.toPath(row.name))}`
                a.click()
            }
        },
        toPath(name) {
            return this.path ? `${this.path}/${name}` : name
        },
        toSize(bytes) {
            if (bytes == null || bytes === 0) {
                return ""
            }
            const units = ["B", "KB", "MB", "GB"]
            let i = 0,
                size = bytes
            while (size >= 1024 && i < units.length - 1) {
                size /= 1024
                i++
            }
            return size.toFixed(1) + " " + units[i]
        },
        toTime(t) {
            if (!t) {
                return ""
            }
            const d = new Date(t)
            return new Date(d - d.getTimezoneOffset() * 60000).toISOString().replace("T", " ").slice(0, 19)
        },
    },
}
</script>

<style scoped>
.items {
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--el-border-radius-base);
    overflow: hidden;
}
.item {
    display: flex;
    align-items: center;
    padding: 10px 16px;
    cursor: pointer;
    transition: background-color 0.2s;
}
.item + .item {
    border-top: 1px solid var(--el-border-color-lighter);
}
.item:hover {
    background-color: var(--el-fill-color-light);
}
.item > * + * {
    margin-left: 12px;
}
.item > span.folder {
    flex: 1;
    color: var(--el-color-primary);
    font-weight: 500;
}
.item > span.file {
    flex: 1;
    color: var(--el-text-color-regular);
}
.item > span.info {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
}
</style>
