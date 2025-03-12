package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"gRPCWebServer/backend/entities"
	pb "gRPCWebServer/backend/generated"
	"gRPCWebServer/backend/middleware"
	"gRPCWebServer/backend/repository"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ActiveUpload представляет информацию о текущей активной загрузке файла
type ActiveUpload struct {
	file       *os.File
	uploadInfo *entities.FileUpload
	mutex      sync.Mutex
}

type FileService struct {
	pb.UnimplementedFileServiceServer
	fileRepo     repository.FileRepository
	userRepo     repository.UserRepository
	chatRepo     repository.ChatRepository
	fileUploads  map[string]*ActiveUpload // uploadID -> активная загрузка
	uploadsMutex sync.RWMutex
	baseFilePath string // Базовый путь для хранения файлов
}

func NewFileService(
	fileRepo repository.FileRepository,
	userRepo repository.UserRepository,
	chatRepo repository.ChatRepository,
	baseFilePath string,
) *FileService {
	// Создаем директории для хранения файлов, если они не существуют
	os.MkdirAll(baseFilePath, 0755)
	os.MkdirAll(filepath.Join(baseFilePath, "temp"), 0755)

	return &FileService{
		fileRepo:     fileRepo,
		userRepo:     userRepo,
		chatRepo:     chatRepo,
		fileUploads:  make(map[string]*ActiveUpload),
		uploadsMutex: sync.RWMutex{},
		baseFilePath: baseFilePath,
	}
}

// InitFileUpload инициализирует загрузку файла
func (s *FileService) InitFileUpload(ctx context.Context, req *pb.InitFileUploadRequest) (*pb.InitFileUploadResponse, error) {
	userID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Требуется аутентификация")
	}

	// Получаем чат
	chat, err := s.chatRepo.GetChatByUsername(ctx, req.ChatUsername)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о чате: %v", err)
	}
	if chat == nil {
		return nil, status.Errorf(codes.NotFound, "Чат не найден")
	}

	// Генерируем уникальный ID для загрузки
	uploadID := uuid.New().String()

	// Определяем рекомендуемый размер чанка (1 МБ)
	chunkSize := 1 * 1024 * 1024

	// Создаем временный файл
	tempDir := filepath.Join(s.baseFilePath, "temp")
	tempFilePath := filepath.Join(tempDir, uploadID)

	file, err := os.Create(tempFilePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при создании временного файла: %v", err)
	}

	// Создаем запись о загрузке в базе данных
	upload := &entities.FileUpload{
		UploadID:       uploadID,
		FileName:       req.Filename,
		MimeType:       req.MimeType,
		TotalSize:      req.TotalSize,
		ReceivedChunks: 0,
		ChunkSize:      chunkSize,
		TempPath:       tempFilePath,
		UserID:         userID,
		ChatID:         chat.ID,
		Status:         "in_progress",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err = s.fileRepo.CreateFileUpload(ctx, upload)
	if err != nil {
		file.Close()
		os.Remove(tempFilePath)
		return nil, status.Errorf(codes.Internal, "Ошибка при создании записи о загрузке: %v", err)
	}

	// Сохраняем информацию об активной загрузке
	s.uploadsMutex.Lock()
	s.fileUploads[uploadID] = &ActiveUpload{
		file:       file,
		uploadInfo: upload,
		mutex:      sync.Mutex{},
	}
	s.uploadsMutex.Unlock()

	return &pb.InitFileUploadResponse{
		UploadId:  uploadID,
		ChunkSize: int32(chunkSize),
	}, nil
}

