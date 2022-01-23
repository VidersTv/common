package rmq

import (
	"context"

	"github.com/streadway/amqp"
	"github.com/viderstv/common/instance"
)

type RmqInst struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func New(ctx context.Context, opts SetupOptions) (instance.RabbitMQ, error) {
	conn, err := amqp.Dial(opts.URI)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	_, err = ch.QueueDeclare(opts.QueueName, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return &RmqInst{
		conn: conn,
		ch:   ch,
	}, nil
}

func (r *RmqInst) RawClient() *amqp.Connection {
	return r.conn
}

func (r *RmqInst) RawChannel() *amqp.Channel {
	return r.ch
}

type SetupOptions struct {
	URI       string
	QueueName string
}
