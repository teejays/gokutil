package ctxutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/teejays/goku-util/panics"
	"github.com/teejays/goku-util/scalars"
)

var ErrValueNotSet = fmt.Errorf("context value is not set")
var ErrUnexpectedValueType = fmt.Errorf("context value stored is of an unexpected type")

type ContextKey int

const (
	_ ContextKey = iota
	EnvironmentKey
	JWTTokenKey
	UserIDKey
)

type environment int

const (
	_ environment = iota
	DEV
	PROD
)

type UserIDType = scalars.ID
type ContextValueType interface {
	UserIDType | environment | string
}

func setValue[T ContextValueType](ctx context.Context, key ContextKey, val T) context.Context {
	return context.WithValue(ctx, key, val)
}

func getValue[T ContextValueType](ctx context.Context, key ContextKey) (T, error) {
	valI := ctx.Value(key)
	if valI == nil {
		var empty T
		return empty, ErrValueNotSet
	}
	val := valI.(T) // panic here is okay!
	return val, nil
}

func SetEnv(ctx context.Context, val environment) (context.Context, error) {
	return setValue(ctx, EnvironmentKey, val), nil
}

func GetEnv(ctx context.Context) environment {
	val, err := getValue[environment](ctx, EnvironmentKey)
	if errors.Is(err, ErrValueNotSet) {
		return DEV // Default to DEV
	}
	panics.IfError(err, "context.GetEnv()")
	return val
}

func SetUserID(ctx context.Context, val UserIDType) context.Context {
	panics.If(val.IsEmpty(), "UserID is nil: cannot set nil UserID in context")
	return setValue(ctx, UserIDKey, val)
}

func GetUserID(ctx context.Context) (UserIDType, error) {
	val, err := getValue[UserIDType](ctx, UserIDKey)
	if err != nil {
		return val, err
	}
	if val.IsEmpty() {
		return val, fmt.Errorf("UserID stored in context is nil")
	}
	return val, nil
}

func SetJWTToken(ctx context.Context, token string) context.Context {
	return setValue(ctx, JWTTokenKey, token)
}

func GetJWTToken(ctx context.Context) (string, error) {
	return getValue[string](ctx, JWTTokenKey)
}
