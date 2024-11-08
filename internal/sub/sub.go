package sub

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/grassrootseconomics/eth-custodial/internal/pub"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/nats-io/nats.go"
)

type (
	SubObts struct {
		Store store.Store
		Pub   *pub.Pub
		JSSub *nats.Subscription
		Logg  *slog.Logger
	}

	Sub struct {
		store    store.Store
		pub      *pub.Pub
		jsSub    *nats.Subscription
		natsConn *nats.Conn
		logg     *slog.Logger
	}
)

func NewSub(o SubObts) *Sub {
	return &Sub{
		store: o.Store,
		pub:   o.Pub,
		jsSub: o.JSSub,
		logg:  o.Logg,
	}
}

func (s *Sub) Process(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgs, err := s.jsSub.Fetch(100, nats.MaxWait(1*time.Second))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			} else if errors.Is(err, nats.ErrConnectionClosed) {
				return nil
			} else {
				return err
			}
		}

		for _, msg := range msgs {
			if err := s.processEvent(context.Background(), msg.Subject, msg.Data); err != nil {
				s.logg.Error("sub error processing nats message", "error", err)
				msg.Nak()
			} else {
				msg.Ack()
			}
		}
	}
}
