# Mock server

1. Create a controller with url `/service/mockd` and method `Any`.
    ```typescript
    //?name=mockd&type=controller&url=mockd&method=&tag=mock
    export default (app => app.run.bind(app))(new class {
        private db = $native("db")
    
        public run(ctx: ServiceContext) {
            const forms = ctx.getForm()
            switch (ctx.getMethod()) {
                case "GET":
                    if ("setup" in forms) {
                        return this.setup()
                    }
                    if ("test" in forms) {
                        return this.test(forms.u?.[0], forms.c?.[0], forms.b?.[0])
                    }
                    return this.get(forms.g?.[0])
                case "POST":
                    return this.post(forms.g?.[0], ctx.getBody().toJson())
                case "DELETE":
                    return this.delete(forms.g?.[0], forms.s?.[0])
                default:
                    return new ServiceResponse(405)
            }
        }
    
        public setup() {
            this.db.exec(`
                DROP TABLE IF EXISTS MockGroup;
                CREATE TABLE IF NOT EXISTS MockGroup (
                    Name VARCHAR(64) PRIMARY KEY NOT NULL,
                    Active BOOLEAN NOT NULL DEFAULT false
                );
    
                DROP TABLE IF EXISTS MockService;
                CREATE TABLE IF NOT EXISTS MockService (
                    GroupId INTEGER NOT NULL,
                    Url VARCHAR(255) NOT NULL,
                    Output TEXT NOT NULL DEFAULT '',
                    Script TEXT NOT NULL DEFAULT '',
                    Setting TEXT NOT NULL DEFAULT '',
                    Active BOOLEAN NOT NULL DEFAULT false
                );
            `);
        }
    
        public test(url: string, callback: string, inputBody?: string) {
            const record = this.db.query(`
                SELECT
                    s.Output output,
                    s.Script script,
                    s.Setting setting
                FROM
                    MockService s
                    LEFT JOIN MockGroup g ON s.GroupId = g.rowid
                WHERE
                    g.Active = 1
                    AND s.Active = 1
                    AND s.Url like ?
                LIMIT 1
            `, "%" + url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, "") + "%")?.pop()
            if (!record) {
                return new ServiceResponse(200, undefined, `mockc.callbacks["${callback}"](${JSON.stringify({ status: 404, })})`)
            }
            const setting = record.setting ? JSON.parse(record.setting) : {}
            if (setting.time) {
                setTimeout(() => {}, setting.time)
            }
            const input = inputBody && JSON.parse(decodeURIComponent(inputBody)),
                output = JSON.parse(record.output),
                body = record.script && (new Function("input", "output", record.script))(input, output) || output
            return new ServiceResponse(200, undefined, `mockc.callbacks["${callback}"](${JSON.stringify({ status: setting.status ?? 200, body, })})`)
        }
    
        public get(groupId: string) {
            let wheres = "WHERE GroupId in (SELECT rowid FROM MockGroup WHERE Active = 1)",
                params = []
            if (groupId) {
                wheres = "WHERE GroupId = ?"
                params.push(groupId)
            }
            return {
                services: (this.db.query(`SELECT rowid, GroupId, Url, Output, Script, Setting, Active FROM MockService ${wheres}`, ...params) ?? []).map(i => {
                    return {
                        id: i.rowid,
                        group: i.GroupId,
                        url: i.Url,
                        output: i.Output,
                        script: i.Script,
                        setting: i.Setting,
                        active: i.Active,
                    }
                }),
                groups: (this.db.query(`SELECT rowid, Name, Active FROM MockGroup`) ?? []).map(i => {
                    return {
                        id: i.rowid,
                        name: i.Name,
                        active: i.Active,
                    }
                }),
            }
        }
    
        public post(groupId: string, jsonBody: any) {
            const { name, services } = jsonBody
            if (!services.length) {
                return 0
            }
            this.db.transaction(tx => {
                const group = groupId && tx.query(`SELECT rowid, Name FROM MockGroup where rowid = ?`, groupId)?.pop()
                if (group) {
                    tx.exec(`DELETE FROM MockService WHERE GroupId = ?`, groupId)
                    tx.exec(`UPDATE MockGroup SET Name = ?, Active = 1 WHERE rowid = ?`, name, groupId)
                } else {
                    tx.exec(`INSERT INTO MockGroup (Name, Active) VALUES (?, 1)`, name)
                    groupId = tx.query("SELECT last_insert_rowid() id")[0].id
                }
                tx.exec(`INSERT INTO MockService (GroupId, Url, Output, Script, Setting, Active) VALUES ${services.map(() => "(?, ?, ?, ?, ?, ?)").join(",")}`, ...services.map(s => [groupId, s.url, s.output, s.script, s.setting, s.active]).flat())
                tx.exec(`UPDATE MockGroup SET Active = 0 WHERE rowid <> ?`, groupId)
            })
            return groupId
        }
    
        public delete(groupId: string, serviceId: string) {
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
        <!-- <script src="https://cdn.bootcdn.net/ajax/libs/vue/3.5.13/vue.global.min.js"></script> -->
        <script src="/libs/element-plus/2.9.1/index.full.min.js"></script>
        <script src="/libs/element-plus-icons-vue/2.3.1/index.iife.min.js"></script>
        <title>mockd</title>
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
                            <el-button :icon="Delete" @click="onGroupDelete" v-if="group.record"></el-button>
                        </el-button-group>
                    </div>
                </el-row>
            </el-card>
            <br />
            <el-card>
                <el-row style="padding-bottom: 10px;">
                    <el-button-group style="padding-left: 5px;">
                        <el-button :icon="Plus" @click="onServiceAdd"></el-button>
                        <el-upload :auto-upload="false" action="" :on-change="onServiceImport" :show-file-list="false" accept=".har" style="display: none;">
                            <el-button ref="UploadRef"></el-button>
                        </el-upload>
                        <el-button :icon="Upload" @click="() => this.$refs.UploadRef.ref.click()"></el-button>
                        <el-button :icon="Download" @click="onServiceExport" v-if="service.records.length"></el-button>
                        <el-button :icon="Delete" @click="onServiceDelete" v-if="service.selections.length"></el-button>
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
                            </template>
                        </el-table-column>
                        <el-table-column label="Size" width="100">
                            <template #default="scope">
                                {{ ((scope.row.output?.length ?? 0) / 1024).toFixed(2) }} KB
                            </template>
                        </el-table-column>
                        <el-table-column label="Operation" width="100">
                            <template #default="scope">
                                <el-switch v-model="scope.row.active" size="small" style="margin-right: 12px;" @change="onServiceActiveSwitch(scope.row)">
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
            <el-dialog v-model="group.dialog.visible">
                <template #header>
                    <el-input v-model="group.dialog.record.name" placeholder="Please input group name"></el-input>
                </template>
                <template #footer>
                    <el-button @click="onGroupSave" type="primary">Confirm</el-button>
                </template>
            </el-dialog>
            <el-dialog v-model="service.dialog.visible" title="Service">
                <template #header>
                    <el-input v-model="service.dialog.record.url" placeholder="Please input service url"></el-input>
                </template>
                <el-form>
                    <el-tabs tab-position="left">
                        <el-tab-pane label="Output">
                            <monaco-editor v-model="this['proxy.service.dialog.record.output']" height="512px" language="json"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Script" lazy>
                            <monaco-editor v-model="this['proxy.service.dialog.record.script']" height="512px" language="javascript"></monaco-editor>
                        </el-tab-pane>
                        <el-tab-pane label="Setting" lazy>
                            <monaco-editor v-model="this['proxy.service.dialog.record.setting']" height="512px" language="json"></monaco-editor>
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
                    const { Delete, Download, Plus, Upload, Check, } = ElementPlusIconsVue
                    return {
                        Delete, Download, Plus, Upload, Check,
                        UploadRef: ref(),
                    }
                },
                computed: {
                    "proxy.service.dialog.record.output": {
                        get() {
                            return this.service.dialog.record.output && JSON.stringify(JSON.parse(this.service.dialog.record.output), undefined, 2)
                        },
                        set(value) {
                            this.service.dialog.record.output = value ? JSON.stringify(JSON.parse(value)) : value
                        },
                    },
                    "proxy.service.dialog.record.script": {
                        get() {
                            return this.service.dialog.record.script ?? "// input, output"
                        },
                        set(value) {
                            this.service.dialog.record.script = value ?? ""
                        },
                    },
                    "proxy.service.dialog.record.setting": {
                        get() {
                            return this.service.dialog.record.setting && JSON.stringify(JSON.parse(this.service.dialog.record.setting), undefined, 2)
                        },
                        set(value) {
                            this.service.dialog.record.setting = value ?? ""
                        },
                    },
                    "proxy.group.id": {
                        get() {
                            return this.group.record?.id
                        },
                        set(value) {
                            this.group.record = this.group.records.find(i => i.id === value) ?? {}
                        },
                    },
                },
                data() {
                    return {
                        group: {
                            record: {},
                            records: [],
                            dialog: {
                                record: {},
                                visiable: false,
                            },
                        },
                        service: {
                            loading: false,
                            records: [],
                            selections: [],
                            highlights: [],
                            dialog: {
                                record: {},
                                visiable: false,
                            },
                        },
                    }
                },
                methods: {
                    onBeforeGroupSave() {
                        this.group.dialog.record = {
                            name: new Date().toISOString().replace(/[-T:\.Z]/g, ""),
                            ...this.group.record,
                        }
                        this.group.dialog.visible = true
                    },
                    onGroupSave() {
                        if (!this.group.dialog.record.name) {
                            ElMessage.warning("Group name is required")
                            return
                        }
                        fetch(`/service/mockd?g=${this["proxy.group.id"] ?? ""}`, {
                            method: "POST",
                            body: JSON.stringify({
                                name: this.group.dialog.record.name,
                                services: this.service.records.map(i => {
                                    return {
                                        url: i.url,
                                        output: i.output ?? "",
                                        script: i.script ?? "",
                                        setting: i.setting ?? "",
                                        active: i.active,
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
                            const records = JSON.parse(this.result).log.entries.filter(i => i._resourceType === "xhr").map(i => {
                                return {
                                    url: i.request.url.replace(/^https?:\/\/[^\/]+/, "").replace(/\?.*$/, ""),
                                    output: i.response.content?.text,
                                    script: "",
                                    setting: JSON.stringify({
                                        time: i.time,
                                        status: i.response.status,
                                    }),
                                    active: false,
                                    input: i.request.postData?.text,
                                }
                            })
                            for (const record of records) {
                                if (!that.service.records.some(i => i.active && i.url === record.url)) {
                                    record.active = true
                                }
                                that.service.records.push(record)
                            }
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
                    onServiceAdd() {
                        this.service.dialog.record = {}
                        this.service.records.push(this.service.dialog.record)
                        this.service.dialog.visible = true
                    },
                    onServiceEdit(record, evt) {
                        this.service.dialog.record = record
                        this.service.dialog.visible = true
                    },
                    onServiceActiveSwitch(record) {
                        if (!record.active) {
                            return
                        }
                        this.service.records.filter(i => i.active && i.url === record.url).forEach(i => {
                            i.active = false
                        })
                        record.active = true
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

5. Inject mock client using JSONP with src `/service/mockd?test&u=...&c=...&b=...`.
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
    You can also reverse the server to android devices like that
    ```bash
    adb reverse tcp:8090 tcp:8090
    ```
