# Mock server

1. Create a controller with url `/service/mockd` and method `Any`.
    ```typescript
    //?name=mockd&type=controller&url=mockd&method=&tag=mock
    type SearchParams = {
        groupId?: number
        serviceId?: number
        setup?: boolean
        test?: boolean
        url?: string
        callback?: string
    }
    
    export default (app => app.run.bind(app))(new class {
        private db = $native("db")
    
        public run(ctx: ServiceContext) {
            const params = this.getSearchParams(ctx)
            switch (ctx.getMethod()) {
                case "GET":
                    if (params.setup) {
                        return this.setup()
                    }
                    if (params.test) {
                        return this.test(params.url, params.callback)
                    }
                    return this.get(params)
                case "PUT":
                    return this.put(params, ctx.getBody().toJson())
                case "POST":
                    return this.post(params)
                case "DELETE":
                    return this.delete(params)
                default:
                    throw new Error("no such method")
            }
        }
    
        public setup() {
            // this.db.exec("alter table MockService add column Sort INTEGER NOT NULL DEFAULT 0")
            this.db.exec(`
                DROP TABLE IF EXISTS MockGroup;
                CREATE TABLE IF NOT EXISTS MockGroup (
                    Name VARCHAR(64) PRIMARY KEY NOT NULL,
                    Active BOOLEAN NOT NULL DEFAULT false,
                    Progress INTEGER NOT NULL DEFAULT 0
                );
    
                DROP TABLE IF EXISTS MockService;
                CREATE TABLE IF NOT EXISTS MockService (
                    GroupId INTEGER NOT NULL,
                    Url VARCHAR(255) NOT NULL,
                    Output TEXT NOT NULL DEFAULT '',
                    Active BOOLEAN NOT NULL DEFAULT false,
                    Sort INTEGER NOT NULL DEFAULT 0
                );
            `);
        }
    
        public get(searchParams: SearchParams) {
            const { groupId } = searchParams
            let wheres = "WHERE GroupId in (SELECT rowid FROM MockGroup WHERE Active = 1)",
                params = []
            if (groupId !== undefined) {
                wheres = "WHERE GroupId = ?"
                params.push(groupId)
            }
            return {
                services: (this.db.query(`SELECT rowid, GroupId, Url, Output, Active, Sort FROM MockService ${wheres}`, ...params) ?? []).map(i => {
                    return {
                        id: i.rowid,
                        group: i.GroupId,
                        url: i.Url,
                        output: i.Output,
                        active: i.Active,
                        sort: i.Sort,
                    }
                }),
                groups: (this.db.query(`SELECT rowid, Name, Active, Progress FROM MockGroup`) ?? []).map(i => {
                    return {
                        id: i.rowid,
                        name: i.Name,
                        active: i.Active,
                        progress: i.Progress,
                    }
                }),
            }
        }
    
        public put(searchParams: SearchParams, jsonBody: any) {
            const { name, services } = jsonBody
            if (!services.length) {
                return 0
            }
            let groupId = searchParams.groupId
            this.db.transaction(tx => {
                const group = groupId && tx.query(`SELECT rowid, Name FROM MockGroup where rowid = ?`, groupId)?.pop()
                if (group) {
                    tx.exec(`DELETE FROM MockService WHERE GroupId = ?`, groupId)
                    tx.exec(`UPDATE MockGroup SET Name = ? WHERE rowid = ?`, name, groupId)
                } else {
                    tx.exec(`INSERT INTO MockGroup (Name, Active, Progress) VALUES (?, 1, 0)`, name)
                    groupId = tx.query("SELECT last_insert_rowid() id")[0].id
                }
                tx.exec(`INSERT INTO MockService (GroupId, Url, Output, Active, Sort) VALUES ${services.map(() => "(?, ?, ?, ?, ?)").join(",")}`, ...services.map(s => [groupId, s.url, s.output, s.active, s.sort]).flat())
                tx.exec(`UPDATE MockGroup SET Active = 0 WHERE rowid <> ?`, groupId)
            })
            return groupId
        }
    
        public post(searchParams: SearchParams) {
            const { groupId } = searchParams
            this.db.transaction(tx => {
                tx.exec(`UPDATE MockGroup SET Active = 1, Progress = 0 WHERE rowid = ?`, groupId)
                tx.exec(`UPDATE MockGroup SET Active = 0 WHERE rowid <> ?`, groupId)
            })
            return
        }
    
        public delete(searchParams: SearchParams) {
            const { groupId, serviceId } = searchParams
            if (serviceId !== undefined) {
                return this.db.exec(`DELETE FROM MockService WHERE rowid = ?`, serviceId)
            }
            if (groupId !== undefined) {
                let effect = 0
                this.db.transaction(tx => {
                    tx.exec(`DELETE FROM MockService WHERE GroupId = ?`, groupId)
                    effect = tx.exec(`DELETE FROM MockGroup WHERE rowid = ?`, groupId)
                })
                return effect
            }
            return 0
        }
    
        public test(url: string, callback: string) {
            url = url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, "")
            const rec = this.db.query(`
                SELECT
                    s.Sort Sort,
                    s.GroupId GroupId,
                    s.Output Output,
                    g.Progress Progress,
                    CASE
                        WHEN s.Sort < g.Progress + 2 THEN (g.Progress - s.rowid)
                        ELSE s.Sort - g.Progress + 999999
                    END Weight
                FROM
                    MockService s
                    LEFT JOIN MockGroup g ON s.GroupId = g.rowid
                WHERE
                    g.Active = 1
                    AND s.Active = 1
                    AND s.Url like ?
                ORDER BY
                    Weight ASC
                    LIMIT 1
            `, "%" + url + "%")?.pop()
            if (!rec) {
                return new ServiceResponse(200, undefined, `mockc.callbacks["${callback}"](${JSON.stringify({ status: 404, })})`)
            }
            if (rec.Sort > rec.Progress) {
                this.db.exec(`UPDATE MockGroup SET Progress = ? WHERE rowid = ?`, rec.Sort, rec.GroupId)
            }
            return new ServiceResponse(200, undefined, `mockc.callbacks["${callback}"](${JSON.stringify({ status: 200, body: JSON.parse(rec.Output), })})`)
        }
    
        private getSearchParams(ctx: ServiceContext) {
            const form = ctx.getForm(),
                output = {} as SearchParams
            if (/^\d+$/.test(form.g?.[0])) {
                output.groupId = Number(form.g?.[0])
            }
            if (/^\d+$/.test(form.s?.[0])) {
                output.serviceId = Number(form.s?.[0])
            }
            if ("setup" in form) {
                output.setup = true
            }
            if ("test" in form) {
                output.test = true
                if ("test" in form) {
                    output.url = form.url?.[0]
                    output.callback = form.callback?.[0]
                }
            }
            return output
        }
    })
    ```

