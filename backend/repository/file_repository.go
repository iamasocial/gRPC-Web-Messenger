package repository

import (
	"context"
	"gRPCWebServer/backend/entities"
	"time"

	"github.com/jmoiron/sqlx"
)

// FileRepository интерфейс для работы с файлами в базе данных
type FileRepository interface {
	// Методы для работы с загрузками файлов
	CreateFileUpload(ctx context.Context, upload *entities.FileUpload) error
	GetFileUpload(ctx context.Context, uploadID string) (*entities.FileUpload, error)
	UpdateFileUpload(ctx context.Context, upload *entities.FileUpload) error
	DeleteFileUpload(ctx context.Context, uploadID string) error

	// Методы для работы с файлами
	CreateFile(ctx context.Context, file *entities.File) error
	GetFileByID(ctx context.Context, fileID string) (*entities.File, error)
	GetFilesByChat(ctx context.Context, chatID uint64, page, pageSize int) ([]*entities.File, int, error)
	DeleteFile(ctx context.Context, fileID string) error
}

type fileRepository struct {
	db *sqlx.DB
}

func NewFileRepository(db *sqlx.DB) FileRepository {
	return &fileRepository{
		db: db,
	}
}

// CreateFileUpload создает запись о загрузке файла в базе данных
func (fr *fileRepository) CreateFileUpload(ctx context.Context, upload *entities.FileUpload) error {
	query := `
		INSERT INTO file_uploads (
			upload_id, file_name, mime_type, total_size, chunk_size, temp_path,
			user_id, chat_id, status, created_at, updated_at
		) VALUES (
			:upload_id, :file_name, :mime_type, :total_size, :chunk_size, :temp_path,
			:user_id, :chat_id, :status, :created_at, :updated_at
		) RETURNING id
	`

	rows, err := fr.db.NamedQueryContext(ctx, query, upload)
	if err != nil {
		return err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&upload.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetFileUpload получает информацию о загрузке файла по ID
func (fr *fileRepository) GetFileUpload(ctx context.Context, uploadID string) (*entities.FileUpload, error) {
	query := `
		SELECT id, upload_id, file_name, mime_type, total_size, received_chunks, 
		chunk_size, temp_path, user_id, chat_id, status, created_at, updated_at
		FROM file_uploads
		WHERE upload_id = $1
	`

	var upload entities.FileUpload
	err := fr.db.GetContext(ctx, &upload, query, uploadID)
	if err != nil {
		return nil, err
	}

	return &upload, nil
}

// UpdateFileUpload обновляет информацию о загрузке файла
func (fr *fileRepository) UpdateFileUpload(ctx context.Context, upload *entities.FileUpload) error {
	upload.UpdatedAt = time.Now()

	query := `
		UPDATE file_uploads
		SET received_chunks = :received_chunks,
			status = :status,
			updated_at = :updated_at
		WHERE upload_id = :upload_id
	`

	_, err := fr.db.NamedExecContext(ctx, query, upload)
	return err
}

// DeleteFileUpload удаляет запись о загрузке файла
func (fr *fileRepository) DeleteFileUpload(ctx context.Context, uploadID string) error {
	query := `DELETE FROM file_uploads WHERE upload_id = $1`
	_, err := fr.db.ExecContext(ctx, query, uploadID)
	return err
}

// CreateFile создает запись о файле в базе данных
func (fr *fileRepository) CreateFile(ctx context.Context, file *entities.File) error {
	query := `
		INSERT INTO files (
			file_id, file_name, mime_type, size, path, uploaded_by, chat_id, checksum, created_at
		) VALUES (
			:file_id, :file_name, :mime_type, :size, :path, :uploaded_by, :chat_id, :checksum, :created_at
		) RETURNING id
	`

	rows, err := fr.db.NamedQueryContext(ctx, query, file)
	if err != nil {
		return err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&file.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetFileByID получает информацию о файле по ID
func (fr *fileRepository) GetFileByID(ctx context.Context, fileID string) (*entities.File, error) {
	query := `
		SELECT id, file_id, file_name, mime_type, size, path, uploaded_by, chat_id, checksum, created_at
		FROM files
		WHERE file_id = $1 AND deleted_at IS NULL
	`

	var file entities.File
	err := fr.db.GetContext(ctx, &file, query, fileID)
	if err != nil {
		return nil, err
	}

	return &file, nil
}

// GetFilesByChat получает список файлов в чате с пагинацией
func (fr *fileRepository) GetFilesByChat(ctx context.Context, chatID uint64, page, pageSize int) ([]*entities.File, int, error) {
	// Сначала получаем общее количество файлов
	countQuery := `
		SELECT COUNT(*) 
		FROM files 
		WHERE chat_id = $1 AND deleted_at IS NULL
	`

	var totalCount int
	err := fr.db.GetContext(ctx, &totalCount, countQuery, chatID)
	if err != nil {
		return nil, 0, err
	}

	// Затем получаем файлы с пагинацией
	offset := (page - 1) * pageSize
	query := `
		SELECT id, file_id, file_name, mime_type, size, path, uploaded_by, chat_id, checksum, created_at
		FROM files
		WHERE chat_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var files []*entities.File
	err = fr.db.SelectContext(ctx, &files, query, chatID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	return files, totalCount, nil
}

// DeleteFile выполняет мягкое удаление файла (устанавливает deleted_at)
func (fr *fileRepository) DeleteFile(ctx context.Context, fileID string) error {
	query := `
		UPDATE files
		SET deleted_at = NOW()
		WHERE file_id = $1
	`
	_, err := fr.db.ExecContext(ctx, query, fileID)
	return err
}
