package repository

import (
	"context"
	"log"
	"messenger/backend/entities"

	"github.com/jmoiron/sqlx"
)

type TokenRepository interface {
	SaveToken(ctx context.Context, token *entities.Token) error
	GetByToken(ctx context.Context, token string) (*entities.Token, error)
	DeleteToken(ctx context.Context, token string) error
	GetActiveToken(ctx context.Context, userID uint64) (*entities.Token, error)
}

type tokenRepository struct {
	db *sqlx.DB
}

func NewTokenRepository(db *sqlx.DB) *tokenRepository {
	return &tokenRepository{db: db}
}

func (tr *tokenRepository) SaveToken(ctx context.Context, token *entities.Token) error {
	query := `INSERT INTO users_tokens (user_id, token, expires_at, created_at) VALUES ($1, $2, $3, NOW())`
	_, err := tr.db.ExecContext(ctx, query, token.UserID, token.Token, token.ExpiresAt)
	log.Println("Token saved successfully")
	return err
}

func (tr *tokenRepository) GetByToken(ctx context.Context, token string) (*entities.Token, error) {
	var t entities.Token
	query := `SELECT id, user_id, token, expires_at, created_at FROM users_tokens WHERE token = $1`
	err := tr.db.GetContext(ctx, &t, query, token)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// func (tr *tokenRepository) GetUserIDByToken(ctx context.Context, token string) (uint64, error) {
// 	var userID uint64
// 	query := `SELECT user_id FROM users_tokens WHERE token = $1 AND expires_at > NOW()`
// 	err := tr.db.QueryRowContext(ctx, query, token).Scan(&userID)
// 	return userID, err
// } // do I really need that?

func (tr *tokenRepository) GetActiveToken(ctx context.Context, userID uint64) (*entities.Token, error) {
	var token entities.Token
	query := `SELECT * FROM users_tokens WHERE user_id = $1 AND expires_at > NOW() LIMIT 1`
	row := tr.db.QueryRowContext(ctx, query, userID)
	err := row.Scan(&token.ID, &token.UserID, &token.Token, &token.ExpiresAt, &token.Created_at)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (tr *tokenRepository) DeleteToken(ctx context.Context, token string) error {
	query := `DELETE FROM users_tokens WHERE token = $1`
	_, err := tr.db.ExecContext(ctx, query, token)
	return err
}

// func (tr *tokenRepository) DeleteTokenByUserID(ctx context.Context, userID uint64) error {
// 	query := `DELETE FROM users_tokens WHERE user_id = $1`
// 	_, err := tr.db.ExecContext(ctx, query, userID)
// 	return err
// } // do I really need that?
