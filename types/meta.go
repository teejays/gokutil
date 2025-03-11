package types

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"slices"

	"github.com/teejays/gokutil/naam"
	"github.com/teejays/gokutil/scalars"
)

/* * * * * *
 * With MetaFields
 * * * * * */

// type IWithMetaFields interface {
// 	GetID() scalars.ID
// 	GetUpdatedAt() scalars.Timestamp
// 	SetUpdatedAt(scalars.Timestamp)
// 	SetDeletedAt(scalars.Timestamp)
// }

// type WithMetaFields struct {
// 	ID        scalars.ID
// 	CreatedAt scalars.Timestamp
// 	UpdatedAt scalars.Timestamp
// 	DeletedAt *scalars.Timestamp
// }

// func (t WithMetaFields) GetID() scalars.ID {
// 	return t.ID
// }

// func (t WithMetaFields) GetCreatedAt() scalars.Timestamp {
// 	return t.CreatedAt
// }

// func (t WithMetaFields) GetUpdatedAt() scalars.Timestamp {
// 	return t.UpdatedAt
// }

// func (t *WithMetaFields) SetUpdatedAt(v scalars.Timestamp) {
// 	t.UpdatedAt = v
// }

// func (t *WithMetaFields) SetDeletedAt(v scalars.Timestamp) {
// 	t.DeletedAt = &v
// }

// type TypeWithMetaFields[T any] struct {
// 	WithMetaFields
// }

/* * * * * *
 * Hooks
 * * * * * */

type HookType string

func (h HookType) Name() naam.Name { return naam.New(string(h)) }

func (h HookType) IsInit() bool       { return h == HookPoint_Init }
func (h HookType) IsCreatePre() bool  { return h == HookPoint_CreatePre }
func (h HookType) IsCreatePost() bool { return h == HookPoint_CreatePost }
func (h HookType) IsUpdatePre() bool  { return h == HookPoint_UpdatePre }
func (h HookType) IsUpdatePost() bool { return h == HookPoint_UpdatePost }
func (h HookType) IsSavePre() bool    { return h == HookPoint_SavePre }
func (h HookType) IsSavePost() bool   { return h == HookPoint_SavePost }
func (h HookType) IsDeletePre() bool  { return h == HookPoint_DeletePre }
func (h HookType) IsDeletePost() bool { return h == HookPoint_DeletePost }
func (h HookType) IsReadPre() bool    { return h == HookPoint_ReadPre }
func (h HookType) IsReadPost() bool   { return h == HookPoint_ReadPost }

const (
	HookPoint_Invalid    HookType = ""
	HookPoint_Init       HookType = "init"
	HookPoint_CreatePre  HookType = "create_pre"
	HookPoint_CreatePost HookType = "create_post"
	HookPoint_UpdatePre  HookType = "update_pre"
	HookPoint_UpdatePost HookType = "update_post"
	HookPoint_SavePre    HookType = "save_pre"
	HookPoint_SavePost   HookType = "save_post"
	HookPoint_DeletePre  HookType = "delete_pre"
	HookPoint_DeletePost HookType = "delete_post"
	HookPoint_ReadPre    HookType = "read_pre"
	HookPoint_ReadPost   HookType = "read_post"
)

type TypeHookFunc[T BasicType] func(context.Context, T) (T, error)

type IWithHooks[T BasicType] interface {
	// Setters
	SetHookInit(TypeHookFunc[T]) error
	SetHookCreatePre(TypeHookFunc[T]) error
	SetHookCreatePost(TypeHookFunc[T]) error
	SetHookUpdatePre(TypeHookFunc[T]) error
	SetHookUpdatePost(TypeHookFunc[T]) error
	SetHookSavePre(TypeHookFunc[T]) error
	SetHookSavePost(TypeHookFunc[T]) error
	SetHookDeletePre(TypeHookFunc[T]) error
	SetHookDeletePost(TypeHookFunc[T]) error
	SetHookReadPre(TypeHookFunc[T]) error
	SetHookReadPost(TypeHookFunc[T]) error

	// Getters
	GetHookInit() TypeHookFunc[T]
	GetHookCreatePre() TypeHookFunc[T]
	GetHookCreatePost() TypeHookFunc[T]
	GetHookUpdatePre() TypeHookFunc[T]
	GetHookUpdatePost() TypeHookFunc[T]
	GetHookSavePre() TypeHookFunc[T]
	GetHookSavePost() TypeHookFunc[T]
	GetHookDeletePre() TypeHookFunc[T]
	GetHookDeletePost() TypeHookFunc[T]
	GetHookReadPre() TypeHookFunc[T]
	GetHookReadPost() TypeHookFunc[T]
}

