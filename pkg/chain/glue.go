package chain

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/praetorian-inc/janus-framework/pkg/chain/cherrors"
)

func Process(link Link, item any) error {
	err := CallForReceiver(link, "Process", item)
	return wrapError(fmt.Sprintf("failed to process item in link %q", link.Name()), err)
}

func Output(outputter Outputter, item any) error {
	err := CallForReceiver(outputter, "Output", item)
	return wrapError(fmt.Sprintf("failed to output item in outputter %q", outputter.Name()), err)
}

func wrapError(msg string, err error) error {
	var newErr error
	switch err.(type) {
	case *cherrors.ProcessError:
		newErr = cherrors.NewProcessErrorf("%s: %v", msg, err)
	case *cherrors.ConversionError:
		newErr = cherrors.NewConversionErrorf("%s: %v", msg, err)
	default:
		newErr = err
	}
	return newErr
}
func CallForReceiver(receiver any, methodName string, item any) error {
	receiverValue := reflect.ValueOf(receiver)
	method := receiverValue.MethodByName(methodName)
	converted, err := convertForReceiver(item, receiver, methodName)
	if err != nil {
		return err
	}
	results := method.Call([]reflect.Value{converted})
	if isErr(results) {
		return cherrors.NewProcessErrorf("process error: %v", results[0].Interface().(error))
	}
	return nil
}

func isErr(result []reflect.Value) bool {
	return len(result) > 0 && !result[0].IsNil()
}

func ConvertForLink(input any, link Link) (any, error) {
	value, err := convertForReceiver(input, link, "Process")
	if err != nil {
		return nil, err
	}
	return value.Interface(), nil
}

func ConvertForOutputter(input any, outputter Outputter) (any, error) {
	value, err := convertForReceiver(input, outputter, "Output")
	if err != nil {
		return nil, err
	}
	return value.Interface(), nil
}

func ConvertForJSON(input string, receiver any) (any, error) {
	value := reflect.ValueOf(receiver)
	method := value.MethodByName("Process")
	if !method.IsValid() {
		return reflect.Value{}, fmt.Errorf("receiver %T does not have method Process", receiver)
	}

	expectedType := method.Type().In(0)
	typeValue := reflect.New(expectedType)
	typeVariable := typeValue.Interface()

	err := json.Unmarshal([]byte(input), typeVariable)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON input: %v", err)
	}

	return typeVariable, nil
}

func convertForReceiver(input any, receiver any, methodName string) (reflect.Value, error) {
	value := reflect.ValueOf(receiver)
	method := value.MethodByName(methodName)
	if !method.IsValid() {
		return reflect.Value{}, fmt.Errorf("receiver %T does not have method %s", receiver, methodName)
	}
	expectedType := method.Type().In(0)

	return Convert(input, expectedType)
}

func Convert(input any, outputType reflect.Type) (reflect.Value, error) {
	inputValue := reflect.ValueOf(input)
	if !inputValue.IsValid() {
		return reflect.Value{}, cherrors.NewDebugErrorf("input is invalid: %v", input)
	}

	if typesMatch(outputType, inputValue.Type()) {
		return inputValue, nil
	}

	if adjusted, ok := adjustForInterface(outputType, inputValue); ok {
		return adjusted, nil
	} else if outputType.Kind() == reflect.Interface {
		return reflect.Value{}, cherrors.NewConversionErrorf("input %v does not implement receiver's interface %v", inputValue.Type(), outputType)
	}

	if !canCopyType(inputValue.Type()) {
		return reflect.Value{}, cherrors.NewConversionErrorf("input %q cannot be copied to output %q", inputValue.Type(), outputType)
	}

	if !canCopyType(outputType) {
		return reflect.Value{}, cherrors.NewConversionErrorf("output %q cannot be copied from input %q", outputType, inputValue.Type())
	}

	if inputValue.Kind() == reflect.Ptr {
		inputValue = inputValue.Elem()
	}

	outputPtr := reflect.New(outputType)
	if outputType.Kind() == reflect.Ptr {
		outputPtr = reflect.New(outputType.Elem())
	}

	err := CopyFields(inputValue, outputPtr)
	if err != nil {
		return reflect.Value{}, err
	}

	if outputType.Kind() != reflect.Ptr {
		outputPtr = outputPtr.Elem()
	}

	return outputPtr, nil
}

func typesMatch(typeA, typeB reflect.Type) bool {
	return typeA == typeB
}

func adjustForInterface(interfaceType reflect.Type, structValue reflect.Value) (reflect.Value, bool) {
	if interfaceType.Kind() != reflect.Interface {
		return reflect.Value{}, false
	}

	structType := structValue.Type()
	if structType.Implements(interfaceType) {
		return structValue, true
	}

	if structType.Kind() == reflect.Ptr && structType.Elem().Implements(interfaceType) {
		return structValue.Elem(), true
	}

	ptrToStructType := reflect.PointerTo(structType)
	if ptrToStructType.Implements(interfaceType) {
		ptrValue := reflect.New(structType)
		ptrValue.Elem().Set(structValue)

		return ptrValue, true
	}

	return reflect.Value{}, false
}

func canCopyType(t reflect.Type) bool {
	return t.Kind() == reflect.Struct || (t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct)
}

func CopyFields(source, destinationPtr reflect.Value) error {
	destination := destinationPtr.Elem()

	for i := 0; i < destination.NumField(); i++ {
		fieldName := destination.Type().Field(i).Name
		dstField := destination.Field(i)
		srcField := source.FieldByName(fieldName)

		if !dstField.CanSet() {
			continue // unexported field
		}

		if !srcField.IsValid() {
			return cherrors.NewConversionErrorf("source item %v does not have field %s of type %s", source.Type(), fieldName, dstField.Type())
		}

		dstField.Set(srcField)
	}
	return nil
}
