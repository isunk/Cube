<template>
    <div>
        <el-row>
            <el-radio-group v-model="mode">
                <el-radio-button value="table">Table</el-radio-button>
                <el-radio-button value="sql">SQL</el-radio-button>
            </el-radio-group>
        </el-row>

        <el-row :gutter="12" v-if="mode === 'table'" style="margin-top: 10px;">
            <el-col :span="5">
                <div v-loading="tableListLoading" style="border: 1px solid #dcdfe6; border-radius: 4px; overflow: hidden;">
                    <div style="padding: 8px 12px; font-weight: 600; font-size: 13px; color: #606266; background: #f5f7fa; border-bottom: 1px solid #dcdfe6;">Tables</div>
                    <div v-if="!tables.length" style="text-align: center; color: #c0c4cc; padding: 20px; font-size: 13px;">No tables</div>
                    <div
                        v-for="table in tables"
                        :key="table"
                        style="display: flex; align-items: center; justify-content: space-between; padding: 6px 12px; cursor: pointer; font-size: 13px; border-bottom: 1px solid #ebeef5;"
                        :style="{ backgroundColor: activeTable === table ? '#ecf5ff' : '', color: activeTable === table ? 'var(--el-color-primary)' : '' }"
                        @click="onSelectTable(table)"
                    >
                        <span>{{ table }}</span>
                        <el-button link type="danger" @click.stop="onDropTable(table)">
                            <el-icon><Delete /></el-icon>
                        </el-button>
                    </div>
                </div>
            </el-col>
            <el-col :span="19">
                <div v-if="activeTable">
                    <el-row>
                        <el-button :icon="Plus" @click="onRowNew">New</el-button>
                        <el-button :icon="Delete" @click="onBatchDelete" :disabled="!checkedRows.length">Batch Delete</el-button>
                        <el-upload :auto-upload="false" action="" :on-change="onImport" :show-file-list="false" accept="application/json" style="display: none;">
                            <el-button ref="UploadRef"></el-button>
                        </el-upload>
                        <el-button-group style="padding-left: 5px;">
                            <el-button :icon="Upload" @click="UploadClick">Import</el-button>
                            <el-button :icon="Download" @click="onExport" :disabled="!records.length">Export</el-button>
                        </el-button-group>
                        <span style="margin-left: auto; color: #909399; font-size: 13px;">Total: {{ total }}</span>
                    </el-row>
                    <el-table v-loading="loading" :data="records" stripe @selection-change="onSelectionChange" table-layout="fixed">
                        <el-table-column type="selection" width="40"></el-table-column>
                        <el-table-column type="index" width="50" label="#"></el-table-column>
                        <el-table-column v-for="col in columns" :key="col" :prop="col" :label="col" :show-overflow-tooltip="true">
                        </el-table-column>
                        <el-table-column label="Operation" width="100">
                            <template #default="scope">
                                <el-button link type="primary" :icon="Edit" @click="onRowEdit(scope.row)"></el-button>
                                <el-button link type="danger" :icon="Delete" @click="onRowDelete(scope.row)"></el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                </div>
                <div v-else style="text-align: center; color: #c0c4cc; padding: 60px; font-size: 13px;">Select a table from the left</div>
            </el-col>
        </el-row>

        <el-row v-else style="margin-top: 10px;">
            <monaco-editor v-model="sql" language="sql" height="120px" style="margin-bottom: 8px;"></monaco-editor>
            <div style="display: flex; justify-content: flex-end; margin-bottom: 8px;">
                <el-button type="primary" @click="onExecute" :loading="executing">Execute</el-button>
            </div>
            <el-table v-if="columns.length" v-loading="executing" :data="records" stripe table-layout="fixed">
                <el-table-column type="index" width="50" label="#"></el-table-column>
                <el-table-column v-for="col in columns" :key="col" :prop="col" :label="col" :show-overflow-tooltip="true">
                </el-table-column>
            </el-table>
            <div v-if="!executing && !columns.length" style="text-align: center; color: #c0c4cc; padding: 60px; font-size: 13px;">Enter SQL and click Execute</div>
        </el-row>

        <el-dialog v-model="dialog.visible" :title="dialog.isNew ? 'New Record' : 'Edit Record'">
            <el-form ref="FormRef" label-position="right" label-width="100px">
                <el-form-item v-for="col in columns" :key="col" :label="col" v-if="col !== 'rowid'">
                    <el-input v-model="dialog.record[col]" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button type="primary" :loading="dialog.loading" @click="onDialogSubmit">Submit</el-button>
                <el-button @click="dialog.visible = false">Cancel</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script>
const { ElMessage, ElMessageBox } = ElementPlus

