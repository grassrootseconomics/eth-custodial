package api

import (
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	ValidatorProvider *validator.Validate
}

// In production we don't expose detailed validation error messages.
func (v *Validator) Validate(i interface{}) error {
	// if err := v.ValidatorProvider.Struct(i); err != nil {
	// 	if _, ok := err.(validator.ValidationErrors); ok {
	// 		return newBadRequestError("Validation failed on one or more fields")
	// 	}
	// }
	// return nil
	return v.ValidatorProvider.Struct(i)
}
