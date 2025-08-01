package chutil

import (
	"fmt"
	"reflect"
)

func ExtractChanInValue(link any) reflect.Value {
	return extractLinkField(link, "ChanIn")
}

func ExtractChanOutValue(link any) reflect.Value {
	return extractLinkField(link, "ChanOut")
}

func ExtractChanIn[T any](link any) chan T {
	val := ExtractChanInValue(link)
	return val.Interface().(chan T)
}

func ExtractChanOut[T any](link any) chan T {
	val := ExtractChanOutValue(link)
	return val.Interface().(chan T)
}

func extractLinkField(link any, fieldName string) reflect.Value {
	linkValue := reflect.ValueOf(link)
	linkElem := linkValue.Elem()
	baseValue := linkElem.FieldByName("Base")
	baseElem := baseValue.Elem()
	field := baseElem.FieldByName(fieldName)
	if !field.IsValid() {
		panic(fmt.Sprintf("link field %s is nil", fieldName))
	}
	return field
}
