//collection of errors
package gover

import "errors"

var (
	StartingPointError = errors.New("Starting point is not valid")
	TimeLocationError  = errors.New("Time location is not defined")
	KeyNotFoundError   = errors.New("Unable to locate this key")
	DuplicateKeyError  = errors.New("This key is identified as duplicate")
	InterfaceTypeError = errors.New("Invalid type interface")
)
