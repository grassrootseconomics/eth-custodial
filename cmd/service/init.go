package main

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/grassrootseconomics/eth-custodial/internal/api"
	"github.com/grassrootseconomics/eth-custodial/internal/gas"
	"github.com/grassrootseconomics/eth-custodial/internal/jetstream"
	"github.com/grassrootseconomics/eth-custodial/internal/pub"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/internal/sub"
	"github.com/grassrootseconomics/eth-custodial/internal/util"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	"github.com/grassrootseconomics/ethutils"
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

	js *jetstream.JetStream
)

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

	gasOracle, err = gas.New(gas.GasOpts{
		OracleType: ko.MustString("gas.oracle_type"),
	})
	if err != nil {
		lo.Error("could not initialize gas oracle", "error", err)
		os.Exit(1)
	}

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

func loadJetStream() *jetstream.JetStream {
	if js != nil {
		return js
	}
	var err error

	js, err = jetstream.NewJetStream(jetstream.JetStreamOpts{
		Logg:            lo,
		Endpoint:        ko.MustString("jetstream.endpoint"),
		JetStreamID:     ko.MustString("jetstream.id"),
		PersistDuration: time.Duration(ko.MustInt("jetstream.persist_duration_hrs")) * time.Hour,
	})
	if err != nil {
		lo.Error("could not initialize jetstream sub", "error", err)
		os.Exit(1)
	}

	return js
}

func loadPub() *pub.Pub {
	if jsPub != nil {
		return jsPub
	}
	if js == nil {
		loadJetStream()
	}

	jsPub = pub.NewPub(pub.PubOpts{
		JSCtx: js.JSCtx,
	})

	return jsPub
}

func initSub() *sub.Sub {
	if jsSub != nil {
		return jsSub
	}
	if js == nil {
		loadJetStream()
	}

	jsSub = sub.NewSub(sub.SubObts{
		Store: loadStore(),
		Pub:   loadPub(),
		JSSub: js.JSSub,
		Logg:  lo,
	})

	return jsSub
}

func initWorker() *worker.WorkerContainer {
	if workerContainer != nil {
		return workerContainer
	}

	workerOpts := worker.WorkerOpts{
		CustodialRegistrationProxy: loadRegistry()[ethutils.CustodialProxy].Hex(),
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

	return api.New(api.APIOpts{
		EnableMetrics: ko.Bool("service.metrics"),
		EnableDocs:    ko.Bool("api.docs"),
		ListenAddress: ko.MustString("api.address"),
		SigningKey:    privateKey,
		VerifyingKey:  publicKey,
		Store:         loadStore(),
		ChainProvider: loadChainProvider(),
		Worker:        initWorker(),
		Logg:          lo,
		Debug:         true,
		BannedTokens:  ko.Strings("chain.banned_tokens"),
	})
}