// UploadFileChunk обрабатывает потоковую загрузку частей файла
func (s *FileService) UploadFileChunk(stream pb.FileService_UploadFileChunkServer) error {
	ctx := stream.Context()

	userID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "Требуется аутентификация")
	}

	var uploadID string
	var currentUpload *ActiveUpload
	var totalChunks int

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			// Все чанки получены
			response := &pb.UploadFileChunkResponse{
				UploadId:       uploadID,
				ReceivedChunks: int32(totalChunks),
				Success:        true,
			}
			return stream.SendAndClose(response)
		}
		if err != nil {
			return status.Errorf(codes.Internal, "Ошибка при получении части файла: %v", err)
		}

		// Получаем ID загрузки из первого чанка
		if uploadID == "" {
			uploadID = chunk.UploadId

			// Проверяем, существует ли такая загрузка
			s.uploadsMutex.RLock()
			currentActive, exists := s.fileUploads[uploadID]
			s.uploadsMutex.RUnlock()

			if !exists {
				// Проверяем, существует ли загрузка в базе данных
				upload, err := s.fileRepo.GetFileUpload(ctx, uploadID)
				if err != nil {
					return status.Errorf(codes.Internal, "Ошибка при получении информации о загрузке: %v", err)
				}

				if upload == nil {
					return status.Errorf(codes.NotFound, "Загрузка не найдена")
				}

				// Проверяем, принадлежит ли загрузка этому пользователю
				if upload.UserID != userID {
					return status.Errorf(codes.PermissionDenied, "У вас нет доступа к этой загрузке")
				}

				// Открываем временный файл
				file, err := os.OpenFile(upload.TempPath, os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					return status.Errorf(codes.Internal, "Ошибка при открытии временного файла: %v", err)
				}

				currentActive = &ActiveUpload{
					file:       file,
					uploadInfo: upload,
					mutex:      sync.Mutex{},
				}

				s.uploadsMutex.Lock()
				s.fileUploads[uploadID] = currentActive
				s.uploadsMutex.Unlock()
			}

			currentUpload = currentActive
		}

		if chunk.UploadId != uploadID {
			return status.Errorf(codes.InvalidArgument, "Все части файла должны иметь один и тот же ID загрузки")
		}

		// Блокируем доступ к загрузке, чтобы избежать одновременной записи
		currentUpload.mutex.Lock()

		// Записываем данные чанка во временный файл
		_, err = currentUpload.file.Write(chunk.Data)
		if err != nil {
			currentUpload.mutex.Unlock()
			return status.Errorf(codes.Internal, "Ошибка при записи данных в файл: %v", err)
		}

		// Обновляем информацию о загрузке
		currentUpload.uploadInfo.ReceivedChunks++
		totalChunks = currentUpload.uploadInfo.ReceivedChunks

		// Обновляем запись в базе данных
		err = s.fileRepo.UpdateFileUpload(ctx, currentUpload.uploadInfo)
		if err != nil {
			currentUpload.mutex.Unlock()
			return status.Errorf(codes.Internal, "Ошибка при обновлении информации о загрузке: %v", err)
		}

		currentUpload.mutex.Unlock()
	}
}

