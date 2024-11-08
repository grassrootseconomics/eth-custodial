package pub

import (
	"context"
	"fmt"
	"time"

	"github.com/grassrootseconomics/eth-custodial/pkg/event"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	PubOpts struct {
		PersistDuration time.Duration
		JS              jetstream.JetStream
		NatsConn        *nats.Conn
	}

	Pub struct {
		js       jetstream.JetStream
		natsConn *nats.Conn
	}
)

const pushStream = "CUSTODIAL"

var pushStreamSubjects = []string{
	"CUSTODIAL.*",
}

func NewPub(o PubOpts) *Pub {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	o.JS.CreateStream(ctx, jetstream.StreamConfig{
		Name:       pushStream,
		Subjects:   pushStreamSubjects,
		MaxAge:     o.PersistDuration,
		Storage:    jetstream.FileStorage,
		Duplicates: time.Minute,
	})

	return &Pub{
		js:       o.JS,
		natsConn: o.NatsConn,
	}
}

func (p *Pub) Close() {
	if p.natsConn != nil {
		p.natsConn.Close()
	}
}

func (p *Pub) Send(ctx context.Context, payload event.Event) error {
	data, err := payload.Serialize()
	if err != nil {
		return err
	}

	_, err = p.js.Publish(
		ctx,
		fmt.Sprintf("%s.%s", pushStream, payload.TrackingID),
		data,
	)
	if err != nil {
		return err
	}

	return nil
}
