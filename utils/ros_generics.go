package utils

import (
	"fmt"
	"reflect"
)

type Result[T any] struct {
	Value T
	Err   error
}

func GenericCallback[T any](msg interface{}, handleFunc func(Result[T])) {
	var result Result[T]
	msgCast, ok := msg.(T)
	if !ok {
		result.Err = fmt.Errorf("failed to cast message to type %T", reflect.TypeOf(*new(T)))
	} else {
		result.Value = msgCast
	}
	handleFunc(result)
}
