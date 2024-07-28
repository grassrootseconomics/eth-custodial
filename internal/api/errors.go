package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func newBadRequestError(message ...interface{}) *echo.HTTPError {
	return echo.NewHTTPError(http.StatusBadRequest, message...)
}

func (a *API) customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	if hErr, ok := err.(*echo.HTTPError); ok {
		var errorMsg string

		if m, ok := hErr.Message.(error); ok {
			errorMsg = m.Error()
		} else if m, ok := hErr.Message.(string); ok {
			errorMsg = m
		}

		c.JSON(hErr.Code, errResp{
			Ok:          false,
			Description: errorMsg,
		})
		return
	}

	a.logg.Error("api: echo error", "path", c.Path(), "err", err)
	c.JSON(http.StatusInternalServerError, errResp{
		Ok:          false,
		Description: "Internal server error.",
	})
}
