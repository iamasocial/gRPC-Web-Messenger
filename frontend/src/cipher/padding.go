package cipher

import (
	"crypto/rand"
	"errors"
	"fmt"
)

// paddingImpl структура для реализации интерфейса Padding
type paddingImpl struct {
	method PaddingMethod
}

// PKCS7Padding представляет метод набивки PKCS#7
func PKCS7Padding() Padding {
	return &paddingImpl{method: PaddingPKCS7}
}

// ISO10126Padding представляет метод набивки ISO/IEC 10126
func ISO10126Padding() Padding {
	return &paddingImpl{method: PaddingISO10126}
}

// ZerosPadding представляет метод набивки нулями без маркера конца
func ZerosPadding() Padding {
	return &paddingImpl{method: PaddingZeros}
}

// ANSIX923Padding представляет метод набивки ANSI X.923
func ANSIX923Padding() Padding {
	return &paddingImpl{method: PaddingANSIX923}
}

// Method возвращает используемый метод набивки
func (p *paddingImpl) Method() PaddingMethod {
	return p.method
}

// Pad добавляет набивку к данным в соответствии с выбранным методом
func (p *paddingImpl) Pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 0 {
		return nil, ErrInvalidBlockSize
	}

	if data == nil {
		return nil, errors.New("данные не могут быть nil")
	}

	// Вычисляем количество байтов, которые нужно добавить
	padLen := blockSize - (len(data) % blockSize)
	if padLen == 0 && p.method != PaddingZeros {
		padLen = blockSize
	}

	// Создаем новый слайс для результата
	paddedData := make([]byte, len(data)+padLen)
	copy(paddedData, data)

	// Добавляем набивку в зависимости от метода
	switch p.method {
	case PaddingPKCS7:
		// PKCS#7: все байты набивки равны количеству добавленных байтов
		for i := 0; i < padLen; i++ {
			paddedData[len(data)+i] = byte(padLen)
		}

	case PaddingISO10126:
		// ISO 10126: случайные байты, последний байт указывает на длину
		randomBytes := make([]byte, padLen-1)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("ошибка при генерации случайных байтов: %w", err)
		}
		for i := 0; i < padLen-1; i++ {
			paddedData[len(data)+i] = randomBytes[i]
		}
		paddedData[len(paddedData)-1] = byte(padLen)

	case PaddingZeros:
		// Zeros Padding: все байты набивки равны нулю, без маркера конца
		// Если данные уже кратны размеру блока, набивка не добавляется
		if padLen == blockSize {
			return data, nil
		}
		for i := 0; i < padLen; i++ {
			paddedData[len(data)+i] = 0
		}

	case PaddingANSIX923:
		// ANSI X.923: заполнение нулями, последний байт указывает на длину
		for i := 0; i < padLen-1; i++ {
			paddedData[len(data)+i] = 0
		}
		paddedData[len(paddedData)-1] = byte(padLen)

	default:
		return nil, fmt.Errorf("неподдерживаемый метод набивки: %s", p.method)
	}

	return paddedData, nil
}

// Unpad удаляет набивку из данных в соответствии с выбранным методом
func (p *paddingImpl) Unpad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 0 {
		return nil, ErrInvalidBlockSize
	}

	if data == nil {
		return nil, errors.New("данные не могут быть nil")
	}

	dataLen := len(data)
	if dataLen == 0 {
		return nil, errors.New("данные имеют нулевую длину")
	}

	// Проверка валидности длины данных
	if dataLen%blockSize != 0 {
		return nil, fmt.Errorf("длина данных (%d) не кратна размеру блока (%d)", dataLen, blockSize)
	}

	switch p.method {
	case PaddingPKCS7:
		// Получаем значение последнего байта, оно должно указывать на длину набивки
		padLen := int(data[dataLen-1])
		if padLen <= 0 || padLen > blockSize {
			return nil, ErrInvalidPadding
		}

		// Проверяем все байты набивки
		for i := dataLen - padLen; i < dataLen; i++ {
			if data[i] != byte(padLen) {
				return nil, ErrInvalidPadding
			}
		}
		return data[:dataLen-padLen], nil

	case PaddingISO10126:
		// Для ISO 10126 нам важен только последний байт, указывающий на длину
		padLen := int(data[dataLen-1])
		if padLen <= 0 || padLen > blockSize {
			return nil, ErrInvalidPadding
		}
		return data[:dataLen-padLen], nil

	case PaddingZeros:
		// Удаляем все нулевые байты в конце данных
		i := dataLen - 1
		for ; i >= 0 && data[i] == 0; i-- {
		}
		return data[:i+1], nil

	case PaddingANSIX923:
		// Последний байт указывает на длину набивки
		padLen := int(data[dataLen-1])
		if padLen <= 0 || padLen > blockSize {
			return nil, ErrInvalidPadding
		}

		// Проверяем, что все байты набивки - нули (кроме последнего)
		for i := dataLen - padLen; i < dataLen-1; i++ {
			if data[i] != 0 {
				return nil, ErrInvalidPadding
			}
		}
		return data[:dataLen-padLen], nil

	default:
		return nil, fmt.Errorf("неподдерживаемый метод набивки: %s", p.method)
	}
}

// Фабрика для создания объектов набивки
type paddingFactory struct{}

// NewPaddingFactory создает новую фабрику для работы с набивкой
func NewPaddingFactory() *paddingFactory {
	return &paddingFactory{}
}

// CreatePadding создает реализацию интерфейса Padding на основе метода
func (f *paddingFactory) CreatePadding(method PaddingMethod) (Padding, error) {
	switch method {
	case PaddingPKCS7:
		return PKCS7Padding(), nil
	case PaddingISO10126:
		return ISO10126Padding(), nil
	case PaddingZeros:
		return ZerosPadding(), nil
	case PaddingANSIX923:
		return ANSIX923Padding(), nil
	default:
		return nil, fmt.Errorf("неподдерживаемый метод набивки: %s", method)
	}
}

// AvailablePaddingMethods возвращает список доступных методов набивки
func (f *paddingFactory) AvailablePaddingMethods() []PaddingMethod {
	return []PaddingMethod{
		PaddingPKCS7,
		PaddingISO10126,
		PaddingZeros,
		PaddingANSIX923,
	}
}
