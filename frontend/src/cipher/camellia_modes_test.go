package cipher

import (
	"bytes"
	"context"
	"fmt"
	"testing"
)

// TestCamelliaCipherModes проверяет шифрование Camellia с различными режимами шифрования
func TestCamelliaCipherModes(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 128 бит
	camellia, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(CamelliaKey128))
	if err != nil {
		t.Fatalf("Ошибка при генерации ключа: %v", err)
	}

	// Устанавливаем ключ
	err = camellia.setupKey(key)
	if err != nil {
		t.Fatalf("Ошибка при установке ключа: %v", err)
	}

	// Генерируем IV
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Ошибка при генерации IV: %v", err)
	}

	// Тестовое сообщение
	plaintext := []byte("Это тестовое сообщение для проверки разных режимов шифрования.")

	// Создаем набивку PKCS7
	padding := PKCS7Padding()

	// Тестируем разные режимы шифрования
	modes := []struct {
		name       BlockCipherMode
		iv         bool // требуется ли IV для этого режима
		doubleSize bool // увеличивается ли размер зашифрованных данных в два раза (для RandomDelta)
	}{
		{ModeECB, false, false},
		{ModeCBC, true, false},
		{ModeCFB, true, false},
		{ModeOFB, true, false},
		{ModeCTR, true, false},
		{ModeRandomDelta, true, true}, // RandomDelta удваивает размер данных
	}

	for _, mode := range modes {
		t.Run(string(mode.name), func(t *testing.T) {
			// Создаем режим шифрования
			cipherMode, err := NewBlockCipherMode(mode.name, camellia.BlockSize())
			if err != nil {
				t.Fatalf("Ошибка при создании режима %s: %v", mode.name, err)
			}

			// Применяем набивку PKCS7 ко всем режимам
			// Хотя для поточных режимов (CFB, OFB, CTR) это необязательно,
			// они могут работать с данными любой длины
			paddedData, err := padding.Pad(plaintext, camellia.BlockSize())
			if err != nil {
				t.Fatalf("Ошибка при применении набивки: %v", err)
			}

			// IV для режимов, которые его используют
			var modeIV []byte
			if mode.iv {
				modeIV = iv
			}

			// Шифруем данные
			encrypted, err := cipherMode.Encrypt(paddedData, modeIV, camellia.createEncryptBlockFunc())
			if err != nil {
				t.Fatalf("Ошибка при шифровании в режиме %s: %v", mode.name, err)
			}

			// Для режима RandomDelta проверяем, что размер зашифрованных данных в два раза больше исходных
			if mode.doubleSize {
				if len(encrypted) != len(paddedData)*2 {
					t.Fatalf("Режим %s: ожидаемый размер зашифрованных данных: %d, получено: %d",
						mode.name, len(paddedData)*2, len(encrypted))
				}
			}

			// Дешифруем данные
			decrypted, err := cipherMode.Decrypt(encrypted, modeIV, camellia.createDecryptBlockFunc())
			if err != nil {
				t.Fatalf("Ошибка при дешифровании в режиме %s: %v", mode.name, err)
			}

			// Убираем набивку для всех режимов
			unpaddedData, err := padding.Unpad(decrypted, camellia.BlockSize())
			if err != nil {
				t.Fatalf("Ошибка при удалении набивки: %v", err)
			}

			// Проверяем, что расшифрованный текст совпадает с исходным
			if !bytes.Equal(plaintext, unpaddedData) {
				t.Fatalf("Режим %s: расшифрованный текст не совпадает с исходным", mode.name)
			}

			// Для RandomDelta проверяем, что повторное шифрование дает другой результат
			if mode.name == ModeRandomDelta {
				encrypted2, err := cipherMode.Encrypt(paddedData, modeIV, camellia.createEncryptBlockFunc())
				if err != nil {
					t.Fatalf("Ошибка при повторном шифровании в режиме %s: %v", mode.name, err)
				}

				// Результаты должны отличаться из-за случайного компонента
				if bytes.Equal(encrypted, encrypted2) {
					t.Errorf("Режим %s: повторное шифрование дало тот же результат, что не ожидается", mode.name)
				}

				// Проверяем, что дешифрование второго результата тоже работает
				decrypted2, err := cipherMode.Decrypt(encrypted2, modeIV, camellia.createDecryptBlockFunc())
				if err != nil {
					t.Fatalf("Ошибка при дешифровании после повторного шифрования: %v", err)
				}

				// Убираем набивку для проверки
				unpaddedData2, err := padding.Unpad(decrypted2, camellia.BlockSize())
				if err != nil {
					t.Fatalf("Ошибка при удалении набивки после повторного шифрования: %v", err)
				}

				if !bytes.Equal(plaintext, unpaddedData2) {
					t.Fatalf("Режим %s после повторного шифрования: расшифрованный текст не совпадает с исходным", mode.name)
				}
			}

			t.Logf("Режим %s: шифрование и дешифрование выполнено успешно", mode.name)
		})
	}
}

