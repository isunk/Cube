package util

import (
	"database/sql"
	"strconv"
	"time"
)

type querier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

func Query(db querier, stmt string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	columnTypes, _ := rows.ColumnTypes()

	dataset := make([]interface{}, len(columns))
	row := make([]interface{}, len(columns))
	for i := range dataset {
		row[i] = &dataset[i] // 将每个值的指针放入接口切片中
	}

	var records []map[string]interface{}
	for rows.Next() {
		rows.Scan(row...)
		record := make(map[string]interface{})
		for i, data := range dataset {
			if bytes, ok := data.([]byte); ok { // 对于使用 MySQL 驱动程序，返回值始终为 []byte，这里根据列类型进行转换（参考 https://github.com/go-sql-driver/mysql/issues/1401）
				value := string(bytes)
				switch columnTypes[i].DatabaseTypeName() {
				case "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "YEAR":
					data, _ = strconv.Atoi(value)
				case "TINYINT", "BOOL", "BOOLEAN", "BIT":
					data, _ = strconv.ParseBool(value)
				case "FLOAT", "DOUBLE", "DECIMAL":
					data, _ = strconv.ParseFloat(value, 64)
				case "DATETIME", "TIMESTAMP":
					data, _ = time.Parse("2006-01-02 15:04:05", value)
				case "DATE":
					data, _ = time.Parse("2006-01-02", value)
				case "TIME":
					data, _ = time.Parse("15:04:05", value)
				case "NULL":
					data = nil
				default:
					data = value
				}
			}
			record[columns[i]] = data
		}
		records = append(records, record)
	}

	return records, rows.Err()
}
