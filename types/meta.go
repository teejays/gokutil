package types

import (
	"context"
	"database/sql/driver"
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
 * Type Meta
 * * * * * */

type ITypeMeta[T BasicType, F Field] interface {
	GetCommonMeta() TypeCommonMeta[T, F]
	SetMetaFieldValues(context.Context, T, time.Time) T

	ConvertTimestampColumnsToUTC(T) T
	SetDefaultFieldValues(T) T
}

type TypeCommonMeta[T BasicType, F Field] struct {
	Name   naam.Name
	Fields []F
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
