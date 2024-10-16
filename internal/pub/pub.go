package pub

import (
	"context"
	"fmt"

	"github.com/grassrootseconomics/eth-custodial/internal/jetstream"
	"github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/nats-io/nats.go"
)

type (
	PubOpts struct {
		JSCtx nats.JetStreamContext
	}

	Pub struct {
		jsCtx nats.JetStreamContext
	}
)

func NewPub(o PubOpts) *Pub {
	return &Pub{
		jsCtx: o.JSCtx,
	}
}

func (p *Pub) Send(_ context.Context, payload event.Event) error {
	data, err := payload.Serialize()
	if err != nil {
		return err
	}

	_, err = p.jsCtx.Publish(
		fmt.Sprintf("%s.%s", jetstream.PushStream, payload.TrackingID),
		data,
	)
	if err != nil {
		return err
	}

	return nil
}
