<template>
    <el-tabs v-model="mode">
        <el-tab-pane label="Table" name="table" lazy>
            <el-row :gutter="12">
                <el-col :span="5">
                    <el-card shadow="never">
                        <template #header>
                            <div style="display: flex; align-items: center; justify-content: space-between;">
                                <span>Tables</span>
                                <div style="display: flex; align-items: center;">
                                    <el-upload :auto-upload="false" action="" :on-change="onTableImport" :show-file-list="false" accept=".json" style="display: none;">
                                        <el-button ref="ColumnUploadRef"></el-button>
                                    </el-upload>
                                    <el-button :icon="Plus" link @click="onTableCreate" />
                                    <el-button :icon="Upload" link @click="onImportClick" />
                                </div>
                            </div>
                        </template>
                        <div v-loading="table.loading">
                            <el-empty v-if="!table.records.length" description="No tables" :image-size="40" />
                            <div v-else class="tables">
                                <div v-for="t in table.records" :key="t" :class="{ active: table.record === t }" @click="onTableSelect(t)">
                                    <span>{{ t }}</span>
                                </div>
                            </div>
                        </div>
                    </el-card>
                </el-col>
                <el-col :span="19">
                    <template v-if="table.record">
                        <el-card shadow="never">
                            <template #header>
                                <div style="display: flex; align-items: center; justify-content: space-between;">
                                    <span>Columns</span>
                                    <div style="display: flex; align-items: center;">
                                        <el-button :icon="Plus" @click="onColumnCreate" :disabled="protected">New</el-button>
                                        <el-button :icon="Download" @click="onColumnExport" :disabled="protected" style="margin-left: 8px;">Export</el-button>
                                        <el-button type="danger" :icon="Delete" @click="onTableDelete(table.record)" :disabled="protected" style="margin-left: 8px;">Drop</el-button>
                                    </div>
                                </div>
                            </template>
                            <el-table v-loading="column.loading" :data="column.records" stripe>
                                <el-table-column prop="name" label="Name" />
                                <el-table-column prop="type" label="Type" />
                                <el-table-column label="PK">
                                    <template #default="scope">
                                        <el-tag v-if="scope.row.pk" type="warning">PK</el-tag>
                                    </template>
                                </el-table-column>
                                <el-table-column label="Null">
                                    <template #default="scope">
                                        <el-tag :type="scope.row.nullable ? 'success' : 'danger'">{{ scope.row.nullable ? "YES" : "NO" }}</el-tag>
                                    </template>
                                </el-table-column>
                                <el-table-column prop="default" label="Default">
                                    <template #default="scope">
                                        <span :style="scope.row.default == null ? 'color: var(--el-text-color-placeholder)' : ''">{{ scope.row.default != null ? scope.row.default : "NULL" }}</span>
                                    </template>
                                </el-table-column>
                                <el-table-column v-if="!protected" width="86">
                                    <template #default="scope">
                                        <el-button link type="primary" :icon="Edit" @click="onColumnEdit(scope.row)" />
                                        <el-button link type="danger" :icon="Delete" @click="onColumnDelete(scope.row)" />
                                    </template>
                                </el-table-column>
                            </el-table>
                        </el-card>
                        <el-card shadow="never" style="margin-top: 12px;">
                            <template #header>
                                <div style="display: flex; align-items: center; justify-content: space-between;">
                                    <span>Records</span>
                                    <div style="display: flex; align-items: center;">
                                        <el-upload :auto-upload="false" action="" :on-change="onRecordImport" :show-file-list="false" accept=".json" style="display: none;">
                                            <el-button ref="UploadRef"></el-button>
                                        </el-upload>
                                        <el-button-group>
                                            <el-button :icon="Plus" @click="onRecordCreate" :disabled="protected">New</el-button>
                                            <el-button :icon="Delete" @click="onRecordDeleteBatch" :disabled="protected || (!data.selection.values.length && !data.selection.reversion)" type="danger">Delete</el-button>
                                        </el-button-group>
                                        <el-button-group style="margin-left: 8px;">
                                            <el-button :icon="Upload" :loading="button.upload.loading" @click="UploadClick" :disabled="protected">Import</el-button>
                                            <el-button :icon="Download" :disabled="protected || (!data.selection.values.length && !data.selection.reversion)" @click="onRecordExport">Export</el-button>
                                        </el-button-group>
                                    </div>
                                </div>
                            </template>
                            <el-table v-loading="data.loading" :data="data.records" stripe>
                                <el-table-column width="40">
                                    <template #header>
                                        <el-checkbox v-model="data.selection.reversion" :indeterminate="data.selection.values.length && data.selection.values.length < data.pagination.count" @change="onRecordSelectAll" />
                                    </template>
                                    <template #default="scope">
                                        <el-checkbox :model-value="data.selection.reversion !== data.selection.values.includes(scope.row.rowid)" @change="(v) => onRecordSelect(v, scope.row.rowid)" />
                                    </template>
                                </el-table-column>
                                <el-table-column prop="rowid" width="60" label="#" />
                                <el-table-column v-for="col in column.records" :key="col.name" :prop="col.name" :label="col.name" :show-overflow-tooltip="true" />
                                <el-table-column v-if="!protected" width="86" fixed="right">
                                    <template #default="scope">
                                        <el-button link type="primary" :icon="Edit" @click="onRecordEdit(scope.row)" />
                                        <el-button link type="danger" :icon="Delete" @click="onRecordDelete(scope.row)" />
                                    </template>
                                </el-table-column>
                            </el-table>
                            <el-pagination v-if="data.pagination.count" v-model:current-page="data.pagination.index" v-model:page-size="data.pagination.size" @size-change="onRecordSizeChange" @current-change="onRecordPageChange" :page-sizes="data.pagination.sizes" layout="total, sizes, prev, pager, next, jumper" :total="data.pagination.count" style="padding: 12px 16px; justify-content: flex-end;" />
                        </el-card>
                    </template>
                    <el-card v-else shadow="never">
                        <el-empty description="Select a table or create a new one" :image-size="60" />
                    </el-card>
                </el-col>
            </el-row>
        </el-tab-pane>

        <el-tab-pane label="SQL" name="sql" lazy>
            <el-card shadow="never">
                <template #header>
                    <div style="display: flex; justify-content: flex-end; margin-bottom: 8px;">
                        <el-button type="primary" :icon="VideoPlay" link @click="onSqlExecute" :loading="sql.executing" :disabled="!sql.statement" />
                    </div>
                    <monaco-editor v-model="sql.statement" language="sql" height="180px" />
                </template>
                <el-table v-if="sql.records.length" :data="sql.records" stripe v-loading="sql.executing">
                    <el-table-column v-for="col in Object.keys(sql.records[0])" :key="col" :prop="col" :label="col" :show-overflow-tooltip="true" />
                </el-table>
                <el-empty v-else-if="!sql.executing" description="No data" />
            </el-card>
        </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="dialog.table.visible" title="New" width="640px">
        <el-form ref="TableFormRef" :model="dialog.table" :rules="rules.table" label-position="right" label-width="96px">
            <el-form-item label="Name" prop="name">
                <el-input v-model="dialog.table.name" placeholder="Enter table name" />
            </el-form-item>
            <el-form-item label="Columns">
                <div v-for="(col, idx) in dialog.table.columns" :key="idx" style="display: flex; gap: 6px; margin-bottom: 8px;">
                    <el-input v-model="col.name" placeholder="Name" style="flex: 1;" />
                    <el-select v-model="col.type" style="width: 110px;">
                        <el-option label="INTEGER" value="INTEGER" />
                        <el-option label="REAL" value="REAL" />
                        <el-option label="TEXT" value="TEXT" />
                        <el-option label="BLOB" value="BLOB" />
                        <el-option label="NUMERIC" value="NUMERIC" />
                    </el-select>
                    <el-checkbox v-model="col.pk" style="margin-right: auto;">PK</el-checkbox>
                    <el-checkbox v-model="col.nullable">Null</el-checkbox>
                    <el-input v-model="col.default" placeholder="Default" style="width: 90px;" />
                    <el-button link type="danger" :icon="Delete" @click="dialog.table.columns.splice(idx, 1)" :disabled="dialog.table.columns.length <= 1" />
                </div>
                <el-button :icon="Plus" @click='dialog.table.columns.push({ name: "", type: "TEXT", pk: false, nullable: true, default: "" })'>Add</el-button>
            </el-form-item>
            <el-form-item>
                <el-button type="primary" :loading="dialog.table.loading" @click="onTableSubmit">Submit</el-button>
                <el-button @click="dialog.table.visible = false">Cancel</el-button>
            </el-form-item>
        </el-form>
    </el-dialog>

    <el-dialog :key="dialog.column.visible" v-model="dialog.column.visible" :title="dialog.column.record ? 'Edit' : 'New'" width="480px">
        <el-form ref="ColumnFormRef" :model="dialog.column.form" :rules="rules.column" label-position="right" label-width="96px">
            <el-form-item label="Name" prop="name">
                <el-input v-model="dialog.column.form.name" placeholder="Column name" />
            </el-form-item>
            <el-form-item label="Type">
                <el-select v-model="dialog.column.form.type" :disabled="dialog.column.record" style="width: 100%;">
                    <el-option label="INTEGER" value="INTEGER" />
                    <el-option label="REAL" value="REAL" />
                    <el-option label="TEXT" value="TEXT" />
                    <el-option label="BLOB" value="BLOB" />
                    <el-option label="NUMERIC" value="NUMERIC" />
                </el-select>
            </el-form-item>
            <!-- SQLite does not support ALTER TABLE ADD COLUMN PRIMARY KEY -->
            <el-form-item label="Nullable">
                <el-checkbox v-model="dialog.column.form.nullable" :disabled="dialog.column.record" />
            </el-form-item>
            <el-form-item label="Default">
                <el-input v-model="dialog.column.form.default" placeholder="Default value" :disabled="dialog.column.record" />
            </el-form-item>
            <el-form-item>
                <el-button type="primary" :loading="dialog.column.loading" @click="onColumnSubmit">Submit</el-button>
                <el-button @click="dialog.column.visible = false">Cancel</el-button>
            </el-form-item>
        </el-form>
    </el-dialog>

    <el-dialog v-model="dialog.record.visible" :title="dialog.record.rowid ? 'Edit' : 'New'" width="480px">
        <el-form label-position="right" label-width="96px">
            <el-form-item v-for="col in column.records" :key="col.name" :label="col.name">
                <el-input v-model="dialog.record.values[col.name]" />
            </el-form-item>
            <el-form-item>
                <el-button type="primary" :loading="dialog.record.loading" @click="onRecordSubmit">Submit</el-button>
                <el-button @click="dialog.record.visible = false">Cancel</el-button>
            </el-form-item>
        </el-form>
    </el-dialog>

