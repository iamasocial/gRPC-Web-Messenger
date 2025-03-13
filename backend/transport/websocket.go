package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	pb "gRPCWebServer/backend/generated"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Message представляет общую структуру сообщения
type Message struct {
	Type           string `json:"type,omitempty"`
	Content        string `json:"content,omitempty"`
	SenderUsername string `json:"senderUsername,omitempty"`
	Timestamp      int64  `json:"timestamp,omitempty"`

	// Поля для файлов
	FileId      string `json:"fileId,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	FileSize    int64  `json:"fileSize,omitempty"`
	MessageType string `json:"messageType,omitempty"`

	// Поля для загрузки файлов
	UploadId       string `json:"uploadId,omitempty"`
	ChunkIndex     int32  `json:"chunkIndex,omitempty"`
	ReceivedChunks int32  `json:"receivedChunks,omitempty"`
	Data           []byte `json:"-"`                  // Не сериализуем в JSON
	DataString     string `json:"data,omitempty"`     // Для строковых данных (например, base64)
	Encoding       string `json:"encoding,omitempty"` // Тип кодирования данных
	IsLastChunk    bool   `json:"isLastChunk,omitempty"`

	// Поля для инициализации загрузки
	MimeType     string `json:"mimeType,omitempty"`
	TotalSize    int64  `json:"totalSize,omitempty"`
	ChatUsername string `json:"chatUsername,omitempty"`

	// Поля для ошибок
	Error string `json:"error,omitempty"`
}

// FileUpload представляет информацию о загружаемом файле
type FileUpload struct {
	UploadId       string
	FileName       string
	MimeType       string
	TotalSize      int64
	ChatUsername   string
	FilePath       string
	ReceivedChunks int32
	UserId         string
}

type WebSocketHandler struct {
	upgrader    websocket.Upgrader
	connections sync.Map
	client      pb.ChatServiceClient
	fileClient  pb.FileServiceClient

	// Для работы с файлами
	fileUploads  map[string]*FileUpload
	uploadsMutex sync.RWMutex
	baseFilePath string

	// Мьютекс для защиты записи в WebSocket
	writeMutex sync.Mutex
}

func NewWebSocketHandler() *WebSocketHandler {
	// Создаем директорию для временных файлов
	tempDir := "./storage/temp"
	os.MkdirAll(tempDir, 0755)

	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		fileUploads:  make(map[string]*FileUpload),
		uploadsMutex: sync.RWMutex{},
		baseFilePath: "./storage",
	}
}

func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Sec-WebSocket-Protocol")

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	grpcConn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to gRPC server: %v", err)
		h.writeMutex.Lock()
		conn.WriteMessage(websocket.TextMessage, []byte("Failed to connect to chat server"))
		h.writeMutex.Unlock()
		return
	}
	defer grpcConn.Close() // Закрываем соединение при выходе из функции

	h.client = pb.NewChatServiceClient(grpcConn)
	h.fileClient = pb.NewFileServiceClient(grpcConn)

	h.connections.Store(1, conn)
	defer h.connections.Delete(1)

	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "Authorization", "Bearer "+token)

	stream, err := h.client.Chat(ctx)
	if err != nil {
		log.Println("Error starting Chat stream:", err)
		h.writeMutex.Lock()
		conn.WriteMessage(websocket.TextMessage, []byte("Errror starting chat stream"))
		h.writeMutex.Unlock()
		return
	}

	// Логируем успешное соединение
	log.Println("WebSocket connection established")

	// sending messages
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading WebSocket message:", err)
				return
			}

			var message Message
			if err := json.Unmarshal(msg, &message); err != nil {
				log.Println("Error parsing JSON:", err)
				continue
			}

			// Если сообщение имеет тип file_chunk и не имеет DataString, попробуем получить поле data
			if message.Type == "file_chunk" && message.DataString == "" {
				var rawMsg map[string]interface{}
				if err := json.Unmarshal(msg, &rawMsg); err == nil {
					if dataVal, ok := rawMsg["data"]; ok {
						// Обработка в зависимости от типа данных
						switch data := dataVal.(type) {
						case string:
							// Это строка base64
							message.DataString = data
							message.Encoding = "base64"
						case []interface{}:
							// Это массив байтов
							byteArray := make([]byte, len(data))
							for i, v := range data {
								if num, ok := v.(float64); ok {
									byteArray[i] = byte(num)
								}
							}
							message.Data = byteArray
						}
					}
				}
			}

			// Обработка различных типов сообщений
			switch message.Type {
			case "file_upload_init":
				// Инициализация загрузки файла
				h.handleFileUploadInit(conn, message, token)

			case "file_chunk":
				// Обработка чанка файла
				h.handleFileChunk(conn, message)

			case "file_download_request":
				// Запрос на скачивание файла
				h.handleFileDownload(conn, message)

			default:
				// Стандартное текстовое сообщение для чата
				chatMessage := pb.ChatMessage{
					Content: message.Content,
				}

				// Если это файловое сообщение, добавляем метаинформацию о файле в Content
				if message.MessageType == "file" && message.FileId != "" {
					// Создаем специальный префикс для идентификации файлового сообщения
					fileInfo := fmt.Sprintf("[FILE:%s:%s:%d]%s",
						message.FileId,
						message.FileName,
						message.FileSize,
						message.Content)

					// Заменяем контент сообщения структурированной информацией о файле
					chatMessage.Content = fileInfo
				}

				if err := stream.Send(&chatMessage); err != nil {
					log.Println("Error sending message to gRPC stream:", err)
					return
				}
			}
		}
	}()

	// receiving messages
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				log.Println("Error receiving message from gRPC stream", err)
				return
			}

			messageJSON, err := json.Marshal(map[string]interface{}{
				"senderUsername": resp.Senderusername,
				"content":        resp.Content,
				"timestamp":      resp.Timestamp,
			})
			if err != nil {
				log.Println("Error marshalling JSON:", err)
				return
			}

			h.writeMutex.Lock()
			err = conn.WriteMessage(websocket.TextMessage, messageJSON)
			h.writeMutex.Unlock()

			if err != nil {
				log.Println("Error sending WebSocket message:", err)
				return
			}
		}
	}()

	select {}
}

// Обработка инициализации загрузки файла
func (h *WebSocketHandler) handleFileUploadInit(conn *websocket.Conn, message Message, token string) {
	// Создаем новую загрузку
	uploadId := message.UploadId
	if uploadId == "" {
		uploadId = uuid.New().String()
	}

	// Создаем временный файл
	tempFilePath := filepath.Join(h.baseFilePath, "temp", uploadId)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		log.Printf("Error creating temp file: %v", err)
		h.sendError(conn, "file_upload_error", uploadId, "Ошибка создания временного файла")
		return
	}
	tempFile.Close()

	h.uploadsMutex.Lock()
	h.fileUploads[uploadId] = &FileUpload{
		UploadId:       uploadId,
		FileName:       message.FileName,
		MimeType:       message.MimeType,
		TotalSize:      message.TotalSize,
		ChatUsername:   message.ChatUsername,
		FilePath:       tempFilePath,
		ReceivedChunks: 0,
		UserId:         token, // используем токен для идентификации пользователя
	}
	h.uploadsMutex.Unlock()

	// Отправляем подтверждение
	response := Message{
		Type:     "file_upload_initialized",
		UploadId: uploadId,
	}
	h.sendMessage(conn, response)
}

// Обработка чанка файла
func (h *WebSocketHandler) handleFileChunk(conn *websocket.Conn, message Message) {
	uploadId := message.UploadId

	h.uploadsMutex.RLock()
	upload, exists := h.fileUploads[uploadId]
	h.uploadsMutex.RUnlock()

	if !exists {
		h.sendError(conn, "file_upload_error", uploadId, "Загрузка не найдена")
		return
	}

	// Открываем файл для добавления данных
	file, err := os.OpenFile(upload.FilePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Error opening temp file: %v", err)
		h.sendError(conn, "file_upload_error", uploadId, "Ошибка открытия временного файла")
		return
	}
	defer file.Close()

	// Данные для записи в файл
	var dataToWrite []byte

	// Проверяем, как закодированы данные
	if message.Encoding == "base64" {
		// Если данные закодированы в base64, декодируем их
		decoded, err := base64.StdEncoding.DecodeString(message.DataString)
		if err != nil {
			log.Printf("Error decoding base64 data: %v", err)
			h.sendError(conn, "file_upload_error", uploadId, "Ошибка декодирования данных")
			return
		}
		dataToWrite = decoded
		log.Printf("Received base64 chunk for upload %s, size after decode: %d bytes", uploadId, len(dataToWrite))
	} else {
		// Совместимость с предыдущей версией - используем бинарные данные
		dataToWrite = message.Data
		log.Printf("Received binary chunk for upload %s, size: %d bytes", uploadId, len(dataToWrite))
	}

	// Проверяем, есть ли данные для записи
	if len(dataToWrite) == 0 {
		log.Printf("Warning: empty chunk data received for upload %s", uploadId)
	}

	// Записываем данные
	if _, err := file.Write(dataToWrite); err != nil {
		log.Printf("Error writing to temp file: %v", err)
		h.sendError(conn, "file_upload_error", uploadId, "Ошибка записи данных")
		return
	}

	// Обновляем счетчик полученных чанков
	h.uploadsMutex.Lock()
	h.fileUploads[uploadId].ReceivedChunks++
	receivedChunks := h.fileUploads[uploadId].ReceivedChunks
	h.uploadsMutex.Unlock()

	// Отправляем подтверждение получения чанка
	response := Message{
		Type:           "file_chunk_received",
		UploadId:       uploadId,
		ChunkIndex:     message.ChunkIndex,
		ReceivedChunks: receivedChunks,
	}
	h.sendMessage(conn, response)

	// Если это последний чанк, финализируем загрузку
	if message.IsLastChunk {
		h.finalizeFileUpload(conn, uploadId)
	}
}

// Финализация загрузки файла
func (h *WebSocketHandler) finalizeFileUpload(conn *websocket.Conn, uploadId string) {
	h.uploadsMutex.RLock()
	upload, exists := h.fileUploads[uploadId]
	h.uploadsMutex.RUnlock()

	if !exists {
		h.sendError(conn, "file_upload_error", uploadId, "Загрузка не найдена")
		return
	}

	// Проверяем размер файла
	fileStats, err := os.Stat(upload.FilePath)
	if err != nil {
		log.Printf("Error checking file stats: %v", err)
	} else {
		log.Printf("Temp file size: %d bytes, expected size: %d bytes", fileStats.Size(), upload.TotalSize)
		if fileStats.Size() == 0 {
			log.Printf("WARNING: File is empty! This indicates a problem with upload.")
		}
	}

	// Перемещаем файл из временной директории в постоянную
	finalDir := filepath.Join(h.baseFilePath, "files")
	os.MkdirAll(finalDir, 0755)

	fileId := uuid.New().String()
	finalPath := filepath.Join(finalDir, fileId)

	if err := os.Rename(upload.FilePath, finalPath); err != nil {
		log.Printf("Error moving file: %v", err)
		h.sendError(conn, "file_upload_error", uploadId, "Ошибка перемещения файла")
		return
	}

	// Создаем и сохраняем метаданные файла
	metaPath := filepath.Join(finalDir, fileId+".meta")
	metaFile, err := os.Create(metaPath)
	if err != nil {
		log.Printf("Error creating metadata file: %v", err)
	} else {
		defer metaFile.Close()
		metaData := struct {
			FileName string `json:"fileName"`
			MimeType string `json:"mimeType"`
			FileSize int64  `json:"fileSize"`
		}{
			FileName: upload.FileName,
			MimeType: upload.MimeType,
			FileSize: upload.TotalSize,
		}

		metaBytes, err := json.Marshal(metaData)
		if err != nil {
			log.Printf("Error marshaling metadata: %v", err)
		} else {
			if _, err := metaFile.Write(metaBytes); err != nil {
				log.Printf("Error writing metadata: %v", err)
			}
		}
	}

	// Отправляем сообщение о завершении загрузки
	response := Message{
		Type:     "file_upload_complete",
		UploadId: uploadId,
		FileId:   fileId,
		FileName: upload.FileName, // Важно: включаем имя файла в ответ
	}
	h.sendMessage(conn, response)

	// Удаляем информацию о загрузке
	h.uploadsMutex.Lock()
	delete(h.fileUploads, uploadId)
	h.uploadsMutex.Unlock()

	// Логируем успешную загрузку
	log.Printf("File uploaded successfully: %s, size: %d, type: %s",
		upload.FileName, upload.TotalSize, upload.MimeType)
}

// Обработка запроса на скачивание файла
func (h *WebSocketHandler) handleFileDownload(conn *websocket.Conn, message Message) {
	fileId := message.FileId

	// Получаем метаданные файла, если они есть
	filePath := filepath.Join(h.baseFilePath, "files", fileId)
	metaPath := filepath.Join(h.baseFilePath, "files", fileId+".meta")

	// Инициализируем информацию о файле со значениями по умолчанию
	fileInfo := Message{
		Type:     "file_info",
		FileId:   fileId,
		FileName: "downloaded_file", // Значение по умолчанию
		MimeType: "application/octet-stream",
		FileSize: 0,
	}

	// Пробуем загрузить метаданные
	if metaBytes, err := os.ReadFile(metaPath); err == nil {
		var metaData struct {
			FileName string `json:"fileName"`
			MimeType string `json:"mimeType"`
			FileSize int64  `json:"fileSize"`
		}

		if err := json.Unmarshal(metaBytes, &metaData); err == nil {
			fileInfo.FileName = metaData.FileName
			fileInfo.MimeType = metaData.MimeType
			fileInfo.FileSize = metaData.FileSize
		}
	} else {
		// Если метаданные не найдены, используем значение по умолчанию
		fileInfo.FileName = filepath.Base(filePath)
	}

	// Проверяем существование файла
	fileStats, err := os.Stat(filePath)
	if err != nil {
		// Если файл не найден, проверяем другие возможные расширения
		found := false
		possibleExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt", ".doc", ".docx", ".xls", ".xlsx", ".zip"}

		for _, ext := range possibleExtensions {
			possiblePath := filepath.Join(h.baseFilePath, "files", fileId+ext)
			if stats, err := os.Stat(possiblePath); err == nil {
				filePath = possiblePath
				fileStats = stats
				found = true
				break
			}
		}

		if !found {
			h.sendError(conn, "file_download_error", fileId, "Файл не найден")
			return
		}
	}

	// Используем размер из статистики файла
	fileInfo.FileSize = fileStats.Size()
	h.sendMessage(conn, fileInfo)

	// Отправляем файл по частям
	file, err := os.Open(filePath)
	if err != nil {
		h.sendError(conn, "file_download_error", fileId, "Ошибка открытия файла")
		return
	}
	defer file.Close()

	buffer := make([]byte, 64*1024) // 64 КБ чанки
	chunkIndex := int32(0)

	for {
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			h.sendError(conn, "file_download_error", fileId, "Ошибка чтения файла")
			return
		}

		// Определяем, последний ли это чанк
		isLastChunk := bytesRead < len(buffer)

		// Логируем информацию о чанке для отладки
		log.Printf("Sending chunk %d for file %s, size: %d bytes", chunkIndex, fileId, bytesRead)

		// Кодируем данные в base64 для безопасной передачи через JSON
		base64Data := base64.StdEncoding.EncodeToString(buffer[:bytesRead])

		// Отправляем чанк
		chunk := Message{
			Type:        "file_chunk",
			FileId:      fileId,
			ChunkIndex:  chunkIndex,
			DataString:  base64Data,
			Encoding:    "base64",
			IsLastChunk: isLastChunk,
		}
		h.sendMessage(conn, chunk)

		chunkIndex++

		// Небольшая задержка, чтобы не перегружать соединение
		time.Sleep(20 * time.Millisecond)
	}

	log.Printf("File %s download complete, sent %d chunks", fileId, chunkIndex)
}

// Отправка сообщения клиенту
func (h *WebSocketHandler) sendMessage(conn *websocket.Conn, message Message) {
	h.writeMutex.Lock()
	defer h.writeMutex.Unlock()

	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling message: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
		log.Printf("Error sending WebSocket message: %v", err)
	}
}

// Отправка сообщения об ошибке
func (h *WebSocketHandler) sendError(conn *websocket.Conn, errorType string, id string, errorMessage string) {
	message := Message{
		Type:  errorType,
		Error: errorMessage,
	}

	// Если указан идентификатор, добавляем его в сообщение
	if id != "" {
		switch errorType {
		case "file_upload_error":
			message.UploadId = id
		case "file_download_error":
			message.FileId = id
		}
	}

	h.sendMessage(conn, message)
}