// TestCamelliaPaddingMethods проверяет шифрование Camellia с различными методами набивки
func TestCamelliaPaddingMethods(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 128 бит
	camellia, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(CamelliaKey128))
	if err != nil {
		t.Fatalf("Ошибка при генерации ключа: %v", err)
	}

	// Устанавливаем ключ
	err = camellia.SetupKey(key)
	if err != nil {
		t.Fatalf("Ошибка при установке ключа: %v", err)
	}

	// Генерируем IV для CBC режима
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Ошибка при генерации IV: %v", err)
	}

	// Создаем режим CBC
	cbcMode, err := NewBlockCipherMode(ModeCBC, camellia.BlockSize())
	if err != nil {
		t.Fatalf("Ошибка при создании режима CBC: %v", err)
	}

	// Создаем тестовый блок размером с блок шифрования (16 байт для Camellia)
	blockSizeMessage := make([]byte, camellia.BlockSize())
	for i := 0; i < len(blockSizeMessage); i++ {
		blockSizeMessage[i] = byte(i)
	}

	// Тестовые сообщения разной длины
	testMessages := []struct {
		name string
		data []byte
	}{
		// Пустое сообщение проблематично для некоторых методов набивки, поэтому исключаем его
		{"Короткое сообщение", []byte("Короткий текст")},
		{"Сообщение размером с блок", blockSizeMessage},
		{"Сообщение длиннее блока", []byte("Это сообщение длиннее размера блока и требует набивки")},
	}

	// Методы набивки для тестирования
	paddingMethods := []struct {
		name        string
		padding     Padding
		skipForSize func(int, int) bool // Функция, определяющая, нужно ли пропустить тест для определенного размера данных
	}{
		{
			name:    "PKCS7",
			padding: PKCS7Padding(),
			skipForSize: func(dataLen, blockSize int) bool {
				return false // PKCS7 работает со всеми размерами данных
			},
		},
		{
			name:    "ISO10126",
			padding: ISO10126Padding(),
			skipForSize: func(dataLen, blockSize int) bool {
				return false // ISO10126 работает со всеми размерами данных
			},
		},
		{
			name:    "Zeros",
			padding: ZerosPadding(),
			skipForSize: func(dataLen, blockSize int) bool {
				return dataLen == 0 // Zeros не работает с пустыми данными
			},
		},
		{
			name:    "ANSIX923",
			padding: ANSIX923Padding(),
			skipForSize: func(dataLen, blockSize int) bool {
				return false // ANSIX923 работает со всеми размерами данных
			},
		},
	}

	// Дополнительно тестируем метод PKCS7 с пустым сообщением отдельно
	t.Run("Пустое_сообщение-PKCS7", func(t *testing.T) {
		emptyData := []byte{}
		padder := PKCS7Padding()

		// Применяем набивку
		paddedData, err := padder.Pad(emptyData, camellia.BlockSize())
		if err != nil {
			t.Fatalf("Ошибка при применении набивки PKCS7 к пустому сообщению: %v", err)
		}

		// Шифруем данные
		encrypted, err := cbcMode.Encrypt(paddedData, iv, camellia.createEncryptBlockFunc())
		if err != nil {
			t.Fatalf("Ошибка при шифровании с набивкой PKCS7: %v", err)
		}

		// Дешифруем данные
		decrypted, err := cbcMode.Decrypt(encrypted, iv, camellia.createDecryptBlockFunc())
		if err != nil {
			t.Fatalf("Ошибка при дешифровании с набивкой PKCS7: %v", err)
		}

		// Убираем набивку
		unpaddedData, err := padder.Unpad(decrypted, camellia.BlockSize())
		if err != nil {
			t.Fatalf("Ошибка при удалении набивки PKCS7: %v", err)
		}

		// Проверяем, что расшифрованный текст совпадает с исходным
		if !bytes.Equal(emptyData, unpaddedData) {
			t.Fatalf("Набивка PKCS7: расшифрованный текст не совпадает с исходным")
		}

		t.Logf("Набивка PKCS7 для пустого сообщения: шифрование и дешифрование выполнено успешно")
	})

	// Аналогично для ISO10126 и ANSIX923, которые также должны поддерживать пустые сообщения
	t.Run("Пустое_сообщение-ISO10126", func(t *testing.T) {
		emptyData := []byte{}
		padder := ISO10126Padding()

		// Применяем набивку
		paddedData, err := padder.Pad(emptyData, camellia.BlockSize())
		if err != nil {
			t.Fatalf("Ошибка при применении набивки ISO10126 к пустому сообщению: %v", err)
		}

		// Шифруем данные
		encrypted, err := cbcMode.Encrypt(paddedData, iv, camellia.createEncryptBlockFunc())
		if err != nil {
			t.Fatalf("Ошибка при шифровании с набивкой ISO10126: %v", err)
		}

		// Дешифруем данные
		decrypted, err := cbcMode.Decrypt(encrypted, iv, camellia.createDecryptBlockFunc())
		if err != nil {
			t.Fatalf("Ошибка при дешифровании с набивкой ISO10126: %v", err)
		}

		// Убираем набивку
		unpaddedData, err := padder.Unpad(decrypted, camellia.BlockSize())
		if err != nil {
			t.Fatalf("Ошибка при удалении набивки ISO10126: %v", err)
		}

		// Проверяем, что расшифрованный текст совпадает с исходным
		if !bytes.Equal(emptyData, unpaddedData) {
			t.Fatalf("Набивка ISO10126: расшифрованный текст не совпадает с исходным")
		}

		t.Logf("Набивка ISO10126 для пустого сообщения: шифрование и дешифрование выполнено успешно")
	})

	t.Run("Пустое_сообщение-ANSIX923", func(t *testing.T) {
		emptyData := []byte{}
		padder := ANSIX923Padding()

		// Применяем набивку
		paddedData, err := padder.Pad(emptyData, camellia.BlockSize())
		if err != nil {
			t.Fatalf("Ошибка при применении набивки ANSIX923 к пустому сообщению: %v", err)
		}

		// Шифруем данные
		encrypted, err := cbcMode.Encrypt(paddedData, iv, camellia.createEncryptBlockFunc())
		if err != nil {
			t.Fatalf("Ошибка при шифровании с набивкой ANSIX923: %v", err)
		}

		// Дешифруем данные
		decrypted, err := cbcMode.Decrypt(encrypted, iv, camellia.createDecryptBlockFunc())
		if err != nil {
			t.Fatalf("Ошибка при дешифровании с набивкой ANSIX923: %v", err)
		}

		// Убираем набивку
		unpaddedData, err := padder.Unpad(decrypted, camellia.BlockSize())
		if err != nil {
			t.Fatalf("Ошибка при удалении набивки ANSIX923: %v", err)
		}

		// Проверяем, что расшифрованный текст совпадает с исходным
		if !bytes.Equal(emptyData, unpaddedData) {
			t.Fatalf("Набивка ANSIX923: расшифрованный текст не совпадает с исходным")
		}

		t.Logf("Набивка ANSIX923 для пустого сообщения: шифрование и дешифрование выполнено успешно")
	})

	// Основные тесты для разных сообщений и методов набивки
	for _, msg := range testMessages {
		for _, padMethod := range paddingMethods {
			// Проверяем, нужно ли пропустить тест для данного размера сообщения
			if padMethod.skipForSize(len(msg.data), camellia.BlockSize()) {
				t.Logf("Пропускаем тест для '%s' с методом набивки %s (несовместимый размер данных)",
					msg.name, padMethod.name)
				continue
			}

			testName := fmt.Sprintf("%s-%s", msg.name, padMethod.name)
			t.Run(testName, func(t *testing.T) {
				// Применяем набивку
				paddedData, err := padMethod.padding.Pad(msg.data, camellia.BlockSize())
				if err != nil {
					t.Fatalf("Ошибка при применении набивки %s: %v", padMethod.name, err)
				}

				// Шифруем данные
				encrypted, err := cbcMode.Encrypt(paddedData, iv, camellia.createEncryptBlockFunc())
				if err != nil {
					t.Fatalf("Ошибка при шифровании с набивкой %s: %v", padMethod.name, err)
				}

				// Дешифруем данные
				decrypted, err := cbcMode.Decrypt(encrypted, iv, camellia.createDecryptBlockFunc())
				if err != nil {
					t.Fatalf("Ошибка при дешифровании с набивкой %s: %v", padMethod.name, err)
				}

				// Убираем набивку
				unpaddedData, err := padMethod.padding.Unpad(decrypted, camellia.BlockSize())
				if err != nil {
					t.Fatalf("Ошибка при удалении набивки %s: %v", padMethod.name, err)
				}

				// Проверяем, что расшифрованный текст совпадает с исходным
				if !bytes.Equal(msg.data, unpaddedData) {
					t.Fatalf("Набивка %s: расшифрованный текст не совпадает с исходным", padMethod.name)
				}

				t.Logf("Набивка %s для '%s': шифрование и дешифрование выполнено успешно",
					padMethod.name, msg.name)
			})
		}
	}
}

