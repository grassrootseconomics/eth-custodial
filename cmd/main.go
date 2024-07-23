package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
		ChainID:   ko.MustInt64("chain.chan_id"),
		GasOracle: gasOracle,
		Store:     store,
		Logg:      lo,
	})
	if err != nil {
		lo.Error("could not initialize signer workers", "error", err)
		os.Exit(1)
	}

	queue, err := queue.New(queue.QueueOpts{
		MaxWorkers: ko.MustInt("workers.max"),
		Logg:       lo,
		PgxPool:    store.Pool(),
		Workers:    workers,
	})

	// init go routines
}

func notifyShutdown() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
}
