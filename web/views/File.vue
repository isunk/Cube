<template>
    <div>
        <div class="toolbar">
            <el-breadcrumb separator="/">
                <el-breadcrumb-item @click="onNavigate('')" style="cursor: pointer;">files</el-breadcrumb-item>
                <el-breadcrumb-item v-for="(part, idx) in pathParts" :key="idx" @click="onNavigate(pathParts.slice(0, idx + 1).join('/'))" style="cursor: pointer;">{{ part }}</el-breadcrumb-item>
            </el-breadcrumb>
            <div style="margin-left: auto; display: flex; gap: 8px;">
                <el-button @click="onNewDir">New Directory</el-button>
                <el-upload :auto-upload="false" action="" :on-change="onFileUpload" :show-file-list="false" style="display: inline-flex;">
                    <el-button type="primary">Upload</el-button>
                </el-upload>
            </div>
        </div>

        <div v-loading="loading" class="file-list">
            <div v-if="!loading && !files.length" class="empty">Empty directory</div>
            <div v-for="file in files" :key="file.name" class="file-row" @click="onRowClick(file)">
                <el-icon :size="18" :style="{ color: file.dir ? 'var(--el-color-warning)' : 'var(--el-color-info)' }">
                    <FolderOpened v-if="file.dir" />
                    <Document v-else />
                </el-icon>
                <span class="file-name">{{ file.name }}</span>
                <span class="file-meta">
                    <span class="file-size" v-if="!file.dir">{{ formatSize(file.size) }}</span>
                    <span class="file-time">{{ file.time }}</span>
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
        const { Delete, Document, FolderOpened } = ElementPlusIconsVue
        return { Delete, Document, FolderOpened }
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

<style scoped>
.toolbar {
    display: flex;
    align-items: center;
    padding-bottom: 12px;
}
.file-list {
    border: 1px solid #e4e7ed;
    border-radius: 4px;
}
.file-row {
    display: flex;
    align-items: center;
    padding: 10px 16px;
    border-bottom: 1px solid #ebeef5;
    cursor: pointer;
    transition: background-color .15s;
}
.file-row:last-child {
    border-bottom: none;
}
.file-row:hover {
    background-color: #f5f7fa;
}
.file-name {
    flex: 1;
    margin-left: 10px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
.file-meta {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-right: 12px;
    color: #909399;
    font-size: 13px;
}
.file-size {
    min-width: 60px;
    text-align: right;
}
.file-time {
    min-width: 140px;
}
.empty {
    text-align: center;
    color: #c0c4cc;
    padding: 40px;
}
</style>
