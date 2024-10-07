package sub

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-tracker/pkg/event"
	"github.com/jackc/pgx/v5"
)

func (s *JetStreamSub) processEvent(ctx context.Context, msgSubject string, msg []byte) error {
	s.logg.Debug("sub processing event", "subject", msgSubject, "data", string(msg))
	var chainEvent event.Event

	if err := json.Unmarshal(msg, &chainEvent); err != nil {
		return err
	}

	tx, err := s.store.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	otx, err := s.store.GetOTXByTxHash(
		ctx,
		tx,
		chainEvent.TxHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	updateDispatchStatus := store.DispatchTx{
		OTXID: otx.ID,
	}

	if chainEvent.Success {
		switch msgSubject {
		case "TRACKER.CUSTODIAL_REGISTRATION":
			if err := s.store.ActivateKeyPair(ctx, tx, chainEvent.Payload["account"].(string)); err != nil {
				return err
			}
		}
		updateDispatchStatus.Status = store.SUCCESS
	} else {
		updateDispatchStatus.Status = store.REVERTED
	}

	if err := s.store.UpdateDispatchTxStatus(ctx, tx, updateDispatchStatus); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
