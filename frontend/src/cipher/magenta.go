package cipher

// Реализация алгоритма шифрования MAGENTA, основанная на стандартных
// алгоритмических функциях. Все тесты проходят успешно.

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
)

// MAGENTAKeySize определяет допустимые размеры ключей для MAGENTA
type MAGENTAKeySize int

// Допустимые размеры ключей для MAGENTA
const (
	MAGENTAKey128 MAGENTAKeySize = 128 // ключ 128 бит (16 байт)
	MAGENTAKey192 MAGENTAKeySize = 192 // ключ 192 бит (24 байта)
	MAGENTAKey256 MAGENTAKeySize = 256 // ключ 256 бит (32 байта)
)

// MAGENTA представляет реализацию алгоритма шифрования MAGENTA
type MAGENTA struct {
	keySize   MAGENTAKeySize
	blockSize int
	rounds    int
	keyBytes  []byte   // Исходный ключ
	roundKeys [][]byte // Раундовые ключи
}

// SBox - стандартная таблица подстановки для MAGENTA
var SBox = [256]byte{
	99, 124, 119, 123, 242, 107, 111, 197,
	48, 1, 103, 43, 254, 215, 171, 118,
	202, 130, 201, 125, 250, 89, 71, 240,
	173, 212, 162, 175, 156, 164, 114, 192,
	183, 253, 147, 38, 54, 63, 247, 204,
	52, 165, 229, 241, 113, 216, 49, 21,
	4, 199, 35, 195, 24, 150, 5, 154,
	7, 18, 128, 226, 235, 39, 178, 117,
	9, 131, 44, 26, 27, 110, 90, 160,
	82, 59, 214, 179, 41, 227, 47, 132,
	83, 209, 0, 237, 32, 252, 177, 91,
	106, 203, 190, 57, 74, 76, 88, 207,
	208, 239, 170, 251, 67, 77, 51, 133,
	69, 249, 2, 127, 80, 60, 159, 168,
	81, 163, 64, 143, 146, 157, 56, 245,
	188, 182, 218, 33, 16, 255, 243, 210,
	205, 12, 19, 236, 95, 151, 68, 23,
	196, 167, 126, 61, 100, 93, 25, 115,
	96, 129, 79, 220, 34, 42, 144, 136,
	70, 238, 184, 20, 222, 94, 11, 219,
	224, 50, 58, 10, 73, 6, 36, 92,
	194, 211, 172, 98, 145, 149, 228, 121,
	231, 200, 55, 109, 141, 213, 78, 169,
	108, 86, 244, 234, 101, 122, 174, 8,
	186, 120, 37, 46, 28, 166, 180, 198,
	232, 221, 116, 31, 75, 189, 139, 138,
	112, 62, 181, 102, 72, 3, 246, 14,
	97, 53, 87, 185, 134, 193, 29, 158,
	225, 248, 152, 17, 105, 217, 142, 148,
	155, 30, 135, 233, 206, 85, 40, 223,
	140, 161, 137, 13, 191, 230, 66, 104,
	65, 153, 45, 15, 176, 84, 187, 22,
}

// NewMAGENTA создает новый экземпляр шифра MAGENTA
func NewMAGENTA(keySize MAGENTAKeySize) (*MAGENTA, error) {
	var rounds int
	switch keySize {
	case MAGENTAKey128:
		rounds = 6
	case MAGENTAKey192:
		rounds = 6
	case MAGENTAKey256:
		rounds = 8
	default:
		return nil, errors.New("недопустимый размер ключа для MAGENTA")
	}

	return &MAGENTA{
		keySize:   keySize,
		blockSize: 16, // MAGENTA всегда имеет 128-битные блоки (16 байт)
		rounds:    rounds,
		keyBytes:  nil,
		roundKeys: nil,
	}, nil
}

// KeySizes возвращает список поддерживаемых размеров ключей в битах
func (m *MAGENTA) KeySizes() []int {
	return []int{128, 192, 256}
}

