package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"cube/internal"
	"cube/internal/util"
)

var READONLY_TABLES = []string{"source"}

var (
	REGEX_DATABASE_IDENTIFIER  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`)   // 数据库标识符（表名/列名/字段名）校验
	REGEX_DATABASE_COLUMN_TYPE = regexp.MustCompile(`^[A-Za-z]\w*(\(\d+(,\d+)?\))?$`) // 数据库列的类型校验（例如 `INTEGER`、`TEXT`、`VARCHAR(255)`、`DECIMAL(10,2)`）
)

func HandleDatabase(w http.ResponseWriter, r *http.Request) {
	p := &util.QueryParams{Values: r.URL.Query()}
	table, column, record := p.Get("table"), p.Get("column"), p.Get("record")

	if table != "" {
		// 验证表名合法性（空表名表示查询列表，跳过校验）
		if err := util.ValidateString(table, REGEX_DATABASE_IDENTIFIER, "invalid table name"); err != nil {
			Error(w, err)
			return
		}
	}

	var (
		data       interface{}
		returnless bool
		err        error
	)

	switch r.Method {
	case http.MethodGet:
		if p.Has("table") && table == "" {
			data, err = handleTableGet()
		} else if table != "" && p.Has("column") && column == "" {
			data, err = handleColumnGet(table)
		} else if table != "" && p.Has("record") {
			data, returnless, err = handleRecordGet(table, record, p)
		} else {
			err = errors.New("invalid request")
		}

	case http.MethodPost:
		if p.Has("sql") {
			data, err = handleSQLExecute(r)
		} else if table != "" && !p.Has("column") && !p.Has("record") {
			data, err = handleTableCreate(r, table)
		} else if table != "" && p.Has("column") {
			data, err = handleColumnCreateOrUpdate(r, table, column)
		} else if table != "" && p.Has("record") {
			data, err = handleRecordCreateOrUpdate(r, table)
		} else {
			err = errors.New("invalid request")
		}

	case http.MethodDelete:
		if table != "" && !p.Has("column") && !p.Has("record") {
			data, err = handleTableDelete(table)
		} else if table != "" && column != "" {
			data, err = handleColumnDelete(table, column)
		} else if table != "" && p.Has("record") {
			data, err = handleRecordDelete(table, record, p)
		} else {
			err = errors.New("invalid request")
		}

	default:
		Error(w, http.StatusMethodNotAllowed)
		return
	}

	if err != nil {
		Error(w, err)
		return
	}

	if !returnless {
		Success(w, data)
	} else {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.Encode(data)
		w.Write(buf.Bytes())
	}
}

func handleTableGet() (interface{}, error) {
	records, err := util.Query(internal.Db, "select name from sqlite_master where type='table' and name not like 'sqlite_%' order by name")
	if err != nil {
		return nil, err
	}
	tables := make([]string, len(records))
	for i, rec := range records {
		tables[i] = rec["name"].(string)
	}
	return tables, nil
}

func handleTableCreate(r *http.Request, table string) (interface{}, error) {
	if slices.Contains(READONLY_TABLES, table) {
		return nil, errors.New(fmt.Sprintf("operation not permitted on system table: %s", table))
	}

	var body map[string]interface{}
	if err := util.UnmarshalWithIoReader(r.Body, &body); err != nil {
		return nil, err
	}

	columns, ok := body["columns"].([]interface{})
	if !ok || len(columns) == 0 {
		return nil, errors.New("columns are required")
	}

	var defs []string
	for _, c := range columns {
		col, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		cname, _ := col["name"].(string)
		ctype, _ := col["type"].(string)
		if cname == "" || ctype == "" {
			return nil, errors.New("column name and type are required")
		}
		// 校验列名合法性
		if err := util.ValidateString(cname, REGEX_DATABASE_IDENTIFIER, "invalid column name"); err != nil {
			return nil, err
		}
		// 校验列类型合法性
		if err := util.ValidateString(ctype, REGEX_DATABASE_COLUMN_TYPE, "invalid column type"); err != nil {
			return nil, err
		}
		defs = append(defs, toSQLColumn(col))
	}

	if len(defs) == 0 {
		return nil, errors.New("no valid columns")
	}

	_, err := internal.Db.Exec(fmt.Sprintf("create table \"%s\" (%s)", table, strings.Join(defs, ", ")))
	if err != nil {
		return nil, err
	}
	return 1, nil
}

func handleTableDelete(table string) (interface{}, error) {
	if slices.Contains(READONLY_TABLES, table) {
		return nil, errors.New(fmt.Sprintf("operation not permitted on system table: %s", table))
	}

	result, err := internal.Db.Exec(fmt.Sprintf("drop table if exists \"%s\"", table))
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

func handleColumnGet(table string) (interface{}, error) {
	records, err := util.Query(internal.Db, fmt.Sprintf("pragma table_info(\"%s\")", table))
	if err != nil {
		return nil, err
	}
	columns := make([]map[string]interface{}, len(records))
	for i, rec := range records {
		columns[i] = toColumn(rec)
	}
	return columns, nil
}

func handleColumnCreateOrUpdate(r *http.Request, table, column string) (interface{}, error) {
	if slices.Contains(READONLY_TABLES, table) {
		return nil, errors.New(fmt.Sprintf("operation not permitted on system table: %s", table))
	}

	var body map[string]interface{}
	if err := util.UnmarshalWithIoReader(r.Body, &body); err != nil {
		return nil, err
	}

	if column == "" {
		cname, _ := body["name"].(string)
		ctype, _ := body["type"].(string)
		if cname == "" || ctype == "" {
			return nil, errors.New("column name and type are required")
		}
		// 校验列名合法性
		if err := util.ValidateString(cname, REGEX_DATABASE_IDENTIFIER, "invalid column name"); err != nil {
			return nil, err
		}
		// 校验列类型合法性
		if err := util.ValidateString(ctype, REGEX_DATABASE_COLUMN_TYPE, "invalid column type"); err != nil {
			return nil, err
		}
		def := toSQLColumn(map[string]interface{}{
			"name":     cname,
			"type":     ctype,
			"nullable": body["nullable"],
			"default":  body["default"],
		})
		_, err := internal.Db.Exec(fmt.Sprintf("alter table \"%s\" add column %s", table, def))
		if err != nil {
			return nil, err
		}
		return 1, nil
	}

	name, _ := body["name"].(string)
	if name == "" {
		return nil, errors.New("name is required")
	}
	// 校验原列名合法性
	if err := util.ValidateString(column, REGEX_DATABASE_IDENTIFIER, "invalid column name"); err != nil {
		return nil, err
	}
	// 校验新列名合法性
	if err := util.ValidateString(name, REGEX_DATABASE_IDENTIFIER, "invalid column name"); err != nil {
		return nil, err
	}
	_, err := internal.Db.Exec(fmt.Sprintf("alter table \"%s\" rename column \"%s\" to \"%s\"", table, column, name))
	if err != nil {
		return nil, err
	}
	return 1, nil
}

func handleColumnDelete(table, column string) (interface{}, error) {
	if slices.Contains(READONLY_TABLES, table) {
		return nil, errors.New(fmt.Sprintf("operation not permitted on system table: %s", table))
	}
	// 校验列名合法性
	if err := util.ValidateString(column, REGEX_DATABASE_IDENTIFIER, "invalid column name"); err != nil {
		return nil, err
	}

	_, err := internal.Db.Exec(fmt.Sprintf("alter table \"%s\" drop column \"%s\"", table, column))
	if err != nil {
		return nil, err
	}
	return 1, nil
}

func handleRecordGet(table, record string, p *util.QueryParams) (interface{}, bool, error) {
	if p.Has("bulk") {
		ids := append(strings.Split(record, ","), "") // 追加空字符串占位，适用于全选但未勾选黑名单的删除或导出场景
		params := make([]interface{}, len(ids))
		for i, v := range ids {
			params[i] = v
		}
		operation := "not in"
		if !p.Has("reversion") {
			operation = "in"
		}
		records, err := util.Query(internal.Db, fmt.Sprintf("select _rowid_ AS rowid, * from \"%s\" where rowid %s (%s)", table, operation, strings.Repeat("?,", len(ids)-1)+"?"), params...)
		if err != nil {
			return nil, false, err
		}
		return records, true, nil
	}

	from, size := p.GetIntOrDefault("from", 0), p.GetIntOrDefault("size", 500)

	records, err := util.Query(internal.Db, fmt.Sprintf("select _rowid_ AS rowid, * from \"%s\" limit ? offset ?", table), size, from)
	if err != nil {
		return nil, false, err
	}

	var total int
	internal.Db.QueryRow(fmt.Sprintf("select count(1) from \"%s\"", table)).Scan(&total)

	return map[string]interface{}{
		"records": records,
		"total":   total,
	}, false, nil
}

func handleRecordCreateOrUpdate(r *http.Request, table string) (interface{}, error) {
	if slices.Contains(READONLY_TABLES, table) {
		return nil, errors.New(fmt.Sprintf("operation not permitted on system table: %s", table))
	}

	records := make([]map[string]interface{}, 0)
	if err := util.UnmarshalWithIoReader(r.Body, &records); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return 0, nil
	}

	var total int
	for _, row := range records {
		columns, params := []string{}, []interface{}{}
		for k, v := range row {
			if k != "rowid" {
				// 校验列名合法性，防止 SQL 注入
				if err := util.ValidateString(k, REGEX_DATABASE_IDENTIFIER, "invalid column name: "+k); err != nil {
					return nil, err
				}
				columns = append(columns, "\""+k+"\"")
				params = append(params, v)
			} else {
				// rowid 作为显式列插入，确保 on conflict 能命中目标行
				columns = append(columns, "rowid")
				params = append(params, v)
			}
		}
		if len(columns) == 0 {
			continue
		}
		statement := fmt.Sprintf("insert into \"%s\" (%s) values (%s)", table, strings.Join(columns, ", "), strings.Repeat("?,", len(columns)-1)+"?")
		if _, ok := row["rowid"]; ok {
			statement += " on conflict(_rowid_) do update set " + strings.Join(columns, " = ?, ") + " = ?"
			for i := range columns {
				params = append(params, params[i])
			}
		}
		_, err := internal.Db.Exec(statement, params...)
		if err != nil {
			return nil, err
		}
		total++
	}
	return total, nil
}

func handleRecordDelete(table, record string, p *util.QueryParams) (interface{}, error) {
	if slices.Contains(READONLY_TABLES, table) {
		return nil, errors.New(fmt.Sprintf("operation not permitted on system table: %s", table))
	}

	ids := append(strings.Split(record, ","), "") // 追加空字符串占位，适用于全选但未勾选黑名单的删除场景
	params := make([]interface{}, len(ids))
	for i, v := range ids {
		params[i] = v
	}
	operation := "not in"
	if !p.Has("reversion") {
		operation = "in"
	}
	result, err := internal.Db.Exec(fmt.Sprintf("delete from \"%s\" where rowid %s (%s)", table, operation, strings.Repeat("?,", len(ids)-1)+"?"), params...)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

func handleSQLExecute(r *http.Request) (interface{}, error) {
	var body map[string]interface{}
	if err := util.UnmarshalWithIoReader(r.Body, &body); err != nil {
		return nil, err
	}

	statement, _ := body["statement"].(string)
	if statement == "" {
		return nil, errors.New("statement is required")
	}

	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(statement)), "SELECT") {
		return nil, errors.New("only select is allowed")
	}

	params, _ := body["params"].([]interface{})

	records, err := util.Query(internal.Db, statement, params...)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"records": records,
		"total":   len(records),
	}, nil
}

func toColumn(r map[string]interface{}) map[string]interface{} {
	pk := int64(0)
	if v, ok := r["pk"].(int64); ok {
		pk = v
	}
	notnull := int64(0)
	if v, ok := r["notnull"].(int64); ok {
		notnull = v
	}
	return map[string]interface{}{
		"name":     r["name"],
		"type":     r["type"],
		"pk":       pk != 0,
		"nullable": notnull == 0,
		"default":  r["dflt_value"],
	}
}

func toSQLColumn(input map[string]interface{}) string {
	cname, _ := input["name"].(string)
	ctype, _ := input["type"].(string)
	output := fmt.Sprintf("\"%s\" %s", cname, ctype)
	if pk, _ := input["pk"].(bool); pk {
		output += " PRIMARY KEY"
	}
	if nullable, ok := input["nullable"].(bool); ok && !nullable {
		output += " NOT NULL"
	}
	if d, ok := input["default"]; ok && d != nil {
		output += " DEFAULT "
		switch val := d.(type) {
		case string:
			if val == "" {
				output += "''"
			} else if _, err := fmt.Sscan(val, new(float64)); err == nil {
				output += val
			} else {
				output += "'" + strings.ReplaceAll(val, "'", "''") + "'"
			}
		case float64:
			output += fmt.Sprintf("%v", val)
		case bool:
			if val {
				output += "1"
			} else {
				output += "0"
			}
		case nil:
			output += "NULL"
		default:
			output += "'" + strings.ReplaceAll(fmt.Sprintf("%v", d), "'", "''") + "'"
		}
	}
	return output
}
