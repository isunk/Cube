# GraphQL

A GraphQL-like query and mutation engine with schema-based data access. Supports nested queries with relationship resolution, conditional filtering with multiple operators, pagination with optional unlimited depth, and automatic foreign key handling. Mutations run in a transactional scope with commit and rollback hooks, and include DSL validation plus chunked IN clauses for large datasets. Provides abstract base class for custom database adapters.

```typescript
//?name=node_modules/graphql&type=module
//#region record

export type Record = {
    [name: string]: any;
}

//#endregion

//#region schema

export type Schema = {
    [name: string]: TableSchema;
}

type SubSchema = {
    [name: string]: SubTableSchema | ColumnSchema;
}

type TableSchema = {
    table: string;
    properties: SubSchema;
}

type SubTableSchema = TableSchema & {
    joined: {
        column: string;
        parent: string;
        isCollection?: false;
    };
}

type ColumnSchema = {
    column: string;
    isPrimaryKey?: true;
}

//#endregion

//#region query

export type QueryDSL = {
    [name: string]: TableQueryDSL | Record | null;
}

type TableQueryDSL = {
    properties: QueryDSL | null;
    conditions?: ConditionQueryDSL[];
    options?: {
        skip?: number;
        limit?: number | null;
    };
}

type ConditionQueryDSL = {
    field: string;
    operator: "eq" | "gt" | "ge" | "lt" | "le" | "ne";
    value: object;
} | {
    field: string;
    operator: "in";
    value: object[];
}

type ExtendedQueryDSL = {
    [name: string]: {
        foreignKeys?: string[];
        conditions?: {
            field: string;
            operator: "in";
            value: object[];
        }[];
        options?: {
            skip?: number;
            limit?: number | null;
        };
    }
}

type SelectClauses = {
    selects: { [name: string]: { alias?: string; display?: boolean; isForeignKey?: true; } };
    subrequests: QueryDSL;
}

//#endregion

//#region mutation

export type MutationDSL = {
    [name: string]: Record | MutationDSL;
}

type MutationRequestData = {
    /**
     * 请求对象
     */
    record: Record;
    /**
     * 数据库对象
     */
    dataset: Record;
}

type MutationRequests = {
    [name: string]: {
        /**
         * 对象集合
         */
        data: MutationRequestData[];
        /**
         * 前置子请求
         */
        subrequests0: MutationRequests;
        /**
         * 后置子请求
         */
        subrequests1: MutationRequests;
    }
};

//#endregion

export abstract class GraphQL {
    private schema: Schema;

    constructor(schema: Schema) {
        this.schema = schema;
    }

    //#region query

    public query(dsl: QueryDSL) {
        if (dsl == null) {
            throw new Error(`dsl can not be null`);
        }
        if (typeof dsl !== "object" || dsl instanceof Array) {
            throw new Error(`dsl must be a object`);
        }
        return this.doQuery(dsl, <SubSchema>this.schema);
    }

    private doQuery(dsl: QueryDSL, schema: SubSchema, exdsl?: ExtendedQueryDSL) {
        return Object.keys(dsl).reduce((result: Record, name: string) => {
            // 查询表模型
            const model = schema[name];
            if (!model || !this.ownProperty(schema, name)) {
                throw new Error(`can not find ${name} in schema`);
            }
            if (!("table" in model)) {
                throw new Error(`${name} is not a table schema`);
            }

            // 初始化请求
            const request = this.toQueryRequest(dsl, name, model, exdsl);

            // 构建查询列集合、子请求
            const { selects, subrequests } = this.toSqlSelectClauses(request.properties, request.foreignKeys, model);

            // 构建查询条件集合
            const { wheres, params } = this.toSqlWhereClauses(<ConditionQueryDSL[]>request.conditions);

            // limit 为 null 表示不限制条数，此时 skip 一并忽略
            const datasets = this.onSelect(`select ${Object.keys(selects).join(", ")} from ${model.table} where ${["1 = 1"].concat(wheres).join(" and ")}${request.options.limit == null ? "" : ` limit ${request.options.skip},${request.options.limit}`}`, { params }),
                records = this.toQueryRecords(datasets, model, { selects, subrequests });

            result[name] = schema === this.schema
                ? records.map(r => r.data) // 隐藏根元素的 keys 记录
                : records;
            return result;
        }, {});
    }

    private toQueryRequest(dsl: QueryDSL, name: string, model: SubTableSchema, exdsl?: ExtendedQueryDSL) {
        const tdsl = this.toTableQueryDSL(dsl, name, model);

        if (tdsl.properties && !Object.keys(tdsl.properties).length) {
            throw new Error(`properties of ${name} can not be empty`);
        }

        if (tdsl.conditions && !Array.isArray(tdsl.conditions)) {
            throw new Error(`conditions of ${name} must be an array`);
        }

        return {
            properties: tdsl.properties || model.properties,
            options: this.toQueryOptions(name, tdsl.options, exdsl?.[name]?.options),
            conditions: (tdsl.conditions || [])
                .map(condition => {
                    if (condition == null) {
                        throw new Error(`condition of ${name} can not be null`);
                    }
                    const { field, operator, value } = condition;
                    // 校验 field
                    if (!field) {
                        throw new Error(`field of condition can not be null or empty`);
                    }
                    const column = model.properties[field];
                    if (!column || !this.ownProperty(model.properties, field)) {
                        throw new Error(`field ${field} does not exist`);
                    }
                    if (!("column" in column)) {
                        throw new Error(`field ${field} is not a column`);
                    }
                    return {
                        field: column.column,
                        operator,
                        value,
                    };
                })
                .concat(exdsl?.[name]?.conditions || []),
            foreignKeys: exdsl?.[name]?.foreignKeys || [],
        };
    }

    private toQueryOptions(
        name: string,
        options: TableQueryDSL["options"],
        extendedOptions: ExtendedQueryDSL[string]["options"],
    ) {
        const merged = {
            skip: 0,
            limit: 5000 as number | null,
            ...extendedOptions,
            ...options,
        };

        if (!Number.isInteger(merged.skip) || merged.skip < 0) {
            throw new Error(`skip of ${name} must be a non-negative integer`);
        }
        if (merged.limit != null && (!Number.isInteger(merged.limit) || merged.limit <= 0)) {
            throw new Error(`limit of ${name} must be a positive integer`);
        }

        return {
            skip: merged.skip,
            limit: merged.limit,
        };
    }

    private toTableQueryDSL(dsl: QueryDSL, name: string, model: SubTableSchema): TableQueryDSL {
        const v = dsl[name];

        // 仅 null 与 undefined 表示默认查询所有字段，0、""、false 等假值属于形状错误
        if (v == null) {
            return {
                properties: model.properties,
            };
        }

        // 查询侧不支持集合与原始类型，否则后续会报出误导性的 property 0 does not exist
        if (typeof v !== "object" || v instanceof Array) {
            throw new Error(`request ${name} must be a object`);
        }

        // 引用 schema 子模型对象表示查询全部字段，默认全量投影路径也会传入
        if (this.ownProperty(v, "table") && this.ownProperty(v, "properties")) {
            return {
                properties: (<SubTableSchema>v).properties,
            };
        }

        // 若 dsl 没有 properties 属性，则将 dsl 作为 record 查询
        if (!("properties" in v)) {
            return {
                properties: v,
                conditions: Object.keys(v)
                    .reduce((p, c) => {
                        const value = (v as Record)[c];
                        // null 与 undefined 均表示不处理该属性，与 mutate 语义一致
                        if (value === null || value === undefined) {
                            return p;
                        }
                        // 数组条件原本会被静默丢弃并退化为无过滤的全表查询
                        if (value instanceof Array) {
                            throw new Error(`condition of ${c} can not be an array`);
                        }
                        // 原始类型作为相等条件，对象值留给子模型处理
                        if (typeof value !== "object") {
                            p.push({ field: c, operator: "eq", value: <object>value });
                        }
                        return p;
                    }, [] as ConditionQueryDSL[]),
            };
        }

        // 表单形式只接受 properties、conditions、options，多余键静默忽略导致全表扫描
        for (const key of Object.keys(v)) {
            if (key !== "properties" && key !== "conditions" && key !== "options") {
                throw new Error(`unknown key ${key} of ${name}, expected properties, conditions or options`);
            }
        }
        if (v.properties != null && (typeof v.properties !== "object" || v.properties instanceof Array)) {
            throw new Error(`properties of ${name} must be a object`);
        }
        if (v.options != null && (typeof v.options !== "object" || v.options instanceof Array)) {
            throw new Error(`options of ${name} must be a object`);
        }

        return <TableQueryDSL>v;
    }

    private toSqlSelectClauses(properties: QueryDSL, foreignKeys: string[], model: SubTableSchema): SelectClauses {
        // 初始化子请求
        const subrequests = <QueryDSL>{};

        const selects = {} as SelectClauses["selects"];

        // 处理外键所在的列
        for (const n of foreignKeys) {
            selects[n] = { display: false, isForeignKey: true };
        }

        // 处理属性所在的列
        const columnOwner: { [column: string]: string } = {};
        for (const n in properties) {
            const property = model.properties[n];

            if (!property || !this.ownProperty(model.properties, n)) {
                throw new Error(`property ${n} does not exist`);
            }
            if ("table" in property) { // 如果当前属性是子模型
                const p = property.joined.parent;
                selects[p] = selects[p] || { display: false };
                subrequests[n] = properties[n];
            } else { // 否则当前属性是列
                const c = property.column;
                // 同一列被两个属性映射时，后者的别名会覆盖前者，结果会静默丢字段
                const owner = columnOwner[c];
                if (owner) {
                    throw new Error(`property ${owner} and ${n} map to the same column ${c}`);
                }
                columnOwner[c] = n;
                selects[c] = {
                    ...selects[c],
                    alias: n,
                    display: true,
                };
            }
        }

        return {
            selects,
            subrequests,
        };
    }

    private toSqlWhereClauses(conditions: ConditionQueryDSL[]) {
        const wheres: string[] = [],
            params: object[] = [];

        for (const condition of conditions) {
            if (condition.value == null) {
                throw new Error(`value of condition ${condition.field} can not be null`);
            }

            const operator = condition.operator as string;
            switch (condition.operator) {
                case "in":
                    if (!Array.isArray(condition.value)) {
                        throw new Error(`value of condition ${condition.field} must be an array for operator in`);
                    }
                    const values = condition.value.filter(v => v != null); // 去除 null 值
                    if (!values.length) {
                        // 全部被过滤时不能省略条件，否则会退化成无过滤的全表扫描
                        wheres.push("1 = 0");
                        break;
                    }
                    const chunkSize = this.inClauseChunkSize;
                    if (chunkSize <= 0 || values.length <= chunkSize) {
                        wheres.push(`${condition.field} in (${values.map(_ => "?").join(", ")})`);
                        params.push(...values);
                    } else {
                        // 超过单条 IN 的变量上限时按块拆分并用 OR 合并，避免触发驱动的参数数量限制
                        const pieces: string[] = [];
                        for (let i = 0; i < values.length; i += chunkSize) {
                            const part = values.slice(i, i + chunkSize);
                            pieces.push(`${condition.field} in (${part.map(_ => "?").join(", ")})`);
                            params.push(...part);
                        }
                        wheres.push(`(${pieces.join(" or ")})`);
                    }
                    break;
                case "eq":
                case "gt":
                case "ge":
                case "lt":
                case "le":
                case "ne": {
                    const op = { eq: "=", gt: ">", ge: ">=", lt: "<", le: "<=", ne: "<>" }[condition.operator];
                    wheres.push(`${condition.field} ${op} ?`);
                    params.push(condition.value);
                    break;
                }
                default:
                    throw new Error(`operator ${operator} is not supported`);
            }
        }

        return {
            wheres,
            params,
        };
    }

    private toQueryRecords(datasets: Record[], model: SubTableSchema, { selects, subrequests }: SelectClauses): { data: Record; keys: Record; }[] {
        if (!datasets.length) {
            return [];
        }

        const records: { data: Record; keys: Record; }[] = [];

        // 遍历子请求，构造外键和外键关联条件
        const subexrequests: ExtendedQueryDSL = {};
        for (const n in subrequests) {
            const { column: c, parent: p } = (<SubTableSchema>model.properties[n]).joined;
            subexrequests[n] = {
                foreignKeys: [c],
                conditions: [{
                    field: c,
                    operator: "in",
                    value: this.uniq(datasets.map(d => <object>d[p])),
                }],
                // 子查询必须取回该父对象集合的全部子行，不能被默认分页截断
                options: { limit: null },
            };
        }

        // 如果有子属性请求，查询子对象
        const subdatasets = this.doQuery(subrequests, model.properties, subexrequests) as { [name: string]: { data: object, keys?: Record; }[] };

        // 先遍历所有子对象，按照外键分类，便于后续回写子对象值
        const subdatasetsmap: { [name: string]: { [key: string]: object[] } } = {};
        for (const n in subdatasets) {
            const { column: c } = (model.properties[n] as SubTableSchema).joined;
            subdatasetsmap[n] = {};
            for (const subdataset of subdatasets[n]) {
                // 外键列缺失或驱动未返回该列时，按外键分组会静默出错，因此在此中断
                if (subdataset.keys?.[c] === undefined) {
                    throw new Error(`property ${c} in sub hide dataset does not exist`);
                }
                subdatasetsmap[n][subdataset.keys[c]] = subdatasetsmap[n][subdataset.keys[c]] || [];
                subdatasetsmap[n][subdataset.keys[c]].push(subdataset.data);
            }
        }

        // 遍历对象，回写属性值
        for (const dataset of datasets) {
            const data = {} as Record, // 数据（需要随最终结果返回）
                keys = {} as Record; // 键值（不需要返回，用于外键关联）

            for (const n in selects) {
                if (selects[n].display) { // 如果需要作为结果返回，属性名称需要重命名
                    data[selects[n].alias || n] = <object>dataset[n];
                }
                if (selects[n].isForeignKey) { // 如果是外键，则将属性放入不可见属性集中
                    keys[n] = <object>dataset[n];
                }
            }

            for (const n in subdatasetsmap) {
                const submodel = model.properties[n] as SubTableSchema;

                // 根据外键过滤子对象集合
                const sub = subdatasetsmap[n][dataset[submodel.joined.parent]];

                // 子对象赋值只写入结果对象，避免改写驱动返回的原始行
                let value: object | undefined;
                if (sub) {
                    if (submodel.joined.isCollection === false) { // 如果子模型非集合，则只允许匹配到一条
                        if ((<object[]>sub).length > 1) {
                            throw new Error(`property ${n} of ${model.table} is not a collection but matched ${(<object[]>sub).length} subdatasets`);
                        }
                        value = (<object[]>sub)[0];
                    } else {
                        value = sub;
                    }
                }

                data[n] = value; // 子模型需要作为结果返回
            }

            records.push({ data, keys });
        }

        return records;
    }

    //#endregion

    //#region mutate

    public mutate(dsl: MutationDSL) {
        if (dsl == null) {
            throw new Error(`dsl can not be null`);
        }
        if (typeof dsl !== "object" || dsl instanceof Array) {
            throw new Error(`dsl must be a object`);
        }

        const requests: MutationRequests = {};

        // 构造请求
        this.toMutationRequests(dsl, <SubSchema>this.schema, requests);

        // 执行请求，任一步骤失败则回滚已写入的数据
        this.onBegin();
        try {
            this.doMutateRequests(requests, <SubSchema>this.schema);
            this.onCommit();
        } catch (e) {
            try {
                this.onRollback();
            } catch {
                // 回滚失败不应掩盖触发回滚的原始错误
            }
            throw e;
        }

        return dsl;
    }

    private toMutationRequests(dsl: MutationDSL, schema: SubSchema, requests: MutationRequests) {
        for (const name in dsl) {
            // 查询表模型
            const model = schema[name];
            if (!model || !this.ownProperty(schema, name)) {
                throw new Error(`${name} in schema does not exist`);
            }
            if (!("table" in model)) {
                throw new Error(`${name} is not a table schema`);
            }

            // 校验请求
            if (!dsl[name]) {
                throw new Error(`request ${name} can not be null`);
            }
            if (typeof (dsl[name]) !== "object") {
                throw new Error(`request ${name} must be a object or collection`);
            }

            // 查询当前模型中保存了子模型主键的属性名称
            const propertiesOfSubPrimaryKey = Object.keys(model.properties)
                .filter(n => {
                    const property = model.properties[n];
                    if ("column" in property) { // 如果是列属性，跳过不处理
                        return false;
                    }
                    const joined = this.joinedProperty(property);
                    if (!joined || (<ColumnSchema>property.properties[joined]).isPrimaryKey !== true) {
                        return false;
                    }
                    if (property.joined.isCollection !== false) { // 前置子请求必须为非集合元素
                        throw new Error(`subschema ${n} can not be a collection for the foreign key ${property.joined.parent} is a primary key of subschema`);
                    }
                    return true;
                });

            const records = dsl[name] instanceof Array ? <Record[]>dsl[name] : [dsl[name]],
                data: MutationRequestData[] = requests[name]?.data || [], // 对象集合
                // 批量父对象下同名模型多次进入，子请求桶必须复用旧值累加，否则前一批的孙对象静默丢失
                subrequests0: MutationRequests = requests[name]?.subrequests0 || {}, // 前置子请求，需要优先入库以生成主键
                subrequests1: MutationRequests = requests[name]?.subrequests1 || {}; // 后置子请求，需要依赖当前请求的主键入库

            for (const record of records) {
                if (!record) {// 原始请求对象
                    throw new Error("request data can not be null");
                }
                // 数组内原始类型元素会走进 record 遍历，报出误导性的 property 0 does not exist 或 no writable fields
                if (typeof record !== "object" || record instanceof Array) {
                    throw new Error(`request data of ${name} must be a object`);
                }

                const dataset: Record = {}; // 数据库对象
                let hasSubrequest = false; // 当前记录是否携带子请求

                for (const subname in record) {
                    const property = model.properties[subname];
                    if (!property || !this.ownProperty(model.properties, subname)) {
                        throw new Error(`property ${subname} in schema ${name} does not exist`);
                    }
                    // null 表示不处理该属性，undefined 表示未提供；其余值（含 0、""、false）都需要写入
                    if (record[subname] === undefined || record[subname] === null) {
                        continue;
                    }

                    // 如果该属性是列，当前的请求对象属性写入到数据库对象属性
                    if ("column" in property) {
                        dataset[property.column] = <unknown>record[subname];
                        continue;
                    }

                    const isCollection = property.joined.isCollection !== false; // isCollection 为 true 或 null 均表示集合类型
                    if (isCollection !== (record[subname] instanceof Array)) {
                        throw new Error(`subrequest ${subname} ${isCollection ? "must" : "can not"} be a collection`);
                    }

                    const subdsl = {
                        [subname]: <MutationDSL>record[subname]
                    };

                    // 缓存前置子请求
                    if (propertiesOfSubPrimaryKey.includes(subname)) { // 如果该属性是子模型，并且子模型的主键关联父模型，则需要先保存子对象以生成子对象主键
                        this.toMutationRequests(subdsl, model.properties, subrequests0);
                        hasSubrequest = true;
                        continue;
                    }

                    // 缓存后置子请求
                    this.toMutationRequests(subdsl, model.properties, subrequests1);
                    hasSubrequest = true;
                }

                // 无可写字段且无子请求时生成非法 SQL；子请求只能按当前记录判断
                if (!Object.keys(dataset).length && !hasSubrequest) {
                    throw new Error(`request ${name} has no writable fields`);
                }

                // 同一记录对象重复出现会让外键回填命中多个条目，产生错挂与重复插入
                if (data.some(d => d.record === record)) {
                    throw new Error(`the same record object of ${name} can not be referenced more than once`);
                }

                data.push({
                    dataset,
                    record,
                });
            }

            requests[name] = {
                data,
                subrequests0,
                subrequests1,
            };
        }
    }

    private doMutateRequests(requests: MutationRequests, schema: SubSchema) {
        for (const name in requests) {
            const model = <SubTableSchema>schema[name],
                { data, subrequests0, subrequests1 } = requests[name];

            // 保存前置子请求
            this.doMutateRequests(subrequests0, model.properties);
            for (const subname in subrequests0) {
                const submodel = <SubTableSchema>model.properties[subname],
                    primaryKey = this.joinedProperty(submodel);
                // 遍历对象，将子对象以及子对象主键值回写至对象中
                for (const { record, dataset } of data) {
                    const subrecord = <Record>record[subname];
                    if (subrecord === undefined || subrecord === null) {
                        continue;
                    }
                    // 子主键缺失时父表外键会被静默写入 null，与后置方向校验不对称
                    if (subrecord[primaryKey] == null) {
                        throw new Error(`value of ${submodel.joined.column} is required`);
                    }
                    dataset[submodel.joined.parent] = <unknown>subrecord[primaryKey];
                }
            }

            // 保存请求并回写主键
            this.doMutationRequestData(data, name, model);

            // 遍历对象，将对象以及对象主键值回写至后置子请求中
            for (const subname in subrequests1) {
                const submodel = <SubTableSchema>model.properties[subname];
                const subPk = this.modelPrimaryKey(submodel);
                for (const { record, dataset } of data) {
                    // 批量请求中部分父对象可能未提供该子属性
                    if (record[subname] === undefined || record[subname] === null) {
                        continue;
                    }

                    const s = subrequests1[subname].data
                        .filter(d => {
                            if (submodel.joined.isCollection === false) {
                                return d.record === record[subname];
                            } else {
                                return (<Record[]>record[subname]).some(i => d.record === i);
                            }
                        });
                    if (!s.length) {
                        continue;
                    }

                    // 仅新增子对象需要回填外键；自带主键的子对象表示更新，外键列保持原值与类型
                    const pendings = s.filter(o => !(subPk && subPk.column in o.dataset));
                    if (!pendings.length) {
                        continue;
                    }
                    // 父对象主键必须已知，undefined 与 null 都表示缺失；0、"" 等假值仍是合法主键
                    if (dataset[submodel.joined.parent] == null) {
                        throw new Error(`value of ${submodel.joined.parent} is required`);
                    }
                    pendings.forEach(o => {
                        o.dataset[submodel.joined.column] = <unknown>dataset[submodel.joined.parent]; // 将外键塞入子对象中
                    });
                }
            }

            // 保存后置子对象
            this.doMutateRequests(subrequests1, model.properties);
        }
    }

    private doMutationRequestData(data: MutationRequestData[], name: string, model: SubTableSchema) {
        if (!data.length) {
            return;
        }

        // 找出当前模型的主键
        const pk = this.modelPrimaryKey(model);
        if (!pk) {
            throw new Error(`primary key of ${name} in schema does not exist`);
        }

        // 以列是否存在区分新增/更新，假值主键仍是更新
        const data2insert: MutationRequestData[] = [],
            data2update: MutationRequestData[] = [];
        for (const d of data) {
            if (pk.column in d.dataset) {
                data2update.push(d);
            } else {
                data2insert.push(d);
            }
        }

        if (data2insert.length) {
            const ids = this.onInsert(model.table, data2insert.map(d => d.dataset));
            if (!Array.isArray(ids) || ids.length !== data2insert.length) {
                throw new Error(`onInsert of ${name} must return an array of ${data2insert.length} primary keys`);
            }
            for (let i = 0; i < data2insert.length; i++) {
                // 回写 id 至原始请求对象和数据库对象中
                data2insert[i].record[pk.property] = ids[i];
                data2insert[i].dataset[pk.column] = ids[i];
            }
        }
        if (data2update.length) {
            this.onUpdate(model.table, data2update.map(d => d.dataset), pk.column);
        }
    }

    //#endregion

    private uniq<T>(arr: T[]): T[] {
        return [...new Set(arr)];
    }

    // 以自有属性判定查表结果，避免 toString、__proto__ 等原型链成员被误认为 schema 条目
    private ownProperty(owner: object, name: string) {
        return Object.prototype.hasOwnProperty.call(owner, name);
    }

    // 子模型中列名为父模型关联列的属性名（即子模型持有父模型主键值的属性）
    private joinedProperty(model: SubTableSchema) {
        return Object.keys(model.properties)
            .filter(n => (<ColumnSchema>model.properties[n]).column === model.joined.column)
            .pop() || "";
    }

    // 模型自身声明的主键
    private modelPrimaryKey(model: SubTableSchema) {
        const property = Object.keys(model.properties)
            .filter(n => (<ColumnSchema>model.properties[n]).isPrimaryKey === true)
            .pop();
        return property
            ? { property, column: (<ColumnSchema>model.properties[property]).column }
            : undefined;
    }

    // 单条 IN 子句的变量上限，超过后按块拆分用 OR 合并；设为 0 不分块，默认 999 贴合 SQLite
    protected inClauseChunkSize = 999;

    protected abstract onSelect(stmt: string, options: { params: object[]; }): object[];

    protected abstract onInsert(table: string, datasets: object[]): string[];

    protected abstract onUpdate(table: string, datasets: object[], primaryKey: string): void;

    // 变更事务钩子，默认无事务支持
    protected onBegin() {
        return null;
    }

    protected onCommit() {
        return null;
    }

    protected onRollback() {
        return null;
    }
}

export class MyGraphQL extends GraphQL {
    private db = $native("db");

    constructor(schema: Schema, initsql?: string) {
        super(schema);
        this.db.exec(initsql);
    }

    protected onSelect(stmt: string, options: { params: any[]; }): object[] {
        return this.db.query(stmt, ...options.params);
    }

    protected onInsert(table: string, datasets: Record[]): string[] {
        const that = this;
        return datasets.map(d => {
            that.db.exec(`insert into ${table} (${Object.keys(d).join(", ")}) values (${Object.keys(d).map(n => "?").join(", ")})`, ...Object.keys(d).map(i => d[i]));
            return that.db.query("select LAST_INSERT_ROWID() ROWID").pop().ROWID;
        });
    }

    protected onUpdate(table: string, datasets: Record[], primaryKey: string): void {
        datasets.forEach(d => {
            this.db.exec(`update ${table} set ${Object.keys(d).filter(n => n !== primaryKey).map(n => `${n} = ?`).join(", ")} where ${primaryKey} = ?`, ...Object.keys(d).filter(n => n !== primaryKey).map(i => d[i]).concat([d[primaryKey]]));
        });
    }

    // 变更事务钩子，确保批量写入的原子性
    protected onBegin() {
        this.db.exec("begin");
    }

    protected onCommit() {
        this.db.exec("commit");
    }

    protected onRollback() {
        this.db.exec("rollback");
    }
}
```