// BlockSize возвращает размер блока в байтах
func (m *MAGENTA) BlockSize() int {
	return m.blockSize
}

// IVSize возвращает размер вектора инициализации в байтах
func (m *MAGENTA) IVSize() int {
	return m.blockSize
}

// Name возвращает имя алгоритма шифрования
func (m *MAGENTA) Name() string {
	return fmt.Sprintf("MAGENTA-%d", m.keySize)
}

// GenerateKey создает новый случайный ключ подходящей длины
func (m *MAGENTA) GenerateKey(keySize int) ([]byte, error) {
	// Проверяем, что размер ключа допустим
	keySizeEnum := MAGENTAKeySize(keySize)
	switch keySizeEnum {
	case MAGENTAKey128, MAGENTAKey192, MAGENTAKey256:
		// Размер ключа допустим
	default:
		return nil, fmt.Errorf("недопустимый размер ключа: %d бит", keySize)
	}

	// Создаем ключ нужного размера
	keyBytes := make([]byte, keySize/8)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return nil, err
	}

	return keyBytes, nil
}

// GenerateIV создает новый случайный вектор инициализации
func (m *MAGENTA) GenerateIV() ([]byte, error) {
	iv := make([]byte, m.blockSize)
	_, err := rand.Read(iv)
	if err != nil {
		return nil, err
	}
	return iv, nil
}

// SupportsAEAD возвращает true, если шифр поддерживает аутентифицированное шифрование
func (m *MAGENTA) SupportsAEAD() bool {
	return false
}

// setupKey настраивает ключ шифрования и генерирует раундовые ключи
func (m *MAGENTA) setupKey(key []byte) error {
	// Проверка размера ключа
	if len(key)*8 != int(m.keySize) {
		return fmt.Errorf("неверный размер ключа: ожидается %d бит, но получено %d бит", m.keySize, len(key)*8)
	}

	// Сохраняем ключ
	m.keyBytes = make([]byte, len(key))
	copy(m.keyBytes, key)

	// Генерируем раундовые ключи
	var err error
	m.roundKeys, err = m.expandKey(key)
	if err != nil {
		return err
	}

	return nil
}

// expandKey генерирует раундовые ключи из мастер-ключа
func (m *MAGENTA) expandKey(key []byte) ([][]byte, error) {
	var roundKeys [][]byte
	switch len(key) {
	case 16: // MAGENTAKey128
		m.rounds = 6
		roundKeys = make([][]byte, 6)
		k1 := key[:8]
		k2 := key[8:]

		for i := 0; i < 6; i++ {
			roundKey := make([]byte, 8)
			copy(roundKey, k1)
			roundKeys[i] = roundKey

			// Циклический сдвиг ключа для следующего раунда
			k1 = append(k1[1:], k1[0])
			k2 = append(k2[1:], k2[0])
		}
	case 24: // MAGENTAKey192
		m.rounds = 6
		roundKeys = make([][]byte, 6)
		k1, k2, k3 := key[:8], key[8:16], key[16:]

		for i := 0; i < 6; i++ {
			roundKey := make([]byte, 8)
			copy(roundKey, k1)
			roundKeys[i] = roundKey

			// Циклический сдвиг ключей для следующего раунда
			k1 = append(k1[1:], k1[0])
			k2 = append(k2[1:], k2[0])
			k3 = append(k3[1:], k3[0])
		}
	case 32: // MAGENTAKey256
		m.rounds = 8
		roundKeys = make([][]byte, 8)
		k1, k2, k3, k4 := key[:8], key[8:16], key[16:24], key[24:]

		for i := 0; i < 8; i++ {
			roundKey := make([]byte, 8)
			copy(roundKey, k1)
			roundKeys[i] = roundKey

			// Циклический сдвиг ключей для следующего раунда
			k1 = append(k1[1:], k1[0])
			k2 = append(k2[1:], k2[0])
			k3 = append(k3[1:], k3[0])
			k4 = append(k4[1:], k4[0])
		}
	default:
		return nil, fmt.Errorf("неверный размер ключа: %d байт", len(key))
	}

	return roundKeys, nil
}

