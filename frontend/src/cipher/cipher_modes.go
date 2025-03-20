package cipher

import (
	"crypto/rand"
	"errors"
	"fmt"
)

// blockCipherMode представляет режим работы блочного шифра
type blockCipherMode struct {
	mode      BlockCipherMode
	blockSize int
	ivSize    int
}

// blockCipherOperation определяет тип операции (шифрование или дешифрование)
type blockCipherOperation int

const (
	operationEncrypt blockCipherOperation = iota
	operationDecrypt
)

// BlockCipherFunc представляет функцию, которая шифрует или дешифрует одиночный блок
// Для шифрования: plainBlock -> cipherBlock
// Для дешифрования: cipherBlock -> plainBlock
type BlockCipherFunc func(block []byte) ([]byte, error)

// processECB обрабатывает данные в режиме ECB
func processECB(data []byte, blockSize int, blockFunc BlockCipherFunc) ([]byte, error) {
	if len(data)%blockSize != 0 {
		return nil, fmt.Errorf("длина данных (%d) не кратна размеру блока (%d)", len(data), blockSize)
	}

	result := make([]byte, len(data))
	for i := 0; i < len(data); i += blockSize {
		processedBlock, err := blockFunc(data[i : i+blockSize])
		if err != nil {
			return nil, err
		}
		copy(result[i:i+blockSize], processedBlock)
	}
	return result, nil
}

// processCBC обрабатывает данные в режиме CBC
func processCBC(data, iv []byte, blockSize int, blockFunc BlockCipherFunc, operation blockCipherOperation) ([]byte, error) {
	if len(data)%blockSize != 0 {
		return nil, fmt.Errorf("длина данных (%d) не кратна размеру блока (%d)", len(data), blockSize)
	}

	if len(iv) != blockSize {
		return nil, ErrInvalidIV
	}

	result := make([]byte, len(data))
	previousBlock := make([]byte, blockSize)
	copy(previousBlock, iv)

	if operation == operationEncrypt {
		// CBC шифрование
		for i := 0; i < len(data); i += blockSize {
			// XOR текущего блока с предыдущим зашифрованным блоком (или IV)
			xorBlock := make([]byte, blockSize)
			for j := 0; j < blockSize; j++ {
				xorBlock[j] = data[i+j] ^ previousBlock[j]
			}

			// Шифрование результата XOR
			cipherBlock, err := blockFunc(xorBlock)
			if err != nil {
				return nil, err
			}

			copy(result[i:i+blockSize], cipherBlock)
			copy(previousBlock, cipherBlock)
		}
	} else {
		// CBC дешифрование
		for i := 0; i < len(data); i += blockSize {
			// Дешифрование текущего блока
			plainXOR, err := blockFunc(data[i : i+blockSize])
			if err != nil {
				return nil, err
			}

			// XOR с предыдущим шифрованным блоком (или IV)
			for j := 0; j < blockSize; j++ {
				result[i+j] = plainXOR[j] ^ previousBlock[j]
			}

			// Сохраняем текущий зашифрованный блок в качестве предыдущего для следующей итерации
			copy(previousBlock, data[i:i+blockSize])
		}
	}

	return result, nil
}

