package module

import (
	"cube/internal/builtin"
)

var Factories = make(map[string]func(ctx Context) interface{})

func register(name string, factory func(ctx Context) interface{}) {
	Factories[name] = factory
}

type Context = builtin.Context
