//collection of errors
package gover

import "errors"

var (
	StartingPointError = errors.New("Starting point is not valid")
	TimeLocationError  = errors.New("Time location is not defined")
)
