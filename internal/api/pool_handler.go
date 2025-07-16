package api

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/labstack/echo/v4"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

var getQuoteFunc = w3.MustNewFunc("getQuote(address,address,uint256)", "uint256")

// poolSwapHandler godoc
//
//	@Summary		Pool swap request
//	@Description	Pool swap request
//	@Tags			Sign
//	@Accept			json
//	@Produce		json
//	@Param			poolSwapRequest	body		apiresp.PoolSwapRequest	true	"Pool swap request"
//	@Success		200				{object}	apiresp.OKResponse
//	@Failure		400				{object}	apiresp.ErrResponse
//	@Failure		500				{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/pool/swap [post]
func (a *API) poolSwapHandler(c echo.Context) error {
	req := apiresp.PoolSwapRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
	}

	if a.isBannedToken(req.FromTokenAddress) || a.isBannedToken(req.ToTokenAddress) {
		return c.JSON(http.StatusForbidden, apiresp.ErrResponse{
			Ok:          false,
			Description: fmt.Sprintf("Not allowed to interact with token"),
			ErrCode:     apiresp.ErrBannedToken,
		})
	}

	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return handlePostgresError(c, err)
	}
	defer tx.Rollback(c.Request().Context())

	exists, err := a.store.CheckKeypair(c.Request().Context(), tx, req.From)
	if err != nil {
		return handlePostgresError(c, err)
	}
	if !exists {
		return c.JSON(http.StatusNotFound, apiresp.ErrResponse{
			Ok:          false,
			Description: fmt.Sprintf("Account %s does not exist or is not yet activated", req.From),
			ErrCode:     apiresp.ErrCodeAccountNotExists,
		})
	}

	trackingID := uuid.NewString()

	_, err = a.queueClient.InsertTx(c.Request().Context(), tx, worker.PoolSwapArgs{
		TrackingID:       trackingID,
		From:             req.From,
		FromTokenAddress: req.FromTokenAddress,
		ToTokenAddress:   req.ToTokenAddress,
		PoolAddress:      req.PoolAddress,
		Amount:           req.Amount,
	}, nil)
	if err != nil {
		return handlePostgresError(c, err)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return handlePostgresError(c, err)
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Pool swap request successfully created",
		Result: map[string]any{
			"trackingId": trackingID,
		},
	})
}

// poolDepositHandler godoc
//
//	@Summary		Pool deposit request
//	@Description	Pool deposit request
//	@Tags			Sign
//	@Accept			json
//	@Produce		json
//	@Param			poolDepositRequest	body		apiresp.PoolDepositRequest	true	"Pool deposit request"
//	@Success		200					{object}	apiresp.OKResponse
//	@Failure		400					{object}	apiresp.ErrResponse
//	@Failure		500					{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/pool/deposit [post]
func (a *API) poolDepositHandler(c echo.Context) error {
	req := apiresp.PoolDepositRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
	}

	if a.isBannedToken(req.TokenAddress) {
		return c.JSON(http.StatusForbidden, apiresp.ErrResponse{
			Ok:          false,
			Description: fmt.Sprintf("Not allowed to interact with token"),
			ErrCode:     apiresp.ErrBannedToken,
		})
	}

	tx, err := a.store.Pool().Begin(c.Request().Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(c.Request().Context())

	exists, err := a.store.CheckKeypair(c.Request().Context(), tx, req.From)
	if err != nil {
		return handlePostgresError(c, err)
	}
	if !exists {
		return c.JSON(http.StatusNotFound, apiresp.ErrResponse{
			Ok:          false,
			Description: fmt.Sprintf("Account %s does not exist or is not yet activated", req.From),
			ErrCode:     apiresp.ErrCodeAccountNotExists,
		})
	}

	trackingID := uuid.NewString()

	_, err = a.queueClient.InsertTx(c.Request().Context(), tx, worker.PoolDepositArgs{
		TrackingID:   trackingID,
		From:         req.From,
		TokenAddress: req.TokenAddress,
		PoolAddress:  req.PoolAddress,
		Amount:       req.Amount,
	}, nil)
	if err != nil {
		return handlePostgresError(c, err)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return handlePostgresError(c, err)
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Pool deposit request successfully created",
		Result: map[string]any{
			"trackingId": trackingID,
		},
	})
}

// TODO: Add fees multiplier to return true final quote value

// poolQuoteHandler godoc
//
//	@Summary		Get a pool swap quote
//	@Description	Get a pool swap quote
//	@Tags			Sign
//	@Accept			json
//	@Produce		json
//	@Param			poolSwapRequest	body		apiresp.PoolSwapRequest	true	"Get a pool swap quote"
//	@Success		200				{object}	apiresp.OKResponse
//	@Failure		400				{object}	apiresp.ErrResponse
//	@Failure		500				{object}	apiresp.ErrResponse
//	@Security		ApiKeyAuth
//	@Router			/pool/quote [post]
func (a *API) poolQuoteHandler(c echo.Context) error {
	req := apiresp.PoolSwapRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
	}

	amount, err := worker.StringToBigInt(req.Amount, false)
	if err != nil {
		return err
	}

	var outValue *big.Int

	if err := a.chainProvider.Client.CallCtx(
		c.Request().Context(),
		eth.CallFunc(
			common.HexToAddress(req.PoolAddress),
			getQuoteFunc,
			common.HexToAddress(req.ToTokenAddress),
			common.HexToAddress(req.FromTokenAddress),
			amount,
		).Returns(&outValue),
	); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Pool swap quote successfully obtained",
		Result: map[string]any{
			"outValue":              outValue.String(),
			"includesFeesDeduction": false,
		},
	})
}
