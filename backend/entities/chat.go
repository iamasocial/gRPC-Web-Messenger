package entities

import "time"

// Chat представляет информацию о чате между двумя пользователями
type Chat struct {
	ID             uint64    `db:"id"`
	FirstUserID    uint64    `db:"first_user_id"`
	SecondUserID   uint64    `db:"second_user_id"`
	FirstUsername  string    `db:"first_username"`
	SecondUsername string    `db:"second_username"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
