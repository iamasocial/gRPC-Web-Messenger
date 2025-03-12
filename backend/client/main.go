package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	pb "client/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	address         = "localhost:50051"
	defaultUsername = "sas"
	defaultPassword = "qwerty"
	testChatUser    = "sas1"
	chunkSize       = 1024 * 1024 // 1 МБ
)

func main() {
	// Параметры командной строки
	upload := flag.Bool("upload", false, "Загрузить файл")
	download := flag.Bool("download", false, "Скачать файл")
	list := flag.Bool("list", false, "Получить список файлов")
	filePath := flag.String("file", "", "Путь к файлу для загрузки")
	fileId := flag.String("id", "", "ID файла для скачивания")
	outputPath := flag.String("output", "./downloads", "Путь для сохранения скачанного файла")
	flag.Parse()

	// Проверяем параметры
	if !*upload && !*download && !*list {
		log.Fatalf("Необходимо указать как минимум один флаг: -upload, -download или -list")
	}

	if *upload && *filePath == "" {
		log.Fatalf("Для загрузки необходимо указать путь к файлу (-file)")
	}

	if *download && *fileId == "" {
		log.Fatalf("Для скачивания необходимо указать ID файла (-id)")
	}

	// Подключаемся к серверу
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Не удалось подключиться: %v", err)
	}
	defer conn.Close()

	// Создаем клиентов для сервисов
	userClient := pb.NewUserServiceClient(conn)
	fileClient := pb.NewFileServiceClient(conn)

	// Аутентификация
	token, err := authenticate(userClient, defaultUsername, defaultPassword)
	if err != nil {
		log.Fatalf("Ошибка аутентификации: %v", err)
	}
	log.Printf("Аутентификация успешна, получен токен: %s\n", token)

	// Создаем контекст с токеном
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", token)),
	)

	// Выполняем операции в зависимости от флагов
	if *upload {
		fileID, err := uploadFile(ctx, fileClient, *filePath, testChatUser)
		if err != nil {
			log.Fatalf("Ошибка при загрузке файла: %v", err)
		}
		log.Printf("Файл успешно загружен, ID: %s\n", fileID)
	}

	if *list {
		files, err := listFiles(ctx, fileClient, testChatUser)
		if err != nil {
			log.Fatalf("Ошибка при получении списка файлов: %v", err)
		}
		log.Printf("Получен список файлов (%d):\n", len(files))
		for i, file := range files {
			log.Printf("%d. %s (ID: %s, размер: %d байт)\n", i+1, file.Filename, file.FileId, file.Size)
		}
	}

	if *download {
		err := downloadFile(ctx, fileClient, *fileId, *outputPath)
		if err != nil {
			log.Fatalf("Ошибка при скачивании файла: %v", err)
		}
		log.Printf("Файл успешно скачан и сохранен в %s\n", *outputPath)
	}
}

// authenticate выполняет аутентификацию и возвращает токен
func authenticate(client pb.UserServiceClient, username, password string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Login(ctx, &pb.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", err
	}

	return resp.Token, nil
}

