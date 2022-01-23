package instance

import "github.com/streadway/amqp"

type RabbitMQ interface {
	RawClient() *amqp.Connection
	RawChannel() *amqp.Channel
}
