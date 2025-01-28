package broker

import (
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type MessageBroker interface {
	PublishMessage(senderUsername, receiverUsername, content string, timestamp time.Time) error
	SubscribeMessages(queueName string, handleMessage func(string, time.Time) error) error
	GetMessagesFromQueueWithoutDeleting(queueName string, handleMessage func(string, time.Time) error) error
	ProcessMessages(queueName string, handleMessage func(string, string, time.Time) error) error
	CheckMessages(queue string) (bool, error)
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

func (mb *messageBroker) PublishMessage(senderUsername, receiverUsername, content string, timestamp time.Time) error {
	queueName := fmt.Sprintf("chat_queue_%s", receiverUsername)
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

	messageBody := content

	err = mb.channel.Publish(
		"",
		queueName,
		false,
		false,
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(messageBody),
			Timestamp:   timestamp,
			Headers: amqp091.Table{
				"sender": senderUsername,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	return nil
}

func (mb *messageBroker) SubscribeMessages(queueName string, handleMessage func(string, time.Time) error) error {
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
		messageBody := string(msg.Body)
		timestamp := msg.Timestamp

		if err := handleMessage(messageBody, timestamp); err != nil {
			return fmt.Errorf("failed to handle message: %v", err)
		}
	}

	return nil
}

func (mb *messageBroker) GetMessagesFromQueueWithoutDeleting(queueName string, handleMessage func(string, time.Time) error) error {
	msgs, err := mb.channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to consume messages: %v", err)
	}

	for msg := range msgs {
		messageBody := string(msg.Body)
		timestamp := msg.Timestamp

		if err := handleMessage(messageBody, timestamp); err != nil {
			return fmt.Errorf("failed to handle message: %v", err)
		}

		err := msg.Ack(false)
		if err != nil {
			return fmt.Errorf("failed to acknowledge message: %v", err)
		}
	}

	return nil
}

func (mb *messageBroker) CheckMessages(queueName string) (bool, error) {
	queue, err := mb.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		if amqpErr, ok := err.(*amqp091.Error); ok && amqpErr.Code == 404 {
			return false, nil
		}
		return false, fmt.Errorf("failed to declare queue %s: %v", queueName, err)
	}

	return queue.Messages > 0, nil
}

func (mb *messageBroker) ProcessMessages(queueName string, handleMessage func(string, string, time.Time) error) error {
	log.Printf("Processing offline message started")

	_, err := mb.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		log.Printf("Failed to declare queue: %s: %v", queueName, err)
		return fmt.Errorf("failed to declare queue %s: %v", queueName, err)
	}

	log.Printf("Queue %s declared", queueName)

	deliveryChan, err := mb.channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		log.Printf("Failed to consume messages from queue %s: %v", queueName, err)
		return fmt.Errorf("failed to consume messages: %v", err)
	}

	log.Printf("Messages from queue %s consumed successfully", queueName)

	done := make(chan bool)
	go mb.queueMonitor(queueName, done)
	log.Printf("Starting processing online messages")

	for {
		select {
		case msg, ok := <-deliveryChan:
			if !ok {
				log.Println("No more messages in the queue")
				return nil
			}

			messageBody := string(msg.Body)
			timestamp := msg.Timestamp
			sender, ok := msg.Headers["sender"].(string)
			if !ok {
				log.Printf("Invalid header type, expected string, got %T", sender)
				msg.Nack(false, false)
				continue
			}

			err := handleMessage(sender, messageBody, timestamp)

			if err != nil {
				log.Printf("Error processing messages from sender %s: %v", sender, err)
				msg.Nack(false, true)
				continue
			}

			if err := msg.Ack(false); err != nil {
				log.Printf("Error: failed to acknowledge message: %v", err)
				msg.Nack(false, true)
				continue
			}
		case <-done:
			log.Println("Queue is empty, exiting")
			return nil
		}
	}
}

func (mb *messageBroker) queueMonitor(queueName string, done chan bool) {
	for {
		queue, err := mb.channel.QueueDeclare(
			queueName,
			true,
			false,
			false,
			false,
			nil,
		)

		if err != nil {
			log.Printf("Failed to declare queue: %v", err)
			done <- true
			return
		}

		log.Printf("Queue %s has %d messages", queue.Name, queue.Messages)

		if queue.Messages == 0 {
			done <- true
			return
		}

		time.Sleep(1 * time.Second)
	}
}

func (mb *messageBroker) Close() {
	mb.channel.Close()
	mb.conn.Close()
}
