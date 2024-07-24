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

	"github.com/grassrootseconomics/celo-custodial/internal/api"
	"github.com/grassrootseconomics/celo-custodial/internal/gas"
	"github.com/grassrootseconomics/celo-custodial/internal/queue"
	"github.com/grassrootseconomics/celo-custodial/internal/store"
	"github.com/grassrootseconomics/celo-custodial/internal/worker"
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

	lo = initLogger()
	ko = initConfig()

	lo.Info("starting celo indexer", "build", build)
}

func main() {
	/*

		+ store
		+ gas
		+ worker
		+ queue
	*/
	var wg sync.WaitGroup
	ctx, stop := notifyShutdown()

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

	gasOracle, err := gas.New(gas.GasOpts{
		OracleType: ko.MustString("gas.oracle_type"),
	})
	if err != nil {
		lo.Error("could not initialize gas oracle", "error", err)
		os.Exit(1)
	}

	workers, err := worker.New(worker.WorkerOpts{
		ChainID:   ko.MustInt64("chain.id"),
		GasOracle: gasOracle,
		Store:     store,
		Logg:      lo,
	})
	if err != nil {
		lo.Error("could not initialize signer workers", "error", err)
		os.Exit(1)
	}

	queueOpts := queue.QueueOpts{
		MaxWorkers: ko.MustInt("workers.max"),
		Logg:       lo,
		PgxPool:    store.Pool(),
		Workers:    workers,
	}
	if ko.Int("workers.max") <= 0 {
		queueOpts.MaxWorkers = runtime.NumCPU() * 2
	}
	queue, err := queue.New(queueOpts)

	apiServer := api.New(api.APIOpts{
		EnableMetrics: ko.Bool("metrics.enable"),
		ListenAddress: ko.MustString("api.address"),
		Logg:          lo,
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
		queue.Start(ctx)
	}()

	<-ctx.Done()
	lo.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultGracefulShutdownPeriod)

	wg.Add(1)
	go func() {
		defer wg.Done()
		apiServer.Stop(shutdownCtx)
		queue.Stop(shutdownCtx)
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
