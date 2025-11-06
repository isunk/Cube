package module

import (
	"cube/internal/builtin"
)

func init() {
	register("db", func(ctx Context) interface{} {
		dbc := builtin.NewDatabaseClient(ctx)
		return &dbc
	})
}