### Usage

```typescript
import { MyGraphQL } from "graphql"

const G = new MyGraphQL({
    offering: {
        table: "Offering",
        properties: {
            spu: {
                column: "Id",
                isPrimaryKey: true
            },
            name: {
                table: "I18n",
                properties: {
                    id: {
                        column: "Id",
                        isPrimaryKey: true
                    },
                    zh: {
                        column: "Zh"
                    },
                    en: {
                        column: "En"
                    }
                },
                joined: {
                    column: "Id",
                    parent: "Name",
                    isCollection: false
                }
            },
            description: {
                column: "Description"
            },
            product: {
                table: "Product",
                properties: {
                    sku: {
                        column: "Id",
                        isPrimaryKey: true
                    },
                    price: {
                        column: "Price"
                    },
                    stock: {
                        column: "Stock"
                    },
                    attribute: {
                        table: "ProductAttribute",
                        properties: {
                            id: {
                                column: "Id",
                                isPrimaryKey: true
                            },
                            code: {
                                column: "Code"
                            },
                            value: {
                                column: "Value"
                            },
                            type: {
                                column: "Type"
                            }
                        },
                        joined: {
                            column: "ProductId",
                            parent: "Id"
                        }
                    }
                },
                joined: {
                    column: "OfferingId",
                    parent: "Id"
                }
            }
        }
    }
}, `
    create table if not exists Offering (
        Id integer primary key autoincrement,
        Name text,
        Description text
    );
    create table if not exists Product (
        Id integer primary key autoincrement,
        Price text,
        Stock text,
        OfferingId text
    );
    create table if not exists ProductAttribute (
        Id integer primary key autoincrement,
        Code text,
        Value text,
        Type text,
        ProductId text
    );
    create table if not exists I18n (
        Id integer primary key autoincrement,
        Zh text,
        En text
    );
`);

export default function (ctx) {
    console.info("Mutate:", JSON.stringify(
        // 保存记录
        G.mutate({
            offering: {
                name: {
                    zh: "球",
                    en: "ball"
                },
                description: "this is a ball",
                product: [{
                    price: 2,
                    attribute: [{
                        code: "color",
                        value: "red"
                    }]
                }, {
                    price: 2.5,
                    attribute: [{
                        code: "color",
                        value: "green"
                    }]
                }]
            }
        }), null, "    "
    ));

    console.info("Query:", JSON.stringify(
        // 查询记录
        G.query({
            offering: {
                properties: null,
                conditions: [{ // 查询条件
                    field: "spu",
                    operator: "eq",
                    value: 1
                }],
                options: { // 分页查询
                    limit: 1,
                    skip: 0
                }
            }
        }), null, "    "
    ));
    console.info("Query:", JSON.stringify(
        // 查询记录
        G.query({
            offering: {
                name: null,
                description: "this is a ball", // 如果属性不为 null，则表示该属性作为查询条件
                product: null
            }
        }), null, "    "
    ));
    return null
}
```