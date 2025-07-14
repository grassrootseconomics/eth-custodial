package api

import (
	"context"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/grassrootseconomics/ethutils"
	"github.com/labstack/echo/v4"
	"github.com/lmittmann/w3/module/eth"
)

// ERC20Handler godoc
//
//	@Summary		ERC20 deploy request
//	@Description	ERC20 deploy request
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			transferRequest	body		apiresp.ERC20DeployRequest	true	"ERC20 deploy request"
//	@Success		200				{object}	apiresp.OKResponse
//	@Failure		400				{object}	apiresp.ErrResponse
//	@Failure		500				{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/contracts/erc20 [post]
func (a *API) contractsERC20Handler(c echo.Context) error {
	req := apiresp.ERC20DeployRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
	}

	exists, err := a.alreadyExists(c.Request().Context(), a.registry[ethutils.TokenIndex], req.Symbol)
	if err != nil {
		return err
	}
	if exists {
		return c.JSON(http.StatusBadRequest, apiresp.ErrResponse{
			Ok:          false,
			Description: "Token with this symbol already exists",
			ErrCode:     apiresp.ErrSymbolAlreadyExists,
		})
	}

	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	trackingID := uuid.NewString()

	_, err = a.queueClient.InsertTx(c.Request().Context(), tx, worker.TokenDeployArgs{
		TrackingID:    trackingID,
		Name:          req.Name,
		Symbol:        req.Symbol,
		Decimals:      req.Decimals,
		InitialSupply: req.InitialSupply,
		InitialMintee: req.InitialMintee,
		Owner:         req.Owner,
	}, nil)
	if err != nil {
		return handlePostgresError(c, err)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return handlePostgresError(c, err)
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "ERC20 deploy request successfully created",
		Result: map[string]any{
			"trackingId": trackingID,
		},
	})
}

// PoolHandler godoc
//
//	@Summary		Pool deploy request
//	@Description	Pool deploy request
//	@Tags			Contracts
//	@Accept			json
//	@Produce		json
//	@Param			poolRequest	body		apiresp.PoolDeployRequest	true	"Pool deploy request"
//	@Success		200			{object}	apiresp.OKResponse
//	@Failure		400			{object}	apiresp.ErrResponse
//	@Failure		500			{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/contracts/pool [post]
func (a *API) contractsPoolHandler(c echo.Context) error {
	req := apiresp.PoolDeployRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
	}

	exists, err := a.alreadyExists(c.Request().Context(), a.registry[ethutils.PoolIndex], req.Symbol)
	if err != nil {
		return err
	}
	if exists {
		return c.JSON(http.StatusBadRequest, apiresp.ErrResponse{
			Ok:          false,
			Description: "Token with this symbol already exists",
			ErrCode:     apiresp.ErrSymbolAlreadyExists,
		})
	}

	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	trackingID := uuid.NewString()

	_, err = a.queueClient.InsertTx(c.Request().Context(), tx, worker.PoolDeployArgs{
		TrackingID: trackingID,
		Name:       req.Name,
		Symbol:     req.Symbol,
		Owner:      req.Owner,
	}, nil)
	if err != nil {
		return handlePostgresError(c, err)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return handlePostgresError(c, err)
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Pool deploy request successfully created",
		Result: map[string]any{
			"trackingId": trackingID,
		},
	})
}

func (a *API) alreadyExists(ctx context.Context, index common.Address, tokenSymbol string) (bool, error) {
	var address common.Address

	if err := a.chainProvider.Client.CallCtx(
		ctx,
		eth.CallFunc(index, worker.Abi[worker.AddressOf], common.BytesToHash(common.RightPadBytes([]byte(tokenSymbol), 32))).Returns(&address),
	); err != nil {
		return false, err
	}

	return address != common.Address{}, nil
}
