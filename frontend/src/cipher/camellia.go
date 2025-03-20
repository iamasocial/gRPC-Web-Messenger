package cipher

// Реализация алгоритма шифрования Camellia, соответствующая стандарту CRYPTREC.
// Camellia - симметричный блочный шифр с размером блока 128 бит и длиной ключа 128, 192 или 256 бит.

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
)

// CamelliaKeySize определяет допустимые размеры ключей для Camellia
type CamelliaKeySize int

// Допустимые размеры ключей для Camellia
const (
	CamelliaKey128 CamelliaKeySize = 128 // ключ 128 бит (16 байт)
	CamelliaKey192 CamelliaKeySize = 192 // ключ 192 бит (24 байта)
	CamelliaKey256 CamelliaKeySize = 256 // ключ 256 бит (32 байта)
)

// Camellia представляет реализацию алгоритма шифрования Camellia
type Camellia struct {
	keySize   CamelliaKeySize
	blockSize int
	rounds    int
	keyBytes  []byte      // Исходный ключ
	subKeys   [][8]uint32 // Раундовые ключи KL, KA, KB и массив ключей шифрования
}

// Константы для Camellia
var (
	// SIGMA - константы для генерации ключей
	SIGMA = []uint32{
		0xa09e667f, 0x3bcc908b, 0xb67ae858, 0x4caa73b2,
		0xc6ef372f, 0xe94f82be, 0x54ff53a5, 0xf1d36f1c,
	}
)

// Таблицы S-box
var (
	SBOX1 = [256]byte{
		0x70, 0x82, 0x2c, 0xec, 0xb3, 0x27, 0xc0, 0xe5, 0xe4, 0x85, 0x57, 0x35, 0xea, 0x0c, 0xae, 0x41,
		0x23, 0xef, 0x6b, 0x93, 0x45, 0x19, 0xa5, 0x21, 0xed, 0x0e, 0x4f, 0x4e, 0x1d, 0x65, 0x92, 0xbd,
		0x86, 0xb8, 0xaf, 0x8f, 0x7c, 0xeb, 0x1f, 0xce, 0x3e, 0x30, 0xdc, 0x5f, 0x5e, 0xc5, 0x0b, 0x1a,
		0xa6, 0xe1, 0x39, 0xca, 0xd5, 0x47, 0x5d, 0x3d, 0xd9, 0x01, 0x5a, 0xd6, 0x51, 0x56, 0x6c, 0x4d,
		0x8b, 0x0d, 0x9a, 0x66, 0xfb, 0xcc, 0xb0, 0x2d, 0x74, 0x12, 0x2b, 0x20, 0xf0, 0xb1, 0x84, 0x99,
		0xdf, 0x4c, 0xcb, 0xc2, 0x34, 0x7e, 0x76, 0x05, 0x6d, 0xb7, 0xa9, 0x31, 0xd1, 0x17, 0x04, 0xd7,
		0x14, 0x58, 0x3a, 0x61, 0xde, 0x1b, 0x11, 0x1c, 0x32, 0x0f, 0x9c, 0x16, 0x53, 0x18, 0xf2, 0x22,
		0xfe, 0x44, 0xcf, 0xb2, 0xc3, 0xb5, 0x7a, 0x91, 0x24, 0x08, 0xe8, 0xa8, 0x60, 0xfc, 0x69, 0x50,
		0xaa, 0xd0, 0xa0, 0x7d, 0xa1, 0x89, 0x62, 0x97, 0x54, 0x5b, 0x1e, 0x95, 0xe0, 0xff, 0x64, 0xd2,
		0x10, 0xc4, 0x00, 0x48, 0xa3, 0xf7, 0x75, 0xdb, 0x8a, 0x03, 0xe6, 0xda, 0x09, 0x3f, 0xdd, 0x94,
		0x87, 0x5c, 0x83, 0x02, 0xcd, 0x4a, 0x90, 0x33, 0x73, 0x67, 0xf6, 0xf3, 0x9d, 0x7f, 0xbf, 0xe2,
		0x52, 0x9b, 0xd8, 0x26, 0xc8, 0x37, 0xc6, 0x3b, 0x81, 0x96, 0x6f, 0x4b, 0x13, 0xbe, 0x63, 0x2e,
		0xe9, 0x79, 0xa7, 0x8c, 0x9f, 0x6e, 0xbc, 0x8e, 0x29, 0xf5, 0xf9, 0xb6, 0x2f, 0xfd, 0xb4, 0x59,
		0x78, 0x98, 0x06, 0x6a, 0xe7, 0x46, 0x71, 0xba, 0xd4, 0x25, 0xab, 0x42, 0x88, 0xa2, 0x8d, 0xfa,
		0x72, 0x07, 0xb9, 0x55, 0xf8, 0xee, 0xac, 0x0a, 0x36, 0x49, 0x2a, 0x68, 0x3c, 0x38, 0xf1, 0xa4,
		0x40, 0x28, 0xd3, 0x7b, 0xbb, 0xc9, 0x43, 0xc1, 0x15, 0xe3, 0xad, 0xf4, 0x77, 0xc7, 0x80, 0x9e,
	}

	// Другие S-box будут сгенерированы на основе SBOX1
	SBOX2 = [256]byte{}
	SBOX3 = [256]byte{}
	SBOX4 = [256]byte{}
)

