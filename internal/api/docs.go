package api

import (
	"net/http"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/labstack/echo/v4"
)

func (a *API) docsHandler(c echo.Context) error {
	htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
		Theme:   scalar.ThemeSaturn,
		SpecURL: "./docs/swagger.json",
		CustomOptions: scalar.CustomOptions{
			PageTitle: "Simple API",
		},
		DarkMode: false,
	})
	if err != nil {
		return err
	}

	return c.HTML(http.StatusOK, htmlContent)
}
