package entities

import "time"

// Chat представляет информацию о чате между двумя пользователями
type Chat struct {
	ID                  uint64    `db:"id"`
	FirstUserID         uint64    `db:"first_user_id"`
	SecondUserID        uint64    `db:"second_user_id"`
	FirstUsername       string    `db:"first_username"`
	SecondUsername      string    `db:"second_username"`
	EncryptionKey       []byte    `db:"encryption_key"`       // Общий ключ шифрования, полученный по протоколу Диффи-Хеллмана
	EncryptionAlgorithm string    `db:"encryption_algorithm"` // Алгоритм шифрования (например, AES)
	EncryptionMode      string    `db:"encryption_mode"`      // Режим шифрования (например, GCM, CBC)
	EncryptionPadding   string    `db:"encryption_padding"`   // Тип набивки (например, PKCS7)
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

// ChatInfoDTO представляет информацию о чате для возврата клиенту
type ChatInfoDTO struct {
	Username            string  `db:"username"`
	EncryptionAlgorithm *string `db:"encryption_algorithm"`
	EncryptionMode      *string `db:"encryption_mode"`
	EncryptionPadding   *string `db:"encryption_padding"`
}