// uploadFile загружает файл на сервер
func uploadFile(ctx context.Context, client pb.FileServiceClient, filePath, chatUsername string) (string, error) {
	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("ошибка при открытии файла: %v", err)
	}
	defer file.Close()

	// Получаем информацию о файле
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("ошибка при получении информации о файле: %v", err)
	}

	// Определяем MIME-тип (упрощенно)
	mimeType := "application/octet-stream"
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".pdf":
		mimeType = "application/pdf"
	case ".txt":
		mimeType = "text/plain"
	}

	// Инициализируем загрузку
	initResp, err := client.InitFileUpload(ctx, &pb.InitFileUploadRequest{
		Filename:     filepath.Base(filePath),
		MimeType:     mimeType,
		TotalSize:    fileInfo.Size(),
		ChatUsername: chatUsername,
	})
	if err != nil {
		return "", fmt.Errorf("ошибка при инициализации загрузки: %v", err)
	}

	log.Printf("Загрузка инициализирована, ID: %s\n", initResp.UploadId)

	// Создаем поток для загрузки чанков
	stream, err := client.UploadFileChunk(ctx)
	if err != nil {
		return "", fmt.Errorf("ошибка при создании потока загрузки: %v", err)
	}

	// Читаем и отправляем файл по частям
	buffer := make([]byte, chunkSize)
	chunkIndex := 0
	hasher := md5.New() // Для вычисления хеша файла

	for {
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ошибка при чтении файла: %v", err)
		}

		// Обновляем хеш
		hasher.Write(buffer[:bytesRead])

		// Отправляем чанк
		err = stream.Send(&pb.FileChunk{
			UploadId:   initResp.UploadId,
			ChunkIndex: int32(chunkIndex),
			Data:       buffer[:bytesRead],
		})
		if err != nil {
			return "", fmt.Errorf("ошибка при отправке чанка: %v", err)
		}

		log.Printf("Отправлен чанк %d, размер: %d байт\n", chunkIndex, bytesRead)
		chunkIndex++
	}

	// Закрываем поток и получаем ответ
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", fmt.Errorf("ошибка при завершении потока загрузки: %v", err)
	}

	log.Printf("Все чанки отправлены, получено: %d\n", resp.ReceivedChunks)

	// Вычисляем MD5-хеш файла
	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Финализируем загрузку
	finalResp, err := client.FinalizeFileUpload(ctx, &pb.FinalizeFileUploadRequest{
		UploadId: initResp.UploadId,
		Checksum: checksum,
	})
	if err != nil {
		return "", fmt.Errorf("ошибка при финализации загрузки: %v", err)
	}

	return finalResp.FileId, nil
}

// listFiles получает список файлов в чате
func listFiles(ctx context.Context, client pb.FileServiceClient, chatUsername string) ([]*pb.FileInfo, error) {
	resp, err := client.GetChatFiles(ctx, &pb.GetChatFilesRequest{
		ChatUsername: chatUsername,
		Page:         1,
		PageSize:     10,
	})
	if err != nil {
		return nil, err
	}

	return resp.Files, nil
}

// downloadFile скачивает файл с сервера
func downloadFile(ctx context.Context, client pb.FileServiceClient, fileID, outputPath string) error {
	// Получаем информацию о файле
	fileInfo, err := client.GetFileInfo(ctx, &pb.GetFileInfoRequest{
		FileId: fileID,
	})
	if err != nil {
		return fmt.Errorf("ошибка при получении информации о файле: %v", err)
	}

	log.Printf("Информация о файле: %s, размер: %d байт\n", fileInfo.Filename, fileInfo.Size)

	// Создаем директорию для скачанных файлов, если она не существует
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		return fmt.Errorf("ошибка при создании директории: %v", err)
	}

	// Создаем файл для записи скачиваемых данных
	outputFilePath := filepath.Join(outputPath, fileInfo.Filename)
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("ошибка при создании файла: %v", err)
	}
	defer outputFile.Close()

	// Создаем поток для скачивания
	stream, err := client.DownloadFile(ctx, &pb.DownloadFileRequest{
		FileId:    fileID,
		ChunkSize: int32(chunkSize),
	})
	if err != nil {
		return fmt.Errorf("ошибка при создании потока скачивания: %v", err)
	}

	// Получаем и записываем чанки
	totalBytes := 0
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ошибка при получении чанка: %v", err)
		}

		// Записываем данные в файл
		_, err = outputFile.Write(chunk.Data)
		if err != nil {
			return fmt.Errorf("ошибка при записи данных: %v", err)
		}

		totalBytes += len(chunk.Data)
		log.Printf("Получен чанк %d, размер: %d байт, всего: %d байт\n", chunk.ChunkIndex, len(chunk.Data), totalBytes)
	}

	log.Printf("Файл успешно скачан, сохранен в %s, общий размер: %d байт\n", outputFilePath, totalBytes)
	return nil
}
