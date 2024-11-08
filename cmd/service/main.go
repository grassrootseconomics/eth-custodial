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
	"sync"
	"syscall"
	"time"

	"github.com/grassrootseconomics/eth-custodial/internal/api"
	"github.com/grassrootseconomics/eth-custodial/internal/sub"
	"github.com/grassrootseconomics/eth-custodial/internal/util"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
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

}

func main() {
	/*
		In "worker" mode, workers rely on:
			- Postgres
			- NATS to publish messages
			- RPC node to dispatch transactions (chainProvider)
			- GasOracle

		In "api" mode, the API server relies on:
			- Postgres
			- RPC node to fetch data (chainProvider)

		In "sub" mode, the JetStream subscriber relies on:
		  	- NATS to subscrib to messages and publish them
	*/

	var (
		wg sync.WaitGroup

		workerComponent *worker.WorkerContainer
		apiComponent    *api.API
		subComponent    *sub.Sub
	)

	ctx, stop := notifyShutdown()

	switch mode := ko.MustString("service.mode"); mode {
	case "worker":
		workerComponent = initWorker()
	case "sub":
		subComponent = initSub()
	case "api":
		apiComponent = initAPI()
	case "standalone":
		workerComponent = initWorker()
		subComponent = initSub()
		apiComponent = initAPI()
	default:
		lo.Error("a service mode that is either standalone, api, sub or worker needs to be explicitly set")
		os.Exit(1)
	}
	lo.Info("starting eth custodial", "build", build, "service_mode", ko.String("service.mode"))

	if apiComponent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := apiComponent.Start(); err != http.ErrServerClosed {
				lo.Error("failed to start HTTP server", "err", fmt.Sprintf("%T", err))
				os.Exit(1)
			}
		}()
	}

	if workerComponent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workerComponent.Start(ctx)
		}()
	}

	if subComponent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			subComponent.Process(ctx)
		}()
	}

	<-ctx.Done()
	lo.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultGracefulShutdownPeriod)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if js != nil {
			js.Close()
		}
		if apiComponent != nil {
			if err := apiComponent.Stop(shutdownCtx); err != nil {
				lo.Error("failed to stop HTTP server", "err", fmt.Sprintf("%T", err))
			}
		}
		if workerComponent != nil {
			workerComponent.Stop(shutdownCtx)
		}
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
