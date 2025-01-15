package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gRPCWebServer/backend/entities"

	"github.com/jmoiron/sqlx"
)

type UserRepository interface {
	Create(ctx context.Context, user entities.User) (uint64, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
}

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *userRepo {
	return &userRepo{db: db}
}

func (ur *userRepo) Create(ctx context.Context, user entities.User) (uint64, error) {
	query := `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`
	var userID uint64
	err := ur.db.GetContext(ctx, &userID, query, user.Username, user.PasswordHash)
	if err != nil {
		return 0, fmt.Errorf("failed to insert user: %v", err)
	}
	return userID, nil
}

func (us *userRepo) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	var user entities.User
	query := `SELECT id, username, password_hash FROM users WHERE username = $1`

	err := us.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get user by username: %v", err)
	}
	return &user, nil
}
