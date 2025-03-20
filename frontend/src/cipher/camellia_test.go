package cipher

import (
	"bytes"
	"context"
	"testing"
)

// TestCamelliaFactory проверяет корректность работы фабрики для создания Camellia
func TestCamelliaFactory(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 128 бит
	camellia, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании объекта Camellia: %v", err)
	}

	// Проверяем доступные размеры ключей
	keySizes := camellia.KeySizes()
	if len(keySizes) != 3 {
		t.Errorf("Ожидалось 3 размера ключа, получено: %d", len(keySizes))
	}

	// Проверяем размер блока
	if camellia.BlockSize() != 16 {
		t.Errorf("Ожидался размер блока 16 байт, получено: %d", camellia.BlockSize())
	}

	// Проверяем название алгоритма
	if camellia.Name() != "camellia" {
		t.Errorf("Ожидалось название 'camellia', получено: %s", camellia.Name())
	}

	// Проверяем GetName
	if camellia.GetName() != "CAMELLIA-128" {
		t.Errorf("Ожидалось 'CAMELLIA-128', получено: %s", camellia.GetName())
	}

	// Проверяем доступные режимы шифрования
	modes := camellia.AvailableModes()
	if len(modes) < 5 {
		t.Errorf("Ожидалось не менее 5 режимов шифрования, получено: %d", len(modes))
	}

	// Проверяем режим по умолчанию
	if camellia.DefaultMode() != ModeCBC {
		t.Errorf("Ожидался режим CBC по умолчанию, получено: %s", camellia.DefaultMode())
	}

	// Проверяем метод набивки по умолчанию
	if camellia.DefaultPadding() != PaddingPKCS7 {
		t.Errorf("Ожидался метод набивки PKCS7 по умолчанию, получено: %s", camellia.DefaultPadding())
	}
}

// TestCamelliaKeySizes проверяет поддержку разных размеров ключей
func TestCamelliaKeySizes(t *testing.T) {
	// Проверяем 128-битный ключ
	camellia128, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia с ключом 128 бит: %v", err)
	}
	if camellia128.GetName() != "CAMELLIA-128" {
		t.Errorf("Неверное название для 128-битного ключа: %s", camellia128.GetName())
	}

	// Проверяем 192-битный ключ
	camellia192, err := NewCamellia(CamelliaKey192)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia с ключом 192 бит: %v", err)
	}
	if camellia192.GetName() != "CAMELLIA-192" {
		t.Errorf("Неверное название для 192-битного ключа: %s", camellia192.GetName())
	}

	// Проверяем 256-битный ключ
	camellia256, err := NewCamellia(CamelliaKey256)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia с ключом 256 бит: %v", err)
	}
	if camellia256.GetName() != "CAMELLIA-256" {
		t.Errorf("Неверное название для 256-битного ключа: %s", camellia256.GetName())
	}

	// Проверяем неправильный размер ключа
	_, err = NewCamellia(CamelliaKeySize(384))
	if err == nil {
		t.Error("Ожидалась ошибка при создании Camellia с недопустимым размером ключа")
	}
}

// TestCamelliaEncryptionDecryption проверяет процесс шифрования и дешифрования
func TestCamelliaEncryptionDecryption(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 128 бит
	keySize := CamelliaKey128
	camellia, err := NewCamellia(keySize)
	if err != nil {
		t.Fatalf("Не удалось создать Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(keySize))
	if err != nil {
		t.Fatalf("Не удалось сгенерировать ключ: %v", err)
	}

	// Устанавливаем ключ
	err = camellia.setupKey(key)
	if err != nil {
		t.Fatalf("Не удалось установить ключ: %v", err)
	}

	// Тестовое сообщение (ровно один блок - 16 байт)
	plaintext := []byte("This is a test..")
	if len(plaintext) != camellia.BlockSize() {
		t.Fatalf("Размер текста должен быть равен размеру блока (%d байт)", camellia.BlockSize())
	}

	// Шифруем данные
	ciphertext, err := camellia.encryptBlock(plaintext)
	if err != nil {
		t.Fatalf("Ошибка при шифровании: %v", err)
	}

	// Убедимся, что шифротекст отличается от исходного текста
	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("Шифротекст не должен быть равен исходному тексту")
	}

	// Дешифруем данные
	decrypted, err := camellia.decryptBlock(ciphertext)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании: %v", err)
	}

	// Проверяем, что дешифрованный текст совпадает с исходным
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Дешифрованный текст не совпадает с исходным.\nОжидалось: %v\nПолучено: %v",
			plaintext, decrypted)
	}

	t.Logf("Шифрование и дешифрование успешно выполнено для ключа размером %d бит", keySize)

	// Проверяем поддержку всех размеров ключей
	if containsKeySize(camellia.KeySizes(), int(CamelliaKey256)) {
		t.Log("Шифрование с ключами 128, 192 и 256 бит полностью реализовано")
	}
}

