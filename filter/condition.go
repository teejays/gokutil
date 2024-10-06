package filter

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/teejays/gokutil/scalars"
)

type Condition interface {
	GetOperator() Operator
	Len() int
	GetValue(int) interface{}
}

func IsConditionSet(f Condition) bool {
	return IsValidOperator(f.GetOperator())
}

func ValidateCondition(f Condition) error {
	var op Operator = f.GetOperator()
	var n int = f.Len()
	// Operator should be valid
	if op < 1 {
		return fmt.Errorf("invalid Operator")
	}
	// Get Info
	info, err := getOperatorInfo(op)
	if err != nil {
		return err
	}
	// Min values
	if !info.NoValues && n < 1 {
		return fmt.Errorf("Operator '%s' expects a value, none provided", info)
	}
	if !info.AllowMultipleValues && !info.NoValues && n != 1 {
		return fmt.Errorf("Operator '%s' expects one comparator value, %d provided", info, n)
	}
	if info.AllowMultipleValues && info.MaxNumValues > 0 && n > info.MaxNumValues {
		return fmt.Errorf("Operator '%s' expects a max of %d values, %d provided", info, info.MaxNumValues, n)
	}
	if info.AllowMultipleValues && info.MinNumValues > 0 && n < info.MinNumValues {
		return fmt.Errorf("Operator '%s' expects a min of %d values, %d provided", info, info.MinNumValues, n)
	}
	return nil
}

func ListConditionValues(f Condition) []interface{} {
	var r []interface{}
	for i := 0; i < f.Len(); i++ {
		r = append(r, f.GetValue(i))
	}
	return r
}

func InjectConditionIntoSqlBuilder(f Condition, sb *goqu.SelectDataset, col string) (*goqu.SelectDataset, error) {
	// Get operator
	op := f.GetOperator()
	info, err := getOperatorInfo(op)
	if err != nil {
		return sb, err
	}
	// Get Values
	values := ListConditionValues(f)
	sb = info.InjectSqlBuilderWhereCond_Goqu(sb, col, values...)
	return sb, nil
}

// Primitive Conditions
type GenericCondition[T comparable] struct {
	Op     Operator
	Values []T
}

func NewGenericCondition[T comparable](op Operator, values ...T) *GenericCondition[T] {
	return &GenericCondition[T]{Op: op, Values: values}
}

func (f GenericCondition[T]) GetOperator() Operator      { return f.Op }
func (f GenericCondition[T]) Len() int                   { return len(f.Values) }
func (f GenericCondition[T]) GetValue(i int) interface{} { return f.Values[i] }

type StringCondition = GenericCondition[string]

func NewStringCondition(op Operator, values ...string) *StringCondition {
	return NewGenericCondition(op, values...)
}

type NumberCondition = GenericCondition[int]

func NewNumberCondition(op Operator, values ...int) *NumberCondition {
	return NewGenericCondition(op, values...)
}

type FloatCondition = GenericCondition[float64]

func NewFloatCondition(op Operator, values ...float64) *FloatCondition {
	return NewGenericCondition(op, values...)
}

type BoolCondition = GenericCondition[bool]

func NewBoolCondition(op Operator, values ...bool) *BoolCondition {
	return NewGenericCondition(op, values...)
}

type IDCondition = GenericCondition[scalars.ID]

func NewIDCondition(op Operator, values ...scalars.ID) *IDCondition {
	return NewGenericCondition(op, values...)
}

type TimestampCondition = GenericCondition[scalars.Timestamp]

func NewTimestampCondition(op Operator, values ...scalars.Timestamp) *TimestampCondition {
	return NewGenericCondition(op, values...)
}

type DateCondition = GenericCondition[scalars.Date]

func NewDateCondition(op Operator, values ...scalars.Date) *DateCondition {
	return NewGenericCondition(op, values...)
}

type EmailCondition = GenericCondition[scalars.Email]

func NewEmailCondition(op Operator, values ...scalars.Email) *EmailCondition {
	return NewGenericCondition(op, values...)
}

type JSONCondition = GenericCondition[scalars.JSON]

func NewJSONCondition(op Operator, values ...scalars.JSON) *JSONCondition {
	return NewGenericCondition(op, values...)
}

func SecretCondition(op Operator, values ...string) *StringCondition {
	return NewStringCondition(op, values...)
}
