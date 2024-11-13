package ctxutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/teejays/gokutil/env"
	"github.com/teejays/gokutil/panics"
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

// type UserIDType = scalars.ID
type ContextValueType = any

type ContextKeyType interface {
	ContextKey | string
}

func setValue[K ContextKeyType, T ContextValueType](ctx context.Context, key K, val T) context.Context {
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

func SetEnv(ctx context.Context, val env.Environment) (context.Context, error) {
	return setValue(ctx, EnvironmentKey, val), nil
}

func GetEnv(ctx context.Context) env.Environment {
	val, err := getValue[env.Environment](ctx, EnvironmentKey)
	if errors.Is(err, ErrValueNotSet) {
		return env.DEV // Default to DEV
	}
	panics.IfError(err, "context.GetEnv()")
	return val
}

func SetValue[V any, K ContextKeyType](ctx context.Context, key K, val V) context.Context {
	return context.WithValue(ctx, key, val)
}

func GetValue[V any, K ContextKeyType](ctx context.Context, key K) (V, error) {
	var empty V

	if ctx == nil {
		return empty, ErrValueNotSet
	}
	valI := ctx.Value(key)
	if valI == nil {
		return empty, ErrValueNotSet
	}
	val, ok := valI.(V)
	if !ok {
		return empty, ErrUnexpectedValueType
	}
	return val, nil
}
