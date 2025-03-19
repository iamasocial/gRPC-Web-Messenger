package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// DHKeyExchange представляет запись обмена ключами в базе данных
type DHKeyExchange struct {
	ID          uint64
	ChatID      uint64
	InitiatorID uint64
	RecipientID uint64
	DHG         sql.NullString // Генератор
	DHP         sql.NullString // Простое число
	DHA         sql.NullString // Публичный ключ инициатора
	DHB         sql.NullString // Публичный ключ получателя
	Status      string         // статус обмена ключами
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// KeyExchangeRepository интерфейс для работы с хранилищем данных обмена ключами
type KeyExchangeRepository interface {
	// Создает новую запись обмена ключами
	CreateKeyExchange(ctx context.Context, chatID, initiatorID, recipientID uint64, g, p, a string) (uint64, error)

	// Обновляет запись обмена ключами с ключом B
	CompleteKeyExchange(ctx context.Context, id uint64, b string) error

	// Получает запись обмена ключами по ID чата
	GetKeyExchangeByChatID(ctx context.Context, chatID uint64) (*DHKeyExchange, error)

	// Получает запись обмена ключами между двумя пользователями
	GetKeyExchangeByUserIDs(ctx context.Context, user1ID, user2ID uint64) (*DHKeyExchange, error)

	// Обновляет статус обмена ключами
	UpdateKeyExchangeStatus(ctx context.Context, id uint64, status string) error

	// Удаляет запись обмена ключами
	DeleteKeyExchange(ctx context.Context, id uint64) error
}

// keyExchangeRepository реализация интерфейса KeyExchangeRepository
type keyExchangeRepository struct {
	db *sqlx.DB
}

// NewKeyExchangeRepository создает новый экземпляр репозитория для обмена ключами
func NewKeyExchangeRepository(db *sqlx.DB) KeyExchangeRepository {
	return &keyExchangeRepository{
		db: db,
	}
}

// CreateKeyExchange создает новую запись обмена ключами
func (r *keyExchangeRepository) CreateKeyExchange(ctx context.Context, chatID, initiatorID, recipientID uint64, g, p, a string) (uint64, error) {
	query := `
		INSERT INTO dh_key_exchanges (chat_id, initiator_id, recipient_id, dh_g, dh_p, dh_a, status) 
		VALUES ($1, $2, $3, $4, $5, $6, 'INITIATED') 
		RETURNING id
	`

	var id uint64
	err := r.db.QueryRowContext(ctx, query, chatID, initiatorID, recipientID, g, p, a).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// CompleteKeyExchange обновляет запись обмена ключами с ключом B
func (r *keyExchangeRepository) CompleteKeyExchange(ctx context.Context, id uint64, b string) error {
	query := `
		UPDATE dh_key_exchanges 
		SET dh_b = $1, status = 'COMPLETED' 
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, b, id)
	return err
}

// GetKeyExchangeByChatID получает запись обмена ключами по ID чата
func (r *keyExchangeRepository) GetKeyExchangeByChatID(ctx context.Context, chatID uint64) (*DHKeyExchange, error) {
	query := `
		SELECT id, chat_id, initiator_id, recipient_id, dh_g, dh_p, dh_a, dh_b, status, created_at, updated_at 
		FROM dh_key_exchanges 
		WHERE chat_id = $1 AND status NOT IN ('FAILED')
		ORDER BY updated_at DESC 
		LIMIT 1
	`

	var exchange DHKeyExchange
	err := r.db.QueryRowContext(ctx, query, chatID).Scan(
		&exchange.ID,
		&exchange.ChatID,
		&exchange.InitiatorID,
		&exchange.RecipientID,
		&exchange.DHG,
		&exchange.DHP,
		&exchange.DHA,
		&exchange.DHB,
		&exchange.Status,
		&exchange.CreatedAt,
		&exchange.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &exchange, nil
}

// GetKeyExchangeByUserIDs получает запись обмена ключами между двумя пользователями
func (r *keyExchangeRepository) GetKeyExchangeByUserIDs(ctx context.Context, user1ID, user2ID uint64) (*DHKeyExchange, error) {
	query := `
		SELECT ke.id, ke.chat_id, ke.initiator_id, ke.recipient_id, ke.dh_g, ke.dh_p, ke.dh_a, ke.dh_b, ke.status, ke.created_at, ke.updated_at 
		FROM dh_key_exchanges ke
		INNER JOIN chats c ON ke.chat_id = c.id
		WHERE (c.user1_id = $1 AND c.user2_id = $2 OR c.user1_id = $2 AND c.user2_id = $1)
		AND ke.status NOT IN ('FAILED')
		ORDER BY ke.updated_at DESC 
		LIMIT 1
	`

	var exchange DHKeyExchange
	err := r.db.QueryRowContext(ctx, query, user1ID, user2ID).Scan(
		&exchange.ID,
		&exchange.ChatID,
		&exchange.InitiatorID,
		&exchange.RecipientID,
		&exchange.DHG,
		&exchange.DHP,
		&exchange.DHA,
		&exchange.DHB,
		&exchange.Status,
		&exchange.CreatedAt,
		&exchange.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &exchange, nil
}

// UpdateKeyExchangeStatus обновляет статус обмена ключами
func (r *keyExchangeRepository) UpdateKeyExchangeStatus(ctx context.Context, id uint64, status string) error {
	query := `
		UPDATE dh_key_exchanges 
		SET status = $1 
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// DeleteKeyExchange удаляет запись обмена ключами
func (r *keyExchangeRepository) DeleteKeyExchange(ctx context.Context, id uint64) error {
	query := `DELETE FROM dh_key_exchanges WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
