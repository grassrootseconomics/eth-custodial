package sub

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/grassrootseconomics/eth-custodial/internal/worker"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	JetStreamOpts struct {
		Endpoint        string
		JetStreamID     string
		Store           store.Store
		Logg            *slog.Logger
		WorkerContainer *worker.WorkerContainer
	}

	JetStreamSub struct {
		durableID       string
		jsConsumer      jetstream.Consumer
		store           store.Store
		workerContainer *worker.WorkerContainer
		natsConn        *nats.Conn
		logg            *slog.Logger
	}
)

const (
	pullStream  = "TRACKER"
	pullSubject = "TRACKER.*"
)

func NewJetStreamSub(o JetStreamOpts) (*JetStreamSub, error) {
	natsConn, err := nats.Connect(o.Endpoint)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(natsConn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := js.Stream(ctx, pullStream)
	if err != nil {
		return nil, err
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   o.JetStreamID,
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, err
	}
	o.Logg.Info("successfully connected to NATS server")

	return &JetStreamSub{
		durableID:       o.JetStreamID,
		store:           o.Store,
		jsConsumer:      consumer,
		workerContainer: o.WorkerContainer,
		natsConn:        natsConn,
		logg:            o.Logg,
	}, nil
}

func (s *JetStreamSub) Close() {
	if s.natsConn != nil {
		s.natsConn.Close()
	}
}

func (s *JetStreamSub) Process() error {
	for {
		events, err := s.jsConsumer.Fetch(100, jetstream.FetchMaxWait(1*time.Second))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			} else if errors.Is(err, nats.ErrConnectionClosed) {
				return nil
			} else {
				return err
			}
		}

		for msg := range events.Messages() {
			if err := s.processEvent(context.Background(), msg.Subject(), msg.Data()); err != nil {
				s.logg.Error("sub error processing nats message", "error", err)
				msg.Nak()
			} else {
				msg.Ack()
			}
		}
	}
}