// TestCamelliaHighLevelAPI проверяет высокоуровневый API для шифрования с разными режимами
func TestCamelliaHighLevelAPI(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 256 бит
	camellia, err := NewCamellia(CamelliaKey256)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(CamelliaKey256))
	if err != nil {
		t.Fatalf("Ошибка при генерации ключа: %v", err)
	}

	// Тестовое сообщение
	plaintext := []byte("Это тестовое сообщение для высокоуровневого API шифрования Camellia")

	// Тестируем обычное шифрование (ECB режим)
	ctx := context.Background()
	encryptedECB, err := camellia.Encrypt(ctx, plaintext, key)
	if err != nil {
		t.Fatalf("Ошибка при шифровании ECB: %v", err)
	}

	// Дешифруем ECB
	decryptedECB, err := camellia.Decrypt(ctx, encryptedECB, key)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании ECB: %v", err)
	}

	// Проверяем результат ECB
	if !bytes.Equal(plaintext, decryptedECB) {
		t.Fatal("ECB: расшифрованный текст не совпадает с исходным")
	}
	t.Log("Высокоуровневый API (ECB): шифрование и дешифрование выполнено успешно")

	// Тестируем шифрование с IV (CBC режим)
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Ошибка при генерации IV: %v", err)
	}

	encryptedCBC, err := camellia.EncryptWithIV(ctx, plaintext, key, iv)
	if err != nil {
		t.Fatalf("Ошибка при шифровании CBC: %v", err)
	}

	// Дешифруем CBC
	decryptedCBC, err := camellia.DecryptWithIV(ctx, encryptedCBC, key, iv)
	if err != nil {
		t.Fatalf("Ошибка при дешифровании CBC: %v", err)
	}

	// Проверяем результат CBC
	if !bytes.Equal(plaintext, decryptedCBC) {
		t.Fatal("CBC: расшифрованный текст не совпадает с исходным")
	}
	t.Log("Высокоуровневый API (CBC): шифрование и дешифрование выполнено успешно")

	// Проверяем, что ECB и CBC дают разные результаты для одинаковых данных
	if bytes.Equal(encryptedECB, encryptedCBC) {
		t.Error("ECB и CBC шифрования не должны совпадать для одних и тех же данных")
	} else {
		t.Log("ECB и CBC дают разные результаты шифрования, как и ожидалось")
	}
}