// FinalizeFileUpload завершает загрузку файла и создает запись о нем
func (s *FileService) FinalizeFileUpload(ctx context.Context, req *pb.FinalizeFileUploadRequest) (*pb.FinalizeFileUploadResponse, error) {
	userID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Требуется аутентификация")
	}

	uploadID := req.UploadId

	// Получаем информацию о загрузке
	s.uploadsMutex.RLock()
	currentUpload, exists := s.fileUploads[uploadID]
	s.uploadsMutex.RUnlock()

	if !exists {
		// Если нет в памяти, пробуем найти в базе данных
		upload, err := s.fileRepo.GetFileUpload(ctx, uploadID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о загрузке: %v", err)
		}

		if upload == nil {
			return nil, status.Errorf(codes.NotFound, "Загрузка не найдена")
		}

		// Проверяем, принадлежит ли загрузка этому пользователю
		if upload.UserID != userID {
			return nil, status.Errorf(codes.PermissionDenied, "У вас нет доступа к этой загрузке")
		}

		// Открываем временный файл
		file, err := os.Open(upload.TempPath)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Ошибка при открытии временного файла: %v", err)
		}

		currentUpload = &ActiveUpload{
			file:       file,
			uploadInfo: upload,
			mutex:      sync.Mutex{},
		}
	}

	// Блокируем доступ к загрузке
	currentUpload.mutex.Lock()
	defer currentUpload.mutex.Unlock()

	// Закрываем файл
	currentUpload.file.Close()

	// Проверяем контрольную сумму файла
	calculatedChecksum, err := calculateMD5(currentUpload.uploadInfo.TempPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при вычислении контрольной суммы: %v", err)
	}

	if calculatedChecksum != req.Checksum {
		// Удаляем временный файл и запись о загрузке
		os.Remove(currentUpload.uploadInfo.TempPath)
		s.fileRepo.DeleteFileUpload(ctx, uploadID)

		// Удаляем из кэша активных загрузок
		s.uploadsMutex.Lock()
		delete(s.fileUploads, uploadID)
		s.uploadsMutex.Unlock()

		return nil, status.Errorf(codes.DataLoss, "Контрольная сумма не совпадает. Ожидалось %s, получено %s", req.Checksum, calculatedChecksum)
	}

	// Получаем информацию о размере файла
	fileInfo, err := os.Stat(currentUpload.uploadInfo.TempPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о файле: %v", err)
	}

	// Если размер не совпадает с ожидаемым, выдаем ошибку
	if fileInfo.Size() != currentUpload.uploadInfo.TotalSize {
		// Удаляем временный файл и запись о загрузке
		os.Remove(currentUpload.uploadInfo.TempPath)
		s.fileRepo.DeleteFileUpload(ctx, uploadID)

		// Удаляем из кэша активных загрузок
		s.uploadsMutex.Lock()
		delete(s.fileUploads, uploadID)
		s.uploadsMutex.Unlock()

		return nil, status.Errorf(codes.DataLoss, "Размер файла не совпадает. Ожидалось %d, получено %d", currentUpload.uploadInfo.TotalSize, fileInfo.Size())
	}

	// Создаем директорию для сохранения файла
	finalDir := filepath.Join(s.baseFilePath, "files", strconv.FormatUint(currentUpload.uploadInfo.ChatID, 10))
	err = os.MkdirAll(finalDir, 0755)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при создании директории: %v", err)
	}

	// Генерируем уникальный ID для файла
	fileID := uuid.New().String()

	// Определяем путь для сохранения файла
	ext := filepath.Ext(currentUpload.uploadInfo.FileName)
	finalPath := filepath.Join(finalDir, fileID+ext)

	// Перемещаем файл из временной директории в конечную
	err = os.Rename(currentUpload.uploadInfo.TempPath, finalPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при перемещении файла: %v", err)
	}

	// Создаем запись о файле в базе данных
	file := &entities.File{
		FileID:     fileID,
		FileName:   currentUpload.uploadInfo.FileName,
		MimeType:   currentUpload.uploadInfo.MimeType,
		Size:       currentUpload.uploadInfo.TotalSize,
		Path:       finalPath,
		UploadedBy: currentUpload.uploadInfo.UserID,
		ChatID:     currentUpload.uploadInfo.ChatID,
		Checksum:   calculatedChecksum,
		CreatedAt:  time.Now(),
	}

	err = s.fileRepo.CreateFile(ctx, file)
	if err != nil {
		// Если не удалось создать запись, удаляем файл
		os.Remove(finalPath)
		return nil, status.Errorf(codes.Internal, "Ошибка при создании записи о файле: %v", err)
	}

	// Удаляем запись о загрузке
	s.fileRepo.DeleteFileUpload(ctx, uploadID)

	// Удаляем из кэша активных загрузок
	s.uploadsMutex.Lock()
	delete(s.fileUploads, uploadID)
	s.uploadsMutex.Unlock()

	// Формируем URL для доступа к файлу
	// В реальном приложении здесь может быть логика для формирования публичного URL
	fileURL := fmt.Sprintf("/api/files/%s", fileID)

	return &pb.FinalizeFileUploadResponse{
		FileId:  fileID,
		Url:     fileURL,
		Success: true,
	}, nil
}

