package jetstream

import (
	"errors"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

type (
	JetStreamOpts struct {
		Logg            *slog.Logger
		JetStreamID     string
		Endpoint        string
		PersistDuration time.Duration
	}

	JetStream struct {
		JSCtx nats.JetStreamContext
		JSSub *nats.Subscription

		natsConn *nats.Conn
		logg     *slog.Logger
	}
)

const (
	PushStream = "CUSTODIAL"

	pullStream  = "TRACKER"
	pullSubject = "TRACKER.*"
)

var pushStreamSubjects = []string{
	"CUSTODIAL.*",
}

func NewJetStream(o JetStreamOpts) (*JetStream, error) {
	natsConn, err := nats.Connect(o.Endpoint)
	if err != nil {
		return nil, err
	}

	js, err := natsConn.JetStream()
	if err != nil {
		return nil, err
	}

	stream, err := js.StreamInfo(PushStream)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return nil, err
	}
	if stream == nil {
		_, err := js.AddStream(&nats.StreamConfig{
			Name:       PushStream,
			MaxAge:     o.PersistDuration,
			Storage:    nats.FileStorage,
			Subjects:   pushStreamSubjects,
			Duplicates: time.Minute,
		})
		if err != nil {
			return nil, err
		}
		o.Logg.Info("successfully created NATS JetStream stream", "stream_name", PushStream)
	}

	sub, err := js.PullSubscribe(pullSubject, o.JetStreamID, nats.AckExplicit())
	if err != nil {
		return nil, err
	}

	return &JetStream{
		JSCtx:    js,
		JSSub:    sub,
		natsConn: natsConn,
		logg:     o.Logg,
	}, nil
}

func (s *JetStream) Close() {
	if s.natsConn != nil {
		s.natsConn.Close()
	}
}
