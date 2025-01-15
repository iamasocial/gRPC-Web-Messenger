package entities

import "time"

type Message struct {
	ID         uint64    `json:"id" db:"id"`
	ChatID     uint64    `json:"chat_id" db:"chat_id"`
	SenderId   uint64    `json:"sender_id" db:"sender_id"`
	ReceiverId uint64    `json:"receiver_id" db:"receiver_id"`
	Content    string    `json:"content" db:"content"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}
