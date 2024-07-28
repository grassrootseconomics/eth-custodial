package api

import "github.com/go-playground/validator/v10"

type Validator struct {
	ValidatorProvider *validator.Validate
}

func (v *Validator) Validate(i interface{}) error {
	if err := v.ValidatorProvider.Struct(i); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			return newBadRequestError(err.(validator.ValidationErrors).Error())
		}
	}
	return nil
}
