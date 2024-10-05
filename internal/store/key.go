package store

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/grassrootseconomics/eth-custodial/internal/keypair"
	"github.com/jackc/pgx/v5"
)

func (pg *Pg) InsertKeyPair(ctx context.Context, tx pgx.Tx, keypair keypair.Key) error {
	_, err := tx.Exec(
		ctx,
		pg.queries.InsertKeyPair,
		keypair.Public,
		keypair.Private,
	)
	if err != nil {
		return err
	}

	return nil
}

func (pg *Pg) ActivateKeyPair(ctx context.Context, tx pgx.Tx, publicKey string) error {
	_, err := tx.Exec(
		ctx,
		pg.queries.ActivateKeyPair,
		publicKey,
	)
	if err != nil {
		return err
	}

	return nil
}

func (pg *Pg) LoadPrivateKey(ctx context.Context, tx pgx.Tx, publicKey string) (keypair.Key, error) {
	var keypair keypair.Key

	row, err := tx.Query(ctx, pg.queries.LoadKey, publicKey)
	if err != nil {
		return keypair, err
	}

	if err := pgxscan.ScanOne(&keypair, row); err != nil {
		return keypair, err
	}

	return keypair, nil
}

func (pg *Pg) CheckKeypair(ctx context.Context, tx pgx.Tx, publicKey string) (bool, error) {
	var active bool

	if err := tx.QueryRow(
		ctx,
		pg.queries.CheckKeypair,
		publicKey,
	).Scan(&active); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return active, nil
}

func (pg *Pg) LoadMasterSignerKey(ctx context.Context, tx pgx.Tx) (keypair.Key, error) {
	var masterKeypair keypair.Key

	row, err := tx.Query(ctx, pg.queries.LoadMasterKey)
	if err != nil {
		return masterKeypair, err
	}

	if err := pgxscan.ScanOne(&masterKeypair, row); err != nil {
		return masterKeypair, err
	}

	return masterKeypair, nil
}

func (pg *Pg) bootstrapMasterSigner(ctx context.Context, tx pgx.Tx) error {
	_, err := pg.LoadMasterSignerKey(ctx, tx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			masterKeyPair, err := keypair.GenerateKeyPair()
			if err != nil {
				return err
			}

			_, err = tx.Exec(
				ctx,
				pg.queries.BootstrapMasterKey,
				masterKeyPair.Public,
				masterKeyPair.Private,
			)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
