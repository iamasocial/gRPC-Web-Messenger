package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"

	pb "dhclient/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Порядок выполнения тестов в Go идет по алфавиту имени функции
// Добавляем префиксы, чтобы гарантировать правильный порядок
const (
	serverAddr = "localhost:50051"
)

// Глобальные переменные для тестирования
var (
	user1Username = "testuser1"
	user1Password = "password123"
	user1Token    string
	user2Username = "testuser2"
	user2Password = "password123"
	user2Token    string

	// Генератор (стандартное значение 2)
	generator = big.NewInt(2)

	// Параметры для Диффи-Хеллмана
	// Большое простое число (2048 бит)
	prime, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF", 16)
)

// Функция для подключения к серверу
func connectToServer() (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к серверу: %v", err)
	}
	return conn, nil
}

// Функция для регистрации пользователя
func registerUser(username, password string) error {
	conn, err := connectToServer()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	_, err = client.Register(context.Background(), &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	return err
}

// Функция для аутентификации пользователя и получения токена
func loginUser(username, password string) (string, error) {
	conn, err := connectToServer()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	resp, err := client.Login(context.Background(), &pb.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}

// Функция для создания чата между пользователями
func createChat(token, receiverUsername string) error {
	conn, err := connectToServer()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Добавляем токен авторизации в метаданные запроса
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

// Функция для генерации приватного ключа и вычисления публичного ключа
func generateDHKeyPair() (*big.Int, *big.Int, error) {
	// Генерируем случайное число a (приватный ключ)
	privateKey, err := rand.Int(rand.Reader, prime)
	if err != nil {
		return nil, nil, err
	}

	// Вычисляем публичный ключ A = g^a mod p
	publicKey := new(big.Int).Exp(generator, privateKey, prime)

	return privateKey, publicKey, nil
}

// Функция для инициирования обмена ключами
func initKeyExchange(token, receiverUsername string, g, p, publicKeyA *big.Int) error {
	conn, err := connectToServer()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Добавляем токен авторизации в метаданные запроса
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

// Функция для получения параметров обмена ключами
func getKeyExchangeParams(token, peerUsername string) (*pb.GetKeyExchangeParamsResponse, error) {
	conn, err := connectToServer()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Добавляем токен авторизации в метаданные запроса
	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("Authorization", "Bearer "+token),
	)

	client := pb.NewKeyExchangeServiceClient(conn)
	return client.GetKeyExchangeParams(ctx, &pb.GetKeyExchangeParamsRequest{
		Username: peerUsername,
	})
}

// Функция для завершения обмена ключами
func completeKeyExchange(token, initiatorUsername string, publicKeyB *big.Int) error {
	conn, err := connectToServer()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Добавляем токен авторизации в метаданные запроса
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

// Тест на регистрацию и авторизацию пользователей
func Test1_UsersSetup(t *testing.T) {
	// Регистрируем первого пользователя
	err := registerUser(user1Username, user1Password)
	if err != nil {
		// Если пользователь уже существует, это нормально, продолжаем
		log.Printf("Пользователь '%s' уже существует или произошла ошибка: %v", user1Username, err)
	}

	// Регистрируем второго пользователя
	err = registerUser(user2Username, user2Password)
	if err != nil {
		log.Printf("Пользователь '%s' уже существует или произошла ошибка: %v", user2Username, err)
	}

	// Логинимся как первый пользователь
	token, err := loginUser(user1Username, user1Password)
	if err != nil {
		t.Fatalf("Ошибка при входе пользователя '%s': %v", user1Username, err)
	}
	user1Token = token
	log.Printf("Пользователь '%s' успешно авторизован, получен токен", user1Username)

	// Логинимся как второй пользователь
	token, err = loginUser(user2Username, user2Password)
	if err != nil {
		t.Fatalf("Ошибка при входе пользователя '%s': %v", user2Username, err)
	}
	user2Token = token
	log.Printf("Пользователь '%s' успешно авторизован, получен токен", user2Username)
}

// Тест на создание чата между пользователями
func Test2_CreateChat(t *testing.T) {
	if user1Token == "" || user2Token == "" {
		t.Fatalf("Токены авторизации отсутствуют, выполните сначала тест Test1_UsersSetup")
	}

	// Создаем чат от первого пользователя ко второму
	err := createChat(user1Token, user2Username)
	if err != nil {
		// Если чат уже существует, это нормально
		log.Printf("Чат между '%s' и '%s' уже существует или произошла ошибка: %v",
			user1Username, user2Username, err)
	} else {
		log.Printf("Чат между '%s' и '%s' успешно создан", user1Username, user2Username)
	}
}

// Тест на инициирование обмена ключами
func Test3_InitKeyExchange(t *testing.T) {
	if user1Token == "" || user2Token == "" {
		t.Fatalf("Токены авторизации отсутствуют, выполните сначала тест Test1_UsersSetup")
	}

	// Генерируем пару ключей для первого пользователя
	privateKeyA, publicKeyA, err := generateDHKeyPair()
	if err != nil {
		t.Fatalf("Ошибка при генерации ключей: %v", err)
	}

	// Первый пользователь инициирует обмен ключами со вторым
	err = initKeyExchange(user1Token, user2Username, generator, prime, publicKeyA)
	if err != nil {
		t.Fatalf("Ошибка при инициировании обмена ключами: %v", err)
	}
	log.Printf("Пользователь '%s' успешно инициировал обмен ключами с '%s'",
		user1Username, user2Username)

	// Сохраняем приватный ключ пользователя для последующих тестов
	err = os.WriteFile("user1_private_key.txt", []byte(privateKeyA.String()), 0644)
	if err != nil {
		t.Fatalf("Ошибка при сохранении приватного ключа: %v", err)
	}
}

// Тест на получение параметров обмена ключами
func Test4_GetKeyExchangeParams(t *testing.T) {
	if user1Token == "" || user2Token == "" {
		t.Fatalf("Токены авторизации отсутствуют, выполните сначала тест Test1_UsersSetup")
	}

	// Второй пользователь получает параметры обмена ключами
	params, err := getKeyExchangeParams(user2Token, user1Username)
	if err != nil {
		t.Fatalf("Ошибка при получении параметров обмена ключами: %v", err)
	}

	if !params.Success {
		t.Fatalf("Не удалось получить параметры обмена ключами")
	}

	if params.Status != pb.KeyExchangeStatus_INITIATED {
		t.Fatalf("Неверный статус обмена ключами: %v", params.Status)
	}

	if params.DhG == "" || params.DhP == "" || params.DhAPublic == "" {
		t.Fatalf("Отсутствуют необходимые параметры Диффи-Хеллмана")
	}

	log.Printf("Пользователь '%s' успешно получил параметры обмена ключами от '%s'",
		user2Username, user1Username)
	log.Printf("Статус обмена ключами: %v", params.Status)
	log.Printf("Генератор (g): %s", params.DhG)
	log.Printf("Простое число (p): %s... (обрезано)", params.DhP[:50])
	log.Printf("Публичный ключ A: %s... (обрезано)", params.DhAPublic[:50])

	// Сохраняем параметры для последующих тестов
	paramsData := fmt.Sprintf("%s\n%s\n%s\n", params.DhG, params.DhP, params.DhAPublic)
	err = os.WriteFile("key_exchange_params.txt", []byte(paramsData), 0644)
	if err != nil {
		t.Fatalf("Ошибка при сохранении параметров: %v", err)
	}
}

// Тест на завершение обмена ключами
func Test5_CompleteKeyExchange(t *testing.T) {
	if user1Token == "" || user2Token == "" {
		t.Fatalf("Токены авторизации отсутствуют, выполните сначала тест Test1_UsersSetup")
	}

	// Читаем параметры обмена ключами
	paramsData, err := os.ReadFile("key_exchange_params.txt")
	if err != nil {
		t.Fatalf("Ошибка при чтении параметров обмена ключами: %v", err)
	}

	var gStr, pStr, aStr string
	_, err = fmt.Sscanf(string(paramsData), "%s\n%s\n%s\n", &gStr, &pStr, &aStr)
	if err != nil {
		t.Fatalf("Ошибка при парсинге параметров обмена ключами: %v", err)
	}

	g := new(big.Int)
	g.SetString(gStr, 10)

	p := new(big.Int)
	p.SetString(pStr, 10)

	a := new(big.Int)
	a.SetString(aStr, 10)

	// Генерируем приватный ключ для второго пользователя
	privateKeyB, err := rand.Int(rand.Reader, p)
	if err != nil {
		t.Fatalf("Ошибка при генерации приватного ключа: %v", err)
	}

	// Вычисляем публичный ключ B = g^b mod p
	publicKeyB := new(big.Int).Exp(g, privateKeyB, p)

	// Второй пользователь завершает обмен ключами
	err = completeKeyExchange(user2Token, user1Username, publicKeyB)
	if err != nil {
		t.Fatalf("Ошибка при завершении обмена ключами: %v", err)
	}

	log.Printf("Пользователь '%s' успешно завершил обмен ключами с '%s'",
		user2Username, user1Username)

	// Вычисляем общий секретный ключ второго пользователя
	sharedSecretB := new(big.Int).Exp(a, privateKeyB, p)
	log.Printf("Общий секретный ключ (вторым пользователем): %s... (обрезано)",
		sharedSecretB.String()[:50])

	// Сохраняем приватный ключ и общий секрет для последующих тестов
	err = os.WriteFile("user2_private_key.txt", []byte(privateKeyB.String()), 0644)
	if err != nil {
		t.Fatalf("Ошибка при сохранении приватного ключа: %v", err)
	}

	err = os.WriteFile("shared_secret_from_user2.txt", []byte(sharedSecretB.String()), 0644)
	if err != nil {
		t.Fatalf("Ошибка при сохранении общего секрета: %v", err)
	}
}

// Тест на проверку параметров завершенного обмена ключами
func Test6_VerifyCompletedKeyExchange(t *testing.T) {
	if user1Token == "" || user2Token == "" {
		t.Fatalf("Токены авторизации отсутствуют, выполните сначала тест Test1_UsersSetup")
	}

	// Первый пользователь получает параметры обмена ключами
	params, err := getKeyExchangeParams(user1Token, user2Username)
	if err != nil {
		t.Fatalf("Ошибка при получении параметров обмена ключами: %v", err)
	}

	if !params.Success {
		t.Fatalf("Не удалось получить параметры обмена ключами")
	}

	if params.Status != pb.KeyExchangeStatus_COMPLETED {
		t.Fatalf("Неверный статус обмена ключами: %v, ожидалось COMPLETED", params.Status)
	}

	if params.DhG == "" || params.DhP == "" || params.DhAPublic == "" || params.DhBPublic == "" {
		t.Fatalf("Отсутствуют необходимые параметры Диффи-Хеллмана")
	}

	log.Printf("Пользователь '%s' успешно получил параметры завершенного обмена ключами",
		user1Username)
	log.Printf("Статус обмена ключами: %v", params.Status)
	log.Printf("Генератор (g): %s", params.DhG)
	log.Printf("Простое число (p): %s... (обрезано)", params.DhP[:50])
	log.Printf("Публичный ключ A: %s... (обрезано)", params.DhAPublic[:50])
	log.Printf("Публичный ключ B: %s... (обрезано)", params.DhBPublic[:50])

	// Читаем приватный ключ первого пользователя
	privateKeyData, err := os.ReadFile("user1_private_key.txt")
	if err != nil {
		t.Fatalf("Ошибка при чтении приватного ключа: %v", err)
	}

	privateKeyA := new(big.Int)
	privateKeyA.SetString(string(privateKeyData), 10)

	// Читаем публичный ключ B
	publicKeyB := new(big.Int)
	publicKeyB.SetString(params.DhBPublic, 10)

	// Читаем p
	p := new(big.Int)
	p.SetString(params.DhP, 10)

	// Вычисляем общий секретный ключ для первого пользователя
	sharedSecretA := new(big.Int).Exp(publicKeyB, privateKeyA, p)
	log.Printf("Общий секретный ключ (первым пользователем): %s... (обрезано)",
		sharedSecretA.String()[:50])

	// Сохраняем общий секрет для последующей проверки
	err = os.WriteFile("shared_secret_from_user1.txt", []byte(sharedSecretA.String()), 0644)
	if err != nil {
		t.Fatalf("Ошибка при сохранении общего секрета: %v", err)
	}
}

// Тест на проверку совпадения общих секретов
func Test7_VerifySharedSecrets(t *testing.T) {
	// Читаем общий секрет первого пользователя
	secretAData, err := os.ReadFile("shared_secret_from_user1.txt")
	if err != nil {
		t.Fatalf("Ошибка при чтении общего секрета первого пользователя: %v", err)
	}

	// Читаем общий секрет второго пользователя
	secretBData, err := os.ReadFile("shared_secret_from_user2.txt")
	if err != nil {
		t.Fatalf("Ошибка при чтении общего секрета второго пользователя: %v", err)
	}

	secretA := new(big.Int)
	secretA.SetString(string(secretAData), 10)

	secretB := new(big.Int)
	secretB.SetString(string(secretBData), 10)

	// Сравниваем общие секреты
	if secretA.Cmp(secretB) != 0 {
		t.Fatalf("Общие секреты не совпадают: %s != %s", secretA.String(), secretB.String())
	}

	log.Printf("Проверка успешна: общие секреты пользователей совпадают!")
	log.Printf("Общий секрет: %s... (обрезано)", secretA.String()[:50])
}

// Точка входа для тестов
func TestMain(m *testing.M) {
	log.Println("Запуск тестов сервиса обмена ключами Диффи-Хеллмана")

	// Запускаем тесты
	exitCode := m.Run()

	log.Println("Тесты завершены")
	os.Exit(exitCode)
}
