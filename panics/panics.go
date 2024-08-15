package panics

import (
	"fmt"
	"log"
	"reflect"
)

func P(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)

	log.Print("Panic: " + s)
	panic(s)
}

func If(b bool, msg string, args ...interface{}) {
	if b {
		P(msg, args...)
	}
}

func IfError(err error, msg string, args ...interface{}) {
	if err != nil {
		msg = msg + ": " + err.Error()
	}
	If(err != nil, msg, args...)
}

func IfNil(v interface{}, msg string, args ...interface{}) {
	isNil := v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil())
	If(isNil, msg, args...)
}

func Invariant(funcName string) {
	P("Invariant method called: %s", funcName)
}
