package cfg

import (
	"fmt"
	"reflect"
)

func As[T any](a any) (T, error) {
	casted, ok := a.(T)
	isNil := a == nil
	canNil := isNil && canBeNil[T]()
	if !ok && !canNil {
		if isNil {
			typeName := reflect.TypeOf((*T)(nil)).Elem().String()
			return casted, fmt.Errorf("argument is nil, but nil is not a usable value for type %s", typeName)
		}
		return casted, fmt.Errorf("requested type %T, but arg value is of type %T", *new(T), a)
	}
	return casted, nil
}

func canBeNil[T any]() bool {
	if isInterface[T]() {
		return false
	}

	dummy := reflect.TypeOf(*new(T))
	kind := dummy.Kind()
	return kind == reflect.Slice || kind == reflect.Map || kind == reflect.Chan
}

func isInterface[T any]() bool {
	return getInterfaceType[T]().Kind() == reflect.Interface
}

func isSettable[T any](value any) bool {
	if !isInterface[T]() && reflect.TypeOf(*new(T)) == reflect.TypeOf(value) {
		return true
	}

	if isInterface[T]() && value != nil {
		interfaceType := getInterfaceType[T]()
		return reflect.TypeOf(value).Implements(interfaceType)
	}

	return false
}

func getInterfaceType[T any]() reflect.Type {
	return reflect.TypeOf(new(T)).Elem()
}
