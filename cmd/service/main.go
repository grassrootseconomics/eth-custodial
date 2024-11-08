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
	"github.com/grassrootseconomics/eth-custodial/internal/jetstream"
	"github.com/grassrootseconomics/eth-custodial/internal/pub"
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

	js, err := jetstream.NewJetStream(jetstream.JetStreamOpts{
		Logg:            lo,
		Endpoint:        ko.MustString("jetstream.endpoint"),
		JetStreamID:     ko.MustString("jetstream.id"),
		PersistDuration: time.Duration(ko.MustInt("jetstream.persist_duration_hrs")) * time.Hour,
	})
	if err != nil {
		lo.Error("could not initialize jetstream sub", "error", err)
		os.Exit(1)
	}
	pub := pub.NewPub(pub.PubOpts{
		JSCtx: js.JSCtx,
	})

	registry, err := chainProvider.RegistryMap(ctx, ethutils.HexToAddress(ko.MustString("chain.ge_registry")))
	if err != nil {
		lo.Error("could not fetch on chain registry", "error", err)
		os.Exit(1)
	}

	workerOpts := worker.WorkerOpts{
		CustodialRegistrationProxy: registry[ethutils.CustodialProxy].Hex(),
		// TODO: Tune max workers based on load type
		MaxWorkers:    ko.Int("workers.max"),
		GasOracle:     gasOracle,
		Store:         store,
		Logg:          lo,
		Pub:           pub,
		ChainProvider: chainProvider,
	}
	if ko.Int("workers.max") <= 0 {
		workerOpts.MaxWorkers = runtime.NumCPU() * 2
	}
	workerContainer, err := worker.New(workerOpts)
	if err != nil {
		lo.Error("could not initialize worker container", "error", err)
		os.Exit(1)
	}

	sub := sub.NewSub(sub.SubObts{
		Store:           store,
		Pub:             pub,
		JSSub:           js.JSSub,
		Logg:            lo,
		WorkerContainer: workerContainer,
	})

	privateKey, publicKey, err := util.LoadSigningKey(ko.MustString("api.private_key"))
	if err != nil {
		lo.Error("could not load private key", "error", err)
		os.Exit(1)
	}
	lo.Info("loaded private key", "key", privateKey)

	apiServer := api.New(api.APIOpts{
		EnableMetrics: ko.Bool("metrics.enable"),
		EnableDocs:    ko.Bool("api.docs"),
		ListenAddress: ko.MustString("api.address"),
		SigningKey:    publicKey,
		Store:         store,
		ChainProvider: chainProvider,
		Worker:        workerContainer,
		Logg:          lo,
		Debug:         true,
		BannedTokens:  ko.Strings("chain.banned_tokens"),
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
		sub.Process(ctx)
	}()

	<-ctx.Done()
	lo.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultGracefulShutdownPeriod)

	wg.Add(1)
	go func() {
		defer wg.Done()
		js.Close()
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