</template>

<script>
    const { ElMessage, ElMessageBox, } = ElementPlus

    export default {
        components: {
            "monaco-editor": $import("/components/MonacoEditor.vue"),
        },
        setup() {
            const { ref } = Vue
            const { Delete, Download, Edit, Plus, Upload, VideoPlay, } = ElementPlusIconsVue
            const UploadRef = ref()
            const ColumnUploadRef = ref()
            return {
                Delete,
                Download,
                Edit,
                Plus,
                Upload,
                VideoPlay,
                UploadRef,
                ColumnUploadRef,
                UploadClick: () => {
                    UploadRef.value.ref.click()
                },
                onImportClick: () => {
                    ColumnUploadRef.value.ref.click()
                },
            }
        },
        data() {
            return {
                mode: "table",
                table: {
                    record: "",
                    records: [],
                    loading: false,
                },
                column: {
                    records: [],
                    loading: false,
                },
                data: {
                    records: [],
                    selection: {
                        values: [],
                        reversion: false,
                    },
                    pagination: {
                        sizes: [10, 20, 50, 100],
                        size: 10,
                        index: 1,
                        count: 0,
                    },
                    loading: false,
                },
                sql: {
                    statement: "",
                    executing: false,
                    records: [],
                },
                button: {
                    upload: {
                        loading: false,
                    },
                },
                dialog: {
                    table: {
                        visible: false,
                        name: "",
                        loading: false,
                        columns: [{ name: "", type: "TEXT", pk: false, nullable: true, default: "" }],
                    },
                    column: {
                        visible: false,
                        record: null,
                        loading: false,
                        form: { name: "", type: "TEXT", nullable: true, default: "" },
                    },
                    record: {
                        visible: false,
                        rowid: null,
                        loading: false,
                    },
                },
                rules: {
                    table: {
                        name: [{ required: true, message: "Table name is required", trigger: "blur", }],
                    },
                    column: {
                        name: [{ required: true, message: "Column name is required", trigger: "blur", }],
                    },
                },
            }
        },
        computed: {
            protected() {
                return this.table.record && this.table.record === "source"
            },
        },
        mounted() {
            this.onTableFetch()
        },
        methods: {
            onColumnCreate() {
                this.dialog.column.record = null
                this.dialog.column.form = { name: "", type: "TEXT", nullable: true, default: "", }
                this.dialog.column.loading = false
                this.dialog.column.visible = true
            },

            onColumnSubmit() {
                const FormRef = this.$refs.ColumnFormRef
                if (!FormRef) {
                    return
                }
                FormRef.validate(valid => {
                    if (!valid) {
                        return
                    }
                this.dialog.column.loading = true
                const form = this.dialog.column.form
                fetch(`database?table=${this.table.record}${!!this.dialog.column.record ? `&column=${this.dialog.column.record.name}` : "&column"}`, {
                    method: "POST",
                    body: JSON.stringify(!!this.dialog.column.record
                        ? { name: form.name.trim(), }
                        : { name: form.name.trim(), type: form.type, nullable: form.nullable, default: form.default || null, }),
                }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Submit succeeded")
                        this.dialog.column.visible = false
                        this.onColumnFetch()
                        this.onRecordFetch(true)
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => {
                    ElMessage.error(e.message)
                }).finally(() => {
                    this.dialog.column.loading = false
                })
                })
            },

            onColumnDelete(col) {
                ElMessageBox.confirm(`Column "${col.name}" will be dropped permanently. Continue?`, "Warning", {
                    confirmButtonText: "Confirm",
                    type: "warning",
                    beforeClose: (action, instance, done) => {
                        if (action === "confirm") {
                            instance.confirmButtonLoading = true
                            instance.confirmButtonText = "Dropping..."
                            fetch(`database?table=${this.table.record}&column=${col.name}`, {
                                method: "DELETE",
                            }).then(r => r.json()).then(r => {
                                if (r.code === "0") {
                                    ElMessage.success("Drop succeeded")
                                    this.onColumnFetch()
                                    this.onRecordFetch(true)
                                } else {
                                    ElMessage.error(r.message)
                                }
                                instance.confirmButtonLoading = false
                            }).catch(e => {
                                ElMessage.error(e.message)
                                instance.confirmButtonLoading = false
                            }).finally(() => {
                                done()
                            })
                        } else {
                            done()
                        }
                    },
                }).catch(() => { })
            },

            onColumnExport() {
                if (!this.table.record) {
                    return
                }
                fetch(`database?table=${this.table.record}&column`).then(r => r.json()).then(r => {
                    if (r.code !== "0") {
                        return ElMessage.error(r.message)
                    }
                    const blob = new Blob([JSON.stringify(r.data, null, 2)], { type: "application/json", })
                    const a = document.createElement("a")
                    a.href = URL.createObjectURL(blob)
                    a.download = `${this.table.record}.column.json`
                    a.click()
                }).catch(e => {
                    ElMessage.error(e.message)
                })
            },

            onColumnFetch() {
                if (!this.table.record) {
                    return
                }
                this.column.loading = true
                fetch(`database?table=${this.table.record}&column`).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        this.column.records = r.data || []
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => {
                    ElMessage.error(e.message)
                }).finally(() => {
                    this.column.loading = false
                })
            },

            onColumnEdit(col) {
                this.dialog.column.record = col
                this.dialog.column.form = { name: col.name, type: col.type, nullable: col.nullable, default: col.default, }
                this.dialog.column.loading = false
                this.dialog.column.visible = true
            },

            onRecordSelect(checked, id) {
                if (this.data.selection.reversion !== checked) {
                    this.data.selection.values.push(id)
                } else {
                    this.data.selection.values.splice(this.data.selection.values.findIndex(v => v === id), 1)
                }
            },

            onRecordSelectAll(checked) {
                this.data.selection.values.length = 0
                this.data.selection.reversion = checked
            },

            onRecordCreate() {
                this.dialog.record.values = {}
                this.dialog.record.rowid = null
                this.data.columns.forEach(c => {
                    this.dialog.record.values[c] = ""
                })
                this.dialog.record.visible = true
            },

            onRecordDelete(row) {
                ElMessageBox.confirm("This record will be deleted permanently. Continue?", "Warning", {
                    confirmButtonText: "Confirm",
                    type: "warning",
                    beforeClose: (action, instance, done) => {
                        if (action === "confirm") {
                            instance.confirmButtonLoading = true
                            instance.confirmButtonText = "Deleting..."
                            fetch(`database?table=${this.table.record}&record=${row.rowid}`, { method: "DELETE", }).then(r => r.json()).then(r => {
                                if (r.code === "0") {
                                    ElMessage.success("Delete succeeded")
                                    this.data.selection.values = this.data.selection.values.filter(v => v !== row.rowid)
                                    this.onRecordFetch()
                                } else {
                                    ElMessage.error(r.message)
                                }
                                instance.confirmButtonLoading = false
                            }).catch(e => {
                                ElMessage.error(e.message)
                                instance.confirmButtonLoading = false
                            }).finally(() => {
                                done()
                            })
                        } else {
                            done()
                        }
                    },
                }).catch(() => { })
            },

            onRecordDeleteBatch() {
                if (!this.data.selection.values.length && !this.data.selection.reversion) {
                    return
                }
                if (this.data.selection.reversion && this.data.selection.values.length === this.data.pagination.count) {
                    ElMessage.warning("No records to delete")
                    return
                }
                ElMessageBox.confirm("Delete records permanently. Continue?", "Warning", {
                    confirmButtonText: "Confirm",
                    type: "warning",
                    beforeClose: (action, instance, done) => {
                        if (action === "confirm") {
                            instance.confirmButtonLoading = true
                            instance.confirmButtonText = "Deleting..."
                            fetch(`database?table=${this.table.record}&record=${this.data.selection.values.join(",")}${this.data.selection.reversion ? "&reversion" : ""}`, { method: "DELETE", }).then(r => r.json()).then(r => {
                                if (r.code === "0") {
                                 ElMessage.success(`Deleted ${r.data || 0} records`)
                                    this.data.selection.values.length = 0
                                    this.data.selection.reversion = false
                                    this.onRecordFetch()
                                } else {
                                    ElMessage.error(r.message)
                                }
                                instance.confirmButtonLoading = false
                            }).catch(e => {
                                ElMessage.error(e.message)
                                instance.confirmButtonLoading = false
                            }).finally(() => {
                                done()
                            })
                        } else {
                            done()
                        }
                    },
                }).catch(() => { })
            },

            onRecordEdit(row) {
                this.dialog.record.values = { ...row, }
                this.dialog.record.rowid = row.rowid
                this.dialog.record.visible = true
            },

            onRecordExport() {
                if (!this.table.record) {
                    return
                }
                const a = document.createElement("a")
                a.href = `database?table=${this.table.record}&record=${this.data.selection.values.join(",")}${this.data.selection.reversion ? "&reversion" : ""}&bulk`
                a.download = `${this.table.record}.json`
                a.click()
            },

            onRecordFetch(reset) {
                if (!this.table.record) {
                    return
                }
                if (reset) {
                    this.data.pagination.index = 1
                    this.data.selection.values.length = 0
                    this.data.selection.reversion = false
                }
                this.data.loading = true
                fetch(`database?table=${this.table.record}&record&from=${(this.data.pagination.index - 1) * this.data.pagination.size}&size=${this.data.pagination.size}`).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        this.data.records = r.data.records || []
                        this.data.columns = this.column.records.map(c => c.name)
                        this.data.pagination.count = r.data.total || 0
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => {
                    ElMessage.error(e.message)
                }).finally(() => {
                    this.data.loading = false
                })
            },

            onRecordImport(file) {
                if (!this.table.record) {
                    return
                }
                const that = this,
                    reader = new FileReader()
                that.button.upload.loading = true
                reader.onload = function () {
                    let parsed
                    try {
                        parsed = JSON.parse(this.result)
                    } catch (e) {
                        ElMessage.error("Invalid JSON format")
                        that.button.upload.loading = false
                        return
                    }
                    fetch(`database?table=${that.table.record}&record`, {
                        method: "POST",
                        body: JSON.stringify(parsed),
                    }).then(r => r.json()).then(r => {
                        if (r.code === "0") {
                            ElMessage.success(`Imported ${r.data || 0} records`)
                            that.onTableFetch()
                            if (that.table.record) {
                                that.onRecordFetch(true)
                            }
                        } else {
                            ElMessage.error(r.message)
                        }
                    }).catch(e => {
                        ElMessage.error(e.message)
                    }).finally(() => {
                        that.button.upload.loading = false
                    })
                }
                reader.readAsText(file.raw, "utf-8")
            },

            onRecordPageChange(v) {
                this.data.pagination.index = v
                this.onRecordFetch()
            },

            onRecordSizeChange(v) {
                this.data.pagination.size = v
                this.onRecordFetch()
            },

            onRecordSubmit() {
                const vals = this.dialog.record.values
                if (!Object.keys(vals).filter(k => k !== "rowid" && vals[k] !== "").length) {
                    return ElMessage.error("At least one field is required")
                }
                this.dialog.record.loading = true
                const body = this.dialog.record.rowid ? [{ ...vals, rowid: this.dialog.record.rowid }] : [vals]
                fetch(`database?table=${this.table.record}&record`, {
                    method: "POST",
                    body: JSON.stringify(body),
                }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        ElMessage.success("Submit succeeded")
                        this.dialog.record.visible = false
                        this.onRecordFetch()
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => {
                    ElMessage.error(e.message)
                }).finally(() => {
                    this.dialog.record.loading = false
                })
            },

            onSqlExecute() {
                if (!this.sql.statement) {
                    return
                }
                this.sql.executing = true
                this.sql.records = []
                fetch("database?sql", {
                    method: "POST",
                    body: JSON.stringify({ statement: this.sql.statement, params: [], }),
                }).then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        this.sql.records = r.data.records
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => {
                    ElMessage.error(e.message)
                }).finally(() => {
                    this.sql.executing = false
                })
            },

            onTableCreate() {
                this.dialog.table.name = ""
                this.dialog.table.columns = [{ name: "", type: "TEXT", pk: false, nullable: true, default: "", }]
                this.dialog.table.loading = false
                this.dialog.table.visible = true
            },

            onTableSubmit() {
                const FormRef = this.$refs.TableFormRef
                if (!FormRef) {
                    return
                }
                FormRef.validate(valid => {
                    if (!valid) {
                        return
                    }
                    const name = this.dialog.table.name.trim(),
                        cols = this.dialog.table.columns.filter(c => c.name.trim())
                    if (!cols.length) {
                        return ElMessage.error("At least one column is required")
                    }
                    this.dialog.table.loading = true
                    fetch(`database?table=${name}`, {
                        method: "POST",
                        body: JSON.stringify({
                            columns: cols.map(c => ({ name: c.name.trim(), type: c.type, pk: c.pk, nullable: c.nullable, default: c.default || null, })),
                        }),
                    }).then(r => r.json()).then(r => {
                        if (r.code === "0") {
                            ElMessage.success("Create succeeded")
                            this.dialog.table.visible = false
                            this.onTableFetch()
                        } else {
                            ElMessage.error(r.message)
                        }
                    }).catch(e => {
                        ElMessage.error(e.message)
                    }).finally(() => {
                        this.dialog.table.loading = false
                    })
                })
            },

            onTableDelete(table) {
                ElMessageBox.confirm(`"${table}" will be dropped permanently. Continue?`, "Warning", {
                    confirmButtonText: "Confirm",
                    type: "warning",
                    beforeClose: (action, instance, done) => {
                        if (action === "confirm") {
                            instance.confirmButtonLoading = true
                            instance.confirmButtonText = "Dropping..."
                            fetch(`database?table=${table}`, { method: "DELETE", }).then(r => r.json()).then(r => {
                                if (r.code === "0") {
                                    ElMessage.success("Drop succeeded")
                                    if (this.table.record === table) {
                                        this.table.record = ""
                                        this.data.columns = []
                                        this.data.records = []
                                        this.data.pagination.count = 0
                                        this.column.records = []
                                    }
                                    this.onTableFetch()
                                } else {
                                    ElMessage.error(r.message)
                                }
                                instance.confirmButtonLoading = false
                            }).catch(e => {
                                ElMessage.error(e.message)
                                instance.confirmButtonLoading = false
                            }).finally(() => {
                                done()
                            })
                        } else {
                            done()
                        }
                    },
                }).catch(() => { })
            },

            onTableFetch() {
                this.table.loading = true
                fetch("database?table").then(r => r.json()).then(r => {
                    if (r.code === "0") {
                        this.table.records = r.data || []
                    } else {
                        ElMessage.error(r.message)
                    }
                }).catch(e => {
                    ElMessage.error(e.message)
                }).finally(() => {
                    this.table.loading = false
                })
            },

            onTableImport(file) {
                const that = this,
                    reader = new FileReader()
                const m = file.name.match(/^([a-zA-Z0-9_]+)\.column\.json$/)
                if (!m) {
                    return ElMessage.error("Invalid file name, expected <name>.column.json")
                }
                const name = m[1]
                reader.onload = function () {
                    let parsed
                    try {
                        parsed = JSON.parse(this.result)
                    } catch (e) {
                        ElMessage.error("Invalid JSON format")
                        return
                    }
                    fetch(`database?table=${name}`, {
                        method: "POST",
                        body: JSON.stringify({ columns: parsed, }),
                    }).then(r => r.json()).then(r => {
                        if (r.code === "0") {
                            ElMessage.success("Import succeeded")
                            that.onTableFetch()
                            that.table.record = name
                            that.onColumnFetch()
                            that.onRecordFetch(true)
                        } else {
                            ElMessage.error(r.message)
                        }
                    }).catch(e => {
                        ElMessage.error(e.message)
                    })
                }
                reader.readAsText(file.raw, "utf-8")
            },

            onTableSelect(table) {
                this.table.record = table
                this.data.selection.values.length = 0
                this.data.selection.reversion = false
                this.onColumnFetch()
                this.onRecordFetch(true)
            },
        },
    }
</script>

<style scoped>
    :deep(.el-pagination__total) {
        flex: auto;
    }
    :deep(.el-card__header) {
        padding: 8px 12px;
    }
    :deep(.el-card__body) {
        padding: 0;
    }
    .tables > div {
        display: flex;
        align-items: center;
        padding: 10px 16px;
        cursor: pointer;
        font-size: 13px;
        transition: background-color .15s;
    }
    .tables > div + div {
        border-top: 1px solid var(--el-border-color-lighter);
    }
    .tables > div:hover {
        background-color: var(--el-fill-color-light);
    }
    .tables > div.active {
        background-color: var(--el-color-primary-light-9);
        color: var(--el-color-primary);
    }
</style>
