package filter

import (
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/teejays/gokutil/log"
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
	switch info.ValuesType {
	case ValuesType_Zero:
		if n > 1 {
			return fmt.Errorf("operator [%s] does not expect a value but [%d] provided", info, n)
		}
	case ValuesType_One:
		if n != 1 {
			return fmt.Errorf("operator [%s] expects one value but [%d] provided", info, n)
		}
	case ValuesType_Multiple:
		if info.MultipleValuesMin > 0 && n < info.MultipleValuesMin {
			return fmt.Errorf("operator [%s] expects a min of [%d] values but [%d] provided", info, info.MultipleValuesMin, n)
		}
		if info.MultipleValuesMax > 0 && n > info.MultipleValuesMax {
			return fmt.Errorf("operator [%s] expects a max of [%d] values but [%d] provided", info, info.MultipleValuesMax, n)
		}
	case ValuesType_Free:
		// Do nothing
	default:
		// Do nothing
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

// type QueryWhereInjectionRequest struct {
// 	Condition                  Condition
// 	Column                     string
// 	ExistingSelectQueryBuilder *goqu.SelectDataset
// 	IsColumnArray              bool
// }

func InjectConditionIntoSqlBuilder(f Condition, sb *goqu.SelectDataset, col string, isColArray bool) (*goqu.SelectDataset, error) {
	// Get operator
	op := f.GetOperator()
	info, err := getOperatorInfo(op)
	if err != nil {
		return nil, err
	}
	// Get Values
	values := ListConditionValues(f)
	log.DebugWithoutCtx("Injecting condition into SQL Builder", "Operator", info, "Column", col, "Values", values, "IsColumnArray", isColArray)

	// Validate some stuff
	err = ValidateCondition(f)
	if err != nil {
		return nil, err
	}

	// Handle when column is a SQL array
	if isColArray {
		sb := sb.Where(
			goqu.L(
				GetRawSQLConditionForArrayColumn(info, col), values[0],
			),
		)
		return sb, nil
	}
	sb = info.InjectSqlBuilderWhereCond_Goqu(sb, col, values...)
	return sb, nil
}

// func GetRawSQLConditionForArrayColumn(f Condition, col string) (string, error) {
// 	// Get operator
// 	op := f.GetOperator()
// 	info, err := getOperatorInfo(op)
// 	if err != nil {
// 		return "", err
// 	}
// 	// Get Values
// 	values := ListConditionValues(f)
// 	log.DebugWithoutCtx("Getting raw SQL condition for array column", "Operator", info, "Column", col, "Values", values)

// 	return info.GetSqlSign(), nil
// }

// Primitive Conditions
type GenericCondition[T comparable] struct {
	Op     Operator `json:"op" yaml:"op"`
	Values []T      `json:"values" yaml:"values"`
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

var False = NewBoolCondition(EQUAL, false)
var True = NewBoolCondition(EQUAL, true)

type IDCondition = GenericCondition[scalars.ID]

func NewIDCondition(op Operator, values ...scalars.ID) *IDCondition {
	return NewGenericCondition(op, values...)
}

type TimestampCondition = GenericCondition[scalars.Timestamp]

func NewTimestampCondition(op Operator, values ...scalars.Timestamp) *TimestampCondition {
	return NewGenericCondition(op, values...)
}

var NullTimestamp = NewTimestampCondition(IS_NULL)

type DateCondition = GenericCondition[scalars.Date]

func NewDateCondition(op Operator, values ...scalars.Date) *DateCondition {
	return NewGenericCondition(op, values...)
}

type EmailCondition = GenericCondition[scalars.Email]

func NewEmailCondition(op Operator, values ...scalars.Email) *EmailCondition {
	return NewGenericCondition(op, values...)
}

type LinkCondition = GenericCondition[scalars.Link]

func NewLinkCondition(op Operator, values ...scalars.Link) *LinkCondition {
	return NewGenericCondition(op, values...)
}

type GenericDataCondition = GenericCondition[scalars.GenericData]

func NewGenericDataCondition(op Operator, values ...scalars.GenericData) *GenericDataCondition {
	return NewGenericCondition(op, values...)
}

func SecretCondition(op Operator, values ...string) *StringCondition {
	return NewStringCondition(op, values...)
}
