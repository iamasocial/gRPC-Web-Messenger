package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, user1ID, user2ID uint64) (uint64, error)
}

type chatRepository struct {
	db *sqlx.DB
}

func NewChatRepository(db *sqlx.DB) *chatRepository {
	return &chatRepository{db: db}
}

func (cr *chatRepository) CreateChat(ctx context.Context, user1ID, user2ID uint64) (uint64, error) {
	// var chatID uint64
	// query := `INSERT INTO chats (username) VALUES (?) RETURNING chat_id`
	// err := cr.db.QueryRowContext(ctx, query, username).Scan(&chatID)
	// if err != nil {
	// 	return 0, err
	// }
	// return chatID, nil
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	query := `INSERT INTO chats (user_1_id, user_2_id) VALUES ($1, $2)
			  ON CONFLICT (user_1_id, user_2_id) DO NOTHING RETURNING id`

	var chatId uint64
	err := cr.db.QueryRowContext(ctx, query, user1ID, user2ID).Scan(&chatId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("chat already exists")
		}
		return 0, err
	}

	return chatId, nil
}
