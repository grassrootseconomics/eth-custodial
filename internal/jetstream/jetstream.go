package jetstream

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type JetStreamOpts struct {
	Endpoint string
}

func NewJetStream(o JetStreamOpts) (*nats.Conn, jetstream.JetStream, error) {
	natsConn, err := nats.Connect(o.Endpoint)
	if err != nil {
		return nil, nil, err
	}

	js, err := jetstream.New(natsConn)
	if err != nil {
		return nil, nil, err
	}

	return natsConn, js, nil
}
