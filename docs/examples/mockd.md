# Mock server

1. Create a controller with url `/service/mockd` and method `Any`.
    ```typescript
    //?name=mockd&type=controller&url=mockd{name}&method=&tag=mock
    import * as JSON5 from "https://unpkg.com/json5@2/dist/index.min.js"

    export default (app => app.run.bind(app))(new class {
        private db = $native("db")

        public run(ctx: ServiceContext) {
            const forms = Object.entries(ctx.getForm()).reduce((p, c) => { p[c[0]] = c[1]?.[0]; return p; }, {}) as { u: string; c: string; b: string; g: string; s: string; },
                name = ctx.getPathVariables().name
            if ("setup" in forms) {
                return this.setup()
            }
            if ("test" in forms || name) {
                if (ctx.getMethod() === "OPTIONS") {
                    return new ServiceResponse(200, { "Access-Control-Allow-Origin": "*", "Access-Control-Allow-Methods": "*", "Access-Control-Allow-Headers": "*" })
                }
                return this.test(name || forms.u, forms.c, forms.b ?? ctx.getBody()?.toString())
            }
            switch (ctx.getMethod()) {
                case "POST":
                    return this.post(ctx.getBody().toJson())
                case "DELETE":
                    return this.delete(forms.g, forms.s)
                case "PUT":
                    return this.put(ctx.getBody().toJson())
                case "GET":
                    return this.get(forms.g)
                default:
                    return new ServiceResponse(405)
            }
        }

        public setup() {
            this.db.exec(`
                DROP TABLE IF EXISTS MockGroup;
                CREATE TABLE IF NOT EXISTS MockGroup (
                    Name VARCHAR(64) PRIMARY KEY NOT NULL,
                    Active BOOLEAN NOT NULL DEFAULT false,
                    Storage TEXT NOT NULL DEFAULT '',
                    PreRequestScript TEXT NOT NULL DEFAULT ''
                );
                DROP TABLE IF EXISTS MockService;
                CREATE TABLE IF NOT EXISTS MockService (
                    GroupId INTEGER NOT NULL,
                    Active BOOLEAN NOT NULL DEFAULT false,
                    URL VARCHAR(255) NOT NULL,
                    StatusCode INTEGER NOT NULL DEFAULT 200,
                    Headers TEXT NOT NULL DEFAULT '',
                    Body TEXT NOT NULL DEFAULT '',
                    PreResponseScript TEXT NOT NULL DEFAULT '',
                    Settings TEXT NOT NULL DEFAULT ''
                );
            `);
        }

        public test(url: string, callback?: string, requestBody?: string) {
            const service = this.db.query(`
                SELECT
                    s.StatusCode status,
                    s.Headers headers,
                    s.Body body,
                    s.PreResponseScript preResponseScript,
                    s.Settings settings,
                    s.GroupId groupId,
                    g.Storage storage,
                    g.PreRequestScript preRequestScript
                FROM
                    MockService s
                    LEFT JOIN MockGroup g ON s.GroupId = g.rowid
                WHERE
                    g.Active = 1
                    AND s.Active = 1
                    AND s.URL like ?
                LIMIT 1
            `, "%" + url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, "") + "%")?.pop()
            if (!service) {
                return callback ? new ServiceResponse(200, undefined, `mockc.callbacks["${callback}"](${JSON.stringify({ status: 404, })})`) : new ServiceResponse(404)
            }
            const settings = service.settings ? JSON.parse(service.settings) : {}
            if (settings.time) {
                setTimeout(() => { }, settings.time)
            }
            const context = {
                request: {
                    body: requestBody && JSON.parse(decodeURIComponent(requestBody)),
                },
                response: {
                    status: service.status || 200,
                    headers: JSON.parse(service.headers || "{}"),
                    body: !!~service.headers.indexOf("json") ? this.json52any(service.body || "{}") : service.body,
                },
                storage: JSON.parse(service.storage || "{}"),
            }
            if (service.preRequestScript) {
                context.response.body = (new Function("$", service.preRequestScript))(context) ?? context.response.body
            }
            if (service.preResponseScript) {
                context.response.body = (new Function("$", service.preResponseScript))(context) ?? context.response.body
            }
            const newStorage = JSON.stringify(context.storage)
            if (newStorage !== service.storage) {
                this.db.exec(`UPDATE MockGroup SET Storage = ? WHERE rowid = ?`, newStorage, service.groupId)
            }
            if (callback) {
                return new ServiceResponse(200, undefined, `mockc.callbacks["${callback}"](${JSON.stringify(context.response)})`)
            }
            return new ServiceResponse(context.response.status, { "Access-Control-Allow-Origin": "*", "Access-Control-Allow-Methods": "*", "Access-Control-Allow-Headers": "*", ...context.response.headers }, JSON.stringify(context.response.body))
        }

        public post(input: Group | Service | Service[]) {
            if (!Array.isArray(input)) {
                if (("group" in input)) {
                    input = [input]
                } else {
                    this.db.transaction(tx => {
                        input = input as Group
                        this.db.exec(`INSERT INTO MockGroup (Name, Active, Storage, PreRequestScript) VALUES (?, ?, ?, ?)`, input.name, input.active, input.storage, input.preRequestScript)
                        if (input.active) {
                            const id = tx.query("SELECT last_insert_rowid() id")[0].id
                            tx.exec(`UPDATE MockGroup SET Active = 0 WHERE rowid <> ?`, id)
                        }
                    })
                    return
                }
            }
            return this.db.exec(`INSERT INTO MockService (GroupId, Active, URL, StatusCode, Headers, Body, PreResponseScript, Settings) VALUES ${input.map(() => "(?, ?, ?, ?, ?, ?, ?, ?)").join(",")}`, ...input.map(s => [
                s.group,
                s.active,
                s.url,
                s.status,
                s.headers,
                s.body,
                s.preResponseScript,
                s.settings,
            ]).flat())
        }

        public delete(group: string, services: string) {
            if (services !== undefined) {
                return this.db.exec(`DELETE FROM MockService WHERE rowid in (${services.split(",").map(() => "?").join(",")})`, ...(services.split(",")))
            }
            if (group !== undefined) {
                let effect = 0
                this.db.transaction(tx => {
                    tx.exec(`DELETE FROM MockService WHERE GroupId = ?`, group)
                    effect = tx.exec(`DELETE FROM MockGroup WHERE rowid = ?`, group)
                })
                return effect
            }
            return 0
        }

        public put(input: Group | Service) {
            if ("group" in input) {
                return this.db.exec("UPDATE MockService SET GroupId = ?, Active = ?, URL = ?, StatusCode = ?, Headers = ?, Body = ?, PreResponseScript = ?, Settings = ? WHERE rowid = ?",
                    input.group,
                    input.active,
                    input.url,
                    input.status,
                    input.headers,
                    input.body,
                    input.preResponseScript,
                    input.settings,
                    input.id,
                )
            }
            return this.db.exec(`UPDATE MockGroup SET Name = ?, Active = ?, Storage = ?, PreRequestScript = ? WHERE rowid = ?`, input.name, input.active, input.storage, input.preRequestScript, input.id)
        }

        public get(group?: string): Group[] | Service[] {
            if (group) {
                return this.db.query(`SELECT rowid, GroupId, Active, URL, StatusCode, Headers, Body, PreResponseScript, Settings FROM MockService WHERE GroupId = ? ORDER BY URL`, group).map(i => {
                    return {
                        id: i.rowid,
                        group: i.GroupId,
                        active: i.Active,
                        url: i.URL,
                        status: i.StatusCode,
                        headers: i.Headers,
                        body: i.Body,
                        preResponseScript: i.PreResponseScript,
                        settings: i.Settings,
                    }
                })
            }
            return this.db.query(`SELECT rowid, Name, Active, Storage, PreRequestScript FROM MockGroup`).map(i => {
                return {
                    id: i.rowid,
                    name: i.Name,
                    active: i.Active,
                    storage: i.Storage,
                    preRequestScript: i.PreRequestScript,
                }
            })
        }

        private json52any(text: string) {
            try {
                return JSON5.parse(text, undefined)
            } catch (e) {
                throw new Error("inavlid json5: " + e.message)
            }
        }
    })

    type Group = {
        id?: number
        name: string
        active: boolean
        storage: string
        preRequestScript: string
    }

    type Service = {
        id?: number
        group: number
        active: boolean
        url: string
        status: number
        headers: string
        body: string
        preResponseScript: string
        settings: string
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
        <link rel="stylesheet" href="/libs/element-plus/2.10.5/index.min.css" />
        <script src="/libs/vue/3.5.18/vue.global.prod.min.js"></script>
        <!-- <script src="https://cdn.bootcdn.net/ajax/libs/vue/3.5.18/vue.global.min.js"></script> -->
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
                        <el-option v-for="group in group.records" :key="group.id" :label="group.name" :value="group.id">
                            <span v-if="group.active" style="font-weight: bolder;">{{ group.name }}</span>
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
                        <el-table-column label="#" width="60">
                            <template #default="scope">
                                <span>{{ scope.$index }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column label="URL" prop="url" :show-overflow-tooltip="true">
                            <template #default="scope">
                                <el-link type="primary" @click="onServiceEdit(scope.row)">
                                    {{ scope.row.url }}
                                </el-link>
                                <el-text v-if="scope.row.settings.name" type="info" style="margin-left: 8px;">
                                    {{ scope.row.settings.name }}
                                </el-text>
                            </template>
                        </el-table-column>
                        <el-table-column label="Status Code" width="120">
                            <template #default="scope">
                                {{ scope.row.status }}
                            </template>
                        </el-table-column>
                        <el-table-column label="Body Size" width="100">
                            <template #default="scope">
                                {{ ((scope.row.body?.length ?? 0) / 1024).toFixed(2) }} KB
                            </template>
                        </el-table-column>
                        <el-table-column label="Operation" width="100">
                            <template #default="scope">
                                <el-switch v-model="scope.row.active" size="small" style="margin-right: 12px;"
                                    @change="onServiceActiveSwitch(scope.row)">
                                </el-switch>
                                <el-button link type="danger" @click="onServiceDelete([scope.row.id])" :icon="Delete">
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
                    <el-input v-model="group.dialog.draft.name" placeholder="Please input group name"></el-input>
                </template>
                <el-form>
                    <el-tabs tab-position="left" style="height: 500px;">
                        <el-tab-pane label="Storage" lazy>
                            <monaco-editor v-model="group.dialog.draft.storage" height="500px"
                                language="json"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Pre-Request Script" lazy>
                            <monaco-editor v-model="group.dialog.draft.preRequestScript" height="500px"
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
                    <el-input v-model="service.dialog.draft.url" placeholder="Please input service url"></el-input>
                </template>
                <el-form>
                    <el-tabs tab-position="left" style="height: 500px;">
                        <el-tab-pane label="Body" lazy>
                            <monaco-editor v-model="service.dialog.draft.body" height="500px"
                                :language="!!~service.dialog.draft.headers.indexOf('json') ? 'json5' : 'html'"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Headers" lazy>
                            <monaco-editor v-model="service.dialog.draft.headers" height="500px"
                                language="json"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Status Code">
                            <el-input v-model.number="service.dialog.draft.status" type="number"
                                autocomplete="off"></el-input>
                        </el-tab-pane>
                        <el-tab-pane label="Pre-Response Script" lazy>
                            <monaco-editor v-model="service.dialog.draft.preResponseScript" height="500px"
                                language="typescript"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Settings" lazy>
                            <el-form label-width="auto" style="max-width: 360px;">
                                <el-form-item label="Name">
                                    <el-input v-model="service.dialog.draft.settings.name"></el-input>
                                </el-form-item>
                                <el-form-item label="Time">
                                    <el-input v-model="service.dialog.draft.settings.time" type="number"></el-input>
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
                            return this.group.record?.id ?? ""
                        },
                        set(value) {
                            this.group.record = this.group.records.find(i => i.id === value)
                            if (value) {
                                this.onServiceFetch(value)
                            }
                            document.title = this.group.record?.name || "Just mock it"
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
                        fetch("/service/mockd").then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            return r.json()
                        }).then(({ data: groups }) => {
                            this.group.records = groups
                            this["proxy.group.id"] = groups.find(i => i.active)?.id
                        }).catch(e => {
                            ElMessage.error(e.message)
                        })
                    },
                    onGroupEdit(record) {
                        this.group.dialog.draft = {
                            name: new Date().toISOString().replace(/[-T:\.Z]/g, ""),
                            storage: "",
                            preRequestScript: "",
                            ...this.group.record,
                            active: true,
                        }
                        ;["storage"].forEach(n => this.group.dialog.draft[n] = JSON.stringify(JSON.parse(this.group.dialog.draft[n] || "{}"), undefined, 2))
                        this.group.dialog.visible = true
                    },
                    onGroupDialogSubmit() {
                        if (!this.group.dialog.draft.name) {
                            ElMessage.warning("Group name is required")
                            return
                        }
                        fetch(`/service/mockd?g=${this.group.dialog.draft.id ?? ""}`, {
                            method: this.group.dialog.draft.id ? "PUT" : "POST",
                            body: JSON.stringify({
                                ...this.group.dialog.draft,
                                storage: JSON.stringify(JSON.parse(this.group.dialog.draft.storage || "{}")),
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
                                    fetch(`/service/mockd?g=${this.group.record.id}`, {
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
                                body: JSON.stringify($group),
                            }).then(r => {
                                if (r.status !== 200) {
                                    throw new Error(r.statusText)
                                }
                                return fetch("/service/mockd", {
                                    method: "POST",
                                    body: JSON.stringify(entries.filter(i => i._resourceType === "xhr").map(i => {
                                        return {
                                            group: $group.id,
                                            active: true,
                                            url: i.request.url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, ""),
                                            status: i.response.status,
                                            headers: JSON.stringify(i.response.headers.filter(i => that.constants.HEADER_WHITELIST.includes(i.name.toUpperCase())).reduce((p, c) => {
                                                p[c.name] = c.value
                                                return p
                                            }, {})),
                                            body: i.response.content?.text,
                                            preResponseScript: i.$preResponseScript,
                                            settings: JSON.stringify(i.$settings),
                                        }
                                    })),
                                }).then(r => {
                                    if (r.status !== 200) {
                                        throw new Error(r.statusText)
                                    }
                                })
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
                                            url: i.url,
                                        },
                                        response: {
                                            status: i.status,
                                            headers: Object.entries(JSON.parse(i.headers)).map(i => { return { name: i[0], value: i[1] } }),
                                            content: {
                                                text: i.body,
                                            },
                                        },
                                        time: i.settings.time ?? 0,
                                        $settings: i.settings,
                                        $preResponseScript: i.preResponseScript,
                                    }
                                })
                            }
                        })], { type: "text/plain" }))
                        a.download = Date.now() + ".har"
                        a.click()
                    },
                    onServiceFetch(group = this.group.record.id) {
                        if (!group) {
                            return
                        }
                        this.service.loading = true
                        fetch(`/service/mockd?g=${group}`).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }
                            return r.json()
                        }).then(({ data: services }) => {
                            this.service.records = services.map(i => {
                                return {
                                    ...i,
                                    settings: JSON.parse(i.settings),
                                }
                            })
                        }).catch(e => {
                            ElMessage.error(e.message)
                        }).finally(() => {
                            this.service.loading = false
                        })
                    },
                    onServiceImport(file) {
                        const that = this,
                            reader = new FileReader()
                        reader.onload = function () {
                            fetch("/service/mockd", {
                                method: "POST",
                                body: JSON.stringify(entries.filter(i => i._resourceType === "xhr").map(i => {
                                    return {
                                        group: this.group.record.id,
                                        active: false,
                                        url: i.request.url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, ""),
                                        status: i.response.status,
                                        headers: JSON.stringify(i.response.headers.filter(i => that.constants.HEADER_WHITELIST.includes(i.name.toUpperCase())).reduce((p, c) => {
                                            p[c.name] = c.value
                                            return p
                                        }, {})),
                                        body: i.response.content?.text,
                                        preResponseScript: i.$preResponseScript,
                                        settings: JSON.stringify(i.$settings),
                                    }
                                })),
                            }).then(r => {
                                if (r.status !== 200) {
                                    throw new Error(r.statusText)
                                }
                                this.onServiceFetch()
                            })
                        }
                        reader.readAsText(file.raw, "utf-8")
                    },
                    onServiceDelete(services) {
                        fetch(`/service/mockd?g=${this.group.record.id}&s=${services.join(",")}`, {
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
                            group: this.group.record.id,
                            active: true,
                            url: "",
                            status: 200,
                            headers: JSON.stringify({
                                "Content-Type":"application/json; charset=utf-8",
                            }, undefined, 2),
                            body: "{}",
                            preResponseScript: "",
                            settings: {
                                name: "",
                                time: 0,
                            },
                            ...record,
                        }
                        ;["headers", "body"].forEach(n => {
                            try {
                                this.service.dialog.draft[n] = JSON.stringify(JSON.parse(this.service.dialog.draft[n] || "{}"), undefined, 2)
                            } catch { }
                        })
                        this.service.dialog.record = record
                        this.service.dialog.visible = true
                    },
                    onServiceActiveSwitch(record) {
                        fetch(`/service/mockd`, {
                            method: "PUT",
                            body: JSON.stringify({
                                ...record,
                                settings: JSON.stringify(record.settings),
                            }),
                        }).then(r => {
                            if (r.status !== 200) {
                                throw new Error(r.statusText)
                            }    
                            if (!record.active) {
                                return
                            }
                            this.service.records.filter(i => i.active && i.url === record.url).forEach(i => {
                                i.active = false
                            })
                            record.active = true
                        }).catch(e => {
                            ElMessage.error(e.message)
                        })
                    },
                    onServiceSelect(record) {
                        this.service.highlights = this.service.records.filter(i => i.url === record.url)
                    },
                    onServiceClass({ row }) {
                        let clz = ""
                        if (!row.active) {
                            clz += " disabled"
                        }
                        if (this.service.highlights.includes(row)) {
                            clz += " current-row"
                        }
                        return clz
                    },
                    onServiceDialogPreview() {
                        if (!this.service.dialog.draft.url) {
                            return
                        }
                        window.open("/service/mockd?test&u=" + encodeURIComponent(this.service.dialog.draft.url))
                    },
                    onServiceDialogSubmit() {
                        if (!this.service.dialog.draft.url) {
                            ElMessage.warning("Service url is required")
                            return
                        }
                        ;["headers"].forEach(n => this.service.dialog.draft[n] = JSON.stringify(JSON.parse(this.service.dialog.draft[n] || "{}")))
                        fetch(`/service/mockd`, {
                            method: this.service.dialog.draft.id ? "PUT" : "POST",
                            body: JSON.stringify({
                                ...this.service.dialog.draft,
                                settings: JSON.stringify(this.service.dialog.draft.settings),
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
                            require("/libs/monaco-editor/0.52.2/min/vs/loader.js")
                                .then(() => {
                                    window.require.config({ paths: { vs: window.location.origin + "/libs/monaco-editor/0.52.2/min/vs" } })
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
                                            monaco.languages.typescript.typescriptDefaults.addExtraLib(`declare var $ = { request?: { body?: any, }, response: { headers: any, body: any, }, storage: any, session?: any }`, "global.ts")
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

3. Setup database tables with visit [`/service/mockd?setup`](/service/mockd?setup)

4. Visit `/resource/mockd` and create a group with uploading a HAR file.

5. Inject mock client.
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

6. You can also reverse the server to android devices like that
    ```bash
    adb reverse tcp:8090 tcp:8090
    ```