export default {
    setup() {
        const { ref } = Vue
        const { Delete, Download, Edit, Plus, Upload } = ElementPlusIconsVue
        const UploadRef = ref()
        return {
            Delete, Download, Edit, Plus, Upload,
            FormRef: ref(),
            UploadRef,
            UploadClick: () => { UploadRef.value.ref.click() },
        }
    },
    data() {
        return {
            mode: "table",
            tables: [],
            activeTable: "",
            columns: [],
            records: [],
            total: 0,
            loading: false,
            tableListLoading: false,
            checkedRows: [],
            sql: "",
            executing: false,
            dialog: { visible: false, isNew: false, record: {}, loading: false },
        }
    },
    methods: {
        onFetchTables() {
            this.tableListLoading = true
            fetch("database").then(r => r.json()).then(r => {
                if (r.code === "0") this.tables = r.data || []
                else ElMessage.error(r.message)
            }).catch(e => ElMessage.error(e.message)).finally(() => this.tableListLoading = false)
        },
        onSelectTable(table) {
            this.activeTable = table
            this.loading = true
            this.checkedRows = []
            fetch(`database?table=${table}`).then(r => r.json()).then(r => {
                if (r.code === "0") {
                    this.columns = r.data.columns || []
                    this.records = r.data.records || []
                    this.total = r.data.total || 0
                } else {
                    ElMessage.error(r.message)
                }
            }).catch(e => ElMessage.error(e.message)).finally(() => this.loading = false)
        },
        onSelectionChange(rows) {
            this.checkedRows = rows.map(r => r.rowid)
        },
        onExecute() {
            if (!this.sql) return
            this.executing = true
            this.columns = []
            this.records = []
            fetch("database", { method: "POST", body: JSON.stringify({ sql: this.sql }) })
                .then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        const d = r.data
                        if (d.columns) {
                            this.columns = d.columns
                            this.records = d.records
                            this.total = d.total
                        } else {
                            ElMessage.success(`Executed, ${d.affected || 0} rows affected`)
                        }
                        this.sql = ""
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => ElMessage.error(e.message)).finally(() => this.executing = false)
        },
        onRowNew() {
            this.dialog.isNew = true
            this.dialog.record = {}
            this.columns.filter(c => c !== 'rowid').forEach(c => { this.dialog.record[c] = "" })
            this.dialog.visible = true
        },
        onRowEdit(row) {
            this.dialog.isNew = false
            this.dialog.record = { ...row }
            this.dialog.visible = true
        },
        onRowDelete(row) {
            ElMessageBox.confirm("This record will be deleted permanently. Continue ?", "Warning", {
                confirmButtonText: "Confirm", type: "warning",
            }).then(() => {
                fetch(`database?table=${this.activeTable}&rowid=${row.rowid}`, { method: "DELETE" }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Delete succeeded")
                        this.onSelectTable(this.activeTable)
                    } else ElMessage.error(r.message)
                })
            }).catch(() => {})
        },
        onBatchDelete() {
            if (!this.checkedRows.length) return
            ElMessageBox.confirm(`${this.checkedRows.length} records will be deleted permanently. Continue ?`, "Warning", {
                confirmButtonText: "Confirm", type: "warning",
            }).then(() => {
                fetch(`database?table=${this.activeTable}&rowids=${this.checkedRows.join(",")}`, { method: "DELETE" }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Batch delete succeeded")
                        this.onSelectTable(this.activeTable)
                    } else ElMessage.error(r.message)
                })
            }).catch(() => {})
        },
        onDialogSubmit() {
            const data = { ...this.dialog.record }
            delete data.rowid
            let sql
            if (this.dialog.isNew) {
                const keys = Object.keys(data).filter(k => data[k] !== "")
                const placeholders = keys.map(() => "?").join(", ")
                sql = `insert into "${this.activeTable}" (${keys.map(k => '"' + k + '"').join(", ")}) values(${placeholders})`
            } else {
                const keys = Object.keys(data).filter(k => data[k] !== "")
                const sets = keys.map(k => `"${k}" = ?`).join(", ")
                sql = `update "${this.activeTable}" set ${sets} where rowid = ${this.dialog.record.rowid}`
            }
            this.dialog.loading = true
            fetch("database", { method: "POST", body: JSON.stringify({ sql }) })
                .then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Submit succeeded")
                        this.dialog.visible = false
                        this.onSelectTable(this.activeTable)
                    } else ElMessage.error(r.message)
                }).catch(e => ElMessage.error(e.message)).finally(() => this.dialog.loading = false)
        },
        onImport(file) {
            const that = this
            const reader = new FileReader()
            reader.onload = function() {
                const records = JSON.parse(this.result)
                if (!Array.isArray(records) || !records.length) {
                    ElMessage.error("Invalid JSON format")
                    return
                }
                let count = 0
                const doNext = (i) => {
                    if (i >= records.length) {
                        ElMessage.success(`${count} records imported`)
                        that.onSelectTable(that.activeTable)
                        return
                    }
                    const r = records[i]
                    const keys = Object.keys(r).filter(k => k !== 'rowid' && r[k] !== undefined)
                    const placeholders = keys.map(() => "?").join(", ")
                    const sql = `insert into "${that.activeTable}" (${keys.map(k => '"' + k + '"').join(", ")}) values(${placeholders})`
                    fetch("database", { method: "POST", body: JSON.stringify({ sql }) })
                        .then(r => r.json()).then(r => { if (r.code === "0") count++; doNext(i + 1) })
                }
                doNext(0)
            }
            reader.readAsText(file.raw, "utf-8")
        },
        onExport() {
            const json = JSON.stringify(this.records, null, 2)
            const blob = new Blob([json], { type: "application/json" })
            const a = document.createElement("a")
            a.href = URL.createObjectURL(blob)
            a.download = `${this.activeTable}.json`
            a.click()
        },
        onDropTable(table) {
            ElMessageBox.confirm(`${table} will be dropped permanently. Continue ?`, "Warning", {
                confirmButtonText: "Confirm", type: "warning",
            }).then(() => {
                fetch(`database?table=${table}`, { method: "DELETE" }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Table dropped")
                        if (this.activeTable === table) {
                            this.activeTable = ""
                            this.columns = []
                            this.records = []
                            this.total = 0
                        }
                        this.onFetchTables()
                    } else ElMessage.error(r.message)
                })
            }).catch(() => {})
        },
    },
    mounted() {
        this.onFetchTables()
    },
    components: {
        "monaco-editor": $import("/components/MonacoEditor.vue"),
    },
}
</script>
