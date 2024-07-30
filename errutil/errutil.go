package errutil

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/teejays/goku-util/panics"
)

var ErrNotFound = fmt.Errorf("not found")
var ErrNothingToUpdate = fmt.Errorf("nothing to update")
var ErrNotAuthorized = fmt.Errorf("you are not authorized to perform this action")

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

var ErrBadCredentials = NewGerror(
	http.StatusUnauthorized,
	"Provided credentials did not match those on the system",
	"",
)

var ErrBadToken = GokuError{
	externalMsg:        "Provided authentication token cannot be verified",
	externalHTTPStatus: http.StatusUnauthorized,
}

type GokuError struct {
	internalError      error  // optional:
	externalMsg        string // optional: if left empty, the internal message will be used
	externalHTTPStatus int
}

func NewGerror(code int, externalMessage string, msg string, args ...interface{}) error {
	err := GokuError{
		internalError:      fmt.Errorf(msg, args...),
		externalMsg:        externalMessage,
		externalHTTPStatus: code,
	}
	if (err.internalError == nil || err.internalError.Error() == "") && err.externalMsg == "" && err.externalHTTPStatus == 0 {
		panics.P("New Goku Error created with missing data")
	}
	return err
}

func WrapGerror(err error, code int, externalMsg string) error {
	if err == nil {
		return nil
	}
	// Already Goku Error
	if gerr, ok := err.(GokuError); ok {
		gerr.externalMsg = externalMsg
		gerr.externalHTTPStatus = code
		return gerr
	}
	// Wrap in Goku Error
	return GokuError{
		internalError:      err,
		externalMsg:        externalMsg,
		externalHTTPStatus: code,
	}
}

func (err GokuError) Error() string {
	if err.internalError != nil {
		return err.internalError.Error()
	}
	return err.externalMsg
}

// GetHTTPStatus returns the status, if set, and defaults to InternalServerError
func (err GokuError) GetExternalMsg() string {
	if err.externalMsg != "" {
		return err.externalMsg
	}
	return err.internalError.Error()
}

// GetHTTPStatus returns the status, if set, and defaults to InternalServerError
func (err GokuError) GetHTTPStatus() int {
	if err.externalHTTPStatus > 0 {
		return err.externalHTTPStatus
	}
	return http.StatusInternalServerError
}

func AsGokuError(err error) (GokuError, bool) {
	var gerr GokuError

	var inErr error = err
	for {
		if inGerr, ok := inErr.(GokuError); ok {
			gerr = inGerr
			break
		}
		inErr = errors.Unwrap(inErr)
		if inErr == nil {
			break
		}
	}

	// If we found a Goku Error, and it has an external message, use it!
	if gerr.internalError != nil {
		return gerr, true
	}

	return GokuError{}, false
}
