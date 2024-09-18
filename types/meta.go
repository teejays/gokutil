package types

import (
	"context"
	"database/sql/driver"
	"fmt"
	"slices"
	"time"

	"github.com/teejays/gokutil/naam"
	"github.com/teejays/gokutil/scalars"
)

/* * * * * *
 * With MetaFields
 * * * * * */

type IWithMetaFields interface {
	GetID() scalars.ID
	GetUpdatedAt() scalars.Timestamp
	SetUpdatedAt(scalars.Timestamp)
	SetDeletedAt(scalars.Timestamp)
}

type WithMetaFields struct {
	ID        scalars.ID
	CreatedAt scalars.Timestamp
	UpdatedAt scalars.Timestamp
	DeletedAt *scalars.Timestamp
}

func (t WithMetaFields) GetID() scalars.ID {
	return t.ID
}

func (t WithMetaFields) GetCreatedAt() scalars.Timestamp {
	return t.CreatedAt
}

func (t WithMetaFields) GetUpdatedAt() scalars.Timestamp {
	return t.UpdatedAt
}

func (t *WithMetaFields) SetUpdatedAt(v scalars.Timestamp) {
	t.UpdatedAt = v
}

func (t *WithMetaFields) SetDeletedAt(v scalars.Timestamp) {
	t.DeletedAt = &v
}

// type TypeWithMetaFields[T any] struct {
// 	WithMetaFields
// }

/* * * * * *
 * Hooks
 * * * * * */

type TypeHookFunc[T BasicType] func(context.Context, T) (T, error)

type IWithHooks[T BasicType] interface {
	// Setters
	SetHookAddBefore(TypeHookFunc[T]) error
	SetHookAddAfter(TypeHookFunc[T]) error
	SetHookUpdateBefore(TypeHookFunc[T]) error
	SetHookUpdateAfter(TypeHookFunc[T]) error
	SetHookSaveBefore(TypeHookFunc[T]) error
	SetHookSaveAfter(TypeHookFunc[T]) error
	SetHookDeleteBefore(TypeHookFunc[T]) error
	SetHookDeleteAfter(TypeHookFunc[T]) error
	SetHookFetchAfter(TypeHookFunc[T]) error

	// Getters
	GetHookAddBefore() TypeHookFunc[T]
	GetHookAddAfter() TypeHookFunc[T]
	GetHookUpdateBefore() TypeHookFunc[T]
	GetHookUpdateAfter() TypeHookFunc[T]
	GetHookSaveBefore() TypeHookFunc[T]
	GetHookSaveAfter() TypeHookFunc[T]
	GetHookDeleteBefore() TypeHookFunc[T]
	GetHookDeleteAfter() TypeHookFunc[T]
	GetHookFetchAfter() TypeHookFunc[T]
}

type WithHooks[T BasicType] struct {
	AddBefore    TypeHookFunc[T]
	AddAfter     TypeHookFunc[T]
	UpdateBefore TypeHookFunc[T]
	UpdateAfter  TypeHookFunc[T]
	SaveBefore   TypeHookFunc[T]
	SaveAfter    TypeHookFunc[T]
	DeleteBefore TypeHookFunc[T]
	DeleteAfter  TypeHookFunc[T]
	FetchAfter   TypeHookFunc[T]
}

func setHookHelper[T BasicType](fn TypeHookFunc[T], dst *TypeHookFunc[T]) error {
	// TODO: Figure this out and document better
	if dst == nil {
		return fmt.Errorf("Destination pointer is nil")
	}
	if *dst != nil {
		return fmt.Errorf("Hook already set")
	}
	*dst = fn
	return nil
}

func NewWithHooks[T BasicType]() IWithHooks[T] {
	return &WithHooks[T]{}
}

func (t *WithHooks[T]) SetHookAddBefore(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.AddBefore)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookAddAfter(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.AddAfter)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookUpdateBefore(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.UpdateBefore)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookUpdateAfter(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.UpdateAfter)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookSaveBefore(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.SaveBefore)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookSaveAfter(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.SaveAfter)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookDeleteBefore(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.DeleteBefore)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookDeleteAfter(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.DeleteAfter)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookFetchAfter(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.FetchAfter)
	if err != nil {
		return err
	}
	return nil
}

func (t WithHooks[T]) GetHookAddBefore() TypeHookFunc[T] {
	return t.AddBefore
}

func (t WithHooks[T]) GetHookAddAfter() TypeHookFunc[T] {
	return t.AddAfter
}

func (t WithHooks[T]) GetHookUpdateBefore() TypeHookFunc[T] {
	return t.UpdateBefore
}

func (t WithHooks[T]) GetHookUpdateAfter() TypeHookFunc[T] {
	return t.UpdateAfter
}

