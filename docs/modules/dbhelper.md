# DbHelper


```typescript
//?name=DbHelper&type=module
export enum ColumnType {
    Boolean,
    Integer,
    String,
    Datetime,
    Text,
}

enum IndexType {
    Index = "INDEX",
    UniqueIndex = "UNIQUE INDEX",
}

interface Column {
    name: string
    type: ColumnType
}

interface Condition {
    conjunction: "AND" | "OR"
    conditions: {
        field: string
        operator: "=" | "<>" | ">" | "<" | "in"
        value?: any
    }[]
}

export class MySQLHelper {
    protected dbc: Database

    constructor(dbc: Database) {
        this.dbc = dbc
    }

    public showTables() {
        return this.dbc.query("SHOW TABLES").map(i => {
            return Object.values(i)[0]
        })
    }

    public createTable(tableName: string, columns: Column[]) {
        return this.dbc.exec(`CREATE TABLE IF NOT EXISTS ${tableName} (
            ${this.onParseIDColumn()},
            ${columns
                .map(i => {
                    const defaultValue = {
                        [ColumnType.Boolean]: "false",
                        [ColumnType.Integer]: "0",
                        [ColumnType.String]: "''",
                    }[i.type]
                    return `${i.name} ${this.onParseColumnType(i.type)} NOT NULL ${defaultValue ? `DEFAULT ${defaultValue}` : ""}`
                })
                .join(",\n")
            }
        ) ${this.onParseTableExtInfo()}`)
    }

    public dropTable(tableName: string) {
        return this.dbc.exec(`DROP TABLE IF EXISTS ${tableName}`)
    }

    public showColumns(tableName: string) {
        return this.dbc.query(`SHOW COLUMNS FROM ${tableName}`).map(i => {
            return {
                name: i.Field,
                type: i.Type,
                key: i.Key,
            }
        })
    }

    public addColumn(tableName: string, columnName: string, columnType: ColumnType) {
        return this.dbc.exec(`ALTER TABLE ${tableName} ADD ${columnName} ${this.onParseColumnType(columnType)}`)
    }

    public alterColumn(tableName: string, columnName: string, columnType: ColumnType) {
        return this.dbc.exec(`ALTER TABLE ${tableName} ALTER COLUMN ${columnName} ${this.onParseColumnType(columnType)}`)
    }

    public dropColumn(tableName: string, columnName: string) {
        return this.dbc.exec(`ALTER TABLE ${tableName} DROP COLUMN ${columnName}`)
    }

    public createIndex(tableName: string, columnName: string, indexName: string, indexType: IndexType = IndexType.Index) {
        return this.dbc.exec(`CREATE ${indexType} ${indexName} ON ${tableName} (${columnName})`)
    }

    public dropIndex(tableName: string, indexName: string) {
        return this.dbc.exec(`DROP INDEX ${indexName} ON ${tableName}`)
    }

    public insert(tableName: string, obj: Record<string, any>) {
        const columns = Object.keys(obj)
        this.dbc.exec(`INSERT INTO ${tableName}(${columns.join(", ")}) VALUES(${columns.map(_ => "?").join(", ")})`, ...columns.map(c => obj[c]))
        return this.dbc.query(`SELECT ${this.onParseLastInsertID()} ID`)[0].ID
    }

    public delete(tableName: string, condition: Condition) {
        const { wheres, params } = this.onParseCondition(condition)
        return this.dbc.exec(`DELETE FROM ${tableName} WHERE ${wheres}`, ...params)
    }

    public update(tableName: string, condition: Condition, obj: Record<string, any>) {
        const columns = Object.keys(obj),
            { wheres, params } = this.onParseCondition(condition)
        return this.dbc.exec(`UPDATE ${tableName} SET ${columns.map(c => c + " = ?").join(", ")} WHERE ${wheres}`, ...columns.map(c => obj[c]), ...params)
    }

    public select(tableName: string, condition?: Condition) {
        const { wheres, params } = this.onParseCondition(condition)
        return this.dbc.query(`SELECT * FROM ${tableName} WHERE ${wheres}`, ...params)
    }

    public query(stmt: string, ...params: any[]) {
        return this.dbc.query(stmt, ...params)
    }

    public exec(stmt: string, ...params: any[]) {
        return this.dbc.exec(stmt, ...params)
    }

    protected onParseTableExtInfo() {
        return "ENGINE=InnoDB CHARSET=utf8"
    }

    protected onParseIDColumn() {
        return "ID INT PRIMARY KEY AUTO_INCREMENT"
    }

    protected onParseLastInsertID() {
        return "LAST_INSERT_ID()"
    }

    protected onParseColumnType(columnType: ColumnType): string {
        switch (columnType) {
            case ColumnType.Boolean:
                return "BOOL"
            case ColumnType.Integer:
                return "INT"
            case ColumnType.String:
                return "VARCHAR(255)"
            case ColumnType.Datetime:
                return "DATETIME"
            case ColumnType.Text:
                return "TEXT"
            default:
                throw new Error("unknown column type")
        }
    }

    private onParseCondition(condition?: Condition) {
        let wheres = [],
            params = []
        if (condition) {
            for (const c of condition.conditions) {
                if (["in"].includes(c.operator)) {
                    wheres.push(`${c.field} ${c.operator} (${c.value.map(() => "?").join(",")})`)
                    params.push(...c.value)
                    continue
                }
                wheres.push(`${c.field} ${c.operator} ?`)
                params.push(c.value)
            }
        }
        return {
            wheres: wheres.join(condition?.conjunction) || "1 = 1",
            params,
        }
    }
}

export class Sqlite3Adpater extends MySQLHelper {    
    protected onParseTableExtInfo() {
        return ""
    }

    protected onParseIDColumn() {
        return "ID INTEGER PRIMARY KEY AUTOINCREMENT"
    }

    protected onParseLastInsertID() {
        return "LAST_INSERT_ROWID()"
    }

    protected onParseColumnType(columnType: ColumnType): string {
        switch (columnType) {
            case ColumnType.Boolean:
                return "BOOLEAN"
            case ColumnType.Integer:
                return "INTEGER"
            case ColumnType.Datetime:
                return "DATETIME"
            case ColumnType.String:
            case ColumnType.Text:
                return "TEXT"
            default:
                throw new Error("unknown column type")
        }
    }
}

export const helper = new MySQLHelper(new Database("mysql", "username:pass@tcp(mysql2.sqlpub.com:3307)/dbname"))
```