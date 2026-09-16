# Mock Server

A comprehensive mock API server for development and testing. Supports service management with collections, pre-request/pre-response scripts, variable storage, and HAR file import. Features a full-featured management UI with Monaco Editor integration.

1. Create a controller with url `/service/mockd` and method `Any`.
    ```typescript
    //?name=mockd&type=controller&url=mockd{name}&method=&tag=mock
    import { helper, ColumnType } from "./DbHelper"

    const db = {
        query(stmt: string, params: any[]) {
            return Promise.resolve({ rows: helper.query(stmt, ...params) })
        }
    }

    export default (app => app.run.bind(app))(new class {
        public run(ctx: ServiceContext) {
            const params = Object.entries(ctx.getForm()).reduce((p, c) => { p[c[0]] = c[1]?.[0]; return p; }, {} as Record<string, any>),
                name = ctx.getPathVariables().name
            if ("setup" in params) {
                helper.dropTable("MockCollection")
                helper.createTable("MockCollection", [
                    { name: "Name", type: ColumnType.String, },
                    { name: "PreRequestScript", type: ColumnType.Text, },
                    { name: "Variables", type: ColumnType.Text, },
                    { name: "Libraries", type: ColumnType.Text, },
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
                return
            }
            if ("test" in params || name) {
                return new MockStrategy(ctx.getMethod(), ctx.getHeader(), ctx.getBody(), params, name).run()
            }
            return new MetadataStrategy(ctx.getMethod(), ["POST", "PUT"].includes(ctx.getMethod()) ? ctx.getBody()?.toJson() : "", params).run()
        }
    })

    interface Collection {
        ID?: number
        Name: string
        PreRequestScript: string
        Variables: string
        Libraries: string
    }

    interface Service {
        ID?: number
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
        constructor(status = 200, headers?: { [name: string]: string | number; }, data?: any) {
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
        run(): any
    }

    class MetadataStrategy implements ServiceStrategy {
        private method: string

        private body: any

        private query: { ID: string, [name: string]: string }

        private table: string

        constructor(method: string, body: Buffer, params: any) {
            this.method = method
            this.body = body
            const { t, ...query } = params
            this.query = query as typeof this.query
            this.table = "Mock" + t
        }

        async run() {
            if (this.table && !["MockCollection", "MockService"].includes(this.table)) {
                throw new Error("invalid table")
            }

            switch (this.method) {
                case "POST":
                    return new ServiceResponse(200, undefined, await this.post(this.table, this.body))
                case "DELETE":
                    return new ServiceResponse(200, undefined, await this.delete(this.table, this.query.ID.split(",")))
                case "PUT":
                    return new ServiceResponse(200, undefined, await this.put(this.table, this.query.ID, this.body))
                case "GET":
                    return new ServiceResponse(200, undefined, await this.get(this.table, this.query))
                default:
                    return new ServiceResponse(405)
            }
        }

        public async post(table: string, input: any | any[]) {
            return Promise.all((Array.isArray(input) ? input : [input]).map(async (i) => {
                const keys = this.columns(Object.keys(i))
                return (await db.query(`INSERT INTO ${table}(${keys.join(", ")}) VALUES(${keys.map(_ => "?").join(", ")})`, keys.map(c => i[c])))
            }))
        }

        public async delete(table: string, ids: string[]) {
            return await db.query(`DELETE FROM ${table} WHERE ID IN (${ids.map(() => "?").join(",")})`, ids)
        }

        public async put(table: string, id: string, input: any) {
            const record = (await db.query(`SELECT * FROM ${table} WHERE ID = ?`, [id])).rows[0]
            if (!record) {
                throw new Error("record not found")
            }
            const data = this.patch(input, record),
                columns = this.columns(Object.keys(data))
            return await db.query(`UPDATE ${table} SET ${columns.map(c => c + " = ?").join(", ")} WHERE ID = ?`, [...columns.map(c => data[c]), id])
        }

        public async get(table: string, input: { [name: string]: string }) {
            const entries = Object.entries(input)
            this.columns(entries.map(([name]) => name))
            return (await db.query(`SELECT * FROM ${table} WHERE ${entries.map(([name]) => `${name} = ?`).join(" AND ") || "1 = 1"} ORDER BY ID DESC`, entries.map(([, value]) => value))).rows
        }

        private columns(keys: string[]) {
            for (const k of keys) {
                if (!/^[A-Za-z][A-Za-z0-9_]*$/.test(k)) {
                    throw new Error(`invalid column: ${k}`)
                }
            }
            return keys
        }

        private patch(data: any, record: any) {
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
                    .filter(([k]) => k !== "ID")
                    .map(([k, v]) => {
                        if (["ResponseBody", "PreRequestScript", "Libraries"].includes(k) && Array.isArray(v) && v.length === 4) {
                            return [k, merge(record[k] ?? "", v as [any, any, any, any])]
                        }
                        return [k, v]
                    })
            )
        }
    }

    class MockStrategy implements ServiceStrategy {
        private method: string

        private body: any

        private name: string

        private callback: string

        private headers: Record<string, string>

        private query: Record<string, any>

        constructor(method: string, headers: Record<string, string>, body: Buffer, params: any, name: string) {
            this.method = method
            this.body = (params.b && JSON.parse(decodeURIComponent(params.b + ''))) || body || {}
            this.name = (name || params.u?.toString())?.replace(/^\//, "")
            this.callback = params.c?.toString()
            this.headers = headers
            const { b, u, c, t, ...query } = params
            this.query = query
        }

        async run() {
            if (this.method === "OPTIONS") {
                return new CorsServiceResponse(200)
            }

            try {
                const response = await this.mock(this.name)
                if (this.callback) {
                    return new ServiceResponse(200, undefined, `mockc.callbacks["${this.callback}"](${JSON.stringify(response)})`)
                }
                return new CorsServiceResponse(response.status, response.headers, response.body)
            } catch (err: any) {
                let status = 500
                if (err.message === "service not found") {
                    status = 404
                }
                return this.callback ? new ServiceResponse(200, undefined, `mockc.callbacks["${this.callback}"](${JSON.stringify({ status, error: err.message })})`) : new CorsServiceResponse(status, undefined, { error: err.message })
            }
        }

        private async mock(url: string) {
            const service = (await db.query(`
                SELECT
                    s.CollectionID AS CollectionID,
                    s.RequestMethod AS RequestMethod,
                    s.ResponseCode AS ResponseCode,
                    s.ResponseHeaders AS ResponseHeaders,
                    s.ResponseBody AS ResponseBody,
                    s.PreResponseScript AS PreResponseScript,
                    c.Variables AS Variables,
                    c.PreRequestScript AS PreRequestScript,
                    c.Libraries AS Libraries
                FROM
                    MockService s
                    LEFT JOIN MockCollection c ON s.CollectionID = c.ID
                WHERE
                    s.Active = 1
                    AND s.RequestURL LIKE ?
                LIMIT 1
            `, ["%" + url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, "").replace(/[\\%_]/g, "\\$&")])).rows[0]
            if (!service) {
                throw new Error("service not found")
            }
            const variables = JSON.parse(service.Variables || "{}"),
                libs = (() => { try { return JSON.parse(service.Libraries || "[]") } catch { return [] } })()
            const require = (name: string) => {
                const lib = libs.find((l: any) => l.name === name)
                if (!lib) throw new Error(`Module '${name}' not found`)
                const exported: string[] = []
                let code = lib.code
                    .replace(/export\s+(function|class|const|let|var)\s+([$A-Za-z_]\w*)/g, (_, kw, n) => { exported.push(n); return `${kw} ${n}` })
                    .replace(/export\s*\{([^}]+)\}/g, (_, names) => {
                        names.split(",").map((s: string) => {
                            const [orig, alias] = s.trim().split(/\s+as\s+/)
                            exported.push(alias ? `${alias}: ${orig}` : orig)
                        })
                        return ""
                    })
                if (exported.length) code += `\nreturn { ${exported.join(", ")} }`
                return (new Function(code))()
            }
            const context = {
                request: {
                    method: this.method,
                    url: "/" + this.name,
                    headers: this.headers,
                    query: this.query,
                    body: this.body,
                },
                response: {
                    status: service.ResponseCode || 200,
                    headers: JSON.parse(service.ResponseHeaders || "{}"),
                    body: service.ResponseHeaders.includes("json") ? this.parse(service.ResponseBody || "{}") : service.ResponseBody,
                },
                variables,
                session: {} as Record<string, any>,
            }
            const imports = (script: string) => script
                .replace(/import\s+(.*?)\s+from\s+['"]([^'"]+)['"]/g, (_, spec, path) => `const ${spec.includes("{") ? spec : spec.replace("* as ", "")} = require('${path}')`)
            const AsyncFunction = Object.getPrototypeOf(async function () { }).constructor
            for (const script of [service.PreRequestScript, service.PreResponseScript]) {
                const code = imports(script ?? "").trim()
                if (code) {
                    await new AsyncFunction("$", "require", code)(context, require)
                }
            }
            const newVariables = JSON.stringify(context.variables)
            if (newVariables !== service.Variables) {
                await db.query('UPDATE MockCollection SET Variables = ? WHERE ID = ?', [newVariables, service.CollectionID])
            }
            return context.response
        }

        private parse(text: string) {
            try {
                const stripped = text.replace(/("(?:[^"\\]|\\.)*")|\/\/.*$|\/\*[\s\S]*?\*\//gm, (_, s) => s || '')
                return JSON.parse(stripped.replace(/,(\s*[}\]])/g, '$1'))
            } catch (e: any) {
                throw new Error("invalid json: " + e.message)
            }
        }
    }
    ```

2. Create a resource with url `/resource/mockd`.
    ```html
    //?name=mockd&type=resource&lang=html&url=mockd&tag=mock
    <!DOCTYPE html>
    <html>

    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="maximum-scale=1.0">
        <link rel="stylesheet" href="https://s4.zstatic.net/ajax/libs/element-plus/2.10.5/index.min.css" />
        <script src="https://s4.zstatic.net/ajax/libs/vue/3.5.18/vue.global.prod.min.js"></script>
        <script src="https://s4.zstatic.net/ajax/libs/element-plus/2.10.5/index.full.min.js"></script>
        <script src="https://s4.zstatic.net/ajax/libs/element-plus-icons-vue/2.3.1/index.iife.min.js"></script>
        <base target="_blank" />
        <style>
            [v-cloak] {
                display: none;
            }
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
            .el-dialog__body .el-tabs {
                height: 500px;
            }
            .el-dialog__body .el-tab-pane {
                height: 100%;
            }
            .is-fullscreen .el-dialog__body .el-tabs {
                height: 100%;
            }
        </style>
    </head>

    <body>
        <div id="app" v-cloak style="padding: 32px; position: relative;">
            <el-card>
                <el-skeleton v-if="collection.loading" animated>
                    <template #template>
                        <div style="display: flex; align-items: center;">
                            <el-skeleton-item variant="text" style="width: 260px; height: 32px;"></el-skeleton-item>
                            <el-skeleton-item variant="text" style="width: 180px; height: 32px; margin-left: auto;"></el-skeleton-item>
                        </div>
                    </template>
                </el-skeleton>
                <el-row v-else>
                    <el-select v-model="collection.ID" placeholder="Select a collection" clearable @change="onCollectionSelect" style="flex-grow: 1; width: fit-content;">
                        <el-option v-for="item in collection.records" :key="item.ID" :label="item.Name" :value="item.ID"></el-option>
                    </el-select>
                    <div style="margin-left: auto; display: inline-flex;">
                        <el-button-group style="padding-left: 5px;">
                            <el-button :disabled="+collection.ID" :icon="Plus" @click="onCollectionDialogOpen()"></el-button>
                            <el-button :disabled="!collection.ID" :icon="Edit" @click="onCollectionDialogOpen(collection.ID)"></el-button>
                            <el-button :disabled="!collection.ID" :icon="Delete" @click="onCollectionDelete"></el-button>
                            <el-button :disabled="+collection.ID" :icon="Upload" @click="onCollectionImportOpen"></el-button><el-upload ref="json" :auto-upload="false" action="" :on-change="onCollectionImport" :show-file-list="false" accept=".json" style="display: none;"></el-upload>
                            <el-button :disabled="!collection.ID" :icon="Download" @click="onCollectionExport"></el-button>
                        </el-button-group>
                    </div>
                </el-row>
            </el-card>
            <el-card v-if="collection.ID" style="margin-top: 32px;">
                <el-row>
                    <el-button-group style="padding-left: 5px;">
                        <el-button :disabled="!collection.ID" :icon="Plus" @click="onServiceDialogOpen()"></el-button>
                        <el-button :disabled="!collection.ID" :icon="Upload" @click="onServiceImportOpen"></el-button><el-upload ref="har" :auto-upload="false" action="" :on-change="onServiceImport" :show-file-list="false" accept=".har" style="display: none;"></el-upload>
                    </el-button-group>
                    <el-input v-model="service.search" placeholder="Search URL" clearable style="width: 240px; margin-left: auto;"></el-input>
                </el-row>
                <el-row style="margin-top: 12px;">
                    <el-skeleton v-if="service.loading" :rows="6" animated></el-skeleton>
                    <template v-else>
                        <el-table :data="services">
                        <el-table-column label="ID" prop="ID" sortable width="80">
                            <template #default="scope">
                                {{ scope.row.ID }}
                            </template>
                        </el-table-column>
                        <el-table-column label="Method" width="80">
                            <template #default="scope">
                                {{ scope.row.RequestMethod || "Any" }}
                            </template>
                        </el-table-column>
                        <el-table-column label="URL" prop="RequestURL" sortable :show-overflow-tooltip="true">
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
                    </template>
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
                        <div style="display: flex; flex-direction: column; height: 100%;">
                            <div style="flex: 1; min-height: 0;">
                                <monaco-editor v-model="collection.dialog.draft.PreRequestScript" language="typescript" :types="types"></monaco-editor>
                            </div>
                        </div>
                    </el-tab-pane>
                    <el-tab-pane label="Variables" lazy>
                        <monaco-editor v-model="collection.dialog.draft.Variables" language="json"></monaco-editor>
                    </el-tab-pane>
                    <el-tab-pane label="Libraries" lazy>
                        <div style="display: flex; height: 100%;">
                            <div style="width: 180px; border-right: 1px solid var(--el-border-color); display: flex; flex-direction: column; flex-shrink: 0;">
                                <div style="display: flex; align-items: center; justify-content: flex-start; padding: 4px 8px;">
                                    <el-button link :icon="Plus" @click="onLibraryAdd" style="padding: 0; width: 24px;"></el-button>
                                </div>
                                <el-scrollbar style="flex: 1;">
                                    <el-menu v-if="libraries.length" :default-active="active !== null ? 'lib' + active : ''" @select="(index) => active = Number(index.slice(3))" style="border-right: none;">
                                        <el-menu-item v-for="(lib, index) in libraries" :key="index" :index="'lib' + index" style="display: flex; align-items: center; height: 36px; padding: 0 8px;">
                                            <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1;">{{ lib.name || "(unnamed)" }}</span>
                                            <el-button link type="danger" @click.stop="onLibraryRemove(index)" style="padding: 0; width: 24px;">
                                                <el-icon :size="12"><component :is="Delete"></component></el-icon>
                                            </el-button>
                                        </el-menu-item>
                                    </el-menu>
                                    <el-empty v-else description="No libraries yet" :image-size="40"></el-empty>
                                </el-scrollbar>
                            </div>
                            <div style="flex: 1; display: flex; flex-direction: column; padding-left: 8px; min-width: 0;">
                                <template v-if="active !== null && libraries[active]">
                                    <el-input v-model="libraries[active].name" placeholder="Library name (e.g. util)" style="flex-shrink: 0; margin-bottom: 8px;"></el-input>
                                    <div style="flex: 1; min-height: 0;">
                                        <monaco-editor v-model="libraries[active].code" language="javascript" :types="types"></monaco-editor>
                                    </div>
                                </template>
                                <el-empty v-else description="Select a library or click + to add one" style="margin: auto;"></el-empty>
                            </div>
                        </div>
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
                        <monaco-editor v-model="service.dialog.draft.ResponseBody" :language="language"></monaco-editor>
                    </el-tab-pane>
                    <el-tab-pane label="Pre-Response Script" lazy>
                        <div style="display: flex; flex-direction: column; height: 100%;">
                            <div style="flex: 1; min-height: 0;">
                                <monaco-editor v-model="service.dialog.draft.PreResponseScript" language="typescript" :types="types"></monaco-editor>
                            </div>
                        </div>
                    </el-tab-pane>
                </el-tabs>
            </el-dialog>
        </div>
        <script>
            const require = (() => {
                const load = src => {
                    const map = globalThis.jsloaders || (globalThis.jsloaders = new Map())
                    if (!map.has(src)) {
                        map.set(src, new Promise((resolve, reject) => {
                            const define = window.define
                            window.define = undefined // 临时摘除 AMD define，避免干扰脚本自身的模块检测
                            const e = document.createElement("script")
                            e.src = src
                            e.addEventListener("load", () => {
                                if (define) {
                                    window.define = define // 仅恢复已有的 define，避免 undefined 覆盖 loader 新注册的 define
                                }
                                resolve()
                            })
                            e.addEventListener("error", () => {
                                if (define) {
                                    window.define = define
                                }
                                document.body.removeChild(e)
                                map.delete(src)
                                console.error("failed to load script:", src)
                                reject()
                            })
                            document.body.append(e)
                        }))
                    }
                    return map.get(src)
                },
                    modules = {
                        monaco: () => load(`https://s4.zstatic.net/ajax/libs/monaco-editor/0.56.0/min/vs/loader.js`).then(() => window.require.config({ paths: { vs: "https://s4.zstatic.net/ajax/libs/monaco-editor/0.56.0/min/vs" } })).then(() => new Promise(resolve => window.require(["vs/editor/editor.main"], resolve))),
                    }
                return name => {
                    if (modules[name]) {
                        return new Promise(resolve => window[name] ? resolve(window[name]) : modules[name]().then(() => resolve(window[name])))
                    }
                    throw new Error(`unknown import: ${name}`)
                }
            })()
        </script>
        <script>
            const { ElMessage, ElMessageBox, } = ElementPlus
            Vue.createApp({
                setup() {
                    const { Check, Delete, Download, Edit, FullScreen, Plus, Position, Upload } = ElementPlusIconsVue
                    return {
                        Check, Delete, Download, Edit, FullScreen, Plus, Position, Upload,
                    }
                },
                computed: {
                    services() {
                        return this.service.records.filter(i => String(i.RequestURL ?? "").toLowerCase().includes(this.service.search.toLowerCase()))
                    },
                    language() {
                        const headers = this.parse(this.service.dialog.draft.ResponseHeaders, {}),
                            contentType = String(Object.entries(headers).find(([k]) => k.toLowerCase() === "content-type")?.[1] ?? "").toLowerCase()
                        return ["json", "html", "xml"].find(lang => contentType.includes(lang)) ?? "plaintext"
                    },
                    types() {
                        return [
                            `declare const $: { request: { method: string; url: string; headers: Record<string, string>; query: Record<string, any>; body: any; }; response: { status: number; headers: Record<string, string>; body: any; }; variables: any; session: any; }; declare const require: (name: string) => any;`,
                            ...[...new Map(this.libraries.filter(l => l.name).map(l => [l.name, l])).values()].map(l => `declare module "${l.name}" {${l.code || ""}}`),
                        ].filter(s => s.trim()).join("\n")
                    },
                },
                data() {
                    return {
                        collection: {
                            ID: "",
                            records: [],
                            loading: true,
                            dialog: {
                                draft: {},
                                visible: false,
                                fullscreen: false,
                            },
                        },
                        service: {
                            records: [],
                            search: "",
                            loading: true,
                            dialog: {
                                draft: {},
                                visible: false,
                                fullscreen: false,
                            },
                        },
                        libraries: [],
                        active: null,
                    }
                },
                methods: {
                    fetch(method, table, params = "", data = undefined) {
                        return fetch(`/service/mockd?t=${table}${params && "&" + params}`, {
                            method,
                            headers: {
                                "Content-Type": "application/json",
                            },
                            ...(data && { body: JSON.stringify(data) }),
                        }).then(r => {
                            if (r.status === 200) {
                                return r.json()
                            }
                            throw new Error(r.statusText)
                        }).then(r => {
                            return r
                        }).catch(e => {
                            ElMessage.error(e.message)
                            throw e
                        })
                    },
                    readFile(file, onLoad) {
                        const reader = new FileReader()
                        reader.onload = () => onLoad(reader.result)
                        reader.readAsText(file, "utf-8")
                    },
                    patch(data, record) {
                        const diff = (a, b) => {
                            let start = 0,
                                enda = a.length, endb = b.length
                            while (start < enda && start < endb && a[start] === b[start]) {
                                start++
                            }
                            while (enda > start && endb > start && a[enda - 1] === b[endb - 1]) {
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
                                .filter(([k]) => data[k] !== record[k])
                                .map(([k, v]) => {
                                    if (["ResponseBody", "PreRequestScript", "Libraries"].includes(k)) {
                                        return [k, diff(record[k] ?? "", v ?? "")]
                                    }
                                    return [k, v]
                                })
                        )
                    },
                    parse(text, fallback) {
                        if (!text) return fallback
                        try {
                            return JSON.parse(text)
                        } catch {
                            return fallback
                        }
                    },
                    onLibraryAdd() {
                        this.libraries.push({ name: "", code: "" })
                        this.active = this.libraries.length - 1
                    },
                    onLibraryRemove(index) {
                        this.libraries.splice(index, 1)
                        if (this.active === index) {
                            this.active = null
                        } else if (this.active !== null && index < this.active) {
                            this.active--
                        }
                    },

                    onCollectionLoad() {
                        this.collection.loading = true
                        return this.fetch("GET", "Collection").then(records => {
                            this.collection.records = records
                            this.collection.loading = false
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
                    onCollectionImportOpen() {
                        this.$refs.json.$el.querySelector("input").click()
                    },
                    onCollectionImport(file) {
                        this.readFile(file.raw, result => {
                            const { collection, services } = JSON.parse(result)
                            delete collection.ID
                            this.fetch("POST", "Collection", "", collection)
                                .then(([CollectionID]) => {
                                    return this.fetch("POST", "Service", "", services.map(i => {
                                        delete i.ID
                                        i.CollectionID = CollectionID
                                        return i
                                    }))
                                })
                                .then(() => {
                                    this.onCollectionLoad()
                                })
                        })
                    },
                    onCollectionExport() {
                        const link = document.createElement("a")
                        link.href = URL.createObjectURL(new Blob([JSON.stringify({
                            collection: this.collection.records.find(i => i.ID === this.collection.ID),
                            services: this.service.records,
                        })], { type: "text/plain" }))
                        link.download = Date.now() + ".json"
                        link.click()
                    },
                    onCollectionDelete() {
                        const name = this.collection.records.find(i => i.ID === this.collection.ID)?.Name
                        return ElMessageBox.prompt(`Please input "${name}" to confirm deletion`, "Warning", {
                            confirmButtonText: "Confirm",
                            type: "warning",
                            inputValidator: value => value === name ? true : "Name mismatch",
                            beforeClose: async (action, instance, done) => {
                                if (action === "confirm") {
                                    instance.confirmButtonLoading = true
                                    instance.confirmButtonText = "Delete..."
                                    await this.fetch("DELETE", "Collection", `ID=${this.collection.ID}`)
                                    this.collection.ID = ""
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
                            Libraries: "[]",
                            ...this.collection.records.find(i => i.ID === ID),
                        }
                        this.libraries = this.parse(this.collection.dialog.draft.Libraries, [])
                        this.active = this.libraries.length ? 0 : null
                        this.collection.dialog.visible = true
                    },
                    onCollectionDialogSubmit() {
                        this.collection.dialog.draft.Libraries = JSON.stringify(this.libraries)
                        return Promise.resolve()
                            .then(() => {
                                if (this.collection.dialog.draft.ID) {
                                    return this.fetch("PUT", "Collection", `ID=${this.collection.dialog.draft.ID}`, this.patch(this.collection.dialog.draft, this.collection.records.find(i => i.ID === this.collection.dialog.draft.ID) ?? {}))
                                }
                                return this.fetch("POST", "Collection", "", this.collection.dialog.draft)
                            })
                            .then(() => {
                                this.collection.dialog.visible = false
                                this.onCollectionLoad()
                            })
                    },
                    onCollectionSelect() {
                        if (!this.collection.ID) {
                            this.service.records = []
                            return Promise.resolve()
                        }
                        this.service.loading = true
                        return this.fetch("GET", "Service", `CollectionID=${this.collection.ID}`).then(records => {
                            this.service.records = records.map(i => {
                                i.Active = !!i.Active
                                return i
                            })
                        }).finally(() => {
                            this.service.loading = false
                        })
                    },

                    onServiceImportOpen() {
                        this.$refs.har.$el.querySelector("input").click()
                    },
                    onServiceImport(file) {
                        const cache = this.service.records.filter(i => i.Active).reduce((p, c) => { p[c.RequestURL] = false; return p; }, {})
                        this.readFile(file.raw, result => {
                            return this.fetch("POST", "Service", "", JSON.parse(result).log.entries.filter(i => i._resourceType === "xhr").map(i => {
                                const url = i.request.url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, "")
                                return {
                                    CollectionID: this.collection.ID,
                                    Active: cache[url] ?? !(cache[url] = false),
                                    RequestMethod: i.request.method,
                                    RequestURL: url,
                                    ResponseCode: i.response.status,
                                    ResponseHeaders: JSON.stringify(i.response.headers.reduce((p, c) => {
                                        p[c.name] = c.value
                                        return p
                                    }, {})),
                                    ResponseBody: i.response.content?.text ?? "",
                                    PreResponseScript: "",
                                }
                            })).then(() => this.onCollectionSelect())
                        })
                    },
                    onServiceDelete(ID) {
                        ElMessageBox.confirm("Service will be deleted permanently. Continue ?", "Warning", {
                            confirmButtonText: "Confirm",
                            type: "warning",
                            beforeClose: async (action, instance, done) => {
                                if (action === "confirm") {
                                    instance.confirmButtonLoading = true
                                    instance.confirmButtonText = "Delete..."
                                    await this.fetch("DELETE", "Service", `ID=${ID}`)
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
                        this.libraries = this.parse(this.collection.records.find(i => i.ID === this.collection.ID)?.Libraries, [])
                        this.service.dialog.visible = true
                    },
                    onServiceDialogSubmit() {
                        return Promise.resolve()
                            .then(() => {
                                if (this.service.dialog.draft.ID) {
                                    return this.fetch("PUT", "Service", `ID=${this.service.dialog.draft.ID}`, this.patch(this.service.dialog.draft, this.service.records.find(i => i.ID === this.service.dialog.draft.ID) ?? {}))
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
                        template: `<div ref="container" style="width: 100%; height: 100%;"><div v-if="loading" style="position:absolute;inset:0;display:flex;align-items:center;justify-content:center;color:var(--el-text-color-secondary)">Loading…</div></div>`,
                        props: {
                            modelValue: { type: String, default: "", },
                            language: { type: String, default: "typescript", },
                            types: { type: String, default: "", },
                        },
                        emits: ["update:modelValue",],
                        data() {
                            return { loading: true, disposed: false, }
                        },
                        watch: {
                            modelValue(newValue, oldValue) {
                                if (this.editor && newValue !== oldValue && newValue !== this.editor.getValue()) {
                                    this.editor.setValue(newValue)
                                }
                            },
                            language: {
                                immediate: true,
                                async handler(language) {
                                    const monaco = await require("monaco")
                                    if (this.editor) monaco.editor.setModelLanguage(this.editor.getModel(), language)
                                    const ts = monaco.languages.typescript
                                    ts.typescriptDefaults.setCompilerOptions({
                                        target: ts.ScriptTarget?.ESNext ?? 99,
                                        module: ts.ModuleKind?.ESNext ?? 99,
                                        moduleResolution: ts.ModuleResolutionKind?.NodeJs ?? 1,
                                        allowNonTsExtensions: true,
                                        noEmit: true,
                                    })
                                    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({ allowComments: true })
                                },
                            },
                        },
                        methods: {
                            setExtraLibs() {
                                window.monaco.languages.typescript.typescriptDefaults?.setExtraLibs([{ content: this.types, filePath: "mockd-types.d.ts" }])
                            },
                        },
                        async created() {
                            const monaco = await require("monaco")
                            if (this.disposed) return // await 期间组件可能已卸载，续跑时不能再创建编辑器
                            // this.editor 保持非响应式：Vue 深度劫持 data() 对象，Monaco 实例庞大含循环引用，getValue 触发劫持 getter 撑爆 CPU
                            // .mts 后缀使脚本作为 ESM 模块编译，多个编辑器 model 的顶层声明互相隔离，避免 redeclare
                            const model = monaco.editor.createModel(this.modelValue, this.language, monaco.Uri.parse("file:///mockd-" + this.$.uid + ".mts"))
                            this.editor = monaco.editor.create(this.$refs.container, { model, automaticLayout: true })
                            this.setExtraLibs()
                            this.editor.onDidFocusEditorText(() => this.setExtraLibs())
                            this.editor.onDidChangeModelContent(() => {
                                this.$emit("update:modelValue", this.editor.getValue())
                            })
                            this.loading = false
                        },
                        beforeUnmount() {
                            this.disposed = true
                            if (this.editor) {
                                this.editor.getModel()?.dispose()
                                this.editor.dispose()
                            }
                        },
                    },
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

