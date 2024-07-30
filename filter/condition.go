package filter

import (
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/teejays/goku-util/scalars"
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

type StringCondition struct {
	Op     Operator
	Values []string
}

func NewStringCondition(op Operator, values ...string) *StringCondition {
	return &StringCondition{Op: op, Values: values}
}

func (f StringCondition) GetOperator() Operator      { return f.Op }
func (f StringCondition) Len() int                   { return len(f.Values) }
func (f StringCondition) GetValue(i int) interface{} { return f.Values[i] }

type IntCondition struct {
	Op     Operator
	Values []int
}

func NewIntCondition(op Operator, values ...int) *IntCondition {
	return &IntCondition{Op: op, Values: values}
}

func (f IntCondition) GetOperator() Operator      { return f.Op }
func (f IntCondition) Len() int                   { return len(f.Values) }
func (f IntCondition) GetValue(i int) interface{} { return f.Values[i] }

type FloatCondition struct {
	Op     Operator
	Values []float64
}

func NewFloatCondition(op Operator, values ...float64) *FloatCondition {
	return &FloatCondition{Op: op, Values: values}
}

func (f FloatCondition) GetOperator() Operator      { return f.Op }
func (f FloatCondition) Len() int                   { return len(f.Values) }
func (f FloatCondition) GetValue(i int) interface{} { return f.Values[i] }

type BoolCondition struct {
	Op     Operator
	Values []bool
}

func NewBoolCondition(op Operator, values ...bool) *BoolCondition {
	return &BoolCondition{Op: op, Values: values}
}

func (f BoolCondition) GetOperator() Operator      { return f.Op }
func (f BoolCondition) Len() int                   { return len(f.Values) }
func (f BoolCondition) GetValue(i int) interface{} { return f.Values[i] }

type UUIDCondition struct {
	Op     Operator
	Values []scalars.ID
}

func NewUUIDCondition(op Operator, values ...scalars.ID) *UUIDCondition {
	return &UUIDCondition{Op: op, Values: values}
}

func (f UUIDCondition) GetOperator() Operator      { return f.Op }
func (f UUIDCondition) Len() int                   { return len(f.Values) }
func (f UUIDCondition) GetValue(i int) interface{} { return f.Values[i] }

type TimestampCondition struct {
	Op     Operator
	Values []time.Time
}

func NewTimestampCondition(op Operator, values ...time.Time) *TimestampCondition {
	return &TimestampCondition{Op: op, Values: values}
}

func (f TimestampCondition) GetOperator() Operator      { return f.Op }
func (f TimestampCondition) Len() int                   { return len(f.Values) }
func (f TimestampCondition) GetValue(i int) interface{} { return f.Values[i] }
