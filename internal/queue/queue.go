package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

const (
	migrationTimeout = 15 * time.Second
)

type (
	QueueOpts struct {
		MaxWorkers int
		Logg       *slog.Logger
		PgxPool    *pgxpool.Pool
		Workers    *river.Workers
	}

	Queue struct {
		client *river.Client[pgx.Tx]
		logg   *slog.Logger
	}
)

func New(o QueueOpts) (*Queue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	tx, err := o.PgxPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	riverPgxDriver := riverpgxv5.New(o.PgxPool)
	riverMigrator := rivermigrate.New(riverPgxDriver, &rivermigrate.Config{
		Logger: o.Logg,
	})

	_, err = riverMigrator.MigrateTx(ctx, tx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return nil, err
	}

	riverClient, err := river.NewClient(riverPgxDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {
				MaxWorkers: o.MaxWorkers,
			},
		},
		Workers: o.Workers,
		Logger:  o.Logg,
	})

	return &Queue{
		client: riverClient,
		logg:   o.Logg,
	}, nil
}

func (q *Queue) Client() *river.Client[pgx.Tx] {
	return q.client
}

func (q *Queue) Start(ctx context.Context) error {
	return q.client.Start(ctx)
}

func (q *Queue) Stop(ctx context.Context) error {
	q.logg.Info("shutting down river queue")
	return q.client.Stop(ctx)
}
