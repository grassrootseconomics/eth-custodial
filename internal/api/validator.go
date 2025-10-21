package api

import (
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	ValidatorProvider *validator.Validate
}

var pretiumAllowedTokens = map[string]struct{}{
	"0x765DE816845861e75A25fCA122bb6898B8B1282a": {},
	"0x48065fbBE25f71C9282ddf5e1cD6D6A887483D5e": {},
	"0xcebA9300f2b948710d2653dD7B07f33A8B32118C": {},
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

func (a *API) isBannedToken(tokenAddress string) bool {
	_, exists := a.bannedTokens[tokenAddress]
	return exists
}

// This is a temporary stopgap to prevent people from accddentally sending their normal vouchers to pretium
// true stops the transfer, false allows it
func (a *API) stopPretiumLeak(toAddress string, tokenAddress string) bool {
	if toAddress != a.pretiumAddress {
		return false
	}

	_, allowed := pretiumAllowedTokens[tokenAddress]
	return !allowed
}