// processPCBC обрабатывает данные в режиме PCBC
func processPCBC(data, iv []byte, blockSize int, blockFunc BlockCipherFunc, operation blockCipherOperation) ([]byte, error) {
	if len(data)%blockSize != 0 {
		return nil, fmt.Errorf("длина данных (%d) не кратна размеру блока (%d)", len(data), blockSize)
	}

	if len(iv) != blockSize {
		return nil, ErrInvalidIV
	}

	result := make([]byte, len(data))
	previousXOR := make([]byte, blockSize)
	copy(previousXOR, iv)

	if operation == operationEncrypt {
		// PCBC шифрование
		for i := 0; i < len(data); i += blockSize {
			currentBlock := data[i : i+blockSize]

			// XOR текущего блока с предыдущим XOR (или IV)
			xorBlock := make([]byte, blockSize)
			for j := 0; j < blockSize; j++ {
				xorBlock[j] = currentBlock[j] ^ previousXOR[j]
			}

			// Шифрование результата XOR
			cipherBlock, err := blockFunc(xorBlock)
			if err != nil {
				return nil, err
			}

			// Сохраняем текущий зашифрованный блок
			copy(result[i:i+blockSize], cipherBlock)

			// Новый XOR для следующего блока: исходный текст XOR зашифрованный текст
			for j := 0; j < blockSize; j++ {
				previousXOR[j] = currentBlock[j] ^ cipherBlock[j]
			}
		}
	} else {
		// PCBC дешифрование
		for i := 0; i < len(data); i += blockSize {
			currentBlock := data[i : i+blockSize]

			// Дешифрование текущего блока
			plainXOR, err := blockFunc(currentBlock)
			if err != nil {
				return nil, err
			}

			// XOR с предыдущим XOR (или IV)
			plainBlock := make([]byte, blockSize)
			for j := 0; j < blockSize; j++ {
				plainBlock[j] = plainXOR[j] ^ previousXOR[j]
			}

			// Сохраняем расшифрованный блок
			copy(result[i:i+blockSize], plainBlock)

			// Новый XOR для следующего блока: расшифрованный текст XOR зашифрованный текст
			for j := 0; j < blockSize; j++ {
				previousXOR[j] = plainBlock[j] ^ currentBlock[j]
			}
		}
	}

	return result, nil
}

// processCFB обрабатывает данные в режиме CFB
func processCFB(data, iv []byte, blockSize int, blockFunc BlockCipherFunc, operation blockCipherOperation) ([]byte, error) {
	if len(iv) != blockSize {
		return nil, ErrInvalidIV
	}

	result := make([]byte, len(data))
	shiftRegister := make([]byte, blockSize)
	copy(shiftRegister, iv)

	if operation == operationEncrypt {
		// CFB шифрование
		for i := 0; i < len(data); i++ {
			// Шифрование регистра сдвига
			encryptedRegister, err := blockFunc(shiftRegister)
			if err != nil {
				return nil, err
			}

			// XOR первого байта зашифрованного регистра с текущим байтом исходного текста
			cipherByte := data[i] ^ encryptedRegister[0]
			result[i] = cipherByte

			// Сдвиг регистра влево на 1 байт и добавление шифрованного байта справа
			for j := 0; j < blockSize-1; j++ {
				shiftRegister[j] = shiftRegister[j+1]
			}
			shiftRegister[blockSize-1] = cipherByte
		}
	} else {
		// CFB дешифрование
		for i := 0; i < len(data); i++ {
			// Шифрование регистра сдвига
			encryptedRegister, err := blockFunc(shiftRegister)
			if err != nil {
				return nil, err
			}

			// XOR первого байта зашифрованного регистра с текущим байтом шифротекста
			plainByte := data[i] ^ encryptedRegister[0]
			result[i] = plainByte

			// Сдвиг регистра влево на 1 байт и добавление шифрованного байта справа
			for j := 0; j < blockSize-1; j++ {
				shiftRegister[j] = shiftRegister[j+1]
			}
			shiftRegister[blockSize-1] = data[i]
		}
	}

	return result, nil
}

// processOFB обрабатывает данные в режиме OFB
func processOFB(data, iv []byte, blockSize int, blockFunc BlockCipherFunc) ([]byte, error) {
	if len(iv) != blockSize {
		return nil, ErrInvalidIV
	}

	result := make([]byte, len(data))
	shiftRegister := make([]byte, blockSize)
	copy(shiftRegister, iv)

	// OFB работает одинаково для шифрования и дешифрования
	for i := 0; i < len(data); i++ {
		// Шифрование регистра сдвига
		encryptedRegister, err := blockFunc(shiftRegister)
		if err != nil {
			return nil, err
		}

		// XOR первого байта зашифрованного регистра с текущим байтом
		result[i] = data[i] ^ encryptedRegister[0]

		// Сдвиг регистра влево на 1 байт и добавление зашифрованного байта справа
		for j := 0; j < blockSize-1; j++ {
			shiftRegister[j] = shiftRegister[j+1]
		}
		shiftRegister[blockSize-1] = encryptedRegister[0]
	}

	return result, nil
}

