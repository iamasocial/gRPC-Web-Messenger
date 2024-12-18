package entities

type User struct {
	ID           uint64 `db:"id"`
	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
}
