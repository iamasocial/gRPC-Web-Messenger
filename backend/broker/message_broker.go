package broker

import (
	"fmt"
	"gRPCWebServer/backend/entities"

	"github.com/rabbitmq/amqp091-go"
)

type MessageBroker interface {
	PublishMessage(message *entities.Message) error
	SubscribeMessages(queueName string, handleMessage func(string) error) error
	Close()
}

type messageBroker struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

func NewMessageBroker(url string) (*messageBroker, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create a channel: %v", err)
	}

	return &messageBroker{
		conn:    conn,
		channel: ch,
	}, nil
}

func (mb *messageBroker) PublishMessage(message *entities.Message) error {
	queueName := fmt.Sprintf("chat_queue_%d", message.ReceiverId)
	_, err := mb.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	messageBody := fmt.Sprintf("Sender: %d, Message: %s", message.SenderId, message.Content)

	err = mb.channel.Publish(
		"",
		queueName,
		false,
		false,
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(messageBody),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	return nil
}

func (mb *messageBroker) SubscribeMessages(queueName string, handleMessage func(string) error) error {
	msgs, err := mb.channel.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to consume messages: %v", err)
	}

	for msg := range msgs {
		if err := handleMessage(string(msg.Body)); err != nil {
			return fmt.Errorf("failed to handle message: %v", err)
		}
	}

	return nil
}

func (mb *messageBroker) Close() {
	mb.channel.Close()
	mb.conn.Close()
}
