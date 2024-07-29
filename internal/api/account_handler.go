package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/grassrootseconomics/celo-custodial/internal/keypair"
	"github.com/labstack/echo/v4"
)

func (a *API) accountCreateHandler(c echo.Context) error {
	generatedKeyPair, err := keypair.GenerateKeyPair()
	if err != nil {
		return err
	}

	trackingId := uuid.NewString()

	return c.JSON(http.StatusOK, okResp{
		Ok:          true,
		Description: "Account creation request successfully created",
		Result: H{
			"publicKey":  generatedKeyPair.Public,
			"trackingId": trackingId,
		},
	})
}
