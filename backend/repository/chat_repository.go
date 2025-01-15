package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, user1ID, user2ID uint64) error
	GetChatByUserIds(ctx context.Context, userId1, userId2 uint64) (uint64, error)
	GetChatsByUserId(ctx context.Context, userId uint64) ([]string, error)
	SendMessage(ctx context.Context, chatId, senderId uint64, content string) error
	DeleteChat(ctx context.Context, chatId uint64) error
}

type chatRepository struct {
	db *sqlx.DB
}

func NewChatRepository(db *sqlx.DB) *chatRepository {
	return &chatRepository{db: db}
}

func (cr *chatRepository) CreateChat(ctx context.Context, user1ID, user2ID uint64) error {
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	// var chatId uint64
	query := `INSERT INTO chats (user_1_id, user_2_id) VALUES ($1, $2)`
	_, err := cr.db.ExecContext(ctx, query, user1ID, user2ID)

	return err
}

func (cr *chatRepository) GetChatByUserIds(ctx context.Context, userId1, userId2 uint64) (uint64, error) {
	var chatId uint64
	query := `SELECT id FROM chats WHERE (user_1_id = $1 and user_2_id = $2) OR (user_1_id = $2 AND user_2_id = $1)`
	err := cr.db.GetContext(ctx, &chatId, query, userId1, userId2)
	if err != nil {
		return 0, err
	}

	return chatId, nil
}

func (cr *chatRepository) GetChatsByUserId(ctx context.Context, userId uint64) ([]string, error) {
	var usernames []string
	query := `
	SELECT 
			CASE 
				WHEN user_1_id = $1 THEN u2.username
				WHEN user_2_id = $1 THEN u1.username
			END AS username
		FROM chats
		JOIN users u1 ON u1.id = chats.user_1_id
		JOIN users u2 ON u2.id = chats.user_2_id
		WHERE user_1_id = $1 OR user_2_id = $1`

	err := cr.db.SelectContext(ctx, &usernames, query, userId)
	if err != nil {
		return nil, err
	}

	return usernames, nil
}

func (cr *chatRepository) DeleteChat(ctx context.Context, chatId uint64) error {
	query := `DELETE FROM chats WHERE id = $1`
	result, err := cr.db.ExecContext(ctx, query, chatId)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("chat with ID %d not found", chatId)
	}

	return nil
}

func (cr *chatRepository) SendMessage(ctx context.Context, chatId, senderId uint64, content string) error {
	query := `INSERT INTO messages (chat_id, sender_id, content, sent_at) VALUES ($1, $2, $3, NOW())`
	_, err := cr.db.ExecContext(ctx, query, chatId, senderId, content)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