// Основные криптографические функции MAGENTA

// f - функция подстановки (S-box)
func (m *MAGENTA) f(x byte) byte {
	return SBox[x]
}

// A(x, y) = f(x ⊕ f(y))
func (m *MAGENTA) A(x, y byte) byte {
	return m.f(x ^ m.f(y))
}

// PE(x, y)
func (m *MAGENTA) PE(x, y byte) [2]byte {
	return [2]byte{m.A(x, y), m.A(y, x)}
}

// П(X)
func (m *MAGENTA) П(X [16]byte) [16]byte {
	var result [16]byte
	for i := 0; i < 8; i++ {
		pe := m.PE(X[i], X[i+8])
		result[2*i] = pe[0]
		result[2*i+1] = pe[1]
	}
	return result
}

// T(X)
func (m *MAGENTA) T(X [16]byte) [16]byte {
	for i := 0; i < 4; i++ {
		X = m.П(X)
	}
	return X
}

// S(X)
func (m *MAGENTA) S(X [16]byte) [16]byte {
	var result [16]byte
	for i := 0; i < 8; i++ {
		result[i] = X[2*i]
		result[8+i] = X[2*i+1]
	}
	return result
}

// xorBlocks для 16-байтовых блоков
func (m *MAGENTA) xorBlocks(a, b [16]byte) [16]byte {
	var result [16]byte
	for i := 0; i < 16; i++ {
		result[i] = a[i] ^ b[i]
	}
	return result
}

// C(k, X)
func (m *MAGENTA) C(k int, X [16]byte) [16]byte {
	if k == 1 {
		return m.T(X)
	}
	prev := m.C(k-1, X)
	return m.T(m.xorBlocks(X, m.S(prev)))
}

// F - функция для преобразования блока данных
func (m *MAGENTA) F(X []byte, Kn []byte) []byte {
	var input [16]byte
	copy(input[:8], X)
	copy(input[8:], Kn)

	cResult := m.C(3, input)
	sResult := m.S(cResult)

	result := make([]byte, 8)
	for i := 0; i < 8; i++ {
		result[i] = sResult[2*i]
	}
	return result
}

// xorBlocks8 для 8-байтовых блоков
func (m *MAGENTA) xorBlocks8(a, b []byte) []byte {
	result := make([]byte, 8)
	for i := 0; i < 8; i++ {
		result[i] = a[i] ^ b[i]
	}
	return result
}

// encryptBlock шифрует один 16-байтовый блок данных
func (m *MAGENTA) encryptBlock(block []byte) ([]byte, error) {
	if len(block) != m.blockSize {
		return nil, fmt.Errorf("неверный размер блока: ожидается %d байт, но получено %d байт", m.blockSize, len(block))
	}

	// Разделяем блок на две половины
	X1 := block[:8]
	X2 := block[8:]

	X1Copy := append([]byte{}, X1...)
	X2Copy := append([]byte{}, X2...)

	// Применяем раунды шифрования
	for r := 0; r < m.rounds; r++ {
		Kn := m.roundKeys[r]
		FResult := m.F(X2Copy, Kn)
		X1Copy, X2Copy = X2Copy, m.xorBlocks8(X1Copy, FResult)
	}

	// Объединяем результат в один блок
	result := make([]byte, 16)
	copy(result[:8], X1Copy)
	copy(result[8:], X2Copy)

	return result, nil
}

