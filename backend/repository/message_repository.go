package repository

import (
	"context"
	"fmt"
	"gRPCWebServer/backend/entities"

	"github.com/jmoiron/sqlx"
)

type MessageRepository interface {
	SaveMessage(message *entities.Message) error
	GetHistory(ctx context.Context, chatId uint64, limit int) ([]entities.Message, error)
}

type messageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *messageRepository {
	return &messageRepository{db: db}
}

// func (mr *messageRepository) SaveMessage(ctx context.Context, chatId, userId uint64, message string) (uint64, error) {
// 	var messageId uint64
// 	query := `INSERT INTO messages (chat_id, sender_id, content, sent_at) VALUES ($1, $2, $3, NOW()) RETURNING id`

// 	err := mr.db.GetContext(ctx, &messageId, query, chatId, userId, message)
// 	if err != nil {
// 		return 0, err
// 	}

//		return messageId, nil
//	}
func (mr *messageRepository) SaveMessage(message *entities.Message) error {
	query := `INSERT INTO messages (chat_id, sender_id, receiver_id, content, timestamp)
			  VALUES ($1, $2, $3, $4, $5)`
	_, err := mr.db.Exec(query, message.ChatID, message.SenderId, message.ReceiverId, message.Content, message.Timestamp)
	if err != nil {
		return err
	}

	return nil
}

func (mr *messageRepository) GetHistory(ctx context.Context, chatId uint64, limit int) ([]entities.Message, error) {
	query := `SELECT id, chat_id, sender_id, receiver_id, content, timestamp
			  FROM (
			  		SELECT id, chat_id, sender_id, receiver_id, content, timestamp
					FROM messages WHERE chat_id = $1 ORDER BY timestamp DESC LIMIT $2
					) subquery
			   ORDER BY timestamp ASC;`

	rows, err := mr.db.QueryContext(ctx, query, chatId, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %v", err)
	}

	defer rows.Close()

	var messages []entities.Message
	for rows.Next() {
		var message entities.Message
		if err := rows.Scan(&message.ID, &message.ChatID, &message.SenderId, &message.ReceiverId, &message.Content, &message.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan message: %v", err)
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}

	return messages, nil
}