func (t WithHooks[T]) GetHookSaveBefore() TypeHookFunc[T] {
	return t.SaveBefore
}

func (t WithHooks[T]) GetHookSaveAfter() TypeHookFunc[T] {
	return t.SaveAfter
}

func (t WithHooks[T]) GetHookDeleteBefore() TypeHookFunc[T] {
	return t.DeleteBefore
}

func (t WithHooks[T]) GetHookDeleteAfter() TypeHookFunc[T] {
	return t.DeleteAfter
}

func (t WithHooks[T]) GetHookFetchAfter() TypeHookFunc[T] {
	return t.FetchAfter
}

/* * * * * *
 * Type Meta
 * * * * * */

type ITypeMeta[T BasicType, F Field] interface {
	ITypeCommonMeta[T, F]

	SetMetaFieldValues(context.Context, T, time.Time) T
	ConvertTimestampColumnsToUTC(T) T
	SetDefaultFieldValues(T) T
}

type ITypeCommonMeta[T BasicType, F Field] interface {
	GetTypeCommonMeta() *TypeCommonMeta[T, F]
	IWithHooks[T]
}

type TypeCommonMeta[T BasicType, F Field] struct {
	Name   naam.Name
	Fields []F
	WithHooks[T]
}

func NewTypeCommonMeta[T BasicType, F Field](name naam.Name, fields []F) ITypeCommonMeta[T, F] {
	return &TypeCommonMeta[T, F]{
		Name:   name,
		Fields: fields,
	}
}

func (t *TypeCommonMeta[T, F]) GetTypeCommonMeta() *TypeCommonMeta[T, F] {
	return t
}

/* * * * * *
 * Basic Type
 * * * * * */

type BasicType interface {
	GetID() scalars.ID
	GetUpdatedAt() scalars.Timestamp
	SetUpdatedAt(scalars.Timestamp)
}

/* * * * * *
 * Entity Type
 * * * * * */

type EntityType interface {
	GetID() scalars.ID
	GetUpdatedAt() scalars.Timestamp
	SetUpdatedAt(scalars.Timestamp)
}

/* * * * * *
 * Filter Type
 * * * * * */

type FilterType interface{}

/* * * * * *
 * Entity Meta
 * * * * * */

type IEntityMeta[T BasicType, F Field] interface {
	GetTypeMeta() ITypeMeta[T, F]
}

type EntityMeta[T BasicType, F Field] struct {
	TypeMeta ITypeMeta[T, F]
}

// NewEntityMeta exists to ensure that EntityMeta implements IEntityMeta
func NewEntityMeta[T BasicType, F Field](dbTableName naam.Name, typeMeta ITypeMeta[T, F]) IEntityMeta[T, F] {
	return EntityMeta[T, F]{
		TypeMeta: typeMeta,
	}
}

func (em EntityMeta[T, F]) GetTypeMeta() ITypeMeta[T, F] {
	return em.TypeMeta
}

/* * * * * *
 * Enum
 * * * * * */

type Enum interface {
	String() string
	Name() naam.Name

	Value() (driver.Value, error)
	Scan(src interface{}) error

	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error

	ImplementsGraphQLType(name string) bool
	UnmarshalGraphQL(input interface{}) error
}

/* * * * * *
 * Field
 * * * * * */

// Field is a generic constraint
// Can also consider Field interface and FieldConstraint separately if needed.
type Field interface {
	comparable
	String() string
	Name() naam.Name
}

// PruneFields syncs the list of fields with the list of allowed and excluded fields
func PruneFields[T Field](columns []T, includeFields []T, excludeFields []T) []T {
	var includeFilteredColumns []T

	// If include fields is provided, add those fields
	if len(includeFields) > 0 {
		for _, col := range columns {
			if slices.Contains(includeFields, col) {
				includeFilteredColumns = append(includeFilteredColumns, col)
			}
		}
	} else {
		// If no include fields provided, assume everything is halal
		includeFilteredColumns = columns
	}

	var excludedFilteredColumns []T

	// If exclude fields is provided, remove those fields
	for _, col := range includeFilteredColumns {
		if !IsFieldInFields(col, excludeFields) {
			excludedFilteredColumns = append(excludedFilteredColumns, col)
		}
	}

	return excludedFilteredColumns

}

func IsFieldInFields[T Field](column T, fields []T) bool {
	return slices.ContainsFunc(fields, func(f T) bool { return f.Name().Equal(column.Name()) })
}

func RemoveFieldFromFields[T Field](column T, fields []T) []T {
	for i, fld := range fields {
		if fld.Name().Equal(column.Name()) {
			fields = append(fields[:i], fields[i+1:]...)
		}
	}
	return fields
}