// TestCamelliaVectors проверяет работу алгоритма Camellia на стандартных тестовых векторах
func TestCamelliaVectors(t *testing.T) {
	// Тестовые векторы из официальной спецификации Camellia
	vectors := []struct {
		keySize     CamelliaKeySize
		key         []byte
		plaintext   []byte
		ciphertext  []byte
		description string
	}{
		{
			keySize:     CamelliaKey128,
			key:         []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			plaintext:   []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			ciphertext:  []byte{0x67, 0x67, 0x31, 0x38, 0x54, 0x96, 0x69, 0x73, 0x08, 0x57, 0x06, 0x56, 0x48, 0xea, 0xbe, 0x43},
			description: "128-битный ключ, тестовый вектор 1",
		},
		{
			keySize:     CamelliaKey192,
			key:         []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77},
			plaintext:   []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			ciphertext:  []byte{0xb4, 0x99, 0x34, 0x01, 0xb3, 0xe9, 0x96, 0xf8, 0x4e, 0xe5, 0xce, 0xe7, 0xd7, 0x9b, 0x09, 0xb9},
			description: "192-битный ключ, тестовый вектор",
		},
		{
			keySize:     CamelliaKey256,
			key:         []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			plaintext:   []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10},
			ciphertext:  []byte{0x9a, 0xcc, 0x23, 0x7d, 0xff, 0x16, 0xd7, 0x6c, 0x20, 0xef, 0x7c, 0x91, 0x9e, 0x3a, 0x75, 0x09},
			description: "256-битный ключ, тестовый вектор",
		},
	}

	for _, vector := range vectors {
		t.Run(vector.description, func(t *testing.T) {
			// Создаем экземпляр Camellia
			camellia, err := NewCamellia(vector.keySize)
			if err != nil {
				t.Fatalf("Ошибка при создании Camellia: %v", err)
			}

			// Устанавливаем ключ
			err = camellia.setupKey(vector.key)
			if err != nil {
				t.Fatalf("Ошибка при установке ключа: %v", err)
			}

			// Шифруем блок данных
			encryptFunc := camellia.createEncryptBlockFunc()
			encrypted, err := encryptFunc(vector.plaintext)
			if err != nil {
				t.Fatalf("Ошибка при шифровании: %v", err)
			}

			// Проверяем, что зашифрованный блок соответствует ожидаемому значению
			if !bytes.Equal(encrypted, vector.ciphertext) {
				t.Fatalf("Зашифрованный блок не соответствует ожидаемому.\nОжидалось: %x\nПолучено: %x", vector.ciphertext, encrypted)
			}

			// Дешифруем блок данных
			decryptFunc := camellia.createDecryptBlockFunc()
			decrypted, err := decryptFunc(encrypted)
			if err != nil {
				t.Fatalf("Ошибка при дешифровании: %v", err)
			}

			// Проверяем, что дешифрованный блок соответствует исходному
			if !bytes.Equal(decrypted, vector.plaintext) {
				t.Fatalf("Дешифрованный блок не соответствует исходному.\nОжидалось: %x\nПолучено: %x", vector.plaintext, decrypted)
			}

			t.Logf("Тестовый вектор успешно проверен")
		})
	}
}

