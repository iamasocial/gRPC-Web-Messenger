package repository

import (
	"context"
	"fmt"
	"gRPCWebServer/backend/entities"

	"github.com/jmoiron/sqlx"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, user1ID, user2ID uint64, encAlgorithm, encMode, encPadding string) error
	GetChatByUserIds(ctx context.Context, userId1, userId2 uint64) (uint64, error)
	GetChatsByUserId(ctx context.Context, userId uint64) ([]entities.ChatInfoDTO, error)
	SendMessage(ctx context.Context, chatId, senderId uint64, content string) error
	DeleteChat(ctx context.Context, chatId uint64) error
	GetChatByUsername(ctx context.Context, username string) (*entities.Chat, error)
	GetChatByID(ctx context.Context, chatID uint64) (*entities.Chat, error)
}

type chatRepository struct {
	db *sqlx.DB
}

func NewChatRepository(db *sqlx.DB) *chatRepository {
	return &chatRepository{db: db}
}

func (cr *chatRepository) CreateChat(ctx context.Context, user1ID, user2ID uint64, encAlgorithm, encMode, encPadding string) error {
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	query := `INSERT INTO chats (user_1_id, user_2_id, encryption_algorithm, encryption_mode, encryption_padding) 
	          VALUES ($1, $2, $3, $4, $5)`
	_, err := cr.db.ExecContext(ctx, query, user1ID, user2ID, encAlgorithm, encMode, encPadding)

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

func (cr *chatRepository) GetChatsByUserId(ctx context.Context, userId uint64) ([]entities.ChatInfoDTO, error) {
	var chats []entities.ChatInfoDTO
	query := `
	SELECT 
		CASE 
			WHEN user_1_id = $1 THEN u2.username
			WHEN user_2_id = $1 THEN u1.username
		END AS username,
		c.encryption_algorithm,
		c.encryption_mode,
		c.encryption_padding
	FROM chats c
	JOIN users u1 ON u1.id = c.user_1_id
	JOIN users u2 ON u2.id = c.user_2_id
	WHERE c.user_1_id = $1 OR c.user_2_id = $1`

	err := cr.db.SelectContext(ctx, &chats, query, userId)
	if err != nil {
		return nil, err
	}

	return chats, nil
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

func (cr *chatRepository) GetChatByUsername(ctx context.Context, username string) (*entities.Chat, error) {
	query := `
	SELECT 
		c.id, c.user_1_id as first_user_id, c.user_2_id as second_user_id,
		u1.username as first_username, u2.username as second_username
	FROM chats c
	JOIN users u1 ON u1.id = c.user_1_id
	JOIN users u2 ON u2.id = c.user_2_id
	WHERE u1.username = $1 OR u2.username = $1`

	var chat entities.Chat
	if err := cr.db.GetContext(ctx, &chat, query, username); err != nil {
		return nil, fmt.Errorf("failed to get chat by username: %w", err)
	}

	return &chat, nil
}

func (cr *chatRepository) GetChatByID(ctx context.Context, chatID uint64) (*entities.Chat, error) {
	query := `
	SELECT 
		c.id, c.user_1_id as first_user_id, c.user_2_id as second_user_id,
		u1.username as first_username, u2.username as second_username
	FROM chats c
	JOIN users u1 ON u1.id = c.user_1_id
	JOIN users u2 ON u2.id = c.user_2_id
	WHERE c.id = $1`

	var chat entities.Chat
	if err := cr.db.GetContext(ctx, &chat, query, chatID); err != nil {
		return nil, fmt.Errorf("failed to get chat by ID: %w", err)
	}

	return &chat, nil
}
