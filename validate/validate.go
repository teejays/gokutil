package validate

import (
	"fmt"

	validator "github.com/go-playground/validator/v10"
	"github.com/teejays/gokutil/errutil"
)

// V is the singleton validator.Validate instance, it caches struct info
var V *validator.Validate

func init() {
	V = validator.New(validator.WithRequiredStructEnabled())
}

func Struct(data interface{}) error {
	err := V.Struct(data)
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *validator.InvalidValidationError:
		return fmt.Errorf("Failed to validate struct: %w", err)
	case validator.ValidationErrors:
		errs := errutil.NewMultiErr()
		for _, e := range err {
			errs.Wrap(e, "Validation Error")
		}
		return errs.ErrOrNil()
	default:
		return fmt.Errorf("Failed to validate struct: %w", err)
	}
}
