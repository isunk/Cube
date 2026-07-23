<template>
    <div>
        <el-row>
            <div style="display: flex; align-items: center;">
                <el-breadcrumb separator="/">
                    <el-breadcrumb-item @click="onNavigate('')" style="cursor: pointer; font-weight: 500;">~</el-breadcrumb-item>
                    <el-breadcrumb-item v-for="(part, idx) in pathParts" :key="idx" @click="onNavigate(pathParts.slice(0, idx + 1).join('/'))" style="cursor: pointer; font-weight: 500;">{{ part }}</el-breadcrumb-item>
                </el-breadcrumb>
            </div>
            <div style="margin-left: auto; display: inline-flex;">
                <el-button @click="onNewDir">New Directory</el-button>
                <el-upload :auto-upload="false" action="" :on-change="onFileUpload" :show-file-list="false" style="display: none;">
                    <el-button ref="UploadRef"></el-button>
                </el-upload>
                <el-button @click="UploadClick" style="margin-left: 0;">Upload</el-button>
            </div>
        </el-row>

        <div v-loading="loading" style="margin-top: 10px;">
            <div v-if="!loading && !files.length" style="text-align: center; color: #c0c4cc; padding: 60px; font-size: 13px;">Empty directory</div>
            <div
                v-for="file in files"
                :key="file.name"
                style="display: flex; align-items: center; padding: 10px 16px; border-bottom: 1px solid #ebeef5; cursor: pointer;"
                :style="{ borderTop: files.indexOf(file) === 0 ? '1px solid #dcdfe6' : '', borderLeft: '1px solid #dcdfe6', borderRight: '1px solid #dcdfe6' }"
                @click="onRowClick(file)"
            >
                <el-icon :size="18" :style="{ color: file.dir ? 'var(--el-color-warning)' : 'var(--el-color-info)' }">
                    <FolderOpened v-if="file.dir" />
                    <Document v-else />
                </el-icon>
                <span style="flex: 1; margin-left: 10px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ file.name }}</span>
                <span style="display: flex; align-items: center; gap: 16px; margin-right: 12px; color: #909399; font-size: 13px;">
                    <span v-if="!file.dir" style="min-width: 60px; text-align: right;">{{ formatSize(file.size) }}</span>
                    <span style="min-width: 140px;">{{ file.time }}</span>
                </span>
                <el-button link type="danger" @click.stop="onFileDelete(file)">
                    <el-icon><Delete /></el-icon>
                </el-button>
            </div>
        </div>
    </div>
</template>

<script>
const { ElMessage, ElMessageBox } = ElementPlus

export default {
    setup() {
        const { ref } = Vue
        const { Delete, Document, FolderOpened } = ElementPlusIconsVue
        const UploadRef = ref()
        return {
            Delete, Document, FolderOpened,
            UploadRef,
            UploadClick: () => { UploadRef.value.ref.click() },
        }
    },
    data() {
        return {
            files: [],
            loading: false,
            currentPath: "",
        }
    },
    computed: {
        pathParts() {
            return this.currentPath ? this.currentPath.split("/") : []
        },
    },
    methods: {
        onFetchFiles() {
            this.loading = true
            fetch("file" + (this.currentPath ? `?path=${encodeURIComponent(this.currentPath)}` : "")).then(r => r.json()).then(r => {
                if (r.code === "0") {
                    this.files = r.data || []
                } else {
                    ElMessage.error(r.message)
                }
            }).catch(e => ElMessage.error(e.message)).finally(() => this.loading = false)
        },
        onNavigate(path) {
            this.currentPath = path
            this.onFetchFiles()
        },
        onRowClick(row) {
            if (row.dir) {
                this.onNavigate(row.name)
            } else {
                const a = document.createElement("a")
                a.href = `file?download=${encodeURIComponent(row.name)}`
                a.click()
            }
        },
        onFileUpload(file) {
            const formData = new FormData()
            formData.append("file", file.raw)
            if (this.currentPath) formData.append("path", this.currentPath)
            fetch("file", { method: "POST", body: formData })
                .then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Upload succeeded")
                        this.onFetchFiles()
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => ElMessage.error(e.message))
        },
        onNewDir() {
            ElMessageBox.prompt("Directory name", "New Directory", {
                confirmButtonText: "Create",
            }).then(({ value }) => {
                if (!value) return
                const formData = new FormData()
                formData.append("type", "directory")
                formData.append("name", value)
                if (this.currentPath) formData.append("path", this.currentPath)
                fetch("file", { method: "POST", body: formData })
                    .then(r => r.json()).then(r => {
                        if (r.code === "0") {
                            ElMessage.success("Directory created")
                            this.onFetchFiles()
                        } else {
                            ElMessage.error(r.message)
                        }
                    })
            }).catch(() => {})
        },
        onFileDelete(file) {
            ElMessageBox.confirm(`${file.name} will be deleted permanently. Continue ?`, "Warning", {
                confirmButtonText: "Confirm",
                type: "warning",
            }).then(() => {
                fetch(`file?name=${encodeURIComponent(file.name)}`, { method: "DELETE" }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Delete succeeded")
                        this.onFetchFiles()
                    } else {
                        ElMessage.error(r.message)
                    }
                })
            }).catch(() => {})
        },
        formatSize(bytes) {
            if (bytes == null || bytes === 0) return ""
            const units = ["B", "KB", "MB", "GB"]
            let i = 0, size = bytes
            while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
            return size.toFixed(1) + " " + units[i]
        },
    },
    mounted() {
        this.onFetchFiles()
    },
}
</script>
