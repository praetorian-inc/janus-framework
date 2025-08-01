package util

import (
	"fmt"
	"strconv"
	"strings"
)

var conversionFuncs = map[string]func(string) (any, error){
	"string":    convertString,
	"int":       convertInt,
	"float64":   convertFloat64,
	"bool":      convertBool,
	"[]string":  convertStringSlice,
	"[]int":     convertIntSlice,
	"[]float64": convertFloat64Slice,
	"[]bool":    convertBoolSlice,
}

func IsConvertable[T any]() bool {
	_, ok := conversionFuncs[fmt.Sprintf("%T", *new(T))]
	return ok
}

func convertString(value string) (any, error) {
	return value, nil
}

func convertInt(value string) (any, error) {
	return strconv.Atoi(value)
}

func convertFloat64(value string) (any, error) {
	return strconv.ParseFloat(value, 64)
}

func convertBool(value string) (any, error) {
	return strconv.ParseBool(value)
}

func convertStringSlice(value string) (any, error) {
	if value == "" {
		return []string{}, nil
	}
	return strings.Split(value, ","), nil
}

func convertIntSlice(value string) (any, error) {
	return convertSlice(value, strconv.Atoi)
}

func convertFloat64Slice(value string) (any, error) {
	return convertSlice(value, func(value string) (float64, error) {
		return strconv.ParseFloat(value, 64)
	})
}

func convertBoolSlice(value string) (any, error) {
	return convertSlice(value, strconv.ParseBool)
}

func convertSlice[T any](value string, converter func(string) (T, error)) (any, error) {
	slice := make([]T, 0)
	for i, v := range strings.Split(value, ",") {
		converted, err := converter(v)
		if err != nil {
			return nil, fmt.Errorf("error at index %d: %w", i, err)
		}
		slice = append(slice, converted)
	}
	return slice, nil
}

func ConvertPrimative(vType, value string) (any, error) {
	converter, ok := conversionFuncs[vType]
	if !ok {
		return nil, fmt.Errorf("no converter found for type %q", vType)
	}
	return converter(value)
}
