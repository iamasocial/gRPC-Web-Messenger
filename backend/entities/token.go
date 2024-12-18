package entities

import "time"

type Token struct {
	ID         uint64    `db:"id"`
	UserID     uint64    `db:"user_id"`
	Token      string    `db:"token"`
	ExpiresAt  time.Time `db:"expires_at"`
	Created_at time.Time `db:"created_at"`
}