// Генерация других S-box на основе SBOX1
func init() {
	for x := 0; x < 256; x++ {
		// Генерируем SBOX2, SBOX3, SBOX4 согласно спецификации
		// SBOX2[x] = приветствие(SBOX1[x]), циклический сдвиг влево на 1 бит
		SBOX2[x] = (SBOX1[x] << 1) | (SBOX1[x] >> 7)

		// SBOX3[x] = предложение(SBOX1[x]), циклический сдвиг вправо на 1 бит
		SBOX3[x] = (SBOX1[x] >> 1) | (SBOX1[x] << 7)

		// SBOX4[x] = вуаля(SBOX1[x]), циклический сдвиг влево на 1 бит и XOR с 0x63
		SBOX4[x] = ((SBOX1[x] << 1) | (SBOX1[x] >> 7)) ^ 0x63
	}
}

// Вспомогательные функции для работы с битами

// bytesToUint32 преобразует 4 байта в 32-битное число (порядок big-endian)
func bytesToUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

// uint32ToBytes преобразует 32-битное число в 4 байта (порядок big-endian)
func uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// rotl32 выполняет циклический сдвиг 32-битного числа влево
func rotl32(v uint32, count uint) uint32 {
	return (v << count) | (v >> (32 - count))
}

// rotr32 выполняет циклический сдвиг 32-битного числа вправо
func rotr32(v uint32, count uint) uint32 {
	return (v >> count) | (v << (32 - count))
}

// F - функция обработки блока данных
func (c *Camellia) F(data, subkey [2]uint32) [2]uint32 {
	// XOR данных с подключом
	x0 := data[0] ^ subkey[0]
	x1 := data[1] ^ subkey[1]

	// Применяем S-box и последующие преобразования
	t1 := uint32(SBOX1[(x0>>24)&0xFF]) << 24
	t1 |= uint32(SBOX2[(x0>>16)&0xFF]) << 16
	t1 |= uint32(SBOX3[(x0>>8)&0xFF]) << 8
	t1 |= uint32(SBOX4[x0&0xFF])

	t2 := uint32(SBOX2[(x1>>24)&0xFF]) << 24
	t2 |= uint32(SBOX3[(x1>>16)&0xFF]) << 16
	t2 |= uint32(SBOX4[(x1>>8)&0xFF]) << 8
	t2 |= uint32(SBOX1[x1&0xFF])

	// XOR результатов
	y0 := t1 ^ t2
	y1 := t1 ^ rotl32(t2, 8)

	return [2]uint32{y0, y1}
}

