package cfg

import (
	"fmt"
	"regexp"

	"github.com/praetorian-inc/janus-framework/pkg/util"
)

type Param interface {
	Name() string
	Description() string
	Value() any
	HasValue() bool
	HasBeenSet() bool
	SetValue(any) (Param, error)
	Shortcode() string
	Flag() string
	Required() bool
	HasDefault() bool
	Type() string
	String() string
	Identifier() string
	Regex() *regexp.Regexp
	isValidForRegex(any) error
	isSettableTo(any) bool
	isValidForShortcode() error
	convertFromCLIString(string) (any, error)
}

type ParamImpl[T any] struct {
	name        string
	description string
	shortcode   string
	required    bool
	value       T
	hasValue    bool
	hasBeenSet  bool
	hasDefault  bool
	converter   func(string) (T, error)
	regex       *regexp.Regexp
}

func NewParam[T any](name string, description string) ParamImpl[T] {
	return ParamImpl[T]{
		name:        name,
		description: description,
	}
}

func (p ParamImpl[T]) Name() string {
	return p.name
}

func (p ParamImpl[T]) Description() string {
	return p.description
}

func (p ParamImpl[T]) Required() bool {
	return p.required
}

func (p ParamImpl[T]) HasDefault() bool {
	return p.hasDefault
}

func (p ParamImpl[T]) Value() any {
	return p.value
}

func (p ParamImpl[T]) HasValue() bool {
	return p.hasValue
}

func (p ParamImpl[T]) HasBeenSet() bool {
	return p.hasBeenSet
}

func (p ParamImpl[T]) Identifier() string {
	i := fmt.Sprintf("%s:%s:%s:%t", p.name, p.description, p.Type(), p.required)
	if p.regex != nil {
		i += fmt.Sprintf(":%s", p.regex.String())
	}
	return i
}

func (p ParamImpl[T]) SetValue(value any) (Param, error) {
	if valueString, ok := value.(string); ok {
		return p.setValueFromString(valueString)
	}

	return p.setValueDirectly(value)
}

func (p ParamImpl[T]) setValueFromString(value string) (Param, error) {
	stringArg, err := p.convertFromCLIString(value)
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %q to type %q: %w", value, p.Type(), err)
	}

	return p.setValueDirectly(stringArg)
}

func (p ParamImpl[T]) setValueDirectly(value any) (Param, error) {
	casted, ok := value.(T)
	if !ok {
		return nil, fmt.Errorf("parameter %q expects type %q, but argument value is type %q", p.Name(), p.Type(), fmt.Sprintf("%T", value))
	}
	p.value = casted
	p.hasValue = true
	p.hasBeenSet = true

	return p, nil
}

func (p ParamImpl[T]) Shortcode() string {
	return p.shortcode
}

func (p ParamImpl[T]) Flag() string {
	if p.shortcode == "" {
		return p.name
	}
	return p.shortcode
}

func (p ParamImpl[T]) isValidForShortcode() error {
	if p.shortcode == "" {
		return nil
	}

	ok := util.IsConvertable[T]()
	if p.converter != nil || ok {
		return nil
	}

	return fmt.Errorf("converter required to use shortcode for type %q", p.Type())
}

func (p ParamImpl[T]) convertFromCLIString(value string) (any, error) {
	var err error
	var converted any

	if p.converter != nil {
		converted, err = p.converter(value)
	} else {
		converted, err = util.ConvertPrimative(p.Type(), value)
	}

	if err != nil {
		return nil, err
	}

	return converted, nil
}

func (p ParamImpl[T]) Type() string {
	if isInterface[T]() {
		return getInterfaceType[T]().String()
	}
	return fmt.Sprintf("%T", *new(T))
}

func (p ParamImpl[T]) isSettableTo(value any) bool {
	return isSettable[T](value)
}

func (p ParamImpl[T]) isValidForRegex(value any) error {
	strValue, isString := value.(string)
	sliceValue, isSlice := value.([]string)
	if !isString && !isSlice || p.regex == nil {
		return nil
	}

	if isString {
		if !p.regex.MatchString(strValue) {
			return fmt.Errorf("value %q does not match regex %q", strValue, p.regex.String())
		}
	}

	for i, v := range sliceValue {
		if !p.regex.MatchString(v) {
			return fmt.Errorf("value %q at index %d does not match regex %q", v, i, p.regex.String())
		}
	}

	return nil
}

func (p ParamImpl[T]) String() string {
	str := fmt.Sprintf("%s: %s", p.name, p.description)
	if p.hasDefault {
		str += fmt.Sprintf(" (default: %s)", p.truncate(p.value))
	}
	if p.required {
		str += " (required)"
	}
	return str
}

func (p ParamImpl[T]) truncate(value any) string {
	str := fmt.Sprintf("%v", value)
	if len(str) > 50 {
		return str[:50] + " ...(truncated)"
	}
	return str
}

func (p ParamImpl[T]) Regex() *regexp.Regexp {
	return p.regex
}

func (p ParamImpl[T]) AsRequired() ParamImpl[T] {
	p.required = true
	return p
}

func (p ParamImpl[T]) WithDefault(value T) ParamImpl[T] {
	p.value = value
	p.hasDefault = true
	p.hasValue = true
	return p
}

func (p ParamImpl[T]) WithShortcode(shortcode string) ParamImpl[T] {
	p.shortcode = shortcode
	return p
}

func (p ParamImpl[T]) WithConverter(converter func(string) (T, error)) ParamImpl[T] {
	p.converter = converter
	return p
}

func (p ParamImpl[T]) WithRegex(regex *regexp.Regexp) ParamImpl[T] {
	p.regex = regex
	return p
}
