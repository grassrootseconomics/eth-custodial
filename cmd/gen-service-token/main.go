package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grassrootseconomics/eth-custodial/internal/api"
	"github.com/grassrootseconomics/eth-custodial/internal/util"
	"github.com/grassrootseconomics/ethutils"
	"github.com/knadh/koanf/v2"
)

const defaultJWTExpiry = 365 * 24 * time.Hour

var (
	build = "dev"

	confFlag    string
	subjectFlag string

	lo *slog.Logger
	ko *koanf.Koanf
)

func init() {
	flag.StringVar(&confFlag, "config", "config.toml", "Config file location")
	flag.StringVar(&subjectFlag, "service", "", "Service identifier")
	flag.Parse()

	lo = util.InitLogger()
	ko = util.InitConfig(lo, confFlag)
}

func main() {
	if subjectFlag == "" {
		lo.Error("service identifier is required")
		os.Exit(1)
	}

	claims := api.JWTCustomClaims{
		Service:   true,
		PublicKey: ethutils.ZeroAddress.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    fmt.Sprintf("eth-custodial-%s", build),
			Subject:   subjectFlag,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(defaultJWTExpiry)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	priv, pub, err := util.LoadSigningKey(ko.MustString("api.private_key"))
	if err != nil {
		lo.Error("could not load private key", "error", err)
		os.Exit(1)
	}

	t, err := token.SignedString(priv)
	if err != nil {
		lo.Error("could not sign token", "error", err)
	}

	tokenVerifier, err := jwt.ParseWithClaims(t, &api.JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	if err != nil {
		lo.Error("could not verify service token", "error", err)
		os.Exit(1)
	}

	if claims, ok := tokenVerifier.Claims.(*api.JWTCustomClaims); ok {
		lo.Debug("service token claims verified", "claims", claims)
	} else {
		lo.Error("could not verify service token claims", "ok", ok, "claims", tokenVerifier.Claims)
		os.Exit(1)
	}

	lo.Info("service token generated", "token", t)
}
