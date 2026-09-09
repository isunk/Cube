package builtin

import (
	"context"
	"database/sql"
	"fmt"

	"cube/internal/cache"
	"cube/internal/util"
	"github.com/dop251/goja"
	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

func init() {
	Factories = append(Factories, func(ctx Context) {
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

			db, err := cache.DB.Get(dtype, connection)
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

type DatabaseTransaction struct {
	t *sql.Tx
}

func (d *DatabaseTransaction) Query(stmt string, params ...interface{}) (*[]map[string]interface{}, error) {
	records, err := util.Query(d.t, stmt, params...)
	if err != nil {
		return nil, err
	}
	return &records, nil
}

func (d *DatabaseTransaction) Exec(stmt string, params ...interface{}) (sql.Result, error) {
	res, err := d.t.Exec(stmt, params...)
	if err != nil {
		return nil, err
	}
	return res, nil
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

func (d *DatabaseClient) Query(stmt string, params ...interface{}) (*[]map[string]interface{}, error) {
	records, err := util.Query(d.ctx.Db, stmt, params...)
	if err != nil {
		return nil, err
	}
	return &records, nil
}

func (d *DatabaseClient) Exec(stmt string, params ...interface{}) (sql.Result, error) {
	res, err := d.ctx.Db.Exec(stmt, params...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (d *DatabaseClient) Transaction(fn goja.Callable, isolation sql.IsolationLevel) (err error) { // 此处提前声明了返回值 err，否则 defer 函数将无法对 err 重新赋值
	if fn == nil {
		err = fmt.Errorf("function required")
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
			err = fmt.Errorf("%v", x) // 从 panic 中恢复错误，并重新赋值给 err
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	_, err = fn(nil, d.ctx.Worker.Runtime().ToValue(&DatabaseTransaction{tx}))

	return
}

func NewDatabaseClient(ctx Context) *DatabaseClient {
	return &DatabaseClient{ctx}
}
