package entities

import "time"

// File представляет информацию о загруженном файле
type File struct {
	ID         uint64    `db:"id"`
	FileID     string    `db:"file_id"`
	FileName   string    `db:"file_name"`
	MimeType   string    `db:"mime_type"`
	Size       int64     `db:"size"`
	Path       string    `db:"path"`
	UploadedBy uint64    `db:"uploaded_by"`
	ChatID     uint64    `db:"chat_id"`
	Checksum   string    `db:"checksum"`
	CreatedAt  time.Time `db:"created_at"`
	DeletedAt  time.Time `db:"deleted_at,omitempty"`
}

// FileUpload представляет информацию о процессе загрузки файла
type FileUpload struct {
	ID             uint64    `db:"id"`
	UploadID       string    `db:"upload_id"`
	FileName       string    `db:"file_name"`
	MimeType       string    `db:"mime_type"`
	TotalSize      int64     `db:"total_size"`
	ReceivedChunks int       `db:"received_chunks"`
	ChunkSize      int       `db:"chunk_size"`
	TempPath       string    `db:"temp_path"`
	UserID         uint64    `db:"user_id"`
	ChatID         uint64    `db:"chat_id"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
