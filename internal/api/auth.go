package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	apiresp "github.com/grassrootseconomics/eth-custodial/pkg/api"
	"github.com/grassrootseconomics/ethutils"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

type JWTCustomClaims struct {
	PublicKey string `json:"publicKey"`
	Service   bool   `json:"service"`
	jwt.RegisteredClaims
}

const (
	cookieName   = "__ge_cust_auth"
	cookieDomain = "sarafu.network"

	defaultExpiryPeriod = 1 * 24 * time.Hour
)

func (a *API) apiJWTAuthConfig() echojwt.Config {
	return echojwt.Config{
		ErrorHandler: func(c echo.Context, err error) error {
			var reason = "An unknown JWT error occurred"

			if errors.Is(err, echojwt.ErrJWTMissing) {
				reason = "token missing from Authorization header or cookie"
			} else if errors.Is(err, echojwt.ErrJWTInvalid) {
				reason = "token is invalid or expired"
			} else {
				a.logg.Error("unknown jwt error caught", "error", err)
			}
			return handleJWTAuthError(c, reason)
		},
		BeforeFunc: func(c echo.Context) {
			a.logg.Info("header", "authorization", c.Request().Header.Get("Authorization"))
		},
		// Note that there is a space after Bearer to correctlty extract the token.
		TokenLookup:   "header:Authorization:Bearer ,cookie:__ge_cust_auth",
		SigningMethod: jwt.SigningMethodEdDSA.Alg(),
		SigningKey:    a.verifyingKey,
	}
}

func (a *API) loginHandler(c echo.Context) error {
	req := apiresp.LoginRequest{}

	if err := c.Bind(&req); err != nil {
		return handleBindError(c)
	}

	if err := c.Validate(req); err != nil {
		return handleValidateError(c)
	}

	issueDate := time.Now()
	expiryDate := issueDate.Add(defaultExpiryPeriod)

	claims := JWTCustomClaims{
		Service: false,
		// FetchPublicKey from auth db
		PublicKey: ethutils.ZeroAddress.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: fmt.Sprintf("eth-custodial-%s", a.build),
			// Use PublicKey here as well
			Subject:   ethutils.ZeroAddress.Hex(),
			IssuedAt:  jwt.NewNumericDate(issueDate),
			ExpiresAt: jwt.NewNumericDate(expiryDate),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	t, err := token.SignedString(a.signingKey)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    t,
		Secure:   true,
		Expires:  expiryDate,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Domain:   cookieDomain,
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Login successful",
		Result:      map[string]any{"token": t},
	})
}

func (a *API) logoutHandler(c echo.Context) error {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Secure:   true,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Domain:   cookieDomain,
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, apiresp.OKResponse{
		Ok:          true,
		Description: "Logout successful",
		Result:      nil,
	})
}
