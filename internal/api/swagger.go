package api

import (
	"net/http"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/labstack/echo/v4"
)

//	@title			ETH Custodial API
//	@version		2.0
//	@description	Interact with the Grassroots Economics Custodial API
//	@termsOfService	https://grassecon.org/pages/terms-and-conditions.html

//	@contact.name	API Support
//	@contact.url	https://grassecon.org/pages/contact-us
//	@contact.email	devops@grassecon.org

//	@license.name	AGPL-3.0
//	@license.url	https://www.gnu.org/licenses/agpl-3.0.en.html

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-GE-KEY
//	@description				Service API Key

//	@host		localhost:5003
//	@BasePath	/api/v2

func (a *API) docsHandler(c echo.Context) error {
	htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
		Layout:  scalar.LayoutModern,
		Theme:   scalar.ThemeSaturn,
		SpecURL: "./docs/swagger.json",
		CustomOptions: scalar.CustomOptions{
			PageTitle: "ETH Custodial API",
		},
		DarkMode: false,
	})
	if err != nil {
		return err
	}

	return c.HTML(http.StatusOK, htmlContent)
}
