package main

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/grassrootseconomics/eth-custodial/internal/api"
	"github.com/grassrootseconomics/eth-custodial/internal/gas"
	internaljs "github.com/grassrootseconomics/eth-custodial/internal/jetstream"
	"github.com/grassrootseconomics/eth-custodial/internal/pub"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/internal/sub"
	"github.com/grassrootseconomics/eth-custodial/internal/util"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	"github.com/grassrootseconomics/ethutils"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	pgStore         store.Store
	gasOracle       gas.GasOracle
	chainProvider   *ethutils.Provider
	jsPub           *pub.Pub
	jsSub           *sub.Sub
	registry        map[string]common.Address
	workerContainer *worker.WorkerContainer
	apiServer       *api.API

	js       jetstream.JetStream
	natsConn *nats.Conn
)

func loadJetStream() jetstream.JetStream {
	if js != nil {
		return js
	}
	var err error

	natsConn, js, err = internaljs.NewJetStream(internaljs.JetStreamOpts{
		Endpoint: ko.MustString("jetstream.endpoint"),
	})
	if err != nil {
		lo.Error("could not initialize jetstream", "error", err)
		os.Exit(1)
	}

	return js
}

func loadStore() store.Store {
	if pgStore != nil {
		return pgStore
	}
	var err error

	pgStore, err = store.NewPgStore(store.PgOpts{
		Logg:                 lo,
		DSN:                  ko.MustString("postgres.dsn"),
		MigrationsFolderPath: migrationsFolderFlag,
		QueriesFolderPath:    queriesFlag,
	})
	if err != nil {
		lo.Error("could not initialize postgres store", "error", err)
		os.Exit(1)
	}
	if err := pgStore.Bootstrap(); err != nil {
		lo.Error("store bootstrap actions failed", "error", err)
		os.Exit(1)
	}

	return pgStore
}

func loadChainProvider() *ethutils.Provider {
	if chainProvider != nil {
		return chainProvider
	}

	chainProvider = ethutils.NewProvider(ko.MustString("chain.rpc_endpoint"), ko.MustInt64("chain.id"))
	return chainProvider
}

func loadGasOracle() gas.GasOracle {
	if gasOracle != nil {
		return gasOracle
	}
	var err error

	switch ko.MustString("gas.oracle_type") {
	case "static":
		gasOracle = &gas.StaticGas{}
	case "rpc":
		gasOracle, err = gas.NewRPCGasOracle(gas.RPCGasOracleOpts{
			Logg:          lo,
			ChainProvider: loadChainProvider(),
		})
		if err != nil {
			lo.Error("could not initialize rpc gas oracle", "error", err)
			os.Exit(1)
		}
	default:
		lo.Error("unknown gas oracle type", "type", ko.MustString("gas.oracle_type"))
		os.Exit(1)
	}
	lo.Debug("loaded gas oracle", "type", ko.MustString("gas.oracle_type"))

	return gasOracle
}

func loadRegistry() map[string]common.Address {
	if registry != nil {
		return registry
	}
	if chainProvider == nil {
		loadChainProvider()
	}
	var err error

	registry, err = chainProvider.RegistryMap(context.Background(), ethutils.HexToAddress(ko.MustString("chain.ge_registry")))
	if err != nil {
		lo.Error("could not fetch on chain registry", "error", err)
		os.Exit(1)
	}

	return registry
}

func loadPub() *pub.Pub {
	if jsPub != nil {
		return jsPub
	}

	jsPub = pub.NewPub(pub.PubOpts{
		PersistDuration: time.Duration(ko.MustInt("jetstream.persist_duration_hrs")) * time.Hour,
		JS:              loadJetStream(),
		NatsConn:        natsConn,
	})

	return jsPub
}

func initSub() *sub.Sub {
	if jsSub != nil {
		return jsSub
	}
	var err error

	jsSub, err = sub.NewSub(sub.SubObts{
		Store:      loadStore(),
		JS:         loadJetStream(),
		ConsumerID: ko.MustString("jetstream.id"),
		Pub:        loadPub(),
		Logg:       lo,
	})
	if err != nil {
		lo.Error("could not load jetstream sub", "error", err)
		os.Exit(1)
	}
	lo.Debug("init: successfuly loaded js sub", "consumer_id", ko.String("jetstream.id"))

	return jsSub
}

func initWorker() *worker.WorkerContainer {
	if workerContainer != nil {
		return workerContainer
	}

	workerOpts := worker.WorkerOpts{
		Registry: loadRegistry(),
		// TODO: Tune max workers based on load type
		MaxWorkers:    ko.Int("workers.max"),
		GasOracle:     loadGasOracle(),
		Store:         loadStore(),
		Logg:          lo,
		Pub:           loadPub(),
		ChainProvider: loadChainProvider(),
	}

	if ko.Int("workers.max") <= 0 {
		workerOpts.MaxWorkers = runtime.NumCPU() * 2
	}
	workerContainer, err := worker.New(workerOpts)
	if err != nil {
		lo.Error("could not initialize worker container", "error", err)
		os.Exit(1)
	}

	return workerContainer
}

func initAPI() *api.API {
	if apiServer != nil {
		return apiServer
	}

	privateKey, publicKey, err := util.LoadSigningKey(ko.MustString("api.private_key"))
	if err != nil {
		lo.Error("could not load private key", "error", err)
		os.Exit(1)
	}
	lo.Debug("loaded private key", "key", privateKey)

	worker := initWorker()

	return api.New(api.APIOpts{
		Prod:          ko.Bool("api.prod"),
		EnableMetrics: ko.Bool("service.metrics"),
		EnableDocs:    ko.Bool("api.docs"),
		JRPC:          ko.Bool("api.jrpc"),
		ListenAddress: ko.MustString("api.address"),
		CORS:          ko.MustStrings("api.cors"),
		Registry:      loadRegistry(),
		SigningKey:    privateKey,
		VerifyingKey:  publicKey,
		GasOracle:     loadGasOracle(),
		Store:         loadStore(),
		ChainProvider: loadChainProvider(),
		QueueClient:   worker.Client(),
		Logg:          lo,
		Debug:         true,
		BannedTokens:  ko.Strings("chain.banned_tokens"),
	})
}
