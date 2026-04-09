package template

import "errors"

var (
	ErrNotFound      = errors.New("template not found")
	ErrInvalidRule   = errors.New("invalid recurrence rule")
	ErrInvalidInput  = errors.New("invalid template input")
)
