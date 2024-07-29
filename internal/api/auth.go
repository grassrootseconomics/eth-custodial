package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	authorizationHeader = "X-GE-KEY"
)

func (a *API) serviceAPIAuthConfig() middleware.KeyAuthConfig {
	return middleware.KeyAuthConfig{
		KeyLookup: "header:" + authorizationHeader,
		Validator: func(auth string, c echo.Context) (bool, error) {
			return auth == a.apiKey, nil
		},
		ErrorHandler: func(_ error, c echo.Context) error {
			return c.JSON(http.StatusUnauthorized, errResp{
				Ok:          false,
				Description: "Invalid API key",
			})
		},
	}
}