// GetFileInfo возвращает информацию о файле
func (s *FileService) GetFileInfo(ctx context.Context, req *pb.GetFileInfoRequest) (*pb.GetFileInfoResponse, error) {
	userID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Требуется аутентификация")
	}

	// Получаем информацию о файле
	file, err := s.fileRepo.GetFileByID(ctx, req.FileId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о файле: %v", err)
	}

	if file == nil {
		return nil, status.Errorf(codes.NotFound, "Файл не найден")
	}

	// Получаем информацию о чате, чтобы проверить права доступа
	chat, err := s.chatRepo.GetChatByID(ctx, file.ChatID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о чате: %v", err)
	}

	if chat == nil {
		return nil, status.Errorf(codes.NotFound, "Чат не найден")
	}

	// Проверяем, есть ли у пользователя доступ к чату
	if chat.FirstUserID != userID && chat.SecondUserID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "У вас нет доступа к этому файлу")
	}

	// Получаем имя пользователя, загрузившего файл
	uploader, err := s.userRepo.GetByID(ctx, file.UploadedBy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о пользователе: %v", err)
	}

	if uploader == nil {
		return nil, status.Errorf(codes.NotFound, "Пользователь не найден")
	}

	// Получаем имя пользователя чата
	var chatUsername string
	if chat.FirstUserID == userID {
		chatUsername = chat.SecondUsername
	} else {
		chatUsername = chat.FirstUsername
	}

	return &pb.GetFileInfoResponse{
		FileId:       file.FileID,
		Filename:     file.FileName,
		MimeType:     file.MimeType,
		Size:         file.Size,
		CreatedAt:    file.CreatedAt.Unix(),
		UploadedBy:   uploader.Username,
		ChatUsername: chatUsername,
	}, nil
}

// DownloadFile обрабатывает скачивание файла по частям
func (s *FileService) DownloadFile(req *pb.DownloadFileRequest, stream pb.FileService_DownloadFileServer) error {
	ctx := stream.Context()

	userID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "Требуется аутентификация")
	}

	// Получаем информацию о файле
	file, err := s.fileRepo.GetFileByID(ctx, req.FileId)
	if err != nil {
		return status.Errorf(codes.Internal, "Ошибка при получении информации о файле: %v", err)
	}

	if file == nil {
		return status.Errorf(codes.NotFound, "Файл не найден")
	}

	// Получаем информацию о чате, чтобы проверить права доступа
	chat, err := s.chatRepo.GetChatByID(ctx, file.ChatID)
	if err != nil {
		return status.Errorf(codes.Internal, "Ошибка при получении информации о чате: %v", err)
	}

	if chat == nil {
		return status.Errorf(codes.NotFound, "Чат не найден")
	}

	// Проверяем, есть ли у пользователя доступ к чату
	if chat.FirstUserID != userID && chat.SecondUserID != userID {
		return status.Errorf(codes.PermissionDenied, "У вас нет доступа к этому файлу")
	}

	// Открываем файл для чтения
	f, err := os.Open(file.Path)
	if err != nil {
		return status.Errorf(codes.Internal, "Ошибка при открытии файла: %v", err)
	}
	defer f.Close()

	// Определяем размер чанка для передачи
	chunkSize := int(req.ChunkSize)
	if chunkSize <= 0 {
		// Если размер чанка не указан или некорректен, используем размер по умолчанию (1 МБ)
		chunkSize = 1 * 1024 * 1024
	}

	// Буфер для чтения данных
	buffer := make([]byte, chunkSize)

	// Обрабатываем запрос на скачивание файла частями
	chunkIndex := 0
	for {
		// Читаем данные из файла
		bytesRead, err := f.Read(buffer)
		if err == io.EOF {
			// Достигнут конец файла
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "Ошибка при чтении файла: %v", err)
		}

		// Отправляем чанк клиенту
		err = stream.Send(&pb.FileChunk{
			UploadId:   req.FileId, // Используем FileId в качестве UploadId для скачивания
			ChunkIndex: int32(chunkIndex),
			Data:       buffer[:bytesRead],
		})
		if err != nil {
			return status.Errorf(codes.Internal, "Ошибка при отправке данных: %v", err)
		}

		chunkIndex++
	}

	return nil
}

