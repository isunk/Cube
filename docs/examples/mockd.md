# Mock server

1. Create a controller with url `/service/mockd` and method `Any`.
    ```typescript
    //?name=mockd&type=controller&url=mockd{name}&method=&tag=mock
    import * as JSON5 from "https://cdn.bootcdn.net/ajax/libs/json5/2.2.3/index.min.js"
    import { helper, ColumnType } from "./DbHelper"

    abstract class Record {
        ID?: number
    }

    class Collection extends Record {
        Name: string
        PreRequestScript: string
        Variables: string
    }

    class Service extends Record {
        CollectionID: number
        Active: boolean
        RequestMethod: string
        RequestURL: string
        ResponseCode: number
        ResponseHeaders: string
        ResponseBody: string
        PreResponseScript: string
    }

    class CorsServiceResponse extends ServiceResponse {
        constructor(status = 200, headers = undefined, data = undefined) {
            super(
                status,
                {
                    "Access-Control-Allow-Origin": "*",
                    "Access-Control-Allow-Methods": "*",
                    "Access-Control-Allow-Headers": "*",
                    ...headers,
                },
                data,
            )
        }
    }

    interface ServiceStrategy {
        run()
    }

    class MetadataStrategy<T> implements ServiceStrategy {
        private method: string

        private requestBody: Buffer

        private params: { ID: string, [name: string]: string }

        private table: string

        private isSetup: boolean

        constructor(method: string, requestBody: Buffer, params: any) {
            this.method = method
            this.requestBody = requestBody
            const { t, ...p } = params
            this.params = p
            this.table = "Mock" + t
            this.isSetup = "setup" in params
        }

        run() {
            if (this.isSetup) {
                return this.setup()
            }
            if (this.table && !["MockCollection", "MockService"].includes(this.table)) {
                throw new Error("invalid table")
            }
            switch (this.method) {
                case "POST":
                    return this.post(this.table, this.requestBody.toJson())
                case "DELETE":
                    return this.delete(this.table, this.params.ID.split(","))
                case "PUT":
                    return this.put(this.table, this.params.ID, this.requestBody.toJson())
                case "GET":
                    return this.get(this.table, this.params)
                default:
                    return new ServiceResponse(405)
            }
        }

        public setup() {
            helper.dropTable("MockCollection")
            helper.createTable("MockCollection", [
                { name: "Name", type: ColumnType.String, },
                { name: "PreRequestScript", type: ColumnType.Text, },
                { name: "Variables", type: ColumnType.Text, },
            ])
            helper.dropTable("MockService")
            helper.createTable("MockService", [
                { name: "CollectionID", type: ColumnType.Integer, },
                { name: "Active", type: ColumnType.Boolean, },
                { name: "RequestMethod", type: ColumnType.String, },
                { name: "RequestURL", type: ColumnType.String, },
                { name: "ResponseCode", type: ColumnType.Integer, },
                { name: "ResponseHeaders", type: ColumnType.Text, },
                { name: "ResponseBody", type: ColumnType.Text, },
                { name: "PreResponseScript", type: ColumnType.Text, },
            ])
        }

        public post(table: string, input: any | any[]) {
            return (Array.isArray(input) ? input : [input]).map(i => helper.insert(table, i))
        }

        public delete(table: string, ids: string[]) {
            return helper.delete(table, {
                conditions: [{ field: "ID", operator: "in", value: ids }],
                conjunction: "AND",
            })
        }

        public put(table: string, id: string, input: any) {
            const record = helper.select(table, {
                conditions: [{ field: "ID", operator: "=", value: id }],
                conjunction: "AND",
            }).pop()
            if (!record) {
                throw new Error("record not found")
            }
            return helper.update(table, {
                conditions: [{ field: "ID", operator: "=", value: id }],
                conjunction: "AND",
            }, this.toPutData(input, record))
        }

        public get(table: string, params: { [name: string]: string }) {
            return helper.select(table, {
                conditions: Object.entries(params).map(([field, value]) => {
                    return { field, operator: "=", value }
                }),
                conjunction: "AND",
            })
        }

        private toPutData(data, record) {
            const merge = (a, [start, del, add, checksum]) => {
                const b = a.slice(0, start) + add + a.slice(start + del)
                let hash = 5381
                for (let i = 0; i < b.length; i++) {
                    hash = (hash << 5) + hash + b.charCodeAt(i)
                }
                if (checksum !== (hash >>> 0) % 65535) {
                    throw new Error("check hash failed")
                }
                return b
            }
            return Object.fromEntries(
                Object.entries(data)
                    .map(([k, v]) => {
                        if (["ResponseBody", "PreRequestScript"].includes(k) && Array.isArray(v) && v.length === 4) {
                            return [k, merge(record[k], v as [any, any, any, any])]
                        }
                        return [k, v]
                    })
            )
        }
    }

    class MockStrategy implements ServiceStrategy {
        private method: string

        private requestBody: string

        private name: string

        private callback: string

        constructor(method: string, requestBody: Buffer, params: any, name: string) {
            this.method = method
            this.requestBody = params.b ?? requestBody?.toString()
            this.name = (name || params.u)?.replace(/^\//, "")
            this.callback = params.c
        }

        run() {
            if (this.method === "OPTIONS") {
                return new CorsServiceResponse(200)
            }

            try {
                const response = this.mock(this.name, this.requestBody && JSON.parse(decodeURIComponent(this.requestBody)))
                if (this.callback) {
                    return new ServiceResponse(200, undefined, `mockc.callbacks["${this.callback}"](${JSON.stringify(response)})`)
                }
                const isJson = /"content-type":"application\/json/i.test(JSON.stringify(response.headers))
                return new CorsServiceResponse(response.status, response.headers, isJson ? JSON.stringify(response.body) : response.body)
            } catch (err) {
                let status = 500
                if (err.message === "service not found") {
                    status = 404
                }
                return this.callback ? new ServiceResponse(200, undefined, `mockc.callbacks["${this.callback}"](${JSON.stringify({ status, error: err.message })})`) : new ServiceResponse(status, undefined, { error: err.message })
            }
        }

        private mock(url: string, requestBody: any): { status: number; headers: any; body: any; } {
            const service = helper.query(`
                SELECT
                    s.CollectionID CollectionID,
                    s.RequestMethod RequestMethod,
                    s.ResponseCode ResponseCode,
                    s.ResponseHeaders ResponseHeaders,
                    s.ResponseBody ResponseBody,
                    s.PreResponseScript PreResponseScript,
                    c.Variables Variables,
                    c.PreRequestScript PreRequestScript
                FROM
                    MockService s
                    LEFT JOIN MockCollection c ON s.CollectionID = c.ID
                WHERE
                    s.Active = 1
                    AND s.RequestURL like ?
                LIMIT 1
            `, "%" + url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, ""))?.pop()
            if (!service) {
                throw new Error("service not found")
            }
            const context = {
                request: {
                    body: requestBody,
                },
                response: {
                    status: service.ResponseCode || 200,
                    headers: JSON.parse(service.ResponseHeaders || "{}"),
                    body: !!~service.ResponseHeaders.indexOf("json") ? this.json52any(service.ResponseBody || "{}") : service.ResponseBody,
                },
                variables: JSON.parse(service.Variables || "{}"),
            }
            if (service.PreRequestScript) {
                context.response.body = (new Function("$", service.PreRequestScript))(context) ?? context.response.body
            }
            if (service.PreResponseScript) {
                context.response.body = (new Function("$", service.PreResponseScript))(context) ?? context.response.body
            }
            const variables = JSON.stringify(context.variables)
            if (variables !== service.Variables) {
                helper.update("MockCollection", {
                    conditions: [{ field: "ID", operator: "=", value: service.CollectionID, }],
                    conjunction: "AND",
                }, { variables: variables })
            }
            return context.response
        }

        private json52any(text: string) {
            try {
                return JSON5.parse(text, undefined)
            } catch (e) {
                throw new Error("invalid json5: " + e.message)
            }
        }
    }

    export default (app => app.run.bind(app))(new class {
        public run(ctx: ServiceContext) {
            const params = Object.entries(ctx.getForm()).reduce((p, c) => { p[c[0]] = c[1]?.[0]; return p; }, {}),
                name = ctx.getPathVariables().name
            if ("test" in params || name) {
                return new MockStrategy(ctx.getMethod(), ctx.getBody(), params, name).run()
            }
            return new MetadataStrategy(ctx.getMethod(), ctx.getBody(), params).run()
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
        <meta name="viewport" content="maximum-scale=1.0">
        <link rel="stylesheet" href="/libs/element-plus/2.10.5/index.min.css" />
        <script src="/libs/vue/3.5.18/vue.global.prod.min.js"></script>
        <script src="/libs/element-plus/2.10.5/index.full.min.js"></script>
        <script src="/libs/element-plus-icons-vue/2.3.1/index.iife.min.js"></script>
        <base target="_blank" />
        <style>
            html, body {
                height: 100%;
                margin: 0;
                background-color: #f0f2f5;
            }
            .el-dialog__header {
                display: flex;
                align-items: center;
            }
            .el-dialog__header .el-link {
                margin-left: 16px;
            }
            .el-dialog__headerbtn {
                top: 10px;
            }
            .el-dialog {
                display: flex;
                flex-direction: column;
            }
            .el-dialog__body {
                flex-grow: 1;
            }
            .el-dialog__body .el-tabs, .el-tab-pane {
                height: 500px;
            }
            .is-fullscreen .el-dialog__body .el-tabs, .el-tab-pane {
                height: 100%;
            }
        </style>
    </head>

    <body>
        <div id="app" v-cloak style="padding: 32px; position: relative;">
            <el-card>
                <el-row>
                    <el-select v-model="collection.ID" placeholder="Select a collection" clearable @change="onCollectionSelect" style="flex-grow: 1; width: fit-content;">
                        <el-option v-for="item in collection.records" :key="item.ID" :label="item.Name" :value="item.ID"></el-option>
                    </el-select>
                    <div style="margin-left: auto; display: inline-flex;">
                        <el-button-group style="padding-left: 5px;">
                            <el-button :disabled="+collection.ID" :icon="Plus" @click="onCollectionDialogOpen()"></el-button>
                            <el-button :disabled="!collection.ID" :icon="Edit" @click="onCollectionDialogOpen(collection.ID)"></el-button>
                            <el-button :disabled="!collection.ID" :icon="Delete" @click="onCollectionDelete"></el-button>
                            <el-button :disabled="+collection.ID" :icon="Upload" @click="() => this.$refs.CollectionUploadRef.ref.click()"></el-button><el-upload :auto-upload="false" action="" :on-change="onCollectionImport" :show-file-list="false" accept=".json" style="display: none;"><el-button ref="CollectionUploadRef"></el-button></el-upload>
                            <el-button :disabled="!collection.ID" :icon="Download" @click="onCollectionExport"></el-button>
                        </el-button-group>
                    </div>
                </el-row>
            </el-card>
            <el-card style="margin-top: 32px;">
                <el-row>
                    <el-button-group style="padding-left: 5px;">
                        <el-button :disabled="!collection.ID" :icon="Plus" @click="onServiceDialogOpen()"></el-button>
                        <el-button :disabled="!collection.ID" :icon="Upload" @click="() => this.$refs.ServiceUploadRef.ref.click()"></el-button><el-upload :auto-upload="false" action="" :on-change="onServiceImport" :show-file-list="false" accept=".har" style="display: none;"><el-button ref="ServiceUploadRef"></el-button></el-upload>
                    </el-button-group>
                </el-row>
                <el-row style="margin-top: 12px;">
                    <el-table v-loading="service.loading" :data="service.records" :row-class-name="onServiceClass" @selection-change="(rows) => this.service.selections = rows" @row-click="onServiceSelect">
                        <el-table-column label="ID" width="60">
                            <template #default="scope">
                                {{ scope.row.ID }}
                            </template>
                        </el-table-column>
                        <el-table-column label="Method" width="80">
                            <template #default="scope">
                                {{ scope.row.RequestMethod || "Any" }}
                            </template>
                        </el-table-column>
                        <el-table-column label="URL" :show-overflow-tooltip="true">
                            <template #default="scope">
                                <el-link type="primary" @click="onServiceDialogOpen(scope.row.ID)">
                                    {{ scope.row.RequestURL }}
                                </el-link>
                            </template>
                        </el-table-column>
                        <el-table-column label="Status Code" width="120">
                            <template #default="scope">
                                {{ scope.row.ResponseCode }}
                            </template>
                        </el-table-column>
                        <el-table-column label="Body Size" width="100">
                            <template #default="scope">
                                {{ ((scope.row.ResponseBody?.length ?? 0) / 1024).toFixed(2) }} KB
                            </template>
                        </el-table-column>
                        <el-table-column label="Operation" width="100">
                            <template #default="scope">
                                <el-switch v-model="scope.row.Active" size="small" @change="fetch('PUT', 'Service', `ID=${scope.row.ID}`, { Active: scope.row.Active })">
                                </el-switch>
                                <el-button link type="danger" @click="onServiceDelete(scope.row.ID)" :icon="Delete">
                                </el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-pagination layout="total" :total="service.records.length" style="margin-top: 12px;">
                    </el-pagination>
                </el-row>
            </el-card>
            <el-dialog v-model="collection.dialog.visible" :fullscreen="collection.dialog.fullscreen">
                <template #header>
                    <el-input v-model="collection.dialog.draft.Name" placeholder="Please input collection name"></el-input>
                    <el-link underline="never" :icon="FullScreen" @click="collection.dialog.fullscreen = !collection.dialog.fullscreen"></el-link>
                    <el-link underline="never" :icon="Check" @click="onCollectionDialogSubmit"></el-link>
                </template>
                <el-tabs tab-position="left">
                    <el-tab-pane label="Pre-Request Script" lazy>
                        <monaco-editor v-model="collection.dialog.draft.PreRequestScript" language="typescript"></monaco-editor>
                    </el-tab-pane>
                    <el-tab-pane label="Variables" lazy>
                        <monaco-editor v-model="collection.dialog.draft.Variables" language="json"></monaco-editor>
                    </el-tab-pane>
                </el-tabs>
            </el-dialog>
            <el-dialog v-model="service.dialog.visible" title="Service" :fullscreen="service.dialog.fullscreen">
                <template #header>
                    <el-input v-model="service.dialog.draft.RequestURL" placeholder="Please input service url"></el-input>
                    <el-link underline="never" :icon="FullScreen" @click="service.dialog.fullscreen = !service.dialog.fullscreen"></el-link>
                    <el-link underline="never" :icon="Position" @click="onServiceDialogPreview"></el-link>
                    <el-link underline="never" :icon="Check" @click="onServiceDialogSubmit"></el-link>
                </template>
                <el-tabs tab-position="left">
                    <el-tab-pane label="Method">
                        <el-input v-model="service.dialog.draft.RequestMethod" placeholder="Any"></el-input>
                    </el-tab-pane>
                    <el-tab-pane label="Status Code">
                        <el-input v-model.number="service.dialog.draft.ResponseCode" type="number"></el-input>
                    </el-tab-pane>
                    <el-tab-pane label="Headers" lazy>
                        <monaco-editor v-model="service.dialog.draft.ResponseHeaders" language="json"></monaco-editor>
                    </el-tab-pane>
                    <el-tab-pane label="Body" lazy>
                        <monaco-editor v-model="service.dialog.draft.ResponseBody" :language="!!~service.dialog.draft.ResponseHeaders?.indexOf('json') ? 'json5' : 'html'"></monaco-editor>
                    </el-tab-pane>
                    <el-tab-pane label="Pre-Response Script" lazy>
                        <monaco-editor v-model="service.dialog.draft.PreResponseScript" language="typescript"></monaco-editor>
                    </el-tab-pane>
                </el-tabs>
            </el-dialog>
        </div>
        <script>
            const { ElMessage, ElMessageBox, } = ElementPlus
            Vue.createApp({
                setup() {
                    const { ref } = Vue
                    const { Check, Delete, Download, Edit, FullScreen, Plus, Position, Upload } = ElementPlusIconsVue
                    return {
                        Check, Delete, Download, Edit, FullScreen, Plus, Position, Upload,
                        CollectionUploadRef: ref(), ServiceUploadRef: ref(),
                    }
                },
                computed: {

                },
                data() {
                    return {
                        collection: {
                            ID: "",
                            records: [],
                            dialog: {
                                draft: {},
                                visible: false,
                                fullscreen: false,
                            },
                        },
                        service: {
                            ID: "",
                            records: [],
                            dialog: {
                                draft: {},
                                visible: false,
                                fullscreen: false,
                            },
                        },
                    }
                },
                methods: {
                    fetch(method, table, params = "", data = undefined) {
                        return fetch(`/service/mockd?t=${table}${params && "&" + params}`, {
                            method,
                            ...(data && { body: JSON.stringify(data) }),
                        }).then(r => {
                            if (r.status === 200) {
                                return r.json()
                            }
                            throw new Error(r.statusText)
    
                        }).then(r => {
                            return r.data
                        }).catch(e => {
                            ElMessage.error(e.message)
                            throw e
                        })
                    },
                    toPutData(data, record) {
                        const diff = (a, b) => {
                            let start = 0,
                                enda = a.length, endb = b.length
                            while (start < enda && start < endb && a[start] === b[start]) {
                                start++
                            }
                            while (enda < start && endb < start && a[enda - 1] === b[endb - 1]) {
                                enda--
                                endb--
                            }
                            let hash = 5381
                            for (let i = 0; i < b.length; i++) {
                                hash = (hash << 5) + hash + b.charCodeAt(i)
                            }
                            return [start, enda - start, b.slice(start, endb), (hash >>> 0) % 65535]
                        }
                        return Object.fromEntries(
                            Object.entries(data)
                                .filter(([k]) => ["ID"].includes(k) || data[k] !== record[k])
                                .map(([k, v]) => {
                                    if (["ResponseBody", "PreRequestScript"].includes(k)) {
                                        return [k, diff(record[k], v)]
                                    }
                                    return [k, v]
                                })
                        )
                    },

                    onCollectionLoad() {
                        return this.fetch("GET", "Collection").then(records => {
                            this.collection.records = records
                            if (!records.length) {
                                this.service.records = []
                                return
                            }
                            if (!this.collection.ID) {
                                this.collection.ID = records.at(0)?.ID
                            }
                            this.onCollectionSelect()
                        })
                    },
                    onCollectionImport(file) {
                        const that = this,
                            reader = new FileReader()
                        reader.onload = function () {
                            const { collection, services } = JSON.parse(this.result)
                            delete(collection.ID)
                            that.fetch("POST", "Collection", "", collection)
                                .then(([CollectionID]) => {
                                    return that.fetch("POST", "Service", "", services.map(i => {
                                        delete(i.ID)
                                        i.CollectionID = CollectionID
                                        return i
                                    }))
                                })
                                .then(() => {
                                    that.onCollectionLoad()
                                })
                        }
                        reader.readAsText(file.raw, "utf-8")
                    },
                    onCollectionExport() {
                        const a = document.createElement("a")
                        a.href = URL.createObjectURL(new Blob([JSON.stringify({
                            collection: this.collection.records.find(i => i.ID === this.collection.ID),
                            services: this.service.records,
                        })], { type: "text/plain" }))
                        a.download = Date.now() + ".json"
                        a.click()
                    },
                    onCollectionDelete() {
                        return ElMessageBox.confirm("Collection will be deleted permanently. Continue ?", "Warning", {
                            confirmButtonText: "Confirm",
                            type: "warning",
                            beforeClose: async (action, instance, done) => {
                                if (action === "confirm") {
                                    instance.confirmButtonLoading = true
                                    instance.confirmButtonText = "Delete..."
                                    const ID = this.service.records.map(i => i.ID)
                                    if (ID.length) {
                                        await this.fetch("DELETE", "Service", `ID=${ID.join(",")}`)
                                    }
                                    await this.fetch("DELETE", "Collection", `ID=${this.collection.ID}`)
                                    await this.onCollectionLoad()
                                }
                                done()
                            },
                        })
                    },
                    onCollectionDialogOpen(ID) {
                        this.collection.dialog.draft = {
                            Name: new Date().toISOString().replace(/[-T:\.Z]/g, ""),
                            PreRequestScript: "",
                            Variables: "{}",
                            ...this.collection.records.find(i => i.ID === ID),
                        }
                        this.collection.dialog.visible = true
                    },
                    onCollectionDialogSubmit() {
                        return Promise.resolve()
                            .then(() => {
                                if (this.collection.dialog.draft.ID) {
                                    return this.fetch("PUT", "Collection", `ID=${this.collection.dialog.draft.ID}`, this.toPutData(this.collection.dialog.draft, this.collection.records.find(i => i.ID === this.collection.dialog.draft.ID) ?? {}))
                                }
                                return this.fetch("POST", "Collection", "", this.collection.dialog.draft)
                            })
                            .then(() => {
                                this.collection.dialog.visible = false
                                this.onCollectionLoad()
                            })
                    },
                    onCollectionSelect() {
                        return !this.collection.ID ? Promise.resolve() : this.fetch("GET", "Service", `CollectionID=${this.collection.ID}`).then(records => {
                            this.service.records = records.map(i => {
                                i.Active = !!i.Active
                                return i
                            })
                        })
                    },

                    onServiceImport(file) {
                        const that = this,
                            reader = new FileReader(),
                            cache = this.service.records.filter(i => i.Active).reduce((p, c) => { p[c.RequestURL] = false; return p; }, {})
                        reader.onload = function () {
                            return that.fetch("POST", "Service", "", JSON.parse(this.result).log.entries.filter(i => i._resourceType === "xhr").map(i => {
                                const RequestURL = i.request.url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, "")
                                return {
                                    CollectionID: that.collection.ID,
                                    Active: cache[RequestURL] ?? !(cache[RequestURL] = false),
                                    RequestMethod: i.request.method,
                                    RequestURL,
                                    ResponseCode: i.response.status,
                                    ResponseHeaders: JSON.stringify(i.response.headers.reduce((p, c) => {
                                        p[c.name] = c.value
                                        return p
                                    }, {})),
                                    ResponseBody: i.response.content?.text ?? "",
                                    PreResponseScript: "",
                                }
                            })).then(r => that.onCollectionSelect())
                        }
                        reader.readAsText(file.raw, "utf-8")
                    },
                    onServiceDelete(...ID) {
                        if (!ID.length) {
                            return
                        }
                        ElMessageBox.confirm("Service will be deleted permanently. Continue ?", "Warning", {
                            confirmButtonText: "Confirm",
                            type: "warning",
                            beforeClose: async (action, instance, done) => {
                                if (action === "confirm") {
                                    instance.confirmButtonLoading = true
                                    instance.confirmButtonText = "Delete..."
                                    await this.fetch("DELETE", "Service", `ID=${ID.join(",")}`)
                                        .then(() => {
                                            this.onCollectionSelect()
                                        })
                                }
                                done()
                            },
                        })
                    },
                    onServiceDialogOpen(ID) {
                        this.service.dialog.draft = {
                            CollectionID: this.collection.ID, Active: true,
                            RequestMethod: "",
                            RequestURL: "",
                            ResponseCode: 200,
                            ResponseHeaders: JSON.stringify({ "Content-Type":"application/json; charset=utf-8" }, undefined, 2),
                            ResponseBody: "{}",
                            PreResponseScript: "",
                            ...this.service.records.find(i => i.ID === ID),
                        }
                        this.service.dialog.visible = true
                    },
                    onServiceDialogSubmit() {
                        return Promise.resolve()
                            .then(() => {
                                if (this.service.dialog.draft.ID) {
                                    return this.fetch("PUT", "Service", `ID=${this.service.dialog.draft.ID}`, this.toPutData(this.service.dialog.draft, this.service.records.find(i => i.ID === this.service.dialog.draft.ID) ?? {}))
                                }
                                return this.fetch("POST", "Service", "", this.service.dialog.draft)
                            })
                            .then(() => {
                                this.service.dialog.visible = false
                                this.onCollectionSelect()
                            })
                    },
                    onServiceDialogPreview() {
                        window.open(`/service/mockd/${this.service.dialog.draft.RequestURL.replace(/^\//, "")}`)
                    },
                },
                mounted() {
                    this.onCollectionLoad()
                },
                components: {
                    "monaco-editor": {
                        template: `<div ref="container" :style="{ width: this.width, height: this.height }"></div>`,
                        props: {
                            modelValue: { type: String, default: "" },
                            width: { type: String, default: "100%" },
                            height: { type: String, default: "100%" },
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
                            require("/libs/monaco-editor/0.54.0/min/vs/loader.js")
                                .then(() => {
                                    window.require.config({ paths: { vs: window.location.origin + "/libs/monaco-editor/0.54.0/min/vs" } })
                                })
                                .then(() => {
                                    window.require(["vs/editor/editor.main"], () => {
                                        monaco.languages.register({ id: "json5" })
                                        // 定义 json5 语法高亮规则
                                        monaco.languages.setMonarchTokensProvider("json5", {
                                            tokenizer: {
                                                root: [
                                                    // 单行注释
                                                    [/\/\/.*/, "comment.single.json5"],
                                                    // 多行注释
                                                    [/\/\*/, "comment.block.json5", "@comment"],
                                                    // 字符串
                                                    [/'/, "string.quoted.json5", "@stringSingle"], // 单引号
                                                    [/"/, "string.quoted.json5", "@stringDouble"], // 双引号
                                                    // 数字
                                                    [/0x[0-9a-fA-F]+/, "constant.hex.numeric.json5"], // 十六进制
                                                    [/[+-]?(\d*\.\d+|\d+)([eE][+-]?\d+)?/, "constant.dec.numeric.json5"], // 小数和整数
                                                    // 关键字常量
                                                    [/\b(?:true|false|null|Infinity|NaN)\b/, "constant.language.json5"],
                                                    // 对象和数组的标点符号
                                                    [/[{}]/, "punctuation.definition.dictionary.json5"],
                                                    [/\[\]/, "punctuation.definition.array.json5"],
                                                    [/,/, "punctuation.separator.json5"],
                                                ],
                                                comment: [
                                                    [/[^*]+/, "comment.block.json5"],
                                                    [/\*\//, "comment.block.json5", "@pop"],
                                                    [/\*/, "comment.block.json5"],
                                                ],
                                                stringSingle: [
                                                    [/[^\\']+/, "string.quoted.json5"],
                                                    [/\\./, "constant.character.escape.json5"],
                                                    [/'/, "string.quoted.json5", "@pop"],
                                                ],
                                                stringDouble: [
                                                    [/[^\\"]+/, "string.quoted.json5"],
                                                    [/\\./, "constant.character.escape.json5"],
                                                    [/"/, "string.quoted.json5", "@pop"],
                                                ],
                                            },
                                        })
                                        // 设置 json5 的自动缩进和括号补全
                                        monaco.languages.setLanguageConfiguration("json5", {
                                            autoClosingPairs: [
                                                { open: "{", close: "}" },
                                                { open: "[", close: "]" },
                                                { open: "\"", close: "\"" },
                                                { open: "'", close: "'" },
                                            ],
                                            brackets: [
                                                ["{", "}"],
                                                ["[", "]"],
                                            ],
                                            surroundingPairs: [
                                                { open: "{", close: "}" },
                                                { open: "[", close: "]" },
                                                { open: "\"", close: "\"" },
                                                { open: "'", close: "'" },
                                            ],
                                            indentationRules: {
                                                // 缩进规则：在 { 或 [ 后换行增加缩进， } 或 ] 前换行减少缩进
                                                increaseIndentPattern: /({|\[)[^\}\]]*$/,
                                                decreaseIndentPattern: /^[ \t]*(\}|\]),?$/,
                                            },
                                            comments: {
                                                lineComment: "//",
                                                blockComment: ["/*", "*/"],
                                            },
                                        })
                                        const editor = monaco.editor.create(container.value, {
                                            language: props.language,
                                            value: props.modelValue,
                                        })
                                        if (props.language === "typescript") {
                                            monaco.languages.typescript.typescriptDefaults.addExtraLib(`declare const $: { request: { body?: any; }; response: { status: number; headers: any; body: any; }; variables: any; [name: string]: any; }`, "global.ts")
                                        }
                                        editor.onDidChangeModelContent(() => {
                                            emit("update:modelValue", editor.getValue())
                                        })
                                        editor.updateOptions({ readOnly: props.readOnly ?? false })
                                        Vue.watch(() => props.modelValue, (newValue, oldValue) => {
                                            if (newValue !== oldValue && newValue !== editor.getValue()) {
                                                editor.setValue(newValue)
                                            }
                                        })
                                        new ResizeObserver((e) => editor.layout()).observe(container.value)
                                    })
                                })
                            return { container }
                        },
                    }
                },
            }).use(ElementPlus).mount("#app")
        </script>
    </body>

    </html>
    ```

3. Import [DbHelper](modules/dbhelper.md).

4. Setup database tables with visit [`/service/mockd?setup`](/service/mockd?setup)

5. Visit `/resource/mockd` and create a group with uploading a HAR file.

6. Inject mock client.
    - Using JSONP request with src `/service/mockd?test&u=...&c=...&b=...`
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
                script.src = `${endpoint}/service/mockd?test&u=${encodeURIComponent(url)}&c=${name}&b=${encodeURIComponent(body)}`
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
    - Using fetch request with src `/service/mockd?test&u=...`
        ```javascript
        window.mockc = (endpoint, url, options) => {
            // return fetch(`${endpoint}/service/mockd?test&u=${encodeURIComponent(url)}`, options)
            return fetch(`${endpoint}/service/mockd${url.replace(/^https?:\/\/[^\/]+/, "").replace(/^(?=[^\/])/, "/")}`, options)
        }
        ```
        For example
        ```javascript
        await mockc("http://127.0.0.1:8090", "/greeting", {
            method: "POST",
            body: JSON.stringify({
                name: "zhangsan",
            }),
        })
        ```

7. You can also reverse the server to android devices like that
    ```bash
    adb reverse tcp:8090 tcp:8090
    ```
