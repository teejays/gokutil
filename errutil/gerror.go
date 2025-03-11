package errutil

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/teejays/gokutil/panics"
)

var ErrBadCredentials = NewGerror("Provided credentials did not match those on the system").
	SetHTTPStatus(http.StatusUnauthorized)

var ErrBadToken = NewGerror("Provided authentication token cannot be verified").
	SetHTTPStatus(http.StatusUnauthorized)

type IGerror interface {
	error
	SetExternalMsg(msg string, args ...interface{}) GokuError
	SetHTTPStatus(code int) GokuError
	GetExternalMsg() string
	GetHTTPStatus() int
}

type GokuError struct {
	internalError      error  // optional:
	externalMsg        string // optional: if left empty, the internal message will be used
	externalHTTPStatus int
}

// NewGerror creates a new GokuError with the given code and message.
// If an external message needs to be set, use SetExternalMsg method.
// If an external HTTP status needs to be set, use SetHTTPStatus method.
func NewGerror(msg string, args ...interface{}) IGerror {
	err := GokuError{
		internalError: fmt.Errorf(msg, args...),
		// externalMsg:        "",
		// externalHTTPStatus: code,
	}
	if (err.internalError == nil || err.internalError.Error() == "") && err.externalMsg == "" && err.externalHTTPStatus == 0 {
		panics.P("New Goku Error created with missing data")
	}
	return err
}

func WrapGerror(err error) IGerror {
	if err == nil {
		return nil
	}
	// Already Goku Error
	if gerr, ok := err.(GokuError); ok {
		return gerr
	}
	// Wrap in Goku Error
	return GokuError{
		internalError: err,
	}
}

func WrapGerrorf(err error, msg string, args ...interface{}) IGerror {
	inerr := Wrap(err, msg, args...)
	return WrapGerror(inerr)
}

func WrapBadRequest(err error) error {
	return WrapGerror(err).SetHTTPStatus(http.StatusBadRequest)
}

func WrapInternalError(err error) error {
	return WrapGerror(err).SetHTTPStatus(http.StatusInternalServerError).SetExternalMsg("We're sorry, something went wrong in our system. We've alerted our team and will be looking into this.")
}

func (err GokuError) SetExternalMsg(msg string, args ...interface{}) GokuError {
	err.externalMsg = fmt.Sprintf(msg, args...)
	return err
}

func (err GokuError) SetHTTPStatus(code int) GokuError {
	err.externalHTTPStatus = code
	return err
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
