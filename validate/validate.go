package validate

import (
	"reflect"

	validator "github.com/go-playground/validator/v10"
	"github.com/teejays/gokutil/errutil"
)

// V is the singleton validator.Validate instance, it caches struct info
var V *validator.Validate

func init() {
	V = validator.New(validator.WithRequiredStructEnabled())
}

func Struct(data interface{}) error {
	// If data is a struct or a pointer to a struct, or an interface with underlying struct, use V.Struct(data) otherwise pass
	if reflect.ValueOf(data).Kind() != reflect.Struct &&
		!(reflect.ValueOf(data).Kind() == reflect.Ptr && reflect.ValueOf(data).Elem().Kind() == reflect.Struct) &&
		!(reflect.ValueOf(data).Kind() == reflect.Interface && reflect.ValueOf(data).Elem().Kind() == reflect.Struct) {
		return nil
	}
	err := V.Struct(data)
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *validator.InvalidValidationError:
		return errutil.Wrap(err, "invalid data format")
	case validator.ValidationErrors:
		errs := errutil.NewMultiErr()
		for _, e := range err {
			errs.AddErr(e)
		}
		return errs.ErrOrNil()
	default:
		return err
	}
}
