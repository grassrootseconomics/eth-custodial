package store

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
	"github.com/knadh/goyesql/v2"
)

type (
	Queries struct {
		InsertKeyPair           string `query:"insert-keypair"`
		LoadKey                 string `query:"load-key"`
		CheckKeypair            string `query:"check-keypair"`
		LoadMasterKey           string `query:"load-master-key"`
		BootstrapMasterKey      string `query:"bootstrap-master-key"`
		PeekNonce               string `query:"peek-nonce"`
		AcquireNonce            string `query:"acquire-nonce"`
		SetAcccountNonce        string `query:"set-account-nonce"`
		InsertOTX               string `query:"insert-otx"`
		GetOTXByTrackingID      string `query:"get-otx-by-tracking-id"`
		GetOTXByAccount         string `query:"get-otx-by-account"`
		GetOTXByAccountNext     string `query:"get-otx-by-account-next"`
		GetOTXByAccountPrevious string `query:"get-otx-by-account-previous"`
		InsertDispatchTx        string `query:"insert-dispatch-tx"`
		UpdateDispatchTxStatus  string `query:"update-dispatch-tx-status"`
	}

	PgOpts struct {
		Logg                 *slog.Logger
		DSN                  string
		MigrationsFolderPath string
		QueriesFolderPath    string
	}

	Pg struct {
		logg    *slog.Logger
		db      *pgxpool.Pool
		queries *Queries
	}
)

func NewPgStore(o PgOpts) (Store, error) {
	parsedConfig, err := pgxpool.ParseConfig(o.DSN)
	if err != nil {
		return nil, err
	}

	dbPool, err := pgxpool.NewWithConfig(context.Background(), parsedConfig)
	if err != nil {
		return nil, err
	}

	queries, err := loadQueries(o.QueriesFolderPath)
	if err != nil {
		return nil, err
	}

	if err := runMigrations(dbPool, o.MigrationsFolderPath); err != nil {
		return nil, err
	}
	o.Logg.Info("migrations ran successfully")

	return &Pg{
		logg:    o.Logg,
		db:      dbPool,
		queries: queries,
	}, nil
}

func (s *Pg) Bootstrap() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	return s.bootstrapMasterSigner(ctx, tx)
}

func (s *Pg) Pool() *pgxpool.Pool {
	return s.db
}

func loadQueries(queriesPath string) (*Queries, error) {
	parsedQueries, err := goyesql.ParseFile(queriesPath)
	if err != nil {
		return nil, err
	}

	loadedQueries := &Queries{}

	if err := goyesql.ScanToStruct(loadedQueries, parsedQueries, nil); err != nil {
		return nil, fmt.Errorf("failed to scan queries %v", err)
	}

	return loadedQueries, nil
}

func runMigrations(dbPool *pgxpool.Pool, migrationsPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := dbPool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	migrator, err := migrate.NewMigrator(ctx, conn.Conn(), "schema_version")
	if err != nil {
		return err
	}

	if err := migrator.LoadMigrations(os.DirFS(migrationsPath)); err != nil {
		return err
	}

	if err := migrator.Migrate(ctx); err != nil {
		return err
	}

	return nil
}