// TestCamelliaRandomDelta проверяет работу режима случайного смещения (Random Delta)
func TestCamelliaRandomDelta(t *testing.T) {
	// Новая реализация RandomDelta производит зашифрованные данные в два раза больше исходных,
	// так как хранит дельту для каждого блока данных

	// Создаем экземпляр Camellia с ключом 128 бит
	camellia, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(CamelliaKey128))
	if err != nil {
		t.Fatalf("Ошибка при генерации ключа: %v", err)
	}

	// Устанавливаем ключ
	err = camellia.SetupKey(key)
	if err != nil {
		t.Fatalf("Ошибка при установке ключа: %v", err)
	}

	// Генерируем IV
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Ошибка при генерации IV: %v", err)
	}

	// Создаем режим Random Delta
	randomDeltaMode, err := NewBlockCipherMode(ModeRandomDelta, camellia.BlockSize())
	if err != nil {
		t.Fatalf("Ошибка при создании режима Random Delta: %v", err)
	}

	// Тестовые сообщения - они должны быть кратны размеру блока для корректной работы RandomDelta
	testMessages := []struct {
		name string
		data []byte
	}{
		{"Сообщение размером с блок", make([]byte, camellia.BlockSize())},
		{"Сообщение в два блока", make([]byte, camellia.BlockSize()*2)},
	}

	// Заполняем тестовые данные
	for i := range testMessages {
		for j := 0; j < len(testMessages[i].data); j++ {
			testMessages[i].data[j] = byte(j % 256)
		}
	}

	for _, msg := range testMessages {
		t.Run(msg.name, func(t *testing.T) {
			// Для RandomDelta данные должны быть кратны размеру блока,
			// поэтому не применяем набивку
			if len(msg.data)%camellia.BlockSize() != 0 {
				t.Fatalf("Для режима RandomDelta данные должны быть кратны размеру блока")
			}

			// Шифруем данные
			encrypted, err := randomDeltaMode.Encrypt(msg.data, iv, camellia.createEncryptBlockFunc())
			if err != nil {
				t.Fatalf("Ошибка при шифровании в режиме Random Delta: %v", err)
			}

			// Проверяем, что зашифрованные данные в два раза больше исходных
			if len(encrypted) != len(msg.data)*2 {
				t.Fatalf("Ожидаемый размер зашифрованных данных: %d, получено: %d", len(msg.data)*2, len(encrypted))
			}

			// Дешифруем данные
			decrypted, err := randomDeltaMode.Decrypt(encrypted, iv, camellia.createDecryptBlockFunc())
			if err != nil {
				t.Fatalf("Ошибка при дешифровании в режиме Random Delta: %v", err)
			}

			// Проверяем, что расшифрованный текст совпадает с исходным
			if !bytes.Equal(msg.data, decrypted) {
				t.Fatalf("Random Delta: расшифрованный текст не совпадает с исходным")
			}

			// Повторное шифрование того же текста должно дать другой результат в Random Delta
			encrypted2, err := randomDeltaMode.Encrypt(msg.data, iv, camellia.createEncryptBlockFunc())
			if err != nil {
				t.Fatalf("Ошибка при повторном шифровании: %v", err)
			}

			// Шифровки должны отличаться из-за случайного компонента
			if bytes.Equal(encrypted, encrypted2) {
				t.Error("Режим Random Delta: повторное шифрование дало тот же результат, что не ожидается от этого режима")
			}

			// Проверяем, что вторая шифровка также корректно дешифруется
			decrypted2, err := randomDeltaMode.Decrypt(encrypted2, iv, camellia.createDecryptBlockFunc())
			if err != nil {
				t.Fatalf("Ошибка при дешифровании второй шифровки: %v", err)
			}

			if !bytes.Equal(msg.data, decrypted2) {
				t.Fatalf("Random Delta (второе шифрование): расшифрованный текст не совпадает с исходным")
			}

			t.Logf("Режим Random Delta успешно протестирован для '%s'", msg.name)
		})
	}

	t.Log("Режим Random Delta успешно протестирован")
}