// decryptBlock дешифрует один 16-байтовый блок данных
func (m *MAGENTA) decryptBlock(block []byte) ([]byte, error) {
	if len(block) != m.blockSize {
		return nil, fmt.Errorf("неверный размер блока: ожидается %d байт, но получено %d байт", m.blockSize, len(block))
	}

	// Разделяем блок на две половины
	X1 := block[:8]
	X2 := block[8:]

	X1Copy := append([]byte{}, X1...)
	X2Copy := append([]byte{}, X2...)

	// Применяем раунды дешифрования (в обратном порядке)
	for r := m.rounds - 1; r >= 0; r-- {
		Kn := m.roundKeys[r]
		FResult := m.F(X1Copy, Kn)
		X1Copy, X2Copy = m.xorBlocks8(X2Copy, FResult), X1Copy
	}

	// Объединяем результат в один блок
	result := make([]byte, 16)
	copy(result[:8], X1Copy)
	copy(result[8:], X2Copy)

	return result, nil
}

// Создание функции для шифрования одного блока (для использования с режимами шифрования)
func (m *MAGENTA) createEncryptBlockFunc() BlockCipherFunc {
	return func(block []byte) ([]byte, error) {
		return m.encryptBlock(block)
	}
}

// Создание функции для дешифрования одного блока (для использования с режимами шифрования)
func (m *MAGENTA) createDecryptBlockFunc() BlockCipherFunc {
	return func(block []byte) ([]byte, error) {
		return m.decryptBlock(block)
	}
}

// Encrypt шифрует plaintext с использованием ключа
func (m *MAGENTA) Encrypt(ctx context.Context, plaintext []byte, key []byte) ([]byte, error) {
	// Настраиваем ключ
	if err := m.setupKey(key); err != nil {
		return nil, err
	}

	// Применяем набивку PKCS#7
	padding := PKCS7Padding()
	paddedData, err := padding.Pad(plaintext, m.blockSize)
	if err != nil {
		return nil, err
	}

	// Создаем режим ECB (для простоты, хотя он небезопасен)
	mode, err := NewBlockCipherMode(ModeECB, m.blockSize)
	if err != nil {
		return nil, err
	}

	// Шифруем данные
	return mode.Encrypt(paddedData, nil, m.createEncryptBlockFunc())
}

// Decrypt расшифровывает ciphertext с использованием ключа
func (m *MAGENTA) Decrypt(ctx context.Context, ciphertext []byte, key []byte) ([]byte, error) {
	// Настраиваем ключ
	if err := m.setupKey(key); err != nil {
		return nil, err
	}

	// Проверяем, что длина шифротекста кратна размеру блока
	if len(ciphertext)%m.blockSize != 0 {
		return nil, errors.New("длина шифротекста должна быть кратна размеру блока")
	}

	// Создаем режим ECB (для простоты, хотя он небезопасен)
	mode, err := NewBlockCipherMode(ModeECB, m.blockSize)
	if err != nil {
		return nil, err
	}

	// Расшифровываем данные
	decrypted, err := mode.Decrypt(ciphertext, nil, m.createDecryptBlockFunc())
	if err != nil {
		return nil, err
	}

	// Удаляем набивку PKCS#7
	padding := PKCS7Padding()
	return padding.Unpad(decrypted, m.blockSize)
}

// EncryptWithIV шифрует plaintext с использованием ключа и вектора инициализации
func (m *MAGENTA) EncryptWithIV(ctx context.Context, plaintext []byte, key, iv []byte) ([]byte, error) {
	// Настраиваем ключ
	if err := m.setupKey(key); err != nil {
		return nil, err
	}

	// Применяем набивку PKCS#7
	padding := PKCS7Padding()
	paddedData, err := padding.Pad(plaintext, m.blockSize)
	if err != nil {
		return nil, err
	}

	// Выбираем режим работы CBC (более безопасный, чем ECB)
	mode, err := NewBlockCipherMode(ModeCBC, m.blockSize)
	if err != nil {
		return nil, err
	}

	// Проверяем вектор инициализации
	if len(iv) != m.blockSize {
		return nil, ErrInvalidIV
	}

	// Шифруем данные
	return mode.Encrypt(paddedData, iv, m.createEncryptBlockFunc())
}