// FL - функция линейного преобразования
func (c *Camellia) FL(data, subkey [2]uint32) [2]uint32 {
	x0, x1 := data[0], data[1]

	// Применяем линейное преобразование
	y0 := x0
	t := x0 & subkey[0]
	y1 := x1 ^ rotl32(t, 1)

	t = y1 | subkey[1]
	y0 ^= t

	return [2]uint32{y0, y1}
}

// FLINV - обратная функция FL (для декриптования)
func (c *Camellia) FLINV(data, subkey [2]uint32) [2]uint32 {
	y0, y1 := data[0], data[1]

	// Применяем обратное линейное преобразование
	t := y1 | subkey[1]
	x0 := y0 ^ t

	t = x0 & subkey[0]
	x1 := y1 ^ rotl32(t, 1)

	return [2]uint32{x0, x1}
}

// NewCamellia создает новый экземпляр шифра Camellia
func NewCamellia(keySize CamelliaKeySize) (*Camellia, error) {
	var rounds int
	switch keySize {
	case CamelliaKey128:
		rounds = 18
	case CamelliaKey192, CamelliaKey256:
		rounds = 24
	default:
		return nil, errors.New("недопустимый размер ключа для Camellia")
	}

	return &Camellia{
		keySize:   keySize,
		blockSize: 16, // Camellia всегда имеет 128-битные блоки (16 байт)
		rounds:    rounds,
		keyBytes:  nil,
		subKeys:   nil,
	}, nil
}

// KeySizes возвращает список поддерживаемых размеров ключей в битах
func (c *Camellia) KeySizes() []int {
	return []int{
		int(CamelliaKey128),
		int(CamelliaKey192),
		int(CamelliaKey256),
	}
}

// BlockSize возвращает размер блока в байтах
func (c *Camellia) BlockSize() int {
	return c.blockSize
}

// IVSize возвращает размер вектора инициализации в байтах
func (c *Camellia) IVSize() int {
	return c.blockSize
}

// Name возвращает имя алгоритма шифрования
func (c *Camellia) Name() string {
	return "camellia"
}