// Вспомогательная функция для проверки наличия размера ключа в списке
func containsKeySize(sizes []int, size int) bool {
	for _, s := range sizes {
		if s == size {
			return true
		}
	}
	return false
}

// TestDummyCamelliaEncryption быстрый тест для проверки базового функционала
func TestDummyCamelliaEncryption(t *testing.T) {
	// Создаем экземпляр Camellia
	camellia, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia: %v", err)
	}

	// Простой тест для проверки методов
	if camellia.BlockSize() != 16 {
		t.Errorf("Ожидался размер блока 16 байт, получено: %d", camellia.BlockSize())
	}

	// Генерируем тестовый ключ
	key := make([]byte, 16) // 128 бит
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}

	// Проверяем установку ключа
	err = camellia.SetupKey(key)
	if err != nil {
		t.Fatalf("Ошибка при установке ключа: %v", err)
	}

	// Убеждаемся, что подключи сформированы
	if camellia.subKeys == nil {
		t.Error("Подключи не были сформированы")
	}

	t.Log("Базовая проверка Camellia успешно пройдена")
}

// TestCamelliaWithIV проверяет шифрование с использованием вектора инициализации
func TestCamelliaWithIV(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 128 бит
	keySize := CamelliaKey128
	camellia, err := NewCamellia(keySize)
	if err != nil {
		t.Fatalf("Не удалось создать Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(keySize))
	if err != nil {
		t.Fatalf("Не удалось сгенерировать ключ: %v", err)
	}

	// Генерируем IV
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Не удалось сгенерировать IV: %v", err)
	}

	// Проверяем размер IV
	if len(iv) != camellia.IVSize() {
		t.Fatalf("Неверный размер IV: ожидается %d байт, получено %d байт",
			camellia.IVSize(), len(iv))
	}

	// Тестовое сообщение
	plaintext := []byte("This is a test message for Camellia with IV!")

	// Шифруем данные с IV
	ciphertext, err := camellia.EncryptWithIV(context.Background(), plaintext, key, iv)
	if err != nil {
		t.Fatalf("Ошибка при шифровании с IV: %v", err)
	}

	// Убедимся, что шифротекст отличается от исходного текста
	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("Шифротекст не должен быть равен исходному тексту")
	}

	// Дешифруем данные с IV
	decrypted, err := camellia.DecryptWithIV(context.Background(), ciphertext, key, iv)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании с IV: %v", err)
	}

	// Проверяем, что дешифрованный текст совпадает с исходным
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Дешифрованный текст не совпадает с исходным.\nОжидалось: %s\nПолучено: %s",
			string(plaintext), string(decrypted))
	}

	t.Logf("Шифрование и дешифрование с IV успешно выполнено")

	// Проверяем, что шифрование с другим IV дает другой результат
	iv2, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Не удалось сгенерировать второй IV: %v", err)
	}

	ciphertext2, err := camellia.EncryptWithIV(context.Background(), plaintext, key, iv2)
	if err != nil {
		t.Fatalf("Ошибка при шифровании со вторым IV: %v", err)
	}

	// Проверяем, что результаты шифрования с разными IV отличаются
	if bytes.Equal(ciphertext, ciphertext2) {
		t.Fatal("Шифротексты с разными IV не должны совпадать")
	}

	t.Logf("Шифрование с разными IV дает разные результаты")
}

// TestCamelliaErrorHandling проверяет обработку ошибок
func TestCamelliaErrorHandling(t *testing.T) {
	camellia, _ := NewCamellia(CamelliaKey128)

	// Тест с неправильным размером ключа
	invalidKey := make([]byte, 20) // Не 16, 24 или 32 байта
	err := camellia.SetupKey(invalidKey)
	if err == nil {
		t.Error("Ожидалась ошибка при установке ключа неправильного размера")
	}

	// Тест с правильным ключом
	validKey := make([]byte, 16)
	err = camellia.SetupKey(validKey)
	if err != nil {
		t.Errorf("Ошибка при установке ключа правильного размера: %v", err)
	}

	// Тест шифрования без установки ключа
	camellia, _ = NewCamellia(CamelliaKey128)
	_, err = camellia.encryptBlock(make([]byte, 16))
	if err == nil {
		t.Error("Ожидалась ошибка при шифровании без установки ключа")
	}

	// Тест с неправильным размером блока
	err = camellia.SetupKey(validKey)
	if err != nil {
		t.Fatalf("Ошибка при установке ключа: %v", err)
	}

	_, err = camellia.encryptBlock(make([]byte, 8)) // Не 16 байт
	if err == nil {
		t.Error("Ожидалась ошибка при шифровании блока неправильного размера")
	}
}