// TestCamelliaModesUpdated исправляет общий тест для режимов, исключая RandomDelta из обычного теста
func TestCamelliaModesUpdated(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 128 бит
	camellia, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(CamelliaKey128))
	if err != nil {
		t.Fatalf("Ошибка при генерации ключа: %v", err)
	}

	// Устанавливаем ключ
	err = camellia.SetupKey(key)
	if err != nil {
		t.Fatalf("Ошибка при установке ключа: %v", err)
	}

	// Генерируем IV
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Ошибка при генерации IV: %v", err)
	}

	// Тестовое сообщение
	plaintext := []byte("Это тестовое сообщение для проверки разных режимов шифрования.")

	// Создаем набивку PKCS7
	padding := PKCS7Padding()

	// Тестируем обычные режимы шифрования (включая RandomDelta)
	modes := []struct {
		name       BlockCipherMode
		iv         bool // требуется ли IV для этого режима
		doubleSize bool // увеличивается ли размер зашифрованных данных в два раза (для RandomDelta)
	}{
		{ModeECB, false, false},
		{ModeCBC, true, false},
		{ModeCFB, true, false},
		{ModeOFB, true, false},
		{ModeCTR, true, false},
		{ModeRandomDelta, true, true}, // RandomDelta удваивает размер данных
	}

	for _, mode := range modes {
		t.Run(string(mode.name), func(t *testing.T) {
			// Создаем режим шифрования
			cipherMode, err := NewBlockCipherMode(mode.name, camellia.BlockSize())
			if err != nil {
				t.Fatalf("Ошибка при создании режима %s: %v", mode.name, err)
			}

			// Применяем набивку PKCS7 ко всем режимам
			// Хотя для поточных режимов (CFB, OFB, CTR) это необязательно,
			// они могут работать с данными любой длины
			paddedData, err := padding.Pad(plaintext, camellia.BlockSize())
			if err != nil {
				t.Fatalf("Ошибка при применении набивки: %v", err)
			}

			// IV для режимов, которые его используют
			var modeIV []byte
			if mode.iv {
				modeIV = iv
			}

			// Шифруем данные
			encrypted, err := cipherMode.Encrypt(paddedData, modeIV, camellia.createEncryptBlockFunc())
			if err != nil {
				t.Fatalf("Ошибка при шифровании в режиме %s: %v", mode.name, err)
			}

			// Для режима RandomDelta проверяем, что размер зашифрованных данных в два раза больше исходных
			if mode.doubleSize {
				if len(encrypted) != len(paddedData)*2 {
					t.Fatalf("Режим %s: ожидаемый размер зашифрованных данных: %d, получено: %d",
						mode.name, len(paddedData)*2, len(encrypted))
				}
			}

			// Дешифруем данные
			decrypted, err := cipherMode.Decrypt(encrypted, modeIV, camellia.createDecryptBlockFunc())
			if err != nil {
				t.Fatalf("Ошибка при дешифровании в режиме %s: %v", mode.name, err)
			}

			// Убираем набивку для всех режимов
			unpaddedData, err := padding.Unpad(decrypted, camellia.BlockSize())
			if err != nil {
				t.Fatalf("Ошибка при удалении набивки: %v", err)
			}

			// Проверяем, что расшифрованный текст совпадает с исходным
			if !bytes.Equal(plaintext, unpaddedData) {
				t.Fatalf("Режим %s: расшифрованный текст не совпадает с исходным", mode.name)
			}

			// Для RandomDelta проверяем, что повторное шифрование дает другой результат
			if mode.name == ModeRandomDelta {
				encrypted2, err := cipherMode.Encrypt(paddedData, modeIV, camellia.createEncryptBlockFunc())
				if err != nil {
					t.Fatalf("Ошибка при повторном шифровании в режиме %s: %v", mode.name, err)
				}

				// Результаты должны отличаться из-за случайного компонента
				if bytes.Equal(encrypted, encrypted2) {
					t.Errorf("Режим %s: повторное шифрование дало тот же результат, что не ожидается", mode.name)
				}

				// Проверяем, что дешифрование второго результата тоже работает
				decrypted2, err := cipherMode.Decrypt(encrypted2, modeIV, camellia.createDecryptBlockFunc())
				if err != nil {
					t.Fatalf("Ошибка при дешифровании после повторного шифрования: %v", err)
				}

				// Убираем набивку для проверки
				unpaddedData2, err := padding.Unpad(decrypted2, camellia.BlockSize())
				if err != nil {
					t.Fatalf("Ошибка при удалении набивки после повторного шифрования: %v", err)
				}

				if !bytes.Equal(plaintext, unpaddedData2) {
					t.Fatalf("Режим %s после повторного шифрования: расшифрованный текст не совпадает с исходным", mode.name)
				}
			}

			t.Logf("Режим %s: шифрование и дешифрование выполнено успешно", mode.name)
		})
	}
}

