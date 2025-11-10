package messaging

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

type MessageBroker struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

func NewMessageBroker(url string) (*MessageBroker, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, err
    }

    ch, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, err
    }

    return &MessageBroker{
        conn:    conn,
        channel: ch,
    }, nil
}

func (mb *MessageBroker) PublishEvent(exchange, routingKey string, event interface{}) error {
    body, err := json.Marshal(event)
    if err != nil {
        return err
    }

    err = mb.channel.Publish(
        exchange,
        routingKey,
        false,
        false,
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
    if err != nil {
        return err
    }

    log.Printf("Published event to %s: %s", routingKey, string(body))
    return nil
}

func (mb *MessageBroker) Close() {
    mb.channel.Close()
    mb.conn.Close()
}