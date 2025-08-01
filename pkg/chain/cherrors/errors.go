package cherrors

import (
	"errors"
	"fmt"
)

var ErrRateLimited = errors.New("rate limited")

type ProcessError struct {
	errorMsg string
}

func (e *ProcessError) Error() string {
	return e.errorMsg
}

func NewProcessErrorf(format string, a ...any) *ProcessError {
	return &ProcessError{errorMsg: fmt.Sprintf(format, a...)}
}

type ConversionError struct {
	errorMsg string
}

func (e *ConversionError) Error() string {
	return e.errorMsg
}

func NewConversionErrorf(format string, a ...any) *ConversionError {
	return &ConversionError{errorMsg: fmt.Sprintf(format, a...)}
}

type DebugError struct {
	errorMsg string
}

func (e *DebugError) Error() string {
	return e.errorMsg
}

func NewDebugErrorf(format string, a ...any) *DebugError {
	return &DebugError{errorMsg: fmt.Sprintf(format, a...)}
}
