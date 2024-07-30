package validate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teejays/gokutil/validate"
)

func TestStruct(t *testing.T) {

	type exampleStruct struct {
		ID     string `validate:"required"`
		FooStr string `validate:"required"`
		FooInt int    `validate:"gte=0"`
	}
	tests := []struct {
		name       string
		toValidate exampleStruct
		wantErr    bool
	}{
		{
			name: "valid",
			toValidate: exampleStruct{
				ID:     "123",
				FooStr: "foo",
				FooInt: 1,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			toValidate: exampleStruct{
				FooStr: "foo",
				FooInt: 1,
			},
			wantErr: true,
		},
		{
			name: "missing FooStr",
			toValidate: exampleStruct{
				ID:     "123",
				FooInt: 1,
			},
			wantErr: true,
		},
		{
			name: "invalid FooInt",
			toValidate: exampleStruct{
				ID:     "123",
				FooStr: "foo",
				FooInt: -1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.toValidate)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
