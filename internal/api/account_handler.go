package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/keypair"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	"github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/labstack/echo/v4"
)

func (a *API) accountCreateHandler(c echo.Context) error {
	generatedKeyPair, err := keypair.GenerateKeyPair()
	if err != nil {
		return err
	}

	trackingID := uuid.NewString()

	_, err = a.worker.QueueClient.Insert(c.Request().Context(), worker.AccountCreateArgs{
		TrackingID: trackingID,
		KeyPair:    generatedKeyPair,
	}, nil)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, api.OKResponse{
		Ok:          true,
		Description: "Account creation request successfully created",
		Result: map[string]any{
			"publicKey":  generatedKeyPair.Public,
			"trackingId": trackingID,
		},
	})
}
