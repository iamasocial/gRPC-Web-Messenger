package tests

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	pb "client/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestKeyExchange(t *testing.T) {
	// Подключаемся к серверу
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	// Создаем клиентов для сервисов
	userClient := pb.NewUserServiceClient(conn)
	keyExchangeClient := pb.NewKeyExchangeServiceClient(conn)

	// Аутентификация
	loginResp, err := userClient.Login(context.Background(), &pb.LoginRequest{
		Username: "sas",
		Password: "qwerty",
	})
	if err != nil {
		t.Fatalf("ошибка при аутентификации: %v", err)
	}

	// Создаем контекст с токеном
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("authorization", "Bearer "+loginResp.Token),
	)

	// Шаг 1: Инициируем обмен ключами
	initResp, err := keyExchangeClient.InitiateKeyExchange(ctx, &pb.InitiateKeyExchangeRequest{})
	if err != nil {
		t.Fatalf("ошибка при инициировании обмена ключами: %v", err)
	}

	// Преобразуем полученные параметры в big.Int
	prime := new(big.Int).SetBytes(initResp.Prime)
	generator := new(big.Int).SetBytes(initResp.Generator)
	serverPublic := new(big.Int).SetBytes(initResp.ServerPublic)

	// Генерируем случайное число b (приватный ключ клиента)
	privateKey := new(big.Int)
	privateKey, err = rand.Int(rand.Reader, prime)
	if err != nil {
		t.Fatalf("ошибка при генерации приватного ключа: %v", err)
	}

	// Вычисляем публичный ключ клиента: B = g^b mod p
	clientPublic := new(big.Int).Exp(generator, privateKey, prime)

	// Шаг 2: Завершаем обмен ключами
	completeResp, err := keyExchangeClient.CompleteKeyExchange(ctx, &pb.CompleteKeyExchangeRequest{
		ClientPublic: clientPublic.Bytes(),
		SessionId:    initResp.SessionId,
	})
	if err != nil {
		t.Fatalf("ошибка при завершении обмена ключами: %v", err)
	}

	if !completeResp.Success {
		t.Fatal("обмен ключами не удался")
	}

	// Проверяем, что общий секрет совпадает с тем, что мы можем вычислить
	sharedSecret := new(big.Int).Exp(serverPublic, privateKey, prime)
	if sharedSecret.Cmp(new(big.Int).SetBytes(completeResp.SharedSecret)) != 0 {
		t.Fatal("общие секреты не совпадают")
	}

	t.Log("Обмен ключами успешно завершен")
	t.Logf("Общий секрет: %x", sharedSecret.Bytes())
}
