package errutil

import (
	"fmt"

	"github.com/teejays/clog"
)

type MultiErr struct {
	errs []error
}

func NewMultiErr() MultiErr {
	return MultiErr{}
}

func (e *MultiErr) IsNil() bool {
	return e == nil || len(e.errs) < 1
}
func (e *MultiErr) Add(err error) {
	e.errs = append(e.errs, err)
}
func (e *MultiErr) AddNew(msg string, args ...interface{}) {
	e.errs = append(e.errs, fmt.Errorf(msg, args...))
}

func (e MultiErr) Error() string {
	return e.ErrorIndent(0)
}

func (e MultiErr) ErrorIndent(ind int) string {
	// Case: When MultiErr is empty?
	if len(e.errs) == 0 {
		return "nil"
	}
	// Case: When MultiErr is just a single error
	if len(e.errs) == 1 {
		return e.errs[0].Error()
	}

	str := fmt.Sprintf("There are %d errors:\n", len(e.errs))
	for i, err := range e.errs {
		// add indentation
		space := ""
		for j := 0; j < ind; j++ {
			space += "\t"
		}
		// clog.Debugf("Type of Error: %v", reflect.TypeOf(err))
		errMessage := ""
		if inErr, ok := err.(MultiErr); ok {
			clog.Debugf("Err is MultiErr")
			errMessage = inErr.ErrorIndent(ind + 1)
		} else {
			errMessage = err.Error()
		}
		str += fmt.Sprintf(" %s- [%d] %s", space, i+1, errMessage)
		if i < len(e.errs)-1 {
			str += "\n"
		}
	}
	return str
}
