package store

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
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

func (pg *Pg) LoadPrivateKey(ctx context.Context, tx pgx.Tx, publicKey string) (*ecdsa.PrivateKey, error) {
	var privateKeyString string

	if err := tx.QueryRow(
		ctx,
		pg.queries.LoadKey,
		publicKey,
	).Scan(&privateKeyString); err != nil {
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func (pg *Pg) LoadMasterSignerKey(ctx context.Context, tx pgx.Tx) (*ecdsa.PrivateKey, error) {
	var privateKeyString string

	if err := tx.QueryRow(
		ctx,
		pg.queries.LoadMasterKey,
	).Scan(&privateKeyString); err != nil {
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func (pg *Pg) bootstrapMasterSigner(ctx context.Context, tx pgx.Tx) error {
	_, err := pg.LoadMasterSignerKey(ctx, tx)
	if err != nil {
		if err == pgx.ErrNoRows {
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