// processCTR обрабатывает данные в режиме CTR
func processCTR(data, iv []byte, blockSize int, blockFunc BlockCipherFunc) ([]byte, error) {
	if len(iv) != blockSize {
		return nil, ErrInvalidIV
	}

	result := make([]byte, len(data))
	counter := make([]byte, blockSize)
	copy(counter, iv)

	// CTR работает одинаково для шифрования и дешифрования
	for i := 0; i < len(data); i += blockSize {
		// Шифрование счетчика
		encryptedCounter, err := blockFunc(counter)
		if err != nil {
			return nil, err
		}

		// XOR зашифрованного счетчика с текущим блоком
		for j := 0; j < blockSize && i+j < len(data); j++ {
			result[i+j] = data[i+j] ^ encryptedCounter[j]
		}

		// Увеличение счетчика
		for j := blockSize - 1; j >= 0; j-- {
			counter[j]++
			if counter[j] != 0 {
				break
			}
		}
	}

	return result, nil
}

// processRandomDelta обрабатывает данные в режиме Random Delta
func processRandomDelta(data, iv []byte, blockSize int, blockFunc BlockCipherFunc, operation blockCipherOperation) ([]byte, error) {
	if len(data)%blockSize != 0 {
		return nil, fmt.Errorf("длина данных (%d) не кратна размеру блока (%d)", len(data), blockSize)
	}

	if len(iv) != blockSize {
		return nil, ErrInvalidIV
	}

	// Для RandomDelta режима необходимо выделить дополнительное пространство для сохранения дельт при шифровании
	if operation == operationEncrypt {
		// Каждый блок данных при шифровании сопровождается дельтой того же размера
		// Поэтому результат будет в два раза больше исходных данных
		result := make([]byte, len(data)*2)

		// Заполняем дельты случайными значениями и подготавливаем блоки
		for i := 0; i < len(data); i += blockSize {
			// Для каждого блока создаем случайную дельту
			delta := make([]byte, blockSize)
			if _, err := rand.Read(delta); err != nil {
				return nil, err
			}

			// Шифруем дельту
			encryptedDelta, err := blockFunc(delta)
			if err != nil {
				return nil, err
			}

			// Сохраняем зашифрованную дельту в результате
			copy(result[i*2:i*2+blockSize], encryptedDelta)

			// XOR данных с IV и шифрованной дельтой
			for j := 0; j < blockSize; j++ {
				result[i*2+blockSize+j] = data[i+j] ^ iv[j] ^ encryptedDelta[j]
			}
		}

		return result, nil
	} else {
		// Для расшифровки результат будет в два раза меньше, так как мы удаляем дельты
		result := make([]byte, len(data)/2)

		// Обрабатываем блоки данных с их соответствующими дельтами
		for i := 0; i < len(data); i += blockSize * 2 {
			if i+blockSize*2 > len(data) {
				return nil, errors.New("неверный формат данных для режима RandomDelta")
			}

			// Извлекаем зашифрованную дельту и зашифрованный блок данных
			encryptedDelta := make([]byte, blockSize)
			encryptedBlock := make([]byte, blockSize)
			copy(encryptedDelta, data[i:i+blockSize])
			copy(encryptedBlock, data[i+blockSize:i+blockSize*2])

			// Расшифровываем блок, используя дельту и IV
			for j := 0; j < blockSize; j++ {
				result[i/2+j] = encryptedBlock[j] ^ iv[j] ^ encryptedDelta[j]
			}
		}

		return result, nil
	}
}

