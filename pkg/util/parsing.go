package util

import (
	"strconv"
	"strings"
)

func ParsePrimative[T string | []string | int | []int | bool](value string) (T, error) {
	var dummy T
	var result any
	var err error

	switch any(dummy).(type) {
	case string:
		result, err = value, nil
	case []string:
		result, err = strings.Split(value, ","), nil
	case int:
		result, err = strconv.Atoi(value)
	case []int:
		result = []int{}
		for _, v := range strings.Split(value, ",") {
			i, err := strconv.Atoi(v)
			if err != nil {
				return result.(T), err
			}
			result = append(result.([]int), i)
		}
	case bool:
		result, err = strconv.ParseBool(value)
	}

	return result.(T), err
}

func MarshalPrimative[T string | []string | int | []int | bool](value any) (string, error) {
	var result string
	var err error

	switch any(value).(type) {
	case string:
		result = value.(string)
	case []string:
		result = strings.Join(value.([]string), ",")
	case int:
		result = strconv.Itoa(value.(int))
	case []int:
		result = strings.Join(value.([]string), ",")
	case bool:
		result = strconv.FormatBool(value.(bool))
	}

	return result, err
}
