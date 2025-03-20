package cipher

import (
	"context"
	"errors"
)

// Ошибки, связанные с шифрованием
var (
	ErrEncryption       = errors.New("ошибка шифрования")
	ErrDecryption       = errors.New("ошибка дешифрования")
	ErrInvalidKey       = errors.New("недействительный ключ")
	ErrInvalidBlockSize = errors.New("недействительный размер блока")
	ErrInvalidIV        = errors.New("недействительный вектор инициализации")
	ErrInvalidPadding   = errors.New("недействительная набивка")
	ErrInvalidMode      = errors.New("недействительный режим шифрования")
)

// BlockCipherMode определяет режим работы блочного шифра
type BlockCipherMode string

// Доступные режимы работы блочного шифра
const (
	ModeECB         BlockCipherMode = "ECB"         // Electronic Codebook (небезопасный, только для тестов)
	ModeCBC         BlockCipherMode = "CBC"         // Cipher Block Chaining
	ModePCBC        BlockCipherMode = "PCBC"        // Propagating Cipher Block Chaining
	ModeCFB         BlockCipherMode = "CFB"         // Cipher Feedback
	ModeOFB         BlockCipherMode = "OFB"         // Output Feedback
	ModeCTR         BlockCipherMode = "CTR"         // Counter
	ModeRandomDelta BlockCipherMode = "RandomDelta" // Random Delta
)

// PaddingMethod определяет метод набивки для блочных шифров
type PaddingMethod string

// Доступные методы набивки
const (
	PaddingPKCS7    PaddingMethod = "PKCS7"    // PKCS#7 набивка
	PaddingISO10126 PaddingMethod = "ISO10126" // ISO/IEC 10126 набивка (случайные байты с последним байтом, указывающим на длину)
	PaddingZeros    PaddingMethod = "ZEROS"    // Набивка нулями (добавление нулей до размера блока, но без маркера конца)
	PaddingANSIX923 PaddingMethod = "ANSIX923" // ANSI X.923 набивка (нулевые байты с последним байтом, указывающим на длину)
)

// SymmetricCipher определяет интерфейс для симметричных алгоритмов шифрования
type SymmetricCipher interface {
	// Encrypt шифрует plaintext с использованием ключа
	// Возвращает зашифрованные данные или ошибку
	Encrypt(ctx context.Context, plaintext []byte, key []byte) ([]byte, error)

	// Decrypt расшифровывает ciphertext с использованием ключа
	// Возвращает дешифрованные данные или ошибку
	Decrypt(ctx context.Context, ciphertext []byte, key []byte) ([]byte, error)

	// EncryptWithIV шифрует plaintext с использованием ключа и вектора инициализации
	// iv - вектор инициализации для режимов CBC/CFB/OFB или счетчик для CTR
	// Возвращает зашифрованные данные или ошибку
	EncryptWithIV(ctx context.Context, plaintext []byte, key, iv []byte) ([]byte, error)

	// DecryptWithIV расшифровывает ciphertext с использованием ключа и вектора инициализации
	// iv - вектор инициализации для режимов CBC/CFB/OFB или счетчик для CTR
	// Возвращает дешифрованные данные или ошибку
	DecryptWithIV(ctx context.Context, ciphertext []byte, key, iv []byte) ([]byte, error)

	// GenerateKey создает новый случайный ключ подходящей длины
	// keySize может быть использован для определения размера ключа в битах
	GenerateKey(keySize int) ([]byte, error)

	// GenerateIV создает новый случайный вектор инициализации
	GenerateIV() ([]byte, error)

	// Name возвращает имя алгоритма шифрования
	Name() string

	// KeySizes возвращает список поддерживаемых размеров ключей в битах
	KeySizes() []int

	// BlockSize возвращает размер блока в байтах для блочных шифров
	// или 0 для потоковых шифров
	BlockSize() int

	// IVSize возвращает размер вектора инициализации в байтах
	IVSize() int
}

// Padding предоставляет методы для добавления и удаления набивки
type Padding interface {
	// Pad добавляет набивку к данным в соответствии с выбранным методом
	// blockSize - размер блока в байтах
	Pad(data []byte, blockSize int) ([]byte, error)

	// Unpad удаляет набивку из данных в соответствии с выбранным методом
	Unpad(data []byte, blockSize int) ([]byte, error)

	// Method возвращает используемый метод набивки
	Method() PaddingMethod
}

// CipherConfig содержит конфигурацию для шифрования
type CipherConfig struct {
	// Algorithm определяет базовый алгоритм шифрования ("aes", "des", "blowfish" и т.д.)
	Algorithm string

	// Mode определяет режим работы шифра
	Mode BlockCipherMode

	// PaddingMethod определяет метод набивки для блочных шифров
	PaddingMethod PaddingMethod

	// KeySize определяет размер ключа в битах
	KeySize int

	// AdditionalOptions содержит дополнительные опции, специфичные для алгоритма
	AdditionalOptions map[string]interface{}
}

// CipherFactory создает экземпляр SymmetricCipher на основе конфигурации
type CipherFactory interface {
	// CreateCipher создает экземпляр SymmetricCipher на основе конфигурации
	CreateCipher(config CipherConfig) (SymmetricCipher, error)

	// CreatePadding создает реализацию интерфейса Padding на основе метода
	CreatePadding(method PaddingMethod) (Padding, error)

	// GetDefaultConfig возвращает конфигурацию по умолчанию для указанного алгоритма
	GetDefaultConfig(algorithm string) CipherConfig

	// AvailableAlgorithms возвращает список доступных базовых алгоритмов шифрования
	AvailableAlgorithms() []string

	// AvailableModes возвращает список доступных режимов для указанного алгоритма
	AvailableModes(algorithm string) []BlockCipherMode

	// AvailablePaddingMethods возвращает список доступных методов набивки
	AvailablePaddingMethods() []PaddingMethod
}

// Utils предоставляет вспомогательные функции для работы с данными
type Utils interface {
	// StringToBytes преобразует строку в []byte
	StringToBytes(s string) []byte

	// BytesToString преобразует []byte в строку
	BytesToString(b []byte) string

	// HexToBytes преобразует шестнадцатеричную строку в []byte
	HexToBytes(hex string) ([]byte, error)

	// BytesToHex преобразует []byte в шестнадцатеричную строку
	BytesToHex(b []byte) string

	// Base64Encode кодирует []byte в строку base64
	Base64Encode(b []byte) string

	// Base64Decode декодирует строку base64 в []byte
	Base64Decode(s string) ([]byte, error)

	// RandomBytes генерирует случайные байты указанной длины
	RandomBytes(length int) ([]byte, error)
}
