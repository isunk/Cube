# Mock server

1. Create a controller with url `/service/mockd` and method `Any`.
    ```typescript
    //?name=mockd&type=controller&url=mockd{name}&method=&tag=mock
    import * as JSON5 from "https://cdn.bootcdn.net/ajax/libs/json5/2.2.3/index.min.js"
    import { helper, ColumnType } from "./DbHelper"

    interface Group {
        ID?: number
        Name: string
        Active: boolean
        Storage: string
        PreRequestScript: string
    }

    interface Service {
        ID?: number
        GroupId: number
        Active: boolean
        URL: string
        StatusCode: number
        Headers: string
        Body: string
        PreResponseScript: string
        Settings: string
    }

    export default (app => app.run.bind(app))(new class {
        private CORD_HEADERS = {
            "Access-Control-Allow-Origin": "*",
            "Access-Control-Allow-Methods": "*",
            "Access-Control-Allow-Headers": "*",
        }

        public run(ctx: ServiceContext) {
            const params = Object.entries(ctx.getForm()).reduce((p, c) => { p[c[0]] = c[1]?.[0]; return p; }, {}) as { [i in ("u" | "c" | "b") | ("group" | "service")]: string; },
                name = ctx.getPathVariables().name

            if ("setup" in params) {
                return this.setup()
            }

            if ("test" in params || name) {
                if (ctx.getMethod() === "OPTIONS") {
                    return new ServiceResponse(200, this.CORD_HEADERS)
                }
                return this.test(name || params.u, params.c, params.b ?? ctx.getBody()?.toString())
            }

            switch (ctx.getMethod()) {
                case "POST":
                    return this.post(params.group, ctx.getBody().toJson())
                case "DELETE":
                    return this.delete(params.group, params.service)
                case "PUT":
                    return this.put(params.group, params.service, ctx.getBody().toJson())
                case "GET":
                    return this.get(params.group)
                default:
                    return new ServiceResponse(405)
            }
        }

        public setup() {
            helper.dropTable("MockGroup")
            helper.createTable("MockGroup", [
                { name: "Name", type: ColumnType.String, },
                { name: "Active", type: ColumnType.Boolean, },
                { name: "Storage", type: ColumnType.Text, },
                { name: "PreRequestScript", type: ColumnType.Text, },
            ])
            helper.dropTable("MockService")
            helper.createTable("MockService", [
                { name: "GroupId", type: ColumnType.Integer, },
                { name: "Active", type: ColumnType.Boolean, },
                { name: "URL", type: ColumnType.String, },
                { name: "StatusCode", type: ColumnType.Integer, },
                { name: "Headers", type: ColumnType.Text, },
                { name: "Body", type: ColumnType.Text, },
                { name: "PreResponseScript", type: ColumnType.Text, },
                { name: "Settings", type: ColumnType.Text, },
            ]);
        }

        public test(url: string, callback?: string, requestBody?: string) {
            try {
                const response = this.mock(url, requestBody && JSON.parse(decodeURIComponent(requestBody)))
                if (callback) {
                    return new ServiceResponse(200, undefined, `mockc.callbacks["${callback}"](${JSON.stringify(response)})`)
                }
                return new ServiceResponse(response.status, { ...this.CORD_HEADERS, ...response.headers }, JSON.stringify(response.body))
            } catch (err) {
                let status = 500
                if (err.message === "service not found") {
                    status = 404
                }
                return callback ? new ServiceResponse(200, undefined, `mockc.callbacks["${callback}"](${JSON.stringify({ status, error: err.message })})`) : new ServiceResponse(status, undefined, { error: err.message })
            }
        }

        public post(group: string | undefined, input: Group | Service | Service[]) {
            if (!group) {
                input = input as Group
                if (input.Active) {
                    helper.update("MockGroup", undefined, { Active: false })
                }
                return input.ID = helper.insert("MockGroup", input)
            }
            return (Array.isArray(input) ? input as Service[] : [input as Service]).map(i => helper.insert("MockService", {
                ...i,
                GroupId: group,
            }))
        }

        public delete(group: string | undefined, services: string | undefined) {
            if (services) {
                return helper.delete("MockService", {
                    conditions: [{
                        field: "ID",
                        operator: "in",
                        value: services.split(","),
                    }],
                    conjunction: "AND",
                })
            }
            if (group) {
                helper.delete("MockService", {
                    conditions: [{
                        field: "GroupId",
                        operator: "=",
                        value: group,
                    }],
                    conjunction: "AND",
                })
                return helper.delete("MockGroup", {
                    conditions: [{
                        field: "ID",
                        operator: "=",
                        value: group,
                    }],
                    conjunction: "AND",
                })
            }
            return 0
        }

        public put(group: string | undefined, service: string | undefined, input: Group | Service) {
            if (service) {
                input = input as Service
                return helper.update("MockService", {
                    conditions: [{
                        field: "ID",
                        operator: "=",
                        value: service,
                    }],
                    conjunction: "AND",
                }, input)
            }
            if (group) {
                return helper.update("MockGroup", {
                    conditions: [{
                        field: "ID",
                        operator: "=",
                        value: group,
                    }],
                    conjunction: "AND",
                }, input)
            }
            return 0
        }

        public get(group: string | undefined): Group[] | Service[] {
            if (group) {
                return helper.select("MockService", {
                    conditions: [{
                        field: "GroupId",
                        operator: "=",
                        value: group,
                    }],
                    conjunction: "AND",
                }).sort((a, b) => a.URL.localeCompare(b.URL)).map(i => i)
            }
            return helper.select("MockGroup").map(i => i)
        }

        private mock(url: string, requestBody: any): { status: number; headers: any; body: any; } {
            const service = helper.query(`
                SELECT
                    s.StatusCode StatusCode,
                    s.Headers Headers,
                    s.Body Body,
                    s.PreResponseScript PreResponseScript,
                    s.Settings Settings,
                    s.GroupId GroupId,
                    g.Storage Storage,
                    g.PreRequestScript PreRequestScript
                FROM
                    MockService s
                    LEFT JOIN MockGroup g ON s.GroupId = g.ID
                WHERE
                    g.Active = 1
                    AND s.Active = 1
                    AND s.URL like ?
                LIMIT 1
            `, "%" + url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, ""))?.pop()
            if (!service) {
                throw new Error("service not found")
            }
            const settings = service.Settings ? JSON.parse(service.Settings) : {}
            if (settings.time) {
                setTimeout(() => { }, settings.time)
            }
            const context = {
                request: {
                    body: requestBody,
                },
                response: {
                    status: service.StatusCode || 200,
                    headers: JSON.parse(service.Headers || "{}"),
                    body: !!~service.Headers.indexOf("json") ? this.json52any(service.Body || "{}") : service.Body,
                },
                storage: JSON.parse(service.Storage || "{}"),
                mock: (url, requestBody?: string) => this.mock(url, requestBody),
            }
            if (service.PreRequestScript) {
                context.response.body = (new Function("$", service.PreRequestScript))(context) ?? context.response.body
            }
            if (service.PreResponseScript) {
                context.response.body = (new Function("$", service.PreResponseScript))(context) ?? context.response.body
            }
            const storage = JSON.stringify(context.storage)
            if (storage !== service.Storage) {
                helper.update("MockGroup", {
                    conditions: [{
                        field: "ID",
                        operator: "=",
                        value: service.GroupId,
                    }],
                    conjunction: "AND",
                }, { Storage: storage })
            }
            return context.response
        }

        private json52any(text: string) {
            try {
                return JSON5.parse(text, undefined)
            } catch (e) {
                throw new Error("inavlid json5: " + e.message)
            }
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
            .el-table {
                border-top: 1px solid #dcdfe6;
            }
            .el-table .disabled {
                border-color: #e4e7ed;
                color: #c0c4cc;
                cursor: not-allowed;
            }
            .el-table .disabled a {
                color: #c0c4cc;
            }
            .el-dialog__headerbtn {
                height: 32px;
                top: unset;
            }
            .el-pagination {
                margin-top: 13px;
            }
        </style>
    </head>

    <body>
        <div id="app" v-cloak style="padding: 32px; position: relative;">
            <el-card style="position: sticky; top: 0; z-index: 999;">
                <el-row style="padding-bottom: 10px;">
                    <el-select v-model="this['proxy.group.id']" placeholder="Select a group" clearable @change="onServiceFetch" style="flex-grow: 1; width: fit-content;">
                        <el-option v-for="group in group.records" :key="group.ID" :label="group.Name" :value="group.ID">
                            <span v-if="!!group.Active" style="font-weight: bolder;">{{ group.Name }}</span>
                        </el-option>
                    </el-select>
                    <div style="margin-left: auto; display: inline-flex;">
                        <el-button-group style="padding-left: 5px;" v-if="group.record">
                            <el-button :icon="Check" @click="onGroupEdit"></el-button>
                            <el-button :icon="Download" @click="onGroupExport"></el-button>
                            <el-button :icon="Delete" @click="onGroupDelete"></el-button>
                        </el-button-group>
                        <el-button-group style="padding-left: 5px;" v-else>
                            <el-button :icon="Plus" @click="onGroupEdit()"></el-button>
                            <el-upload :auto-upload="false" action="" :on-change="onGroupImport" :show-file-list="false" accept=".har" style="display: none;">
                                <el-button ref="GroupUploadRef"></el-button>
                            </el-upload>
                            <el-button :icon="Upload" @click="() => this.$refs.GroupUploadRef.ref.click()"></el-button>
                        </el-button-group>
                    </div>
                </el-row>
            </el-card>
            <br />
            <el-card v-if="group.record">
                <el-row style="padding-bottom: 10px;">
                    <el-button-group style="padding-left: 5px;">
                        <el-button :icon="Plus" @click="onServiceEdit()"></el-button>
                        <el-upload :auto-upload="false" action="" :on-change="onServiceImport" :show-file-list="false" accept=".har" style="display: none;">
                            <el-button ref="ServiceUploadRef"></el-button>
                        </el-upload>
                        <el-button :icon="Upload" @click="() => this.$refs.ServiceUploadRef.ref.click()"></el-button>
                        <el-button :icon="Delete" @click="onServiceDelete(service.selections.map(i => i.id))" v-if="service.selections.length"></el-button>
                    </el-button-group>
                </el-row>
                <el-row>
                    <el-table v-loading="service.loading" :data="service.records" :row-class-name="onServiceClass" @selection-change="(rows) => this.service.selections = rows" @row-click="onServiceSelect">
                        <el-table-column type="selection" width="40">
                        </el-table-column>
                        <el-table-column label="ID" width="60">
                            <template #default="scope">
                                {{ scope.row.ID }}
                            </template>
                        </el-table-column>
                        <el-table-column label="URL" :show-overflow-tooltip="true">
                            <template #default="scope">
                                <el-link type="primary" @click="onServiceEdit(scope.row)">
                                    {{ scope.row.URL }}
                                </el-link>
                                <el-text v-if="scope.row.Settings.name" type="info" style="margin-left: 8px;">
                                    {{ scope.row.Settings.name }}
                                </el-text>
                            </template>
                        </el-table-column>
                        <el-table-column label="Status Code" width="120">
                            <template #default="scope">
                                {{ scope.row.StatusCode }}
                            </template>
                        </el-table-column>
                        <el-table-column label="Body Size" width="100">
                            <template #default="scope">
                                {{ ((scope.row.Body?.length ?? 0) / 1024).toFixed(2) }} KB
                            </template>
                        </el-table-column>
                        <el-table-column label="Operation" width="100">
                            <template #default="scope">
                                <el-switch v-model="scope.row.Active" size="small" style="margin-right: 12px;"
                                    @change="onServiceActiveSwitch(scope.row)">
                                </el-switch>
                                <el-button link type="danger" @click="onServiceDelete([scope.row.ID])" :icon="Delete">
                                </el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-pagination layout="total" :total="service.records.length">
                    </el-pagination>
                </el-row>
            </el-card>
            <el-dialog v-model="group.dialog.visible">
                <template #header>
                    <el-input v-model="group.dialog.draft.Name" placeholder="Please input group name"></el-input>
                </template>
                <el-form>
                    <el-tabs tab-position="left" style="height: 500px;">
                        <el-tab-pane label="Storage" lazy>
                            <monaco-editor v-model="group.dialog.draft.Storage" height="500px"
                                language="json"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Pre-Request Script" lazy>
                            <monaco-editor v-model="group.dialog.draft.PreRequestScript" height="500px"
                                language="typescript"></monaco-editor>
                        </el-tab-pane>
                    </el-tabs>
                    <el-form-item style="margin: 12px 0 0 0;">
                        <div style="width: 100%; display: flex; justify-content: flex-end;">
                            <el-button @click="onGroupDialogSubmit" type="primary">Submit</el-button>
                            <el-button @click="group.dialog.visible = !group.dialog.visible">Cancel</el-button>
                        </div>
                    </el-form-item>
                </el-form>
            </el-dialog>
            <el-dialog v-model="service.dialog.visible" title="Service">
                <template #header>
                    <el-input v-model="service.dialog.draft.URL" placeholder="Please input service url"></el-input>
                </template>
                <el-form>
                    <el-tabs tab-position="left" style="height: 500px;">
                        <el-tab-pane label="Body" lazy>
                            <monaco-editor v-model="service.dialog.draft.Body" height="500px"
                                :language="!!~service.dialog.draft.Headers.indexOf('json') ? 'json5' : 'html'"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Headers" lazy>
                            <monaco-editor v-model="service.dialog.draft.Headers" height="500px"
                                language="json"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Status Code">
                            <el-input v-model.number="service.dialog.draft.StatusCode" type="number"
                                autocomplete="off"></el-input>
                        </el-tab-pane>
                        <el-tab-pane label="Pre-Response Script" lazy>
                            <monaco-editor v-model="service.dialog.draft.PreResponseScript" height="500px"
                                language="typescript"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Settings" lazy>
                            <el-form label-width="auto" style="max-width: 360px;">
                                <el-form-item label="Name">
                                    <el-input v-model="service.dialog.draft.Settings.name"></el-input>
                                </el-form-item>
                                <el-form-item label="Time">
                                    <el-input v-model="service.dialog.draft.Settings.time" type="number"></el-input>
                                </el-form-item>
                            </el-form>
                        </el-tab-pane>
                    </el-tabs>
                    <el-form-item style="margin: 12px 0 0 0;">
                        <div style="width: 100%; display: flex; justify-content: flex-end;">
                            <el-button @click="onServiceDialogPreview">Preview</el-button>
                            <el-button type="primary" @click="onServiceDialogSubmit">Submit</el-button>
                            <el-button @click="service.dialog.visible = !service.dialog.visible">Cancel</el-button>
                        </div>
                    </el-form-item>
                </el-form>
            </el-dialog>
        </div>
        <script>
            const { ElMessage, ElMessageBox, } = ElementPlus
            Vue.createApp({
                setup() {
                    const { ref } = Vue
                    const { Delete, Download, Plus, Upload, Check, } = ElementPlusIconsVue
                    return {
                        Delete, Download, Plus, Upload, Check,
                        GroupUploadRef: ref(), ServiceUploadRef: ref(),
                    }
                },
                computed: {
                    "proxy.group.id": {
                        get() {
                            return this.group.record?.ID ?? ""
                        },
                        set(value) {
                            this.group.record = this.group.records.find(i => i.ID === value)
                            document.title = this.group.record?.Name || "Just mock it"
                        },
                    },
                },
                data() {
                    return {
                        group: {
                            record: undefined,
                            records: [],
                            dialog: {
                                draft: {},
                                visiable: false,
                            },
                        },
                        service: {
                            loading: false,
                            records: [],
                            selections: [],
                            highlights: [],
                            dialog: {
                                draft: {},
                                record: {},
                                visiable: false,
                            },
                        },
                        constants: {
                            HEADER_WHITELIST: ["Content-Type"].map(i => i.toUpperCase()),
                        },
                    }
                },
                methods: {
                    onGroupFetch() {
                        return fetch("/service/mockd").then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            return r.json()
                        }).then(({ data: groups }) => {
                            this.group.records = groups
                            this["proxy.group.id"] = groups.find(i => i.Active)?.ID
                            this.onServiceFetch()
                        }).catch(e => {
                            ElMessage.error(e.message)
                        })
                    },
                    onGroupEdit(record) {
                        this.group.dialog.draft = {
                            Name: new Date().toISOString().replace(/[-T:\.Z]/g, ""),
                            Storage: "",
                            PreRequestScript: "",
                            ...this.group.record,
                            Active: true,
                        }
                        ;["Storage"].forEach(n => this.group.dialog.draft[n] = JSON.stringify(JSON.parse(this.group.dialog.draft[n] || "{}"), undefined, 2))
                        this.group.dialog.visible = true
                    },
                    onGroupDialogSubmit() {
                        if (!this.group.dialog.draft.Name) {
                            ElMessage.warning("Group name is required")
                            return
                        }
                        fetch(`/service/mockd?group=${this.group.dialog.draft.ID ?? ""}`, {
                            method: this.group.dialog.draft.ID ? "PUT" : "POST",
                            body: JSON.stringify({
                                ...this.group.dialog.draft,
                                storage: JSON.stringify(JSON.parse(this.group.dialog.draft.Storage || "{}")),
                            })
                        }).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            this.group.dialog.visible = false
                            ElMessage.success("Save succeeded")
                            this.onGroupFetch()
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
                                    fetch(`/service/mockd?group=${this["proxy.group.id"]}`, {
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
                    onGroupImport(file) {
                        const that = this,
                            reader = new FileReader()
                        reader.onload = function () {
                            const { $group, entries } = JSON.parse(this.result).log
                            fetch("/service/mockd", {
                                method: "POST",
                                body: JSON.stringify($group ?? {
                                    Name: Date.now() + "",
                                    Active: true,
                                    Storage: "",
                                    PreRequestScript: "",
                                }),
                            }).then(r => {
                                if (r.status !== 200) {
                                    throw new Error(r.statusText)
                                }
                                return r.json()
                            }).then(r => {
                                return that.onServiceImport(undefined, undefined, entries, r.data)
                            }).then(() => {
                                that.onGroupFetch()
                            }).catch(e => {
                                ElMessage.error(e.message)
                            })
                        }
                        reader.readAsText(file.raw, "utf-8")
                    },
                    onGroupExport() {
                        if (!this.group.record) {
                            return
                        }
                        const a = document.createElement("a")
                        a.href = URL.createObjectURL(new Blob([JSON.stringify({
                            log: {
                                creator: { name: "mockd", version: "0.1" },
                                $group: this.group.record,
                                entries: this.service.records.map(i => {
                                    return {
                                        _resourceType: "xhr",
                                        request: {
                                            url: i.URL,
                                        },
                                        response: {
                                            status: i.StatusCode,
                                            headers: Object.entries(JSON.parse(i.Headers)).map(i => { return { name: i[0], value: i[1] } }),
                                            content: {
                                                text: i.Body,
                                            },
                                        },
                                        time: i.Settings.time ?? 0,
                                        $settings: i.Settings,
                                        $preResponseScript: i.PreResponseScript,
                                    }
                                })
                            }
                        })], { type: "text/plain" }))
                        a.download = Date.now() + ".har"
                        a.click()
                    },
                    onServiceFetch(group = this["proxy.group.id"]) {
                        if (!group) {
                            return
                        }
                        this.service.loading = true
                        fetch(`/service/mockd?group=${group}`).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            return r.json()
                        }).then(({ data: services }) => {
                            this.service.records = services.map(i => {
                                return {
                                    ...i,
                                    Active: !!i.Active,
                                    Settings: JSON.parse(i.Settings),
                                }
                            })
                        }).catch(e => {
                            ElMessage.error(e.message)
                        }).finally(() => {
                            this.service.loading = false
                        })
                    },
                    onServiceImport(file, _, entries, group) {
                        if (!file && !_ && entries && group) {
                            const cache = this.service.records.filter(i => i.active).reduce((p, c) => { p[c.url] = false; return p; }, {})
                            return fetch("/service/mockd?group=" + group, {
                                method: "POST",
                                body: JSON.stringify(entries.filter(i => i._resourceType === "xhr").map(i => {
                                    const URL = i.request.url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, "")
                                    return {
                                        Active: cache[URL] ?? !(cache[URL] = false),
                                        URL,
                                        StatusCode: i.response.status,
                                        Headers: JSON.stringify(i.response.headers.filter(i => this.constants.HEADER_WHITELIST.includes(i.name.toUpperCase())).reduce((p, c) => {
                                            p[c.name] = c.value
                                            return p
                                        }, {})),
                                        Body: i.response.content?.text ?? "",
                                        PreResponseScript: i.$preResponseScript ?? "",
                                        Settings: JSON.stringify(i.$settings ?? "{}"),
                                    }
                                })),
                            }).then(r => {
                                if (r.status !== 200) {
                                    throw new Error(r.statusText)
                                }
                            })
                        }
                        const that = this,
                            reader = new FileReader()
                        reader.onload = function () {
                            that.onServiceImport(undefined, undefined, JSON.parse(this.result).log.entries, that.group.record.id)
                                .then(r => that.onServiceFetch())
                        }
                        reader.readAsText(file.raw, "utf-8")
                    },
                    onServiceDelete(services) {
                        fetch(`/service/mockd?group=${this.group.record.id}&service=${services.join(",")}`, {
                            method: "DELETE",
                        }).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            this.onServiceFetch()
                        }).catch(e => {
                            ElMessage.error(e.message)
                        })
                    },
                    onServiceEdit(record) {
                        this.service.dialog.draft = {
                            GroupId: this.group.record.ID,
                            Active: true,
                            URL: "",
                            StatusCode: 200,
                            Headers: JSON.stringify({
                                "Content-Type":"application/json; charset=utf-8",
                            }, undefined, 2),
                            Body: "{}",
                            PreResponseScript: "",
                            Settings: {
                                name: "",
                                time: 0,
                            },
                            ...record,
                        }
                        ;["Headers", "Body"].forEach(n => {
                            try {
                                this.service.dialog.draft[n] = JSON.stringify(JSON.parse(this.service.dialog.draft[n] || "{}"), undefined, 2)
                            } catch { }
                        })
                        this.service.dialog.record = record
                        this.service.dialog.visible = true
                    },
                    onServiceActiveSwitch(record) {
                        fetch(`/service/mockd?group=${this["proxy.group.id"]}&service=${record.ID ?? ""}`, {
                            method: "PUT",
                            body: JSON.stringify({
                                ...record,
                                Settings: JSON.stringify(record.Settings),
                            }),
                        }).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }    
                            if (!record.active) {
                                return
                            }
                            this.service.records.filter(i => i.Active && i.URL === record.URL).forEach(i => {
                                i.Active = false
                            })
                            record.Active = true
                        }).catch(e => {
                            ElMessage.error(e.message)
                        })
                    },
                    onServiceSelect(record) {
                        this.service.highlights = this.service.records.filter(i => i.URL === record.URL)
                    },
                    onServiceClass({ row }) {
                        let clz = ""
                        if (!row.Active) {
                            clz += " disabled"
                        }
                        if (this.service.highlights.includes(row)) {
                            clz += " current-row"
                        }
                        return clz
                    },
                    onServiceDialogPreview() {
                        if (!this.service.dialog.draft.URL) {
                            return
                        }
                        window.open("/service/mockd?test&u=" + encodeURIComponent(this.service.dialog.draft.URL))
                    },
                    onServiceDialogSubmit() {
                        if (!this.service.dialog.draft.URL) {
                            ElMessage.warning("Service url is required")
                            return
                        }
                        ;["Headers"].forEach(n => this.service.dialog.draft[n] = JSON.stringify(JSON.parse(this.service.dialog.draft[n] || "{}")))
                        fetch(`/service/mockd?group=${this["proxy.group.id"]}&service=${this.service.dialog.draft.ID ?? ""}`, {
                            method: this.service.dialog.draft.ID ? "PUT" : "POST",
                            body: JSON.stringify({
                                ...this.service.dialog.draft,
                                Settings: JSON.stringify(this.service.dialog.draft.Settings),
                            }),
                        }).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            this.service.dialog.visible = false
                            this.onServiceFetch()
                        }).catch(e => {
                            ElMessage.error(e.message)
                        })
                    },
                },
                mounted() {
                    this.onGroupFetch()
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
                                            monaco.languages.typescript.typescriptDefaults.addExtraLib(`declare type MockResponse = { status: number; headers: any; body: any; }; declare const $: { request?: { body?: any; }; response: MockResponse; storage: any; mock: (url: string, requestBody: any) => MockResponse; };`, "global.ts")
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