// TestCamelliaDifferentMessages проверяет шифрование сообщений разной длины
func TestCamelliaDifferentMessages(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 128 бит
	keySize := CamelliaKey128
	camellia, err := NewCamellia(keySize)
	if err != nil {
		t.Fatalf("Не удалось создать Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(keySize))
	if err != nil {
		t.Fatalf("Не удалось сгенерировать ключ: %v", err)
	}

	// Генерируем IV
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Не удалось сгенерировать IV: %v", err)
	}

	// Тестовые сообщения разных размеров
	testMessages := []struct {
		name     string
		message  []byte
		expected bool
	}{
		{"Empty message", []byte{}, true},
		{"Short message", []byte("Hello"), true},
		{"Block size message", []byte("16 bytes message!"), true},
		{"Multi-block message", []byte("This is a longer message that spans multiple blocks for testing"), true},
		{"Non-ASCII message", []byte("Привет, мир! 你好，世界！"), true},
	}

	for _, test := range testMessages {
		t.Run(test.name, func(t *testing.T) {
			// Шифруем данные
			ciphertext, err := camellia.EncryptWithIV(context.Background(), test.message, key, iv)
			if err != nil {
				t.Fatalf("Ошибка при шифровании '%s': %v", test.name, err)
			}

			// Убедимся, что шифротекст отличается от исходного текста (если сообщение не пустое)
			if len(test.message) > 0 && bytes.Equal(test.message, ciphertext) {
				t.Fatalf("Шифротекст не должен быть равен исходному тексту для '%s'", test.name)
			}

			// Дешифруем данные
			decrypted, err := camellia.DecryptWithIV(context.Background(), ciphertext, key, iv)
			if err != nil {
				t.Fatalf("Ошибка при дешифровании '%s': %v", test.name, err)
			}

			// Проверяем, что дешифрованный текст совпадает с исходным
			if !bytes.Equal(test.message, decrypted) {
				t.Fatalf("Дешифрованный текст не совпадает с исходным для '%s'.\nОжидалось: %s\nПолучено: %s",
					test.name, string(test.message), string(decrypted))
			}

			t.Logf("Тест для '%s' успешно пройден", test.name)
		})
	}

	t.Log("Шифрование и дешифрование различных сообщений выполнено успешно")
}

// TestCamelliaBlockEncryption проверяет шифрование и дешифрование одного блока
func TestCamelliaBlockEncryption(t *testing.T) {
	// Создаем экземпляр Camellia
	camellia, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia: %v", err)
	}

	// Генерируем тестовый ключ
	key := make([]byte, 16) // 128 бит
	for i := 0; i < len(key); i++ {
		key[i] = byte(i)
	}

	// Устанавливаем ключ
	err = camellia.SetupKey(key)
	if err != nil {
		t.Fatalf("Ошибка при установке ключа: %v", err)
	}

	// Создаем блок данных для шифрования
	block := make([]byte, camellia.BlockSize())
	for i := 0; i < len(block); i++ {
		block[i] = byte(i + 1) // Заполняем блок тестовыми данными
	}

	// Шифруем блок
	encryptedBlock, err := camellia.encryptBlock(block)
	if err != nil {
		t.Fatalf("Ошибка при шифровании блока: %v", err)
	}

	// Проверяем, что шифрованный блок отличается от исходного
	if bytes.Equal(block, encryptedBlock) {
		t.Error("Шифрованный блок не должен совпадать с исходным")
	}

	// Дешифруем блок
	decryptedBlock, err := camellia.decryptBlock(encryptedBlock)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании блока: %v", err)
	}

	// Проверяем, что дешифрованный блок совпадает с исходным
	if !bytes.Equal(block, decryptedBlock) {
		t.Errorf("Дешифрованный блок должен совпадать с исходным")
		t.Errorf("Исходный блок:      %v", block)
		t.Errorf("Дешифрованный блок: %v", decryptedBlock)
	} else {
		t.Logf("Блочное шифрование и дешифрование работает корректно")
	}
}

