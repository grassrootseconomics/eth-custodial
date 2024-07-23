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
	// https://github.com/riverqueue/river/blob/master/CHANGELOG.md
	riverMigrationVersion = 4
	migrationTimeout      = 15 * time.Second
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
	}
)

func New(o QueueOpts) (*Queue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	tx, err := o.PgxPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	riverPgxDriver := riverpgxv5.New(o.PgxPool)
	riverMigrator := rivermigrate.New(riverPgxDriver, &rivermigrate.Config{
		Logger: o.Logg,
	})

	_, err = riverMigrator.MigrateTx(ctx, tx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{
		TargetVersion: riverMigrationVersion,
	})
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
	})

	return &Queue{
		client: riverClient,
	}, nil
}

func (q *Queue) Queue(ctx context.Context, tx pgx.Tx, jobArgs river.JobArgs) error {
	_, err := q.client.InsertTx(ctx, tx, jobArgs, nil)
	return err
}

func (q *Queue) Start(ctx context.Context) error {
	return q.client.Start(ctx)
}

func (q *Queue) Stop(ctx context.Context) error {
	return q.client.Stop(ctx)
}
