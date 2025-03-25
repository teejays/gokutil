package errutil

import (
	"fmt"
	"strings"

	"github.com/teejays/gokutil/log"
)

type MultiErr struct {
	errs []error
}

func NewMultiErr() MultiErr {
	return MultiErr{}
}

func (e MultiErr) Len() int {
	return len(e.errs)
}

func (e *MultiErr) IsNil() bool {
	return e == nil || len(e.errs) < 1
}
func (e *MultiErr) AddErr(err error) {
	e.errs = append(e.errs, err)
}
func (e *MultiErr) AddNew(msg string, args ...interface{}) {
	e.errs = append(e.errs, NewErrorTree("", fmt.Errorf(msg, args...)))
}

func (e *MultiErr) Wrap(err error, msg string, args ...interface{}) {
	if err == nil {
		log.WarnWithoutCtx("Wrapping a nil error")
	}
	e.errs = append(e.errs, NewErrorTree(fmt.Sprintf(msg, args...), err))
}

func (e MultiErr) Error() string {
	return e.ErrorIndent()
}

func (e MultiErr) ErrOrNil() error {
	if e.IsNil() {
		return nil
	}
	return e
}

var _stdIndentGap = " "

func (e MultiErr) ErrorIndent() string {

	rootIndent := ""
	childrenIndent := rootIndent + _stdIndentGap

	// Case: When MultiErr is empty?
	if e.IsNil() {
		return rootIndent + "nil"
	}

	rootMsg := fmt.Sprintf(rootIndent+"There are [%d] errors:", len(e.errs))
	if len(e.errs) == 1 {
		return rootIndent + e.errs[0].Error()
	}

	var childrenMsg string
	for i, err := range e.errs {

		// Children indentation
		childMsg := fmt.Sprintf("- [%d] %s", i+1, err.Error())
		// Add indentation to all lines other than the first line in the string
		lines := strings.Split(childMsg, "\n")
		for i, line := range lines {
			if i == 0 {
				continue
			}
			lines[i] = fmt.Sprintf("%s%s", childrenIndent, line)
		}
		childrenMsg += "\n" + childMsg
	}

	childrenMsg = strings.TrimSpace(childrenMsg)

	return fmt.Sprintf("%s\n%s", rootMsg, childrenMsg)
}
