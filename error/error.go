package error

import (
	"fmt"
)

type AppError struct {
	Code     string      `json:"code,omitempty"`
	Message  string      `json:"title,omitempty"`
	Data     interface{} `json:"-"`
	Internal bool
	Errors   []error
}

// Ensure error implements the error interface.
var _ Error = (*AppError)(nil)

func (e AppError) GetCode() string {
	return e.Code
}

func (e AppError) GetMessage() string {
	return e.Message
}

func (e AppError) GetData() interface{} {
	return e.Data
}

func (e AppError) IsInternal() bool {
	return e.Internal
}

func (e AppError) GetErrors() []error {
	return e.Errors
}

func (e AppError) AddError(err error) {
	e.Errors = append(e.Errors, err)
}

func (e AppError) Error() string {
	s := e.Code
	if e.Message != "" {
		s += ": " + e.Message
	}

	if e.Data != nil {
		s += "\n" + fmt.Sprintf("%+v", e.Data)
	}

	return s
}
