package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, user1ID, user2ID uint64) (uint64, error)
	GetByUserIds(ctx context.Context, userId1, userId2 uint64) (uint64, error)
	GetChatIdsByUserId(ctx context.Context, userId uint64) ([]uint64, error)
	IsUserInChat(ctx context.Context, userId, chatId uint64) (bool, error)
	DeleteChat(ctx context.Context, chatId uint64) error
}

type chatRepository struct {
	db *sqlx.DB
}

func NewChatRepository(db *sqlx.DB) *chatRepository {
	return &chatRepository{db: db}
}

func (cr *chatRepository) CreateChat(ctx context.Context, user1ID, user2ID uint64) (uint64, error) {
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	query := `INSERT INTO chats (user_1_id, user_2_id) VALUES ($1, $2) RETURNING id`

	var chatId uint64
	err := cr.db.GetContext(ctx, &chatId, query, user1ID, user2ID)
	if err != nil {
		return 0, err
	}

	return chatId, nil
}

func (cr *chatRepository) GetByUserIds(ctx context.Context, userId1, userId2 uint64) (uint64, error) {
	var chatId uint64
	query := `SELECT id FROM chats WHERE (user_1_id = $1 and user_2_id = $2) OR (user_1_id = $2 AND user_2_id = $1)`
	err := cr.db.GetContext(ctx, &chatId, query, userId1, userId2)
	if err != nil {
		return 0, err
	}

	return chatId, nil
}

func (cr *chatRepository) GetChatIdsByUserId(ctx context.Context, userId uint64) ([]uint64, error) {
	var chatIds []uint64
	query := `SELECT id FROM chats WHERE user_1_id = $1 OR user_2_id = $1`
	err := cr.db.SelectContext(ctx, &chatIds, query, userId)
	if err != nil {
		return nil, err
	}

	return chatIds, nil
}

func (cr *chatRepository) IsUserInChat(ctx context.Context, userId, chatId uint64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (
								SELECT 1 FROM chats WHERE id = $1 AND (user_1_id = $2 OR user_2_id = $2)
							)`

	err := cr.db.GetContext(ctx, &exists, query, chatId, userId)
	if err != nil {
		return false, err
	}

	return exists, nil
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