2. Create a resource with url `/resource/mockd`.
    ```html
    //?name=mockd&type=resource&lang=html&url=mockd&tag=mock
    <!DOCTYPE html>
    <html>
    
    <head>
        <meta charset="UTF-8">
        <link rel="stylesheet" href="/libs/element-plus/2.9.1/index.min.css" />
        <script src="/libs/vue/3.5.13/vue.global.prod.min.js"></script>
        <script src="/libs/element-plus/2.9.1/index.full.min.js"></script>
        <script src="/libs/element-plus-icons-vue/2.3.1/index.iife.min.js"></script>
        <title>mockd</title>
        <base target="_blank" />
        <style>
            html,
            body {
                height: 100%;
                margin: 0;
                background-color: #f0f2f5;
            }
    
            .el-table {
                border-top: 1px solid #dcdfe6;
            }
    
            .el-pagination {
                flex: auto;
                margin-top: 13px;
            }
    
            .el-pagination .is-first {
                flex: auto;
            }
    
            .el-table .disabled {
                border-color: #e4e7ed;
                color: #c0c4cc;
                cursor: not-allowed;
            }
    
            .el-dialog {
                max-width: 720px;
            }
    
            .el-dialog .el-input-group__prepend .el-checkbox-group {
                margin: 0 -20px;
            }
    
            .el-dialog .el-input-group__prepend .el-checkbox-button__inner {
                border-right: 0;
                border-top-right-radius: 0 !important;
                border-bottom-right-radius: 0 !important;
                text-decoration: line-through;
                color: var(--el-disabled-text-color);
            }
    
            .el-dialog .el-input-group__prepend .is-checked .el-checkbox-button__inner {
                text-decoration: none;
                color: white;
            }
    
            .el-dialog .el-select {
                max-width: 180px;
            }
    
            .el-tag {
                max-width: 160px;
            }
    
            .el-tag .el-tag__content {
                overflow: hidden;
                text-overflow: ellipsis;
                line-height: 1rem;
            }
        </style>
    </head>
    
    <body>
        <div id="app" v-cloak style="padding: 32px; position: relative;">
            <el-card>
                <el-row style="padding-bottom: 10px;">
                    <el-select v-model="this['proxy.group.id']" placeholder="Select a group" clearable @change="(value) => value && onServiceFetch()" style="width: 240px">
                        <el-option v-for="group in group.records" :key="group.id" :label="group.name" :value="group.id">
                            <span v-if="group.active" style="font-weight: bolder;">{{ group.name }}</span>
                        </el-option>
                    </el-select>
                    <div style="margin-left: auto; display: inline-flex;">
                        <el-button-group style="padding-left: 5px;">
                            <el-button :icon="Check" @click="onBeforeGroupSave" v-if="service.records.length"></el-button>
                            <el-button :icon="VideoPlay" @click="onGroupPlay" v-if="group.record && service.records.length"></el-button>
                            <el-button :icon="Delete" @click="onGroupDelete" type="danger" v-if="group.record"></el-button>
                        </el-button-group>
                    </div>
                </el-row>
            </el-card>
            <br />
            <el-card>
                <el-row style="padding-bottom: 10px;">
                    <el-button-group style="padding-left: 5px;">
                        <el-upload :auto-upload="false" action="" :on-change="onServiceImport" :show-file-list="false" accept=".har" style="display: none;">
                            <el-button ref="UploadRef"></el-button>
                        </el-upload>
                        <el-button :icon="Upload" @click="() => this.$refs.UploadRef.ref.click()"></el-button>
                        <el-button :icon="Download" @click="onServiceExport" v-if="service.records.length"></el-button>
                        <el-button :icon="Delete" @click="onServiceDelete" v-if="service.selections.length"></el-button>
                    </el-button-group>
                </el-row>
                <el-row>
                    <el-table v-loading="service.loading" :data="service.records" :row-class-name="({ row }) => row.active ? '' : 'disabled'" @selection-change="(rows) => this.service.selections = rows">
                        <el-table-column type="selection" :selectable="(row) => row.active" width="40">
                        </el-table-column>
                        <el-table-column label="#" width="60">
                            <template #default="scope">
                                <span :style="{ color: group.record?.active && group.record?.progress === scope.row.sort ? '#409eff' : '#c0c4cc' }">{{ scope.row.sort = scope.$index }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column label="URL" prop="url" :show-overflow-tooltip="true">
                            <template #default="scope">
                                <el-link type="primary" @click="onServiceEdit(scope.row)">
                                    {{ scope.row.url }}
                                </el-link>
                            </template>
                        </el-table-column>
                        <el-table-column label="Size" width="100">
                            <template #default="scope">
                                {{ (scope.row.output.length / 1024).toFixed(2) }} KB
                            </template>
                        </el-table-column>
                        <el-table-column label="Group" prop="group" :show-overflow-tooltip="true" width="160">
                            <template #default="scope">
                                {{ group.records.find(i => i.id === scope.row.group)?.name }}
                            </template>
                        </el-table-column>
                        <el-table-column label="Operation" width="100">
                            <template #default="scope">
                                <el-switch v-model="scope.row.active" size="small" style="margin-right: 12px;">
                                </el-switch>
                                <el-button link type="danger" @click="service.records = service.records.filter(i => i != scope.row)" :icon="Delete">
                                </el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-pagination layout="total" :total="service.records.length">
                    </el-pagination>
                </el-row>
            </el-card>
            <el-dialog v-model="group.dialog.visible" :title="group.dialog.title">
                <el-form>
                    <el-form-item label="Name" style="margin: 12px 16px 16px 16px;">
                        <el-input v-model="group.dialog.name" />
                    </el-form-item>
                </el-form>
                <template #footer>
                    <el-button @click="group.dialog.visible = false">Cancel</el-button>
                    <el-button @click="onGroupSave" type="primary">Confirm</el-button>
                </template>
            </el-dialog>
            <el-dialog v-model="service.dialog.visible" title="Service">
                <el-form>
                    <el-form-item label="Name" style="margin: 12px 16px 16px 16px;">
                        <el-input v-model="service.dialog.record.url" />
                    </el-form-item>
                    <el-tabs tab-position="left">
                        <el-tab-pane label="Output">
                            <monaco-editor v-model="this['proxy.service.dialog.record.output']" height="512px" language="json"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Script" lazy>
                            <monaco-editor v-model="this['proxy.service.dialog.record.script']" height="512px" language="javascript"></monaco-editor>
                        </el-tab-pane>
                    </el-tabs>
                </el-form>
            </el-dialog>
        </div>
        <script>
            const { ElMessage, ElMessageBox, } = ElementPlus
            Vue.createApp({
                setup() {
                    const { ref } = Vue
                    const { Delete, Download, Edit, Search, Plus, Position, Upload, VideoPause, VideoPlay, Check, MostlyCloudy, } = ElementPlusIconsVue
                    return {
                        Delete, Download, Edit, Search, Plus, Position, Upload, VideoPause, VideoPlay, Check, MostlyCloudy,
                        FormRef: ref(),
                        UploadRef: ref(),
                    }
                },
                computed: {
                    "proxy.service.dialog.record.output": {
                        get() {
                            return JSON.stringify(JSON.parse(this.service.dialog.record.output), undefined, 2)
                        },
                        set(value) {
                            this.service.dialog.record.output = JSON.stringify(JSON.parse(value))
                        },
                    },
                    "proxy.service.dialog.record.script": {
                        get() {
                            return ""
                        },
                        set(value) {
                            
                        },
                    },
                    "proxy.group.id": {
                        get() {
                            return this.group.record?.id
                        },
                        set(value) {
                            this.group.record = this.group.records.find(i => i.id === value)
                        },
                    },
                },
                data() {
                    return {
                        group: {
                            record: {},
                            records: [],
                            dialog: {
                                title: "",
                                name: "",
                                visiable: false,
                            },
                        },
                        service: {
                            loading: false,
                            records: [],
                            selections: [],
                            dialog: {
                                record: {},
                                visiable: false,
                            },
                        },
                    }
                },
                methods: {
                    onBeforeGroupSave() {
                        const group = this.group.record
                        if (group) {
                            this.group.dialog.name = group.name
                            this.group.dialog.title = group.name
                        } else {
                            this.group.dialog.name = new Date().toISOString().replace(/[-T:\.Z]/g, "")
                            this.group.dialog.title = "New a group"
                        }
                        this.group.dialog.visible = true
                    },
                    onGroupSave() {
                        if (!this.group.dialog.name) {
                            ElMessage.warning("Group name is required")
                            return
                        }
                        fetch(`/service/mockd?g=${this["proxy.group.id"] ?? ""}`, {
                            method: "PUT",
                            body: JSON.stringify({
                                name: this.group.dialog.name,
                                services: this.service.records.map(i => {
                                    return {
                                        url: i.url,
                                        output: i.output,
                                        active: i.active,
                                        sort: i.sort,
                                    }
                                })
                            })
                        }).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            this.group.dialog.visible = false
                            ElMessage.success("Save succeeded")
                            this.onServiceFetch()
                        }).catch(e => {
                            ElMessage.error(e.message)
                        })
                    },
                    onGroupPlay() {
                        fetch(`/service/mockd?g=${this["proxy.group.id"] ?? ""}`, {
                            method: "POST",
                        }).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            this.group.dialog.visible = false
                            ElMessage.success("Active or reactive succeeded")
                            this.onServiceFetch()
                        }).catch(e => {
                            ElMessage.error(e.message)
                        })
                    },
                    onGroupDelete() {
                        ElMessageBox.confirm(`Group will be deleted permanently. Continue ?`, "Warning", {
                            confirmButtonText: "Confirm",
                            type: "warning",
                            beforeClose: (action, instance, done) => {
                                if (action === "confirm") {
                                    instance.confirmButtonLoading = true
                                    instance.confirmButtonText = "Delete..."
                                    fetch(`/service/mockd?g=${this["proxy.group.id"] ?? ""}`, {
                                        method: "DELETE",
                                    }).then(r => {
                                        if (r.status !== 200) {
                                            throw new Error(r.statusText)
                                        }
                                        this.service.selections = []
                                        this.service.records = []
                                        this.group.records = this.group.records.filter(i => i != this.group.record)
                                        this["proxy.group.id"] = undefined
                                        ElMessage.success("Delete succeeded")
                                    }).catch(e => {
                                        ElMessage.error(e.message)
                                    }).finally(() => {
                                        instance.confirmButtonLoading = false
                                    })
                                }
                                done()
                            },
                        }).catch(() => { })
                    },
                    onServiceImport(file) {
                        const that = this,
                            reader = new FileReader()
                        reader.onload = function () {
                            that.service.records.push(...JSON.parse(this.result).log.entries.filter(i => i._resourceType === "xhr").map(i => {
                                return {
                                    url: i.request.url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, ""),
                                    output: i.response.content?.text,
                                    active: true,
                                    input: i.request.postData?.text,
                                }
                            }))
                        }
                        reader.readAsText(file.raw, "utf-8")
                    },
                    onServiceExport() {
                        if (!this.service.records.length) {
                            return
                        }
                        const a = document.createElement("a")
                        a.href = URL.createObjectURL(new Blob([JSON.stringify({
                            log: {
                                creator: {
                                    name: "mockd",
                                    version: "0.1"
                                },
                                entries: this.service.records.map(i => {
                                    return {
                                        _resourceType: "xhr",
                                        request: {
                                            url: i.url,
                                            postData: {
                                                text: i.input,
                                            }
                                        },
                                        response: {
                                            status: 200,
                                            content: {
                                                text: i.output,
                                            }
                                        },
                                    }
                                })
                            }
                        })], { type: "text/plain" }))
                        a.download = Date.now() + ".har"
                        a.click()
                    },
                    onServiceFetch() {
                        this.service.loading = true
                        fetch(`/service/mockd?g=${this["proxy.group.id"] ?? ""}`).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            return r.json()
                        }).then(r => {
                            this.service.records = r.data.services
                            this.group.records = r.data.groups
                            this.group.record = r.data.groups.find(i => i.id === this["proxy.group.id"]) ?? r.data.groups.find(i => i.active)
                        }).catch(e => {
                            ElMessage.error(e.message)
                        }).finally(() => {
                            this.service.loading = false
                        })
                    },
                    onServiceDelete() {
                        this.service.records = this.service.records.filter(i => !this.service.selections.some(s => s == i))
                    },
                    onServiceEdit(record) {
                        this.service.dialog.record = record
                        this.service.dialog.visible = true
                    },
                },
                mounted() {
                    this.onServiceFetch()
                },
                components: {
                    "monaco-editor": {
                        template: `<div ref="container" :style="{ width: this.width, height: this.height }"></div>`,
                        props: {
                            modelValue: { type: String, default: "" },
                            width: { type: String, default: "100%" },
                            height: { type: String, default: "180px" },
                            language: { type: String, default: "typescript" },
                            readOnly: { type: Boolean, default: false },
                        },
                        emits: [
                            "update:modelValue",
                        ],
                        setup(props, { emit }) {
                            const container = Vue.ref(),
                                require = (src) => {
                                    const map = globalThis.jsloaders || (globalThis.jsloaders = new Map())
                                    if (!map.has(src)) {
                                        map.set(src, new Promise((resolve, reject) => {
                                            const e = document.createElement("script")
                                            e.src = src
                                            document.body.append(e)
                                            e.addEventListener("load", () => resolve(true))
                                            e.onerror = () => {
                                                document.body.removeChild(e)
                                                reject()
                                            }
                                        }))
                                    }
                                    return map.get(src)
                                }
                            require("/libs/monaco-editor/0.52.2/min/vs/loader.js")
                                .then(() => {
                                    window.require.config({ paths: { vs: window.location.origin + "/libs/monaco-editor/0.52.2/min/vs" } })
                                })
                                .then(() => {
                                    window.require(["vs/editor/editor.main"], () => {
                                        const editor = window.monaco.editor.create(container.value, {
                                            language: props.language,
                                            value: props.modelValue,
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
                                    })
                                })
                            return {
                                container,
                            }
                        },
                    }
                },
            }).use(ElementPlus).mount("#app")
        </script>
    </body>
    
    </html>
    ```