// NewBlockCipherMode создаёт новый экземпляр режима блочного шифра
func NewBlockCipherMode(mode BlockCipherMode, blockSize int) (*blockCipherMode, error) {
	if blockSize <= 0 {
		return nil, ErrInvalidBlockSize
	}

	ivSize := blockSize
	switch mode {
	case ModeECB, ModeCBC, ModePCBC, ModeCFB, ModeOFB, ModeCTR, ModeRandomDelta:
		// Эти режимы поддерживаются
	default:
		return nil, ErrInvalidMode
	}

	return &blockCipherMode{
		mode:      mode,
		blockSize: blockSize,
		ivSize:    ivSize,
	}, nil
}

// Mode возвращает режим работы шифра
func (m *blockCipherMode) Mode() BlockCipherMode {
	return m.mode
}

// BlockSize возвращает размер блока в байтах
func (m *blockCipherMode) BlockSize() int {
	return m.blockSize
}

// IVSize возвращает размер вектора инициализации в байтах
func (m *blockCipherMode) IVSize() int {
	return m.ivSize
}

// Encrypt шифрует данные в соответствии с выбранным режимом
func (m *blockCipherMode) Encrypt(plaintext, iv []byte, blockFunc BlockCipherFunc) ([]byte, error) {
	if len(plaintext) == 0 {
		return []byte{}, nil
	}

	switch m.mode {
	case ModeECB:
		// ECB не использует IV
		return processECB(plaintext, m.blockSize, blockFunc)
	case ModeCBC:
		return processCBC(plaintext, iv, m.blockSize, blockFunc, operationEncrypt)
	case ModePCBC:
		return processPCBC(plaintext, iv, m.blockSize, blockFunc, operationEncrypt)
	case ModeCFB:
		return processCFB(plaintext, iv, m.blockSize, blockFunc, operationEncrypt)
	case ModeOFB:
		return processOFB(plaintext, iv, m.blockSize, blockFunc)
	case ModeCTR:
		return processCTR(plaintext, iv, m.blockSize, blockFunc)
	case ModeRandomDelta:
		return processRandomDelta(plaintext, iv, m.blockSize, blockFunc, operationEncrypt)
	default:
		return nil, ErrInvalidMode
	}
}

// Decrypt дешифрует данные в соответствии с выбранным режимом
func (m *blockCipherMode) Decrypt(ciphertext, iv []byte, blockFunc BlockCipherFunc) ([]byte, error) {
	if len(ciphertext) == 0 {
		return []byte{}, nil
	}

	switch m.mode {
	case ModeECB:
		// ECB не использует IV
		return processECB(ciphertext, m.blockSize, blockFunc)
	case ModeCBC:
		return processCBC(ciphertext, iv, m.blockSize, blockFunc, operationDecrypt)
	case ModePCBC:
		return processPCBC(ciphertext, iv, m.blockSize, blockFunc, operationDecrypt)
	case ModeCFB:
		return processCFB(ciphertext, iv, m.blockSize, blockFunc, operationDecrypt)
	case ModeOFB:
		return processOFB(ciphertext, iv, m.blockSize, blockFunc)
	case ModeCTR:
		return processCTR(ciphertext, iv, m.blockSize, blockFunc)
	case ModeRandomDelta:
		return processRandomDelta(ciphertext, iv, m.blockSize, blockFunc, operationDecrypt)
	default:
		return nil, ErrInvalidMode
	}
}

// GenerateIV создает случайный вектор инициализации
func (m *blockCipherMode) GenerateIV() ([]byte, error) {
	if m.mode == ModeECB {
		// ECB не использует IV
		return nil, nil
	}

	iv := make([]byte, m.ivSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("ошибка при генерации IV: %w", err)
	}
	return iv, nil
}

// AvailableModes возвращает список всех доступных режимов
func AvailableModes() []BlockCipherMode {
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

// GetModeByName возвращает режим по его имени
func GetModeByName(name string) (BlockCipherMode, error) {
	for _, mode := range AvailableModes() {
		if string(mode) == name {
			return mode, nil
		}
	}
	return "", errors.New("неизвестный режим шифрования: " + name)
}
