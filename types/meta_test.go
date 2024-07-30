package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teejays/goku-util/naam"
)

type TestField int

const (
	TestField_INVALID TestField = 0
	TestField_1       TestField = 1
	TestField_2       TestField = 2
	TestField_3       TestField = 3
)

func (f TestField) String() string {
	switch f {
	case TestField_INVALID:
		return "INVALID"
	case TestField_1:
		return "1"
	case TestField_2:
		return "2"
	case TestField_3:
		return "3"
	default:
		panic(fmt.Sprintf("'%d' is not a valid type '%s'", f, "TestField"))
	}
}

func (f TestField) Name() naam.Name {
	switch f {
	case TestField_INVALID:
		return naam.New("INVALID")
	case TestField_1:
		return naam.New("1")
	case TestField_2:
		return naam.New("2")
	case TestField_3:
		return naam.New("3")
	default:
		panic(fmt.Sprintf("'%d' is not a valid type '%s'", f, "TestField"))
	}
}

func TestPruneFields(t *testing.T) {
	tests := []struct {
		name          string
		columns       []TestField
		includeFields []TestField
		excludeFields []TestField
		want          []TestField
	}{
		{
			name:          "No fields provided",
			columns:       []TestField{},
			includeFields: []TestField{},
			excludeFields: []TestField{},
			want:          []TestField{},
		},
		{
			name:          "All cols with no include or exclude fields - all should be used",
			columns:       []TestField{TestField_1, TestField_2, TestField_3},
			includeFields: []TestField{},
			excludeFields: []TestField{},
			want:          []TestField{TestField_1, TestField_2, TestField_3},
		},
		{
			name:          "All cols with one include field and no exclude fields - one should be used",
			columns:       []TestField{TestField_1, TestField_2, TestField_3},
			includeFields: []TestField{TestField_2},
			excludeFields: []TestField{},
			want:          []TestField{TestField_2},
		},
		{
			name:          "All cols with no include field and one exclude fields - all but one should be used",
			columns:       []TestField{TestField_1, TestField_2, TestField_3},
			includeFields: []TestField{},
			excludeFields: []TestField{TestField_2},
			want:          []TestField{TestField_1, TestField_3},
		},
		{
			name:          "All cols with two include field and one exclude fields - two but one should be used",
			columns:       []TestField{TestField_1, TestField_2, TestField_3},
			includeFields: []TestField{TestField_1, TestField_2},
			excludeFields: []TestField{TestField_2},
			want:          []TestField{TestField_1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PruneFields(tt.columns, tt.includeFields, tt.excludeFields)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}
