package messaging

import (
	"encoding/json"
	"fmt"
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

    // Declare exchange for order events
    err = ch.ExchangeDeclare(
        "order_events", // exchange name
        "topic",        // exchange type
        true,           // durable
        false,          // auto-deleted
        false,          // internal
        false,          // no-wait
        nil,            // arguments
    )
    if err != nil {
        ch.Close()
        conn.Close()
        return nil, fmt.Errorf("failed to declare exchange: %v", err)
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