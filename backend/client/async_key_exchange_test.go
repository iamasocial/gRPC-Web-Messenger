package main

import (
	"context"
	"crypto/rand"
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	pb "dhclient/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Константы и переменные для асинхронного тестирования
const (
	asyncServerAddr = "localhost:50051"
)

var (
	asyncUser1Username = "async_user1"
	asyncUser1Password = "password123"
	asyncUser1Token    string // Будет получен при первой авторизации
	asyncUser1Token2   string // Будет получен при повторной авторизации

	asyncUser2Username = "async_user2"
	asyncUser2Password = "password123"
	asyncUser2Token    string

	// Для хранения приватных ключей и параметров
	asyncUser1PrivateKey *big.Int
	asyncUser2PrivateKey *big.Int
	asyncGenerator       = big.NewInt(2)
	asyncPrime, _        = new(big.Int).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF", 16)
)

// Функция для имитации параметров Диффи-Хеллмана
type DHParams struct {
	G       string // Генератор
	P       string // Простое число
	A       string // Публичный ключ A
	B       string // Публичный ключ B
	Status  pb.KeyExchangeStatus
	Success bool
}

// Тест на асинхронный обмен ключами
func TestAsynchronousKeyExchange(t *testing.T) {
	// Подготовка - регистрация пользователей
	setupAsyncUsers(t)

	log.Println("======= НАЧАЛО ТЕСТА АСИНХРОННОГО ОБМЕНА КЛЮЧАМИ =======")

	// ШАГ 1: Пользователь 1 авторизуется и инициирует обмен ключами
	log.Println("ШАГ 1: Пользователь 1 авторизуется")
	asyncUser1Token = loginAndGetToken(t, asyncUser1Username, asyncUser1Password)
	log.Printf("Пользователь '%s' успешно авторизован, получен токен", asyncUser1Username)

	// Создаем чат между пользователями
	err := createChatBetweenUsers(t, asyncUser1Token, asyncUser2Username)
	if err != nil {
		log.Printf("Чат между '%s' и '%s' уже существует или произошла ошибка: %v",
			asyncUser1Username, asyncUser2Username, err)
	} else {
		log.Printf("Чат между '%s' и '%s' успешно создан", asyncUser1Username, asyncUser2Username)
	}

	// Инициируем обмен ключами
	log.Println("ШАГ 2: Пользователь 1 инициирует обмен ключами")
	privateKeyA, publicKeyA, err := generateAsyncDHKeyPair()
	if err != nil {
		t.Fatalf("Ошибка при генерации ключей: %v", err)
	}
	asyncUser1PrivateKey = privateKeyA

	err = initiateKeyExchange(t, asyncUser1Token, asyncUser2Username, asyncGenerator, asyncPrime, publicKeyA)
	if err != nil {
		t.Fatalf("Ошибка при инициировании обмена ключами: %v", err)
	}
	log.Printf("Пользователь '%s' успешно инициировал обмен ключами с '%s'",
		asyncUser1Username, asyncUser2Username)

	// Сохраняем идентификатор и закрываем соединение
	log.Println("ШАГ 3: Пользователь 1 отключается")
	log.Println("Пользователь 1 сохранил свой приватный ключ и вышел из системы")
	// В реальном приложении здесь бы произошел выход из системы и закрытие соединения
	// В нашем тесте мы просто "забываем" токен, будто пользователь вышел
	asyncUser1Token = ""

	// Имитация задержки - в реальном мире пользователи обычно не действуют мгновенно
	time.Sleep(1 * time.Second)

	// ШАГ 4: Пользователь 2 авторизуется и получает параметры обмена ключами
	log.Println("ШАГ 4: Пользователь 2 авторизуется")
	asyncUser2Token = loginAndGetToken(t, asyncUser2Username, asyncUser2Password)
	log.Printf("Пользователь '%s' успешно авторизован, получен токен", asyncUser2Username)

	// Получаем параметры обмена ключами
	log.Println("ШАГ 5: Пользователь 2 получает параметры обмена ключами")
	params, err := getKeyExchangeParameters(t, asyncUser2Token, asyncUser1Username)
	if err != nil {
		t.Fatalf("Ошибка при получении параметров обмена ключами: %v", err)
	}

	if !params.Success {
		t.Fatalf("Не удалось получить параметры обмена ключами")
	}

	if params.Status != pb.KeyExchangeStatus_INITIATED {
		t.Fatalf("Неверный статус обмена ключами: %v, ожидалось INITIATED", params.Status)
	}

	log.Printf("Пользователь '%s' успешно получил параметры обмена ключами от '%s'",
		asyncUser2Username, asyncUser1Username)
	log.Printf("Статус обмена ключами: %v", params.Status)

	// ШАГ 6: Пользователь 2 завершает обмен ключами
	log.Println("ШАГ 6: Пользователь 2 завершает обмен ключами своей стороны")
	// Генерируем ключи для второго пользователя
	g := new(big.Int)
	g.SetString(params.DhG, 10)

	p := new(big.Int)
	p.SetString(params.DhP, 10)

	a := new(big.Int)
	a.SetString(params.DhAPublic, 10)

	// Генерируем приватный ключ для второго пользователя
	privateKeyB, err := rand.Int(rand.Reader, p)
	if err != nil {
		t.Fatalf("Ошибка при генерации приватного ключа: %v", err)
	}
	asyncUser2PrivateKey = privateKeyB

	// Вычисляем публичный ключ B = g^b mod p
	publicKeyB := new(big.Int).Exp(g, privateKeyB, p)

	// Завершаем обмен ключами
	err = completeKeyExchangeProcess(t, asyncUser2Token, asyncUser1Username, publicKeyB)
	if err != nil {
		t.Fatalf("Ошибка при завершении обмена ключами: %v", err)
	}

	log.Printf("Пользователь '%s' успешно завершил обмен ключами с '%s'",
		asyncUser2Username, asyncUser1Username)

	// Вычисляем общий секретный ключ для пользователя 2
	sharedSecretB := new(big.Int).Exp(a, privateKeyB, p)
	log.Printf("Общий секретный ключ (вычисленный пользователем 2): %s... (обрезано)",
		sharedSecretB.String()[:50])

	// Сохраняем этот секретный ключ для последующего сравнения
	err = os.WriteFile("async_shared_secret_user2.txt", []byte(sharedSecretB.String()), 0644)
	if err != nil {
		t.Fatalf("Ошибка при сохранении общего секрета: %v", err)
	}

	// ШАГ 7: Пользователь 2 отключается
	log.Println("ШАГ 7: Пользователь 2 отключается")
	asyncUser2Token = ""

	// Имитация задержки
	time.Sleep(1 * time.Second)

	// ШАГ 8: Пользователь 1 заново авторизуется
	log.Println("ШАГ 8: Пользователь 1 заново авторизуется")
	asyncUser1Token2 = loginAndGetToken(t, asyncUser1Username, asyncUser1Password)
	log.Printf("Пользователь '%s' успешно авторизован заново, получен новый токен", asyncUser1Username)

	// ШАГ 9: Пользователь 1 проверяет статус обмена ключами
	log.Println("ШАГ 9: Пользователь 1 проверяет статус обмена ключами")
	params, err = getKeyExchangeParameters(t, asyncUser1Token2, asyncUser2Username)
	if err != nil {
		t.Fatalf("Ошибка при получении параметров обмена ключами: %v", err)
	}

	if !params.Success {
		t.Fatalf("Не удалось получить параметры обмена ключами")
	}

	if params.Status != pb.KeyExchangeStatus_COMPLETED {
		t.Fatalf("Неверный статус обмена ключами: %v, ожидалось COMPLETED", params.Status)
	}

	log.Printf("Пользователь '%s' успешно получил параметры завершенного обмена ключами",
		asyncUser1Username)
	log.Printf("Статус обмена ключами: %v", params.Status)

	// Теперь пользователь 1 может вычислить общий секретный ключ
	publicKeyB = new(big.Int)
	publicKeyB.SetString(params.DhBPublic, 10)

	// Вычисляем общий секретный ключ для пользователя 1
	sharedSecretA := new(big.Int).Exp(publicKeyB, asyncUser1PrivateKey, p)
	log.Printf("Общий секретный ключ (вычисленный пользователем 1): %s... (обрезано)",
		sharedSecretA.String()[:50])

	// Сохраняем этот секретный ключ для последующего сравнения
	err = os.WriteFile("async_shared_secret_user1.txt", []byte(sharedSecretA.String()), 0644)
	if err != nil {
		t.Fatalf("Ошибка при сохранении общего секрета: %v", err)
	}

	// ШАГ 10: Проверяем, совпадают ли общие секретные ключи
	log.Println("ШАГ 10: Проверяем совпадение общих секретных ключей")
	// Читаем общий секрет пользователя 1
	secretAData, err := os.ReadFile("async_shared_secret_user1.txt")
	if err != nil {
		t.Fatalf("Ошибка при чтении общего секрета пользователя 1: %v", err)
	}

	// Читаем общий секрет пользователя 2
	secretBData, err := os.ReadFile("async_shared_secret_user2.txt")
	if err != nil {
		t.Fatalf("Ошибка при чтении общего секрета пользователя 2: %v", err)
	}

	secretA := new(big.Int)
	secretA.SetString(string(secretAData), 10)

	secretB := new(big.Int)
	secretB.SetString(string(secretBData), 10)

	// Сравниваем общие секреты
	if secretA.Cmp(secretB) != 0 {
		t.Fatalf("Общие секреты не совпадают: %s != %s", secretA.String(), secretB.String())
	}

	log.Println("ТЕСТ ПРОЙДЕН: Общие секретные ключи пользователей совпадают даже при асинхронном обмене!")
	log.Printf("Общий секрет: %s... (обрезано)", secretA.String()[:50])
	log.Println("======= ЗАВЕРШЕНИЕ ТЕСТА АСИНХРОННОГО ОБМЕНА КЛЮЧАМИ =======")
}

// Вспомогательные функции для асинхронного теста

// Регистрация тестовых пользователей
func setupAsyncUsers(t *testing.T) {
	// Регистрируем первого пользователя
	conn, err := grpc.Dial(asyncServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	_, err = client.Register(context.Background(), &pb.RegisterRequest{
		Username: asyncUser1Username,
		Password: asyncUser1Password,
	})
	if err != nil {
		log.Printf("Пользователь '%s' уже существует или произошла ошибка: %v", asyncUser1Username, err)
	}

	// Регистрируем второго пользователя
	_, err = client.Register(context.Background(), &pb.RegisterRequest{
		Username: asyncUser2Username,
		Password: asyncUser2Password,
	})
	if err != nil {
		log.Printf("Пользователь '%s' уже существует или произошла ошибка: %v", asyncUser2Username, err)
	}
}

// Авторизация и получение токена
func loginAndGetToken(t *testing.T, username, password string) string {
	conn, err := grpc.Dial(asyncServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	resp, err := client.Login(context.Background(), &pb.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		t.Fatalf("Ошибка при входе пользователя '%s': %v", username, err)
	}
	return resp.Token
}

// Создание чата между пользователями
func createChatBetweenUsers(t *testing.T, token, receiverUsername string) error {
	conn, err := grpc.Dial(asyncServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("Authorization", "Bearer "+token),
	)

	client := pb.NewChatServiceClient(conn)
	_, err = client.CreateChat(ctx, &pb.CreateChatRequest{
		Username: receiverUsername,
	})
	return err
}

// Генерация пары ключей для Диффи-Хеллмана
func generateAsyncDHKeyPair() (*big.Int, *big.Int, error) {
	privateKey, err := rand.Int(rand.Reader, asyncPrime)
	if err != nil {
		return nil, nil, err
	}
	publicKey := new(big.Int).Exp(asyncGenerator, privateKey, asyncPrime)
	return privateKey, publicKey, nil
}

// Инициирование обмена ключами
func initiateKeyExchange(t *testing.T, token, receiverUsername string, g, p, publicKeyA *big.Int) error {
	conn, err := grpc.Dial(asyncServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("Authorization", "Bearer "+token),
	)

	client := pb.NewKeyExchangeServiceClient(conn)
	_, err = client.InitKeyExchange(ctx, &pb.InitKeyExchangeRequest{
		Username:  receiverUsername,
		DhG:       g.String(),
		DhP:       p.String(),
		DhAPublic: publicKeyA.String(),
	})
	return err
}

// Получение параметров обмена ключами
func getKeyExchangeParameters(t *testing.T, token, peerUsername string) (*pb.GetKeyExchangeParamsResponse, error) {
	conn, err := grpc.Dial(asyncServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("Authorization", "Bearer "+token),
	)

	client := pb.NewKeyExchangeServiceClient(conn)
	return client.GetKeyExchangeParams(ctx, &pb.GetKeyExchangeParamsRequest{
		Username: peerUsername,
	})
}

// Завершение обмена ключами
func completeKeyExchangeProcess(t *testing.T, token, initiatorUsername string, publicKeyB *big.Int) error {
	conn, err := grpc.Dial(asyncServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("Authorization", "Bearer "+token),
	)

	client := pb.NewKeyExchangeServiceClient(conn)
	_, err = client.CompleteKeyExchange(ctx, &pb.CompleteKeyExchangeRequest{
		Username:  initiatorUsername,
		DhBPublic: publicKeyB.String(),
	})
	return err
}