// TestCamellia192BitKey проверяет работу шифрования с 192-битным ключом
func TestCamellia192BitKey(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 192 бит
	keySize := CamelliaKey192
	camellia, err := NewCamellia(keySize)
	if err != nil {
		t.Fatalf("Не удалось создать Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(keySize))
	if err != nil {
		t.Fatalf("Не удалось сгенерировать ключ: %v", err)
	}

	// Проверяем размер ключа
	if len(key) != int(keySize)/8 {
		t.Fatalf("Неверный размер ключа: ожидается %d байт, получено %d байт", int(keySize)/8, len(key))
	}

	// Устанавливаем ключ
	err = camellia.setupKey(key)
	if err != nil {
		t.Fatalf("Не удалось установить ключ: %v", err)
	}

	// Тестовое сообщение (ровно один блок - 16 байт)
	plaintext := []byte("This is a test..")
	if len(plaintext) != camellia.BlockSize() {
		t.Fatalf("Размер текста должен быть равен размеру блока (%d байт)", camellia.BlockSize())
	}

	// Шифруем данные
	ciphertext, err := camellia.encryptBlock(plaintext)
	if err != nil {
		t.Fatalf("Ошибка при шифровании: %v", err)
	}

	// Убедимся, что шифротекст отличается от исходного текста
	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("Шифротекст не должен быть равен исходному тексту")
	}

	// Дешифруем данные
	decrypted, err := camellia.decryptBlock(ciphertext)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании: %v", err)
	}

	// Проверяем, что дешифрованный текст совпадает с исходным
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Дешифрованный текст не совпадает с исходным.\nОжидалось: %v\nПолучено: %v",
			plaintext, decrypted)
	}

	t.Logf("Шифрование и дешифрование успешно выполнено для ключа размером %d бит", keySize)

	// Теперь проверим работу с IV
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Не удалось сгенерировать IV: %v", err)
	}

	// Шифруем больший объем данных с IV
	bigPlaintext := []byte("This is a larger test message that requires multiple blocks for encryption.")

	ciphertext, err = camellia.EncryptWithIV(context.Background(), bigPlaintext, key, iv)
	if err != nil {
		t.Fatalf("Ошибка при шифровании с IV: %v", err)
	}

	// Дешифруем с тем же IV
	decrypted, err = camellia.DecryptWithIV(context.Background(), ciphertext, key, iv)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании с IV: %v", err)
	}

	// Проверяем результат
	if !bytes.Equal(bigPlaintext, decrypted) {
		t.Fatalf("Дешифрованный текст не совпадает с исходным при использовании IV.\nОжидалось: %s\nПолучено: %s",
			string(bigPlaintext), string(decrypted))
	}

	t.Logf("Шифрование и дешифрование с IV успешно выполнено для ключа размером %d бит", keySize)
}

// TestCamellia256BitKey проверяет работу шифрования с 256-битным ключом
func TestCamellia256BitKey(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 256 бит
	keySize := CamelliaKey256
	camellia, err := NewCamellia(keySize)
	if err != nil {
		t.Fatalf("Не удалось создать Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(keySize))
	if err != nil {
		t.Fatalf("Не удалось сгенерировать ключ: %v", err)
	}

	// Проверяем размер ключа
	if len(key) != int(keySize)/8 {
		t.Fatalf("Неверный размер ключа: ожидается %d байт, получено %d байт", int(keySize)/8, len(key))
	}

	// Устанавливаем ключ
	err = camellia.setupKey(key)
	if err != nil {
		t.Fatalf("Не удалось установить ключ: %v", err)
	}

	// Тестовое сообщение (ровно один блок - 16 байт)
	plaintext := []byte("This is a test..")
	if len(plaintext) != camellia.BlockSize() {
		t.Fatalf("Размер текста должен быть равен размеру блока (%d байт)", camellia.BlockSize())
	}

	// Шифруем данные
	ciphertext, err := camellia.encryptBlock(plaintext)
	if err != nil {
		t.Fatalf("Ошибка при шифровании: %v", err)
	}

	// Убедимся, что шифротекст отличается от исходного текста
	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("Шифротекст не должен быть равен исходному тексту")
	}

	// Дешифруем данные
	decrypted, err := camellia.decryptBlock(ciphertext)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании: %v", err)
	}

	// Проверяем, что дешифрованный текст совпадает с исходным
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Дешифрованный текст не совпадает с исходным.\nОжидалось: %v\nПолучено: %v",
			plaintext, decrypted)
	}

	t.Logf("Шифрование и дешифрование успешно выполнено для ключа размером %d бит", keySize)

	// Теперь проверим работу с IV
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Не удалось сгенерировать IV: %v", err)
	}

	// Шифруем больший объем данных с IV
	bigPlaintext := []byte("This is a larger test message that requires multiple blocks for encryption.")

	ciphertext, err = camellia.EncryptWithIV(context.Background(), bigPlaintext, key, iv)
	if err != nil {
		t.Fatalf("Ошибка при шифровании с IV: %v", err)
	}

	// Дешифруем с тем же IV
	decrypted, err = camellia.DecryptWithIV(context.Background(), ciphertext, key, iv)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании с IV: %v", err)
	}

	// Проверяем результат
	if !bytes.Equal(bigPlaintext, decrypted) {
		t.Fatalf("Дешифрованный текст не совпадает с исходным при использовании IV.\nОжидалось: %s\nПолучено: %s",
			string(bigPlaintext), string(decrypted))
	}

	t.Logf("Шифрование и дешифрование с IV успешно выполнено для ключа размером %d бит", keySize)
}
