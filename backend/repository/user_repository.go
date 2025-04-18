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
	GetUserNameById(ctx context.Context, userId uint64) (string, error)
	GetByID(ctx context.Context, userID uint64) (*entities.User, error)
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

func (ur *userRepo) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	var user entities.User
	query := `SELECT id, username, password_hash FROM users WHERE username = $1`

	err := ur.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get user by username: %v", err)
	}
	return &user, nil
}

func (ur *userRepo) GetUserNameById(ctx context.Context, userId uint64) (string, error) {
	var username string
	query := `SELECT username FROM users WHERE id = $1`

	err := ur.db.GetContext(ctx, &username, query, userId)
	if err != nil {
		return "", fmt.Errorf("error fetching username for user ID %d: %w", userId, err)
	}

	return username, nil
}

func (ur *userRepo) GetByID(ctx context.Context, userID uint64) (*entities.User, error) {
	var user entities.User
	query := `SELECT id, username, password_hash FROM users WHERE id = $1`

	err := ur.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get user by ID: %v", err)
	}
	return &user, nil
}
