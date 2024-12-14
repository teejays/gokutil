package errutil

import (
	"database/sql"
	"fmt"
)

var ErrNotFound = fmt.Errorf("not found")
var ErrNothingToUpdate = fmt.Errorf("nothing to update")
var ErrNotAuthorized = fmt.Errorf("you are not authorized to perform this action")

func New(msg string, args ...interface{}) error {
	// using fmt since it supports wrapping
	return fmt.Errorf(msg, args...)
}
func Combine(errs ...error) error {
	return MultiErr{errs: removeNilErrs(errs)}
}

func removeNilErrs(errs []error) []error {
	var clean []error
	for _, err := range errs {
		if err != nil {
			clean = append(clean, err)
		}
	}
	return clean
}

func IsErrNoRows(err error) bool {
	return err == sql.ErrNoRows
}
