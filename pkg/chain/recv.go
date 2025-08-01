package chain

import (
	"fmt"
	"log/slog"
	"reflect"
)

func RecvAs[T any](link Link) (T, bool) {
	if chain, ok := link.(*BaseChain); ok {
		chain.startIfUnstarted()
	}

	v, ok := <-link.channel()
	if !ok {
		return *new(T), false
	}

	outputType := reflect.TypeOf(*new(T))
	if outputType == nil {
		outputType = reflect.TypeOf((*any)(nil)).Elem()
	}

	cast, err := Convert(v, outputType)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to receive value from: %T", link), "error", err)
		return *new(T), false
	}

	return cast.Interface().(T), true
}
