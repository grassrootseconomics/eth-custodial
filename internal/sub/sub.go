package sub

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/grassrootseconomics/eth-custodial/internal/pub"
	"github.com/grassrootseconomics/eth-custodial/internal/store"
	"github.com/nats-io/nats.go/jetstream"
)

type (
	SubObts struct {
		Store      store.Store
		JS         jetstream.JetStream
		ConsumerID string
		Pub        *pub.Pub
		Logg       *slog.Logger
	}

	Sub struct {
		store  store.Store
		js     jetstream.JetStream
		jsIter jetstream.MessagesContext
		pub    *pub.Pub
		logg   *slog.Logger
	}
)

const (
	pullStream  = "TRACKER"
	pullSubject = "TRACKER.*"
)

func NewSub(o SubObts) (*Sub, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := o.JS.Stream(ctx, pullStream)
	if err != nil {
		return nil, err
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       o.ConsumerID,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: pullSubject,
	})
	if err != nil {
		return nil, err
	}

	iter, err := consumer.Messages(
		jetstream.WithMessagesErrOnMissingHeartbeat(false),
		jetstream.PullMaxMessages(10),
	)
	if err != nil {
		return nil, err
	}

	return &Sub{
		store:  o.Store,
		js:     o.JS,
		jsIter: iter,
		pub:    o.Pub,
		logg:   o.Logg,
	}, nil
}

func (s *Sub) Close() {
	s.logg.Debug("sub: closing js sub iterator")
	s.jsIter.Stop()
}

func (s *Sub) Process() {
	s.logg.Debug("sub: starting js sub iterator processor")
	for {
		msg, err := s.jsIter.Next()
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) {
				s.logg.Debug("sub: iterator closed")
				return
			} else {
				s.logg.Debug("sub: unknown iterator error", "error", err)
				continue
			}
		}

		s.logg.Debug("processing nats message", "subject", msg.Subject())
		if err := s.processEvent(context.Background(), msg.Subject(), msg.Data()); err != nil {
			s.logg.Error("jetstream: router: error processing nats message", "error", err)
			msg.Nak()
		} else {
			msg.Ack()
		}
	}
}
