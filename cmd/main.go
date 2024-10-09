package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/grassrootseconomics/eth-custodial/internal/api"
	"github.com/grassrootseconomics/eth-custodial/internal/gas"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/internal/sub"
	"github.com/grassrootseconomics/eth-custodial/internal/util"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	"github.com/grassrootseconomics/ethutils"
	"github.com/knadh/koanf/v2"
)

const defaultGracefulShutdownPeriod = time.Second * 20

var (
	build = "dev"

	confFlag             string
	migrationsFolderFlag string
	queriesFlag          string

	lo *slog.Logger
	ko *koanf.Koanf
)

func init() {
	flag.StringVar(&confFlag, "config", "config.toml", "Config file location")
	flag.StringVar(&migrationsFolderFlag, "migrations", "migrations/", "Migrations folder location")
	flag.StringVar(&queriesFlag, "queries", "queries.sql", "Queries file location")
	flag.Parse()

	lo = util.InitLogger()
	ko = util.InitConfig(lo, confFlag)

	lo.Info("starting eth custodial", "build", build)
}

func main() {
	var wg sync.WaitGroup
	ctx, stop := notifyShutdown()

	chainProvider := ethutils.NewProvider(ko.MustString("chain.rpc_endpoint"), ko.MustInt64("chain.id"))

	store, err := store.NewPgStore(store.PgOpts{
		Logg:                 lo,
		DSN:                  ko.MustString("postgres.dsn"),
		MigrationsFolderPath: migrationsFolderFlag,
		QueriesFolderPath:    queriesFlag,
	})
	if err != nil {
		lo.Error("could not initialize postgres store", "error", err)
		os.Exit(1)
	}
	if err := store.Bootstrap(); err != nil {
		lo.Error("store bootstrap actions failed", "error", err)
		os.Exit(1)
	}

	gasOracle, err := gas.New(gas.GasOpts{
		OracleType: ko.MustString("gas.oracle_type"),
	})
	if err != nil {
		lo.Error("could not initialize gas oracle", "error", err)
		os.Exit(1)
	}

	workerOpts := worker.WorkerOpts{
		MaxWorkers:                 ko.Int("workers.max"),
		ChainProvider:              chainProvider,
		CustodialRegistrationProxy: ko.MustString("chain.custodial_registration_proxy"),
		GasOracle:                  gasOracle,
		Store:                      store,
		Logg:                       lo,
	}
	if ko.Int("workers.max") <= 0 {
		workerOpts.MaxWorkers = runtime.NumCPU() * 2
	}
	workerContainer, err := worker.New(workerOpts)
	if err != nil {
		lo.Error("could not initialize worker container", "error", err)
		os.Exit(1)
	}

	jetStreamSub, err := sub.NewJetStreamSub(sub.JetStreamOpts{
		Logg:            lo,
		Store:           store,
		WorkerContainer: workerContainer,
		Endpoint:        ko.MustString("jetstream.endpoint"),
		JetStreamID:     ko.MustString("jetstream.id"),
	})
	if err != nil {
		lo.Error("could not initialize jetstream sub", "error", err)
		os.Exit(1)
	}

	apiServer := api.New(api.APIOpts{
		APIKey:        ko.MustString("api.key"),
		EnableMetrics: ko.Bool("metrics.enable"),
		EnableDocs:    ko.Bool("api.docs"),
		ListenAddress: ko.MustString("api.address"),
		Store:         store,
		ChainProvider: chainProvider,
		Worker:        workerContainer,
		Logg:          lo,
		Debug:         true,
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := apiServer.Start(); err != http.ErrServerClosed {
			lo.Error("failed to start HTTP server", "err", fmt.Sprintf("%T", err))
			os.Exit(1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		workerContainer.Start(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		jetStreamSub.Process()
	}()

	<-ctx.Done()
	lo.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultGracefulShutdownPeriod)

	wg.Add(1)
	go func() {
		defer wg.Done()
		jetStreamSub.Close()
		if err := apiServer.Stop(shutdownCtx); err != nil {
			lo.Error("failed to stop HTTP server", "err", fmt.Sprintf("%T", err))
		}
		workerContainer.Stop(shutdownCtx)
	}()

	go func() {
		wg.Wait()
		stop()
		cancel()
		os.Exit(0)
	}()

	<-shutdownCtx.Done()
	if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
		stop()
		cancel()
		lo.Error("graceful shutdown period exceeded, forcefully shutting down")
	}
	os.Exit(1)
}

func notifyShutdown() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
}
