package errutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/teejays/gokutil/log"
)

type ErrorTree struct {
	rootMsg string
	branch  error
}

func NewErrorTree(msg string, err error) ErrorTree {
	return ErrorTree{
		rootMsg: msg,
		branch:  err,
	}
}

func Wrap(err error, msg string, args ...interface{}) error {
	if err == nil {
		log.WarnWithoutCtx("Wrapping a nil error")
		return nil
	}
	return NewErrorTree(fmt.Sprintf(msg, args...), err)
}

func (e ErrorTree) Unwrap() error {
	return e.branch
}

func (e ErrorTree) Error() string {
	return e.ErrorIndent()
}

func (e ErrorTree) IsNil() bool {
	return e.rootMsg == "" && e.branch == nil
}

func (e ErrorTree) ErrorIndent() string {
	// Case: When MultiErr is empty?
	if e.IsNil() {
		return "nil"
	}

	// indentation
	rootIndent := ""
	childIndent := rootIndent + _stdIndentGap

	var rootMsg = e.rootMsg

	// Get any underlying message
	inErr := errors.Unwrap(e)
	if inErr == nil {
		return rootMsg
	}
	var childMsg = inErr.Error()

	// Add indentation to all lines in the string
	lines := strings.Split(childMsg, "\n")
	for i, line := range lines {
		lines[i] = fmt.Sprintf("%s%s", childIndent, line)
	}
	childMsg = strings.Join(lines, "\n")
	childMsg = strings.TrimSpace(childMsg)

	return rootMsg + "\n" + childMsg
}