// TestCamelliaAllModesAllPaddings проверяет работу всех режимов шифрования Camellia со всеми методами набивки
func TestCamelliaAllModesAllPaddings(t *testing.T) {
	// Создаем экземпляр Camellia с ключом 128 бит
	camellia, err := NewCamellia(CamelliaKey128)
	if err != nil {
		t.Fatalf("Ошибка при создании Camellia: %v", err)
	}

	// Генерируем ключ
	key, err := camellia.GenerateKey(int(CamelliaKey128))
	if err != nil {
		t.Fatalf("Ошибка при генерации ключа: %v", err)
	}

	// Устанавливаем ключ
	err = camellia.setupKey(key)
	if err != nil {
		t.Fatalf("Ошибка при установке ключа: %v", err)
	}

	// Генерируем IV
	iv, err := camellia.GenerateIV()
	if err != nil {
		t.Fatalf("Ошибка при генерации IV: %v", err)
	}

	// Тестовые сообщения разной длины
	testMessages := []struct {
		name string
		data []byte
	}{
		{"Пустое сообщение", []byte{}},
		{"Короткое сообщение", []byte("Короткий текст")},
		{"Сообщение размером с блок", make([]byte, camellia.BlockSize())},
		{"Сообщение длиннее блока", []byte("Это сообщение длиннее размера блока и требует набивки для корректной работы")},
	}

	// Заполняем сообщение размером с блок разными значениями
	for i := 0; i < camellia.BlockSize(); i++ {
		testMessages[2].data[i] = byte(i)
	}

	// Все режимы шифрования для тестирования
	modes := []struct {
		name       BlockCipherMode
		iv         bool // требуется ли IV для этого режима
		doubleSize bool // увеличивается ли размер зашифрованных данных в два раза (для RandomDelta)
	}{
		{ModeECB, false, false},       // ECB не использует IV
		{ModeCBC, true, false},        // CBC требует IV
		{ModeCFB, true, false},        // CFB требует IV
		{ModeOFB, true, false},        // OFB требует IV
		{ModeCTR, true, false},        // CTR требует IV (как начальное значение счетчика)
		{ModeRandomDelta, true, true}, // RandomDelta требует IV и удваивает размер результата
	}

	// Все методы набивки для тестирования
	paddingMethods := []struct {
		name        string
		padding     Padding
		skipForSize func(int, int) bool // Функция, определяющая, нужно ли пропустить тест для определенного размера данных
	}{
		{
			name:    "PKCS7",
			padding: PKCS7Padding(),
			skipForSize: func(dataLen, blockSize int) bool {
				return false // PKCS7 работает со всеми размерами данных
			},
		},
		{
			name:    "ISO10126",
			padding: ISO10126Padding(),
			skipForSize: func(dataLen, blockSize int) bool {
				return false // ISO10126 работает со всеми размерами данных
			},
		},
		{
			name:    "Zeros",
			padding: ZerosPadding(),
			skipForSize: func(dataLen, blockSize int) bool {
				return dataLen == 0 // Zeros не работает с пустыми данными
			},
		},
		{
			name:    "ANSIX923",
			padding: ANSIX923Padding(),
			skipForSize: func(dataLen, blockSize int) bool {
				return false // ANSIX923 работает со всеми размерами данных
			},
		},
	}

	// Создаем необходимые режимы шифрования заранее
	for _, mode := range modes {
		for _, msg := range testMessages {
			for _, padMethod := range paddingMethods {
				// Пропускаем несовместимые сочетания
				if padMethod.skipForSize(len(msg.data), camellia.BlockSize()) {
					continue
				}

				testName := fmt.Sprintf("%s-%s-%s", mode.name, msg.name, padMethod.name)
				t.Run(testName, func(t *testing.T) {
					// Создаем режим шифрования
					cipherMode, err := NewBlockCipherMode(mode.name, camellia.BlockSize())
					if err != nil {
						t.Fatalf("Ошибка при создании режима %s: %v", mode.name, err)
					}

					// Применяем набивку
					paddedData, err := padMethod.padding.Pad(msg.data, camellia.BlockSize())
					if err != nil {
						t.Fatalf("Ошибка при применении набивки %s: %v", padMethod.name, err)
					}

					// IV для режимов, которые его используют
					var modeIV []byte
					if mode.iv {
						modeIV = iv
					}

					// Шифруем данные
					encrypted, err := cipherMode.Encrypt(paddedData, modeIV, camellia.createEncryptBlockFunc())
					if err != nil {
						t.Fatalf("Ошибка при шифровании в режиме %s с набивкой %s: %v",
							mode.name, padMethod.name, err)
					}

					// Для режима RandomDelta проверяем, что размер зашифрованных данных в два раза больше исходных
					if mode.doubleSize {
						if len(encrypted) != len(paddedData)*2 {
							t.Fatalf("Режим %s: ожидаемый размер зашифрованных данных: %d, получено: %d",
								mode.name, len(paddedData)*2, len(encrypted))
						}
					}

					// Дешифруем данные
					decrypted, err := cipherMode.Decrypt(encrypted, modeIV, camellia.createDecryptBlockFunc())
					if err != nil {
						t.Fatalf("Ошибка при дешифровании в режиме %s с набивкой %s: %v",
							mode.name, padMethod.name, err)
					}

					// Убираем набивку
					unpaddedData, err := padMethod.padding.Unpad(decrypted, camellia.BlockSize())
					if err != nil {
						t.Fatalf("Ошибка при удалении набивки %s: %v", padMethod.name, err)
					}

					// Проверяем, что расшифрованный текст совпадает с исходным
					if !bytes.Equal(msg.data, unpaddedData) {
						t.Fatalf("Режим %s с набивкой %s: расшифрованный текст не совпадает с исходным",
							mode.name, padMethod.name)
					}

					// Для RandomDelta также проверяем, что повторное шифрование дает другой результат
					if mode.name == ModeRandomDelta {
						encrypted2, err := cipherMode.Encrypt(paddedData, modeIV, camellia.createEncryptBlockFunc())
						if err != nil {
							t.Fatalf("Ошибка при повторном шифровании в режиме %s: %v", mode.name, err)
						}

						// Проверяем, что результаты различаются
						if bytes.Equal(encrypted, encrypted2) {
							t.Errorf("Режим %s: повторное шифрование дало тот же результат, что не ожидается", mode.name)
						}

						// Проверяем, что дешифрование второго результата также работает корректно
						decrypted2, err := cipherMode.Decrypt(encrypted2, modeIV, camellia.createDecryptBlockFunc())
						if err != nil {
							t.Fatalf("Ошибка при дешифровании после повторного шифрования: %v", err)
						}

						unpaddedData2, err := padMethod.padding.Unpad(decrypted2, camellia.BlockSize())
						if err != nil {
							t.Fatalf("Ошибка при удалении набивки после повторного шифрования: %v", err)
						}

						if !bytes.Equal(msg.data, unpaddedData2) {
							t.Fatalf("Режим %s после повторного шифрования: расшифрованный текст не совпадает с исходным", mode.name)
						}
					}

					// Сообщаем об успешном выполнении теста
					t.Logf("Режим %s с набивкой %s для '%s': шифрование и дешифрование выполнено успешно",
						mode.name, padMethod.name, msg.name)
				})
			}
		}
	}
}