type WithHooks[T BasicType] struct {
	Init       TypeHookFunc[T]
	CreatePre  TypeHookFunc[T]
	CreatePost TypeHookFunc[T]
	UpdatePre  TypeHookFunc[T]
	UpdatePost TypeHookFunc[T]
	SavePre    TypeHookFunc[T]
	SavePost   TypeHookFunc[T]
	DeletePre  TypeHookFunc[T]
	DeletePost TypeHookFunc[T]
	ReadPre    TypeHookFunc[T]
	ReadPost   TypeHookFunc[T]
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

func (t *WithHooks[T]) SetHookInit(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.Init)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookCreatePre(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.CreatePre)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookCreatePost(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.CreatePost)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookUpdatePre(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.UpdatePre)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookUpdatePost(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.UpdatePost)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookSavePre(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.SavePre)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookSavePost(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.SavePost)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookDeletePre(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.DeletePre)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookDeletePost(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.DeletePost)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookReadPre(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.ReadPre)
	if err != nil {
		return err
	}
	return nil
}

func (t *WithHooks[T]) SetHookReadPost(fn TypeHookFunc[T]) error {
	err := setHookHelper(fn, &t.ReadPost)
	if err != nil {
		return err
	}
	return nil
}

func (t WithHooks[T]) GetHookInit() TypeHookFunc[T] {
	return t.Init
}

func (t WithHooks[T]) GetHookCreatePre() TypeHookFunc[T] {
	return t.CreatePre
}

func (t WithHooks[T]) GetHookCreatePost() TypeHookFunc[T] {
	return t.CreatePost
}

func (t WithHooks[T]) GetHookUpdatePre() TypeHookFunc[T] {
	return t.UpdatePre
}

func (t WithHooks[T]) GetHookUpdatePost() TypeHookFunc[T] {
	return t.UpdatePost
}

func (t WithHooks[T]) GetHookSavePre() TypeHookFunc[T] {
	return t.SavePre
}

func (t WithHooks[T]) GetHookSavePost() TypeHookFunc[T] {
	return t.SavePost
}

func (t WithHooks[T]) GetHookDeletePre() TypeHookFunc[T] {
	return t.DeletePre
}

func (t WithHooks[T]) GetHookDeletePost() TypeHookFunc[T] {
	return t.DeletePost
}

func (t WithHooks[T]) GetHookReadPre() TypeHookFunc[T] {
	return t.ReadPost
}

func (t WithHooks[T]) GetHookReadPost() TypeHookFunc[T] {
	return t.ReadPost
}

/* * * * * *
 * Type Meta
 * * * * * */

type ITypeMeta[T BasicType, F Field] interface {
	ITypeCommonMeta[T, F]

	SetMetaFieldValues(context.Context, T, scalars.Timestamp) T
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

// BasicType should really have been called BasicTypeWithMeta (since it has meta fields)
type BasicType interface {
	GetID() scalars.ID
	GetUpdatedAt() scalars.Timestamp
	GetDeletedAt() *scalars.Timestamp
}

// BasicTypeMutable is usually the pointer version of BasicType.
type BasicTypeMutable interface {
	BasicType
	SetUpdatedAt(scalars.Timestamp)
	SetDeletedAt(scalars.Timestamp)
}

/* * * * * *
 * Entity Type
 * * * * * */

type EntityType interface {
	BasicType
}

type EntityTypeMutable interface {
	BasicTypeMutable
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

/* * * * * *
 * Empty Struct
 * * * * * */

// EmptyStruct allows to have a common type for empty structs (so the same type can be passed around without conversions)
type EmptyStruct struct{}

// DerefInterface expects an interface with an underlying type that is a pointer, and returns the value that the pointer points to.
func DerefInterface(i interface{}) interface{} {
	v := reflect.ValueOf(i)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Interface()
}