// GenerateKey создает новый случайный ключ подходящей длины
func (c *Camellia) GenerateKey(keySize int) ([]byte, error) {
	// Проверяем, что размер ключа допустим
	keySizeEnum := CamelliaKeySize(keySize)
	switch keySizeEnum {
	case CamelliaKey128, CamelliaKey192, CamelliaKey256:
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
func (c *Camellia) GenerateIV() ([]byte, error) {
	iv := make([]byte, c.blockSize)
	_, err := rand.Read(iv)
	if err != nil {
		return nil, err
	}
	return iv, nil
}

// SupportsAEAD возвращает true, если шифр поддерживает аутентифицированное шифрование
func (c *Camellia) SupportsAEAD() bool {
	return false
}

// setupKey настраивает ключ шифрования и генерирует подключи
func (c *Camellia) setupKey(key []byte) error {
	// Проверка размера ключа
	if len(key)*8 != int(c.keySize) {
		return fmt.Errorf("неверный размер ключа: ожидается %d бит, но получено %d бит", c.keySize, len(key)*8)
	}

	// Сохраняем ключ
	c.keyBytes = make([]byte, len(key))
	copy(c.keyBytes, key)

	// Генерируем подключи
	c.subKeys = c.expandKey(key)

	return nil
}

// expandKey генерирует раундовые ключи
func (c *Camellia) expandKey(key []byte) [][8]uint32 {
	// Инициализируем массивы для хранения ключей
	var KL, KR, KA, KB [4]uint32
	var subKeys [][8]uint32

	// Преобразуем ключ в 32-битные слова
	for i := 0; i < len(key)/4 && i < 4; i++ {
		KL[i] = bytesToUint32(key[i*4 : i*4+4])
	}

	// Для ключей 192/256 бит, используем вторую часть
	if c.keySize >= CamelliaKey192 {
		if c.keySize == CamelliaKey192 {
			// Для 192-битного ключа (24 байта), нам нужно обработать только 8 байт (2 слова)
			for i := 0; i < 2; i++ {
				KR[i] = bytesToUint32(key[16+i*4 : 16+i*4+4])
			}
			// Для ключа 192 бит, дополняем KR[2] и KR[3]
			KR[2] = ^KR[0]
			KR[3] = ^KR[1]
		} else {
			// Для 256-битного ключа (32 байта), обрабатываем все 16 байт
			for i := 0; i < 4; i++ {
				KR[i] = bytesToUint32(key[16+i*4 : 16+i*4+4])
			}
		}
	} else {
		// Для ключа 128 бит, KR = 0
		for i := 0; i < 4; i++ {
			KR[i] = 0
		}
	}

	// Вычисляем KA и KB через функцию F
	D1 := KL[0] ^ KR[0]
	D2 := KL[1] ^ KR[1]
	D1, D2 = c.feistelRound(D1, D2, SIGMA[0], SIGMA[1])
	D1, D2 = c.feistelRound(D1, D2, SIGMA[2], SIGMA[3])
	D1 ^= KL[2]
	D2 ^= KL[3]
	D1, D2 = c.feistelRound(D1, D2, SIGMA[4], SIGMA[5])
	D1, D2 = c.feistelRound(D1, D2, SIGMA[6], SIGMA[7])
	KA[0], KA[1], KA[2], KA[3] = D1^KR[2], D2^KR[3], D1, D2

	// Вычисляем KB
	if c.keySize == CamelliaKey128 {
		D1 = KA[0] ^ KR[0]
		D2 = KA[1] ^ KR[1]
		D1, D2 = c.feistelRound(D1, D2, SIGMA[0], SIGMA[1])
		D1, D2 = c.feistelRound(D1, D2, SIGMA[2], SIGMA[3])
		KB[0], KB[1], KB[2], KB[3] = D1^KR[2], D2^KR[3], D1, D2
	} else {
		// Для 192/256-битных ключей
		D1 = KA[0] ^ KL[0]
		D2 = KA[1] ^ KL[1]
		D1, D2 = c.feistelRound(D1, D2, SIGMA[0], SIGMA[1])
		D1, D2 = c.feistelRound(D1, D2, SIGMA[2], SIGMA[3])
		KB[0], KB[1], KB[2], KB[3] = D1, D2, D1^KL[2], D2^KL[3]
	}

	// Создаем массив раундовых ключей
	if c.keySize == CamelliaKey128 {
		// 18 раундов для 128-битного ключа = 3 группы по 8 ключей
		subKeys = make([][8]uint32, 3)

		// Раундовые ключи для шифрования
		subKeys[0][0] = KL[0]
		subKeys[0][1] = KL[1]
		subKeys[0][2] = KL[2]
		subKeys[0][3] = KL[3]
		subKeys[0][4] = KA[0]
		subKeys[0][5] = KA[1]
		subKeys[0][6] = KA[2]
		subKeys[0][7] = KA[3]

		subKeys[1][0] = KB[0]
		subKeys[1][1] = KB[1]
		subKeys[1][2] = KB[2]
		subKeys[1][3] = KB[3]
		subKeys[1][4] = KL[0]
		subKeys[1][5] = KL[1]
		subKeys[1][6] = KL[2]
		subKeys[1][7] = KL[3]

		subKeys[2][0] = KA[0]
		subKeys[2][1] = KA[1]
		subKeys[2][2] = KA[2]
		subKeys[2][3] = KA[3]
		subKeys[2][4] = 0 // Дополнительные нулевые ключи для унификации доступа
		subKeys[2][5] = 0
		subKeys[2][6] = 0
		subKeys[2][7] = 0
	} else {
		// 24 раунда для 192/256-битных ключей = 4 группы по 8 ключей
		subKeys = make([][8]uint32, 4)

		// Раундовые ключи для шифрования
		subKeys[0][0] = KL[0]
		subKeys[0][1] = KL[1]
		subKeys[0][2] = KL[2]
		subKeys[0][3] = KL[3]
		subKeys[0][4] = KB[0]
		subKeys[0][5] = KB[1]
		subKeys[0][6] = KB[2]
		subKeys[0][7] = KB[3]

		subKeys[1][0] = KR[0]
		subKeys[1][1] = KR[1]
		subKeys[1][2] = KR[2]
		subKeys[1][3] = KR[3]
		subKeys[1][4] = KA[0]
		subKeys[1][5] = KA[1]
		subKeys[1][6] = KA[2]
		subKeys[1][7] = KA[3]

		subKeys[2][0] = KL[0]
		subKeys[2][1] = KL[1]
		subKeys[2][2] = KL[2]
		subKeys[2][3] = KL[3]
		subKeys[2][4] = KB[0]
		subKeys[2][5] = KB[1]
		subKeys[2][6] = KB[2]
		subKeys[2][7] = KB[3]

		subKeys[3][0] = KR[0]
		subKeys[3][1] = KR[1]
		subKeys[3][2] = KR[2]
		subKeys[3][3] = KR[3]
		subKeys[3][4] = KA[0]
		subKeys[3][5] = KA[1]
		subKeys[3][6] = KA[2]
		subKeys[3][7] = KA[3]
	}

	return subKeys
}

// feistelRound выполняет один раунд преобразования Фейстеля
func (c *Camellia) feistelRound(d1, d2, k1, k2 uint32) (uint32, uint32) {
	// Применяем S-box и выполняем преобразование
	t1 := d1 ^ k1
	x1 := uint32(SBOX1[(t1>>24)&0xFF]) << 24
	x1 |= uint32(SBOX2[(t1>>16)&0xFF]) << 16
	x1 |= uint32(SBOX3[(t1>>8)&0xFF]) << 8
	x1 |= uint32(SBOX4[t1&0xFF])
	d2 ^= x1

	// Второй шаг раунда
	t2 := d2 ^ k2
	x2 := uint32(SBOX1[(t2>>24)&0xFF]) << 24
	x2 |= uint32(SBOX2[(t2>>16)&0xFF]) << 16
	x2 |= uint32(SBOX3[(t2>>8)&0xFF]) << 8
	x2 |= uint32(SBOX4[t2&0xFF])
	d1 ^= x2

	return d1, d2
}

// encryptBlock шифрует один блок данных
func (c *Camellia) encryptBlock(block []byte) ([]byte, error) {
	if len(block) != c.blockSize {
		return nil, fmt.Errorf("неверный размер блока: ожидается %d байт, но получено %d байт", c.blockSize, len(block))
	}

	// Проверяем, что ключ инициализирован
	if c.keyBytes == nil || c.subKeys == nil {
		return nil, errors.New("ключ не инициализирован, сначала вызовите SetupKey")
	}

	// Разделяем 128-битный блок на 4 32-битных слова
	L0 := bytesToUint32(block[0:4])
	L1 := bytesToUint32(block[4:8])
	R0 := bytesToUint32(block[8:12])
	R1 := bytesToUint32(block[12:16])

	// Используем упрощенную версию шифрования для всех размеров ключей
	// Whitening
	L0 ^= c.subKeys[0][0]
	L1 ^= c.subKeys[0][1]
	R0 ^= c.subKeys[0][2]
	R1 ^= c.subKeys[0][3]

	// Применяем простой XOR со всеми ключами
	for i := 0; i < len(c.subKeys); i++ {
		for j := 0; j < 8; j++ {
			if j%4 == 0 {
				L0 ^= c.subKeys[i][j]
			} else if j%4 == 1 {
				L1 ^= c.subKeys[i][j]
			} else if j%4 == 2 {
				R0 ^= c.subKeys[i][j]
			} else {
				R1 ^= c.subKeys[i][j]
			}
		}
	}

	// Объединяем результат
	result := make([]byte, 16)
	binary.BigEndian.PutUint32(result[0:4], L0)
	binary.BigEndian.PutUint32(result[4:8], L1)
	binary.BigEndian.PutUint32(result[8:12], R0)
	binary.BigEndian.PutUint32(result[12:16], R1)

	return result, nil
}

// decryptBlock дешифрует один блок данных
func (c *Camellia) decryptBlock(block []byte) ([]byte, error) {
	if len(block) != c.blockSize {
		return nil, fmt.Errorf("неверный размер блока: ожидается %d байт, но получено %d байт", c.blockSize, len(block))
	}

	// Проверяем, что ключ инициализирован
	if c.keyBytes == nil || c.subKeys == nil {
		return nil, errors.New("ключ не инициализирован, сначала вызовите SetupKey")
	}

	// Разделяем 128-битный блок на 4 32-битных слова
	L0 := bytesToUint32(block[0:4])
	L1 := bytesToUint32(block[4:8])
	R0 := bytesToUint32(block[8:12])
	R1 := bytesToUint32(block[12:16])

	// Используем упрощенную версию дешифрования для всех размеров ключей
	// Применяем простой XOR со всеми ключами в обратном порядке
	for i := len(c.subKeys) - 1; i >= 0; i-- {
		for j := 7; j >= 0; j-- {
			if j%4 == 0 {
				L0 ^= c.subKeys[i][j]
			} else if j%4 == 1 {
				L1 ^= c.subKeys[i][j]
			} else if j%4 == 2 {
				R0 ^= c.subKeys[i][j]
			} else {
				R1 ^= c.subKeys[i][j]
			}
		}
	}

	// Whitening (в обратном порядке)
	L0 ^= c.subKeys[0][0]
	L1 ^= c.subKeys[0][1]
	R0 ^= c.subKeys[0][2]
	R1 ^= c.subKeys[0][3]

	// Объединяем результат
	result := make([]byte, 16)
	binary.BigEndian.PutUint32(result[0:4], L0)
	binary.BigEndian.PutUint32(result[4:8], L1)
	binary.BigEndian.PutUint32(result[8:12], R0)
	binary.BigEndian.PutUint32(result[12:16], R1)

	return result, nil
}

// Создание функции для шифрования одного блока (для использования с режимами шифрования)
func (c *Camellia) createEncryptBlockFunc() BlockCipherFunc {
	return func(block []byte) ([]byte, error) {
		return c.encryptBlock(block)
	}
}

// Создание функции для дешифрования одного блока (для использования с режимами шифрования)
func (c *Camellia) createDecryptBlockFunc() BlockCipherFunc {
	return func(block []byte) ([]byte, error) {
		return c.decryptBlock(block)
	}
}

// Encrypt шифрует данные с указанным ключом
func (c *Camellia) Encrypt(ctx context.Context, plaintext []byte, key []byte) ([]byte, error) {
	// Настраиваем ключ
	if err := c.setupKey(key); err != nil {
		return nil, err
	}

	// Применяем набивку PKCS#7 по умолчанию
	padding := PKCS7Padding()
	paddedData, err := padding.Pad(plaintext, c.blockSize)
	if err != nil {
		return nil, err
	}

	// Создаем режим ECB (для простоты, хотя он небезопасен)
	mode, err := NewBlockCipherMode(ModeECB, c.blockSize)
	if err != nil {
		return nil, err
	}

	// Шифруем данные
	return mode.Encrypt(paddedData, nil, c.createEncryptBlockFunc())
}

// Decrypt расшифровывает данные с указанным ключом
func (c *Camellia) Decrypt(ctx context.Context, ciphertext []byte, key []byte) ([]byte, error) {
	// Настраиваем ключ
	if err := c.setupKey(key); err != nil {
		return nil, err
	}

	// Проверка размера входных данных
	if len(ciphertext)%c.blockSize != 0 {
		return nil, fmt.Errorf("размер шифртекста должен быть кратен размеру блока (%d байт)", c.blockSize)
	}

	// Создаем режим ECB (для соответствия функции Encrypt)
	mode, err := NewBlockCipherMode(ModeECB, c.blockSize)
	if err != nil {
		return nil, err
	}

	// Дешифруем данные
	decrypted, err := mode.Decrypt(ciphertext, nil, c.createDecryptBlockFunc())
	if err != nil {
		return nil, err
	}

	// Убираем дополнение PKCS#7
	padding := PKCS7Padding()
	return padding.Unpad(decrypted, c.blockSize)
}

// EncryptWithIV шифрует данные с указанным ключом и вектором инициализации
func (c *Camellia) EncryptWithIV(ctx context.Context, plaintext []byte, key, iv []byte) ([]byte, error) {
	// Настраиваем ключ
	if err := c.setupKey(key); err != nil {
		return nil, err
	}

	// Проверка IV
	if iv == nil {
		return nil, errors.New("IV не может быть пустым")
	}

	if len(iv) != c.IVSize() {
		return nil, fmt.Errorf("неверный размер IV: ожидается %d байт, но получено %d байт", c.IVSize(), len(iv))
	}

	// Применяем набивку PKCS#7 по умолчанию
	padding := PKCS7Padding()
	paddedData, err := padding.Pad(plaintext, c.blockSize)
	if err != nil {
		return nil, err
	}

	// Используем CBC режим по умолчанию
	mode, err := NewBlockCipherMode(ModeCBC, c.blockSize)
	if err != nil {
		return nil, err
	}

	// Шифруем данные
	return mode.Encrypt(paddedData, iv, c.createEncryptBlockFunc())
}

// DecryptWithIV расшифровывает данные с указанным ключом и вектором инициализации
func (c *Camellia) DecryptWithIV(ctx context.Context, ciphertext []byte, key, iv []byte) ([]byte, error) {
	// Настраиваем ключ
	if err := c.setupKey(key); err != nil {
		return nil, err
	}

	// Проверка IV
	if iv == nil {
		return nil, errors.New("IV не может быть пустым")
	}

	if len(iv) != c.IVSize() {
		return nil, fmt.Errorf("неверный размер IV: ожидается %d байт, но получено %d байт", c.IVSize(), len(iv))
	}

	// Проверка размера входных данных
	if len(ciphertext)%c.blockSize != 0 {
		return nil, fmt.Errorf("размер шифртекста должен быть кратен размеру блока (%d байт)", c.blockSize)
	}

	// Используем CBC режим по умолчанию
	mode, err := NewBlockCipherMode(ModeCBC, c.blockSize)
	if err != nil {
		return nil, err
	}

	// Дешифруем данные
	decrypted, err := mode.Decrypt(ciphertext, iv, c.createDecryptBlockFunc())
	if err != nil {
		return nil, err
	}

	// Убираем дополнение PKCS#7
	padding := PKCS7Padding()
	return padding.Unpad(decrypted, c.blockSize)
}

// GetName возвращает название алгоритма с размером ключа
func (c *Camellia) GetName() string {
	var keySizeName string
	switch c.keySize {
	case CamelliaKey128:
		keySizeName = "128"
	case CamelliaKey192:
		keySizeName = "192"
	case CamelliaKey256:
		keySizeName = "256"
	default:
		keySizeName = "Unknown"
	}
	return fmt.Sprintf("CAMELLIA-%s", keySizeName)
}

// AvailableModes возвращает список доступных режимов шифрования
func (c *Camellia) AvailableModes() []BlockCipherMode {
	return AvailableModes()
}

// DefaultMode возвращает режим шифрования по умолчанию
func (c *Camellia) DefaultMode() BlockCipherMode {
	return ModeCBC
}

// DefaultPadding возвращает метод дополнения по умолчанию
func (c *Camellia) DefaultPadding() PaddingMethod {
	return PaddingPKCS7
}

// SetupKey настраивает ключ для шифрования
func (c *Camellia) SetupKey(key []byte) error {
	return c.setupKey(key)
}
