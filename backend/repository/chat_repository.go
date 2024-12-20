package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, user1ID, user2ID uint64) (uint64, error)
	GetByUserIds(ctx context.Context, userId1, userId2 uint64) (uint64, error)
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