3. Setup database tables with visit [`/service/mockd?setup`](/service/mockd?setup)

4. Visit `/resource/mockd` and create a group with uploading a HAR file.

5. Inject mock client using JSONP with src `/service/mockd?test&url=...&callback=...`.
    ```javascript
    window.mockc = (endpoint, url, options) => {
        mockc.id = (mockc.id ?? -1) + 1
        mockc.callbacks = mockc.callbacks || []
        return new Promise((resolve, reject) => {
            const name = "C" + mockc.id,
                body = options?.body ?? "",
                script = document.createElement("script"),
                cleanup = () => {
                    document.body.removeChild(script)
                    delete mockc.callbacks[name]
                }
            script.src = `${endpoint}/service/mockd?test&url=${encodeURIComponent(url)}&callback=${name}&body=${encodeURIComponent(body)}`
            mockc.callbacks[name] = data => {
                resolve(data)
                cleanup()
            }
            script.onerror = () => {
                reject(new Error("Mock(JSONP) request failed"))
                cleanup()
            }
            document.body.appendChild(script)
        })
    }
    ```
    For example
    ```javascript
    const { status, body } = await mockc("http://127.0.0.1:8090", "/greeting", {
        body: JSON.stringify({
            name: "zhangsan",
        }),
    })
    ```
    You can also reverse the server to android devices like that
    ```bash
    adb reverse tcp:8090 tcp:8090
    ```
