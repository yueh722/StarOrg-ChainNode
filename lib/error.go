package lib

import (
	"fmt"
	"strconv"
	"strings"
)

type MultiError struct {
	Errors map[string]error
	Code   int
	seq    int
}

func NewMultiError() *MultiError {
	return &MultiError{Errors: make(map[string]error), Code: 1000}
}

func (m *MultiError) ErrorCode() int {
	return m.Code
}

func (m *MultiError) AddError(err error) {
	m.Errors[strconv.Itoa(m.seq)] = err
	m.seq++
}

func (m *MultiError) Clear() {
	m.Errors = nil
}

func (m *MultiError) ErrorOrNil() error {
	if len(m.Errors) > 0 {
		return m
	} else {
		return nil
	}
}

func (m *MultiError) Error() string {
	//errStr, err := json.MarshalIndent(m.Errors, "", "  ")

	errStr := []string{}
	for key, err := range m.Errors {
		errStr = append(errStr, key+": "+err.Error())
	}

	return fmt.Sprintf("{ %s }", strings.Join(errStr, ",\n"))
}