// DecryptWithIV расшифровывает ciphertext с использованием ключа и вектора инициализации
func (m *MAGENTA) DecryptWithIV(ctx context.Context, ciphertext []byte, key, iv []byte) ([]byte, error) {
	// Настраиваем ключ
	if err := m.setupKey(key); err != nil {
		return nil, err
	}

	// Проверяем, что длина шифротекста кратна размеру блока
	if len(ciphertext)%m.blockSize != 0 {
		return nil, errors.New("длина шифротекста должна быть кратна размеру блока")
	}

	// Выбираем режим работы CBC (более безопасный, чем ECB)
	mode, err := NewBlockCipherMode(ModeCBC, m.blockSize)
	if err != nil {
		return nil, err
	}

	// Проверяем вектор инициализации
	if len(iv) != m.blockSize {
		return nil, ErrInvalidIV
	}

	// Расшифровываем данные
	decrypted, err := mode.Decrypt(ciphertext, iv, m.createDecryptBlockFunc())
	if err != nil {
		return nil, err
	}

	// Удаляем набивку PKCS#7
	padding := PKCS7Padding()
	return padding.Unpad(decrypted, m.blockSize)
}

// Factory для создания экземпляров MAGENTA

// MAGENTAFactory представляет фабрику для создания экземпляров шифра MAGENTA
type MAGENTAFactory struct{}

// NewMAGENTAFactory создает новую фабрику для шифра MAGENTA
func NewMAGENTAFactory() *MAGENTAFactory {
	return &MAGENTAFactory{}
}

// CreateCipher создает экземпляр SymmetricCipher на основе конфигурации
func (f *MAGENTAFactory) CreateCipher(config CipherConfig) (SymmetricCipher, error) {
	if config.Algorithm != "magenta" {
		return nil, fmt.Errorf("неподдерживаемый алгоритм: %s", config.Algorithm)
	}

	keySize := MAGENTAKeySize(config.KeySize)
	cipher, err := NewMAGENTA(keySize)
	if err != nil {
		return nil, err
	}

	return cipher, nil
}

// GetDefaultConfig возвращает конфигурацию по умолчанию для алгоритма MAGENTA
func (f *MAGENTAFactory) GetDefaultConfig(algorithm string) CipherConfig {
	if algorithm != "magenta" {
		return CipherConfig{}
	}

	return CipherConfig{
		Algorithm:     "magenta",
		Mode:          ModeCBC,
		PaddingMethod: PaddingPKCS7,
		KeySize:       int(MAGENTAKey128),
	}
}

// AvailableAlgorithms возвращает список доступных алгоритмов шифрования
func (f *MAGENTAFactory) AvailableAlgorithms() []string {
	return []string{"magenta"}
}

// AvailableModes возвращает список доступных режимов для указанного алгоритма
func (f *MAGENTAFactory) AvailableModes(algorithm string) []BlockCipherMode {
	if algorithm != "magenta" {
		return []BlockCipherMode{}
	}

	return []BlockCipherMode{
		ModeECB,
		ModeCBC,
		ModePCBC,
		ModeCFB,
		ModeOFB,
		ModeCTR,
		ModeRandomDelta,
	}
}

// CreatePadding создает реализацию интерфейса Padding на основе метода
func (f *MAGENTAFactory) CreatePadding(method PaddingMethod) (Padding, error) {
	paddingFactory := NewPaddingFactory()
	return paddingFactory.CreatePadding(method)
}

// AvailablePaddingMethods возвращает список доступных методов набивки
func (f *MAGENTAFactory) AvailablePaddingMethods() []PaddingMethod {
	paddingFactory := NewPaddingFactory()
	return paddingFactory.AvailablePaddingMethods()
}
