package repository

import (
	"gRPCWebServer/backend/entities"

	"github.com/jmoiron/sqlx"
)

type MessageRepository interface {
	SaveMessage(message *entities.Message) error
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
