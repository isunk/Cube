package builtin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/dop251/goja"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	Builtins = append(Builtins, func(ctx Context) {
		runtime := ctx.Worker.Runtime()

		runtime.Set("Database", func(call goja.ConstructorCall) *goja.Object {
			dtype, ok := call.Argument(0).Export().(string)
			if !ok {
				panic(runtime.NewTypeError("invalid database type: not a string"))
			}

			connection, ok := call.Argument(1).Export().(string)
			if !ok {
				panic(runtime.NewTypeError("invalid connection: not a string"))
			}

			db, err := ctx.Cache.GetDbSource(dtype, connection)
			if err != nil {
				panic(runtime.NewTypeError("invalid connection: connect failed"))
			}

			output := NewDatabaseClient(Context{
				Worker: ctx.Worker,
				Db:     db,
			})

			iv := runtime.ToValue(output).(*goja.Object)
			iv.SetPrototype(call.This.Prototype())
			return iv
		})
	})
}

func NewDatabaseClient(ctx Context) *DatabaseClient {
	return &DatabaseClient{ctx}
}

func ExportDatabaseRows(rows *sql.Rows) ([]interface{}, error) {
	defer rows.Close()

	columns, _ := rows.Columns()
	columnTypes, _ := rows.ColumnTypes()
	dataset, row := make([]interface{}, len(columns)), make([]interface{}, len(columns))
	for index := range dataset {
		row[index] = &dataset[index] // 将每个值的指针放入接口切片中
	}

	var records []interface{}

	for rows.Next() {
		rows.Scan(row...)

		record := make(map[string]interface{})
		for index, data := range dataset {
			if bytes, ok := data.([]byte); ok { // 对于使用 MySQL 驱动程序，返回值始终为 []byte，这里根据列类型进行转换（参考 https://github.com/go-sql-driver/mysql/issues/1401）
				value := string(bytes)
				switch columnTypes[index].DatabaseTypeName() {
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
			record[columns[index]] = data
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

type DatabaseTransaction struct {
	t *sql.Tx
}

func (d *DatabaseTransaction) Query(stmt string, params ...interface{}) ([]interface{}, error) {
	rows, err := d.t.Query(stmt, params...)
	if err != nil {
		return nil, err
	}
	return ExportDatabaseRows(rows)
}

func (d *DatabaseTransaction) Exec(stmt string, params ...interface{}) (int64, error) {
	res, err := d.t.Exec(stmt, params...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *DatabaseTransaction) Commit() error {
	return d.t.Commit()
}

func (d *DatabaseTransaction) Rollback() error {
	return d.t.Rollback()
}

type DatabaseClient struct {
	ctx Context
}

func (d *DatabaseClient) Query(stmt string, params ...interface{}) ([]interface{}, error) {
	rows, err := d.ctx.Db.Query(stmt, params...)
	if err != nil {
		return nil, err
	}
	return ExportDatabaseRows(rows)
}

func (d *DatabaseClient) Exec(stmt string, params ...interface{}) (int64, error) {
	res, err := d.ctx.Db.Exec(stmt, params...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *DatabaseClient) Transaction(fn goja.Callable, isolation sql.IsolationLevel) (err error) { // 此处提前声明了返回值 err，否则 defer 函数将无法对 err 重新赋值
	if fn == nil {
		err = errors.New("function required")
		return
	}

	// 开启一个新事务
	tx, err := d.ctx.Db.BeginTx(context.Background(), &sql.TxOptions{Isolation: isolation})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		if x := recover(); x != nil {
			err = errors.New(fmt.Sprint(x)) // 从 panic 中恢复错误，并重新赋值给 err
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	_, err = fn(nil, d.ctx.Worker.Runtime().ToValue(&DatabaseTransaction{tx}))

	return
}
