package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"cube/internal"
	"cube/internal/util"
)

func HandleDatabase(w http.ResponseWriter, r *http.Request) {
	var (
		data interface{}
		err  error
	)
	switch r.Method {
	case http.MethodGet:
		data, err = handleDatabaseGet(r)
	case http.MethodPost:
		data, err = handleDatabasePost(r)
	case http.MethodDelete:
		err = handleDatabaseDelete(r)
	default:
		Error(w, http.StatusMethodNotAllowed)
		return
	}
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, data)
}

func handleDatabaseGet(r *http.Request) (interface{}, error) {
	table := r.URL.Query().Get("table")
	if table == "" {
		rows, err := internal.Db.Query("select name from sqlite_master where type='table' and name not like 'sqlite_%' order by name")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var tables []string
		for rows.Next() {
			var name string
			rows.Scan(&name)
			tables = append(tables, name)
		}
		return tables, nil
	}

	columns, err := getTableColumns(table)
	if err != nil {
		return nil, err
	}

	rows, err := internal.Db.Query("select rowid, * from \"" + table + "\" limit 500")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []map[string]interface{}
	colCount := len(columns) + 1
	for rows.Next() {
		values := make([]interface{}, colCount)
		ptrs := make([]interface{}, colCount)
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		record := make(map[string]interface{})
		for i, col := range columns {
			record[col] = values[i+1]
		}
		record["rowid"] = values[0]
		records = append(records, record)
	}

	var rowCount int
	internal.Db.QueryRow("select count(1) from \"" + table + "\"").Scan(&rowCount)

	return map[string]interface{}{
		"columns": columns,
		"records": records,
		"total":   rowCount,
	}, nil
}

func handleDatabasePost(r *http.Request) (interface{}, error) {
	var body map[string]string
	if err := util.UnmarshalWithIoReader(r.Body, &body); err != nil {
		return nil, err
	}

	sqlStmt := body["sql"]
	if sqlStmt == "" {
		return nil, errors.New("sql is required")
	}

	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sqlStmt)), "SELECT") {
		rows, err := internal.Db.Query(sqlStmt)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}

		var records []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				continue
			}
			record := make(map[string]interface{})
			for i, col := range cols {
				record[col] = values[i]
			}
			records = append(records, record)
		}

		return map[string]interface{}{
			"columns": cols,
			"records": records,
			"total":   len(records),
		}, nil
	}

	result, err := internal.Db.Exec(sqlStmt)
	if err != nil {
		return nil, err
	}

	affected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()
	return map[string]interface{}{
		"affected": affected,
		"lastId":   lastID,
	}, nil
}

func handleDatabaseDelete(r *http.Request) error {
	table := r.URL.Query().Get("table")
	rowids := r.URL.Query().Get("rowids")

	if rowids != "" && table != "" {
		ids := strings.Split(rowids, ",")
		placeholders := make([]string, len(ids))
		args := make([]interface{}, len(ids))
		for i, id := range ids {
			placeholders[i] = "?"
			args[i] = id
		}
		_, err := internal.Db.Exec("delete from \""+table+"\" where rowid in ("+strings.Join(placeholders, ",")+")", args...)
		return err
	}

	rowid := r.URL.Query().Get("rowid")
	if rowid != "" && table != "" {
		_, err := internal.Db.Exec("delete from \""+table+"\" where rowid = ?", rowid)
		return err
	}

	if table == "" {
		return errors.New("table is required")
	}

	_, err := internal.Db.Exec("drop table if exists \"" + table + "\"")
	return err
}

func getTableColumns(table string) ([]string, error) {
	rows, err := internal.Db.Query("pragma table_info(\"" + table + "\")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var defaultVal *string
		rows.Scan(&cid, &name, &typ, &notnull, &defaultVal, &pk)
		columns = append(columns, name)
	}
	return columns, nil
}

var _ = sql.ErrNoRows
