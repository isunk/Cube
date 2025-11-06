package module

import (
	"cube/internal/builtin"
)

func init() {
	register("db", func(ctx Context) interface{} {
		return builtin.NewDatabaseClient(ctx)
	})
}