// GetChatFiles возвращает список файлов в чате
func (s *FileService) GetChatFiles(ctx context.Context, req *pb.GetChatFilesRequest) (*pb.GetChatFilesResponse, error) {
	userID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Требуется аутентификация")
	}

	// Получаем информацию о чате
	chat, err := s.chatRepo.GetChatByUsername(ctx, req.ChatUsername)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о чате: %v", err)
	}

	if chat == nil {
		return nil, status.Errorf(codes.NotFound, "Чат не найден")
	}

	// Проверяем, есть ли у пользователя доступ к чату
	if chat.FirstUserID != userID && chat.SecondUserID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "У вас нет доступа к этому чату")
	}

	// Определяем параметры пагинации
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10 // Значение по умолчанию
	}

	// Получаем список файлов
	files, totalCount, err := s.fileRepo.GetFilesByChat(ctx, chat.ID, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении списка файлов: %v", err)
	}

	// Формируем ответ
	response := &pb.GetChatFilesResponse{
		TotalCount: int32(totalCount),
		Files:      make([]*pb.FileInfo, 0, len(files)),
	}

	// Кэш пользователей для оптимизации запросов
	userCache := make(map[uint64]*entities.User)

	for _, file := range files {
		// Получаем информацию о пользователе, загрузившем файл
		uploader, ok := userCache[file.UploadedBy]
		if !ok {
			uploader, err = s.userRepo.GetByID(ctx, file.UploadedBy)
			if err != nil || uploader == nil {
				// Если не удалось получить информацию о пользователе, пропускаем файл
				continue
			}
			userCache[file.UploadedBy] = uploader
		}

		response.Files = append(response.Files, &pb.FileInfo{
			FileId:     file.FileID,
			Filename:   file.FileName,
			MimeType:   file.MimeType,
			Size:       file.Size,
			CreatedAt:  file.CreatedAt.Unix(),
			UploadedBy: uploader.Username,
		})
	}

	return response, nil
}

// DeleteFile удаляет файл
func (s *FileService) DeleteFile(ctx context.Context, req *pb.DeleteFileRequest) (*pb.DeleteFileResponse, error) {
	userID, ok := ctx.Value(middleware.TokenKey("user_id")).(uint64)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Требуется аутентификация")
	}

	// Получаем информацию о файле
	file, err := s.fileRepo.GetFileByID(ctx, req.FileId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о файле: %v", err)
	}

	if file == nil {
		return nil, status.Errorf(codes.NotFound, "Файл не найден")
	}

	// Проверяем, является ли пользователь владельцем файла
	if file.UploadedBy != userID {
		// Получаем информацию о чате, чтобы проверить права доступа
		chat, err := s.chatRepo.GetChatByID(ctx, file.ChatID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Ошибка при получении информации о чате: %v", err)
		}

		if chat == nil {
			return nil, status.Errorf(codes.NotFound, "Чат не найден")
		}

		// Проверяем, есть ли у пользователя доступ к чату
		if chat.FirstUserID != userID && chat.SecondUserID != userID {
			return nil, status.Errorf(codes.PermissionDenied, "У вас нет доступа к этому файлу")
		}
	}

	// Удаляем файл из базы данных (мягкое удаление)
	err = s.fileRepo.DeleteFile(ctx, req.FileId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка при удалении файла из базы данных: %v", err)
	}

	// Возвращаем успешный ответ
	return &pb.DeleteFileResponse{
		Success: true,
	}, nil
}

// calculateMD5 вычисляет MD5-хеш файла
func calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashInBytes := hash.Sum(nil)
	md5String := hex.EncodeToString(hashInBytes)

	return md5String, nil
}
