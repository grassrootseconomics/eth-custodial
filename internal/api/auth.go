package api

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grassrootseconomics/eth-custodial/pkg/api"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type jwtCustomClaims struct {
	PublicKey string `json:"publicKey"`
	Admin     bool   `json:"admin"`
	jwt.RegisteredClaims
}

// TODO: Choose edDSA Signing algo
const (
	authorizationHeader = "X-GE-KEY"

	testPublicKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEpl+G3km4UIHLgSe54RIl0EW/Z2ON
3VudoQCszl+yoTkTYp1GD5LK+0ZkqHFB2FuDTjaSiFsDo36FuVXvX5Hnug==
-----END PUBLIC KEY-----`
	testPrivateKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIJ+GMQ6MFC0KHL7izvCz0sR0CR2dAr6GgJCVTvPzNYaToAoGCCqGSM49
AwEHoUQDQgAEpl+G3km4UIHLgSe54RIl0EW/Z2ON3VudoQCszl+yoTkTYp1GD5LK
+0ZkqHFB2FuDTjaSiFsDo36FuVXvX5Hnug==
-----END EC PRIVATE KEY-----`
)

func (a *API) serviceAPIAuthConfig() middleware.KeyAuthConfig {
	return middleware.KeyAuthConfig{
		KeyLookup: "header:" + authorizationHeader,
		Validator: func(auth string, c echo.Context) (bool, error) {
			return auth == a.apiKey, nil
		},
		ErrorHandler: func(_ error, c echo.Context) error {
			return c.JSON(http.StatusUnauthorized, api.ErrResponse{
				Ok:          false,
				ErrCode:     api.ErrCodeInvalidAPIKey,
				Description: "Invalid API key",
			})
		},
	}
}

func (a *API) userAPIJWTAuthConfig() echojwt.Config {
	return echojwt.Config{
		TokenLookup:   "header:Authorization:Bearer,cookie:__ge_auth",
		SigningMethod: "ES256",
		SigningKey:    testPublicKey,
	}
}

func (a *API) testLogin(c echo.Context) error {
	claims := &jwtCustomClaims{
		"0x0000000000000000000000000000000000000000",
		true,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	t, err := token.SignedString([]byte(testPrivateKey))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"token": t,
	})
}

func (a *API) testRestircted(c echo.Context) error {
	return c.JSON(http.StatusOK, api.OKResponse{
		Ok:          true,
		Description: "You are seeing a restricted page!",
	})
}
