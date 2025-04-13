package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"

	"enveloup/cipher"
)

// Глобальная фабрика для создания шифров
var (
	// Фабрики для создания экземпляров шифров
	camelliaFactory *CamelliaFactory
	magentaFactory  *MAGENTAFactory
	paddingFactory  *PaddingFactory
)

// Определяем структуры фабрик, которые будут использоваться
type CamelliaFactory struct{}
type MAGENTAFactory struct{}
type PaddingFactory struct{}

// NewCamelliaFactory создает новую фабрику для Camellia
func NewCamelliaFactory() *CamelliaFactory {
	return &CamelliaFactory{}
}

// CreateCipher создает шифр Camellia на основе конфигурации
func (f *CamelliaFactory) CreateCipher(config cipher.CipherConfig) (cipher.SymmetricCipher, error) {
	camellia, err := cipher.NewCamellia(cipher.CamelliaKeySize(config.KeySize))
	if err != nil {
		return nil, err
	}
	return camellia, nil
}

// NewMAGENTAFactory создает новую фабрику для MAGENTA
func NewMAGENTAFactory() *MAGENTAFactory {
	return &MAGENTAFactory{}
}

// CreateCipher создает шифр MAGENTA на основе конфигурации
func (f *MAGENTAFactory) CreateCipher(config cipher.CipherConfig) (cipher.SymmetricCipher, error) {
	magenta, err := cipher.NewMAGENTA(cipher.MAGENTAKeySize(config.KeySize))
	if err != nil {
		return nil, err
	}
	return magenta, nil
}

// NewPaddingFactory создает новую фабрику для методов набивки
func NewPaddingFactory() *PaddingFactory {
	return &PaddingFactory{}
}

// CreatePadding создает метод набивки на основе указанного метода
func (f *PaddingFactory) CreatePadding(method cipher.PaddingMethod) (cipher.Padding, error) {
	switch method {
	case cipher.PaddingPKCS7:
		return cipher.PKCS7Padding(), nil
	case cipher.PaddingISO10126:
		return cipher.ISO10126Padding(), nil
	case cipher.PaddingZeros:
		return cipher.ZerosPadding(), nil
	case cipher.PaddingANSIX923:
		return cipher.ANSIX923Padding(), nil
	default:
		return nil, fmt.Errorf("Неподдерживаемый метод набивки: %s", method)
	}
}

// Инициализируем фабрики
func init() {
	// Создаем фабрики для каждого типа шифров
	camelliaFactory = NewCamelliaFactory()
	magentaFactory = NewMAGENTAFactory()
	paddingFactory = NewPaddingFactory()
}

// createCipher создает экземпляр шифра на основе JSON-параметров
func createCipher(this js.Value, args []js.Value) interface{} {
	fmt.Println("[WASM] Начало createCipher")
	if len(args) < 1 {
		fmt.Println("[WASM] Ошибка: недостаточно аргументов")
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "Недостаточно аргументов",
		})
	}

	// Получаем JSON параметры
	paramsJSON := args[0].String()
	fmt.Printf("[WASM] Получены параметры: %s\n", paramsJSON)

	// Парсим JSON
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		fmt.Printf("[WASM] Ошибка парсинга JSON: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ошибка парсинга JSON: %v", err),
		})
	}

	fmt.Printf("[WASM] Распарсенные параметры: %+v\n", params)

	// Проверяем обязательные параметры
	algorithm, ok := params["algorithm"].(string)
	if !ok || algorithm == "" {
		fmt.Println("[WASM] Ошибка: не указан алгоритм")
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "Не указан алгоритм",
		})
	}

	// Получаем режим шифрования
	modeStr, _ := params["mode"].(string)
	if modeStr == "" {
		modeStr = "cbc" // Значение по умолчанию
	}

	// Получаем метод набивки
	paddingStr, _ := params["padding"].(string)
	if paddingStr == "" {
		paddingStr = "pkcs7" // Значение по умолчанию
	}

	// Получаем размер ключа
	var keySize int
	if keySizeFloat, ok := params["keySize"].(float64); ok {
		keySize = int(keySizeFloat)
	} else {
		// Размер ключа по умолчанию
		switch strings.ToLower(algorithm) {
		case "camellia":
			keySize = 256
		case "magenta":
			keySize = 128
		default:
			keySize = 256
		}
	}

	fmt.Printf("[WASM] Обработанные параметры: algorithm=%s, mode=%s, padding=%s, keySize=%d\n",
		algorithm, modeStr, paddingStr, keySize)

	// Создаем конфигурацию шифра
	var cipherConfig cipher.CipherConfig
	cipherConfig.Algorithm = strings.ToLower(algorithm)
	cipherConfig.Mode = cipher.BlockCipherMode(strings.ToLower(modeStr))
	cipherConfig.PaddingMethod = cipher.PaddingMethod(strings.ToLower(paddingStr))
	cipherConfig.KeySize = keySize

	fmt.Printf("[WASM] Создана конфигурация шифра: %+v\n", cipherConfig)

	// Создаем шифр в зависимости от алгоритма
	var cipherInstance cipher.SymmetricCipher
	var err error

	switch strings.ToLower(algorithm) {
	case "camellia":
		cipherInstance, err = camelliaFactory.CreateCipher(cipherConfig)
	case "magenta":
		cipherInstance, err = magentaFactory.CreateCipher(cipherConfig)
	default:
		fmt.Printf("[WASM] Неподдерживаемый алгоритм: %s\n", algorithm)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Неподдерживаемый алгоритм: %s", algorithm),
		})
	}

	if err != nil {
		fmt.Printf("[WASM] Ошибка создания шифра: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ошибка создания шифра: %v", err),
		})
	}

	fmt.Println("[WASM] Шифр успешно создан")

	// Создаем информацию о шифре
	cipherInfo := map[string]interface{}{
		"algorithm":     strings.ToLower(cipherConfig.Algorithm),
		"mode":          strings.ToLower(string(cipherConfig.Mode)),
		"paddingMethod": strings.ToLower(string(cipherConfig.PaddingMethod)),
		"keySize":       cipherConfig.KeySize,
		"blockSize":     cipherInstance.BlockSize(),
		"ivSize":        cipherInstance.IVSize(),
		"name":          strings.ToLower(cipherInstance.Name()),
	}

	// Генерируем IV, если требуется
	if generateIV, ok := params["generateIV"].(bool); ok && generateIV {
		ivBytes, err := cipherInstance.GenerateIV()
		if err != nil {
			fmt.Printf("[WASM] Ошибка генерации IV: %v\n", err)
			return js.ValueOf(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Ошибка генерации IV: %v", err),
			})
		}
		ivStr := base64.StdEncoding.EncodeToString(ivBytes)
		cipherInfo["iv"] = ivStr
		fmt.Printf("[WASM] IV сгенерирован, длина: %d, base64: %s\n", len(ivBytes), ivStr)
	}

	// Возвращаем результат
	result := map[string]interface{}{
		"success": true,
		"cipher":  cipherInfo,
	}

	fmt.Printf("[WASM] Возвращаем результат: %+v\n", result)
	return js.ValueOf(result)
}

// encrypt шифрует данные с указанными параметрами
func encrypt(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return js.ValueOf(map[string]interface{}{
			"error": "Требуется 3 аргумента: JSON с параметрами шифра, текст и ключ",
		})
	}

	// Получаем параметры
	paramsJson := args[0].String()
	plaintext := args[1].String()
	key := args[2].String()

	// Создаем шифр на основе параметров
	result := createCipher(this, []js.Value{js.ValueOf(paramsJson)})
	resultMap, ok := result.(js.Value)
	if !ok || !resultMap.Get("success").Bool() {
		return result // Возвращаем ошибку создания шифра
	}

	// Получаем параметры шифра
	cipherInfo := resultMap.Get("cipher")
	algorithm := cipherInfo.Get("algorithm").String()

	// Декодирование параметров из Base64
	plaintextBytes, err := base64.StdEncoding.DecodeString(plaintext)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка декодирования plaintext: %v", err),
		})
	}

	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка декодирования ключа: %v", err),
		})
	}

	// Проверяем размер ключа
	expectedKeySize := cipherInfo.Get("keySize").Int()
	actualKeySize := len(keyBytes) * 8
	if actualKeySize != expectedKeySize {
		// Если размер ключа не соответствует, пытаемся адаптировать его
		if actualKeySize > expectedKeySize {
			// Обрезаем ключ до нужного размера
			keyBytes = keyBytes[:expectedKeySize/8]
		} else {
			// Дополняем ключ нулями до нужного размера
			newKey := make([]byte, expectedKeySize/8)
			copy(newKey, keyBytes)
			keyBytes = newKey
		}
	}

	// Создаем конфигурацию шифра
	cipherConfig := cipher.CipherConfig{
		Algorithm:     algorithm,
		Mode:          cipher.BlockCipherMode(strings.ToLower(cipherInfo.Get("mode").String())),
		PaddingMethod: cipher.PaddingMethod(strings.ToLower(cipherInfo.Get("paddingMethod").String())),
		KeySize:       expectedKeySize,
	}

	// Создаем экземпляр шифра
	var cipherInstance cipher.SymmetricCipher

	switch algorithm {
	case "camellia":
		cipherInstance, err = camelliaFactory.CreateCipher(cipherConfig)
	case "magenta":
		cipherInstance, err = magentaFactory.CreateCipher(cipherConfig)
	default:
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Неподдерживаемый алгоритм: %s", algorithm),
		})
	}

	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка создания шифра: %v", err),
		})
	}

	// Шифруем данные
	var encryptedBytes []byte
	ctx := context.Background()

	encryptedBytes, err = cipherInstance.Encrypt(ctx, plaintextBytes, keyBytes)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка шифрования: %v", err),
		})
	}

	// Кодирование результата в Base64
	resultBase64 := base64.StdEncoding.EncodeToString(encryptedBytes)

	return js.ValueOf(map[string]interface{}{
		"success": true,
		"result":  resultBase64,
	})
}

// encryptWithIV шифрует данные с вектором инициализации
func encryptWithIV(this js.Value, args []js.Value) interface{} {
	fmt.Println("[WASM] Начало encryptWithIV")
	if len(args) < 4 {
		fmt.Printf("[WASM] Ошибка: недостаточно аргументов (получено %d, ожидается 4)\n", len(args))
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "Недостаточно аргументов",
		})
	}

	// Получаем параметры
	paramsJSON := args[0].String()
	plaintext := args[1].String()
	key := args[2].String()
	iv := args[3].String()

	fmt.Printf("[WASM] Получены параметры:\n")
	fmt.Printf("  paramsJSON: %s\n", paramsJSON)
	fmt.Printf("  plaintext length: %d\n", len(plaintext))
	fmt.Printf("  key length: %d\n", len(key))
	fmt.Printf("  iv length: %d\n", len(iv))

	// Проверяем, что параметры не пустые
	if plaintext == "" {
		fmt.Println("[WASM] Ошибка: пустой plaintext")
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "Пустой plaintext",
		})
	}
	if key == "" {
		fmt.Println("[WASM] Ошибка: пустой ключ")
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "Пустой ключ",
		})
	}
	if iv == "" {
		fmt.Println("[WASM] Ошибка: пустой IV")
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   "Пустой IV",
		})
	}

	// Декодируем Base64
	plaintextBytes, err := base64.StdEncoding.DecodeString(plaintext)
	if err != nil {
		fmt.Printf("[WASM] Ошибка декодирования plaintext: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ошибка декодирования plaintext: %v", err),
		})
	}

	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		fmt.Printf("[WASM] Ошибка декодирования ключа: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ошибка декодирования ключа: %v", err),
		})
	}

	ivBytes, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		fmt.Printf("[WASM] Ошибка декодирования IV: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ошибка декодирования IV: %v", err),
		})
	}

	fmt.Printf("[WASM] Декодированные данные:\n")
	fmt.Printf("  plaintext length: %d\n", len(plaintextBytes))
	fmt.Printf("  key length: %d байт (%d бит)\n", len(keyBytes), len(keyBytes)*8)
	fmt.Printf("  iv length: %d\n", len(ivBytes))

	// Парсим JSON параметры
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		fmt.Printf("[WASM] Ошибка парсинга JSON: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ошибка парсинга JSON: %v", err),
		})
	}

	// Получаем параметры
	algorithm, _ := params["algorithm"].(string)
	if algorithm == "" {
		algorithm = "camellia" // По умолчанию
	}

	modeStr, _ := params["mode"].(string)
	if modeStr == "" {
		modeStr = "cbc" // По умолчанию
	}

	paddingStr, _ := params["padding"].(string)
	if paddingStr == "" {
		paddingStr = "pkcs7" // По умолчанию
	}

	var keySize int
	if keySizeFloat, ok := params["keySize"].(float64); ok {
		keySize = int(keySizeFloat)
	} else {
		// Размер ключа по умолчанию
		switch strings.ToLower(algorithm) {
		case "camellia":
			keySize = 256
		case "magenta":
			keySize = 128
		default:
			keySize = 256
		}
	}

	fmt.Printf("[WASM] Распарсенные параметры: algorithm=%s, mode=%s, padding=%s, keySize=%d\n",
		algorithm, modeStr, paddingStr, keySize)

	// Адаптируем размер ключа
	expectedKeyBytes := keySize / 8
	if len(keyBytes) != expectedKeyBytes {
		fmt.Printf("[WASM] Предупреждение: размер ключа не соответствует ожидаемому: ожидается %d бит (%d байт), получено %d бит (%d байт)\n",
			keySize, expectedKeyBytes, len(keyBytes)*8, len(keyBytes))

		if len(keyBytes) > expectedKeyBytes {
			// Если ключ больше, обрезаем его
			fmt.Printf("[WASM] Обрезаем ключ до нужного размера\n")
			keyBytes = keyBytes[:expectedKeyBytes]
		} else {
			// Если ключ меньше, дополняем его нулями
			fmt.Printf("[WASM] Дополняем ключ нулями\n")
			newKey := make([]byte, expectedKeyBytes)
			copy(newKey, keyBytes)
			keyBytes = newKey
		}

		fmt.Printf("[WASM] Адаптированный размер ключа: %d байт (%d бит)\n", len(keyBytes), len(keyBytes)*8)
	}

	// Создаем конфигурацию шифра
	var cipherConfig cipher.CipherConfig
	cipherConfig.Algorithm = strings.ToLower(algorithm)
	cipherConfig.Mode = cipher.BlockCipherMode(strings.ToLower(modeStr))
	cipherConfig.PaddingMethod = cipher.PaddingMethod(strings.ToLower(paddingStr))
	cipherConfig.KeySize = keySize

	fmt.Printf("[WASM] Создана конфигурация шифра: %+v\n", cipherConfig)

	// Создаем экземпляр шифра
	var cipherInstance cipher.SymmetricCipher
	switch strings.ToLower(algorithm) {
	case "camellia":
		cipherInstance, err = camelliaFactory.CreateCipher(cipherConfig)
	case "magenta":
		cipherInstance, err = magentaFactory.CreateCipher(cipherConfig)
	default:
		fmt.Printf("[WASM] Неподдерживаемый алгоритм: %s\n", algorithm)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Неподдерживаемый алгоритм: %s", algorithm),
		})
	}

	if err != nil {
		fmt.Printf("[WASM] Ошибка создания шифра: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ошибка создания шифра: %v", err),
		})
	}

	fmt.Println("[WASM] Шифр успешно создан")

	// Шифруем данные с IV
	ctx := context.Background()
	encryptedBytes, err := cipherInstance.EncryptWithIV(ctx, plaintextBytes, keyBytes, ivBytes)
	if err != nil {
		fmt.Printf("[WASM] Ошибка шифрования: %v\n", err)
		return js.ValueOf(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ошибка шифрования: %v", err),
		})
	}

	fmt.Printf("[WASM] Шифрование успешно, длина шифротекста: %d\n", len(encryptedBytes))

	// Кодируем результат в Base64
	resultBase64 := base64.StdEncoding.EncodeToString(encryptedBytes)
	fmt.Printf("[WASM] Результат закодирован в Base64, длина: %d\n", len(resultBase64))

	return js.ValueOf(map[string]interface{}{
		"success": true,
		"result":  resultBase64,
	})
}

// decrypt расшифровывает данные
func decrypt(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return js.ValueOf(map[string]interface{}{
			"error": "Требуется 3 аргумента: JSON с параметрами шифра, шифротекст и ключ",
		})
	}

	// Получаем параметры
	paramsJson := args[0].String()
	ciphertext := args[1].String()
	key := args[2].String()

	// Создаем шифр на основе параметров
	result := createCipher(this, []js.Value{js.ValueOf(paramsJson)})
	resultMap, ok := result.(js.Value)
	if !ok || !resultMap.Get("success").Bool() {
		return result // Возвращаем ошибку создания шифра
	}

	// Получаем параметры шифра
	cipherInfo := resultMap.Get("cipher")
	algorithm := cipherInfo.Get("algorithm").String()

	// Декодирование параметров из Base64
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка декодирования ciphertext: %v", err),
		})
	}

	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка декодирования ключа: %v", err),
		})
	}

	// Создаем конфигурацию шифра
	cipherConfig := cipher.CipherConfig{
		Algorithm:     algorithm,
		Mode:          cipher.BlockCipherMode(cipherInfo.Get("mode").String()),
		PaddingMethod: cipher.PaddingMethod(cipherInfo.Get("paddingMethod").String()),
		KeySize:       cipherInfo.Get("keySize").Int(),
	}

	// Создаем экземпляр шифра
	var cipherInstance cipher.SymmetricCipher

	switch algorithm {
	case "camellia":
		cipherInstance, err = camelliaFactory.CreateCipher(cipherConfig)
	case "magenta":
		cipherInstance, err = magentaFactory.CreateCipher(cipherConfig)
	default:
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Неподдерживаемый алгоритм: %s", algorithm),
		})
	}

	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка создания шифра: %v", err),
		})
	}

	// Дешифруем данные
	var decryptedBytes []byte
	ctx := context.Background()

	decryptedBytes, err = cipherInstance.Decrypt(ctx, ciphertextBytes, keyBytes)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка дешифрования: %v", err),
		})
	}

	// Кодирование результата в Base64
	resultBase64 := base64.StdEncoding.EncodeToString(decryptedBytes)

	return js.ValueOf(map[string]interface{}{
		"success": true,
		"result":  resultBase64,
	})
}

// decryptWithIV расшифровывает данные с вектором инициализации
func decryptWithIV(this js.Value, args []js.Value) interface{} {
	if len(args) < 4 {
		return js.ValueOf(map[string]interface{}{
			"error": "Требуется 4 аргумента: JSON с параметрами шифра, шифротекст, ключ и IV",
		})
	}

	// Получаем параметры
	paramsJson := args[0].String()
	ciphertext := args[1].String()
	key := args[2].String()
	iv := args[3].String()

	// Создаем шифр на основе параметров
	result := createCipher(this, []js.Value{js.ValueOf(paramsJson)})
	resultMap, ok := result.(js.Value)
	if !ok || !resultMap.Get("success").Bool() {
		return result // Возвращаем ошибку создания шифра
	}

	// Получаем параметры шифра
	cipherInfo := resultMap.Get("cipher")
	algorithm := cipherInfo.Get("algorithm").String()

	// Декодирование параметров из Base64
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка декодирования ciphertext: %v", err),
		})
	}

	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка декодирования ключа: %v", err),
		})
	}

	ivBytes, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка декодирования IV: %v", err),
		})
	}

	// Создаем конфигурацию шифра
	cipherConfig := cipher.CipherConfig{
		Algorithm:     algorithm,
		Mode:          cipher.BlockCipherMode(cipherInfo.Get("mode").String()),
		PaddingMethod: cipher.PaddingMethod(cipherInfo.Get("paddingMethod").String()),
		KeySize:       cipherInfo.Get("keySize").Int(),
	}

	// Создаем экземпляр шифра
	var cipherInstance cipher.SymmetricCipher

	switch algorithm {
	case "camellia":
		cipherInstance, err = camelliaFactory.CreateCipher(cipherConfig)
	case "magenta":
		cipherInstance, err = magentaFactory.CreateCipher(cipherConfig)
	default:
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Неподдерживаемый алгоритм: %s", algorithm),
		})
	}

	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка создания шифра: %v", err),
		})
	}

	// Дешифруем данные с IV
	var decryptedBytes []byte
	ctx := context.Background()

	decryptedBytes, err = cipherInstance.DecryptWithIV(ctx, ciphertextBytes, keyBytes, ivBytes)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка дешифрования с IV: %v", err),
		})
	}

	// Кодирование результата в Base64
	resultBase64 := base64.StdEncoding.EncodeToString(decryptedBytes)

	return js.ValueOf(map[string]interface{}{
		"success": true,
		"result":  resultBase64,
	})
}

// generateKey генерирует ключ для шифра
func generateKey(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return js.ValueOf(map[string]interface{}{
			"error": "Требуется 2 аргумента: JSON с параметрами шифра и размер ключа",
		})
	}

	// Получаем параметры
	paramsJson := args[0].String()
	keySize := args[1].Int()

	// Создаем шифр на основе параметров
	params := make(map[string]interface{})
	if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка разбора JSON: %v", err),
		})
	}
	params["keySize"] = float64(keySize)

	updatedParams, err := json.Marshal(params)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка сериализации JSON: %v", err),
		})
	}

	result := createCipher(this, []js.Value{js.ValueOf(string(updatedParams))})
	resultMap, ok := result.(js.Value)
	if !ok || !resultMap.Get("success").Bool() {
		return result // Возвращаем ошибку создания шифра
	}

	// Получаем параметры шифра
	cipherInfo := resultMap.Get("cipher")
	algorithm := cipherInfo.Get("algorithm").String()

	// Создаем конфигурацию шифра
	cipherConfig := cipher.CipherConfig{
		Algorithm:     algorithm,
		Mode:          cipher.BlockCipherMode(cipherInfo.Get("mode").String()),
		PaddingMethod: cipher.PaddingMethod(cipherInfo.Get("paddingMethod").String()),
		KeySize:       keySize,
	}

	// Создаем экземпляр шифра
	var cipherInstance cipher.SymmetricCipher

	switch algorithm {
	case "camellia":
		cipherInstance, err = camelliaFactory.CreateCipher(cipherConfig)
	case "magenta":
		cipherInstance, err = magentaFactory.CreateCipher(cipherConfig)
	default:
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Неподдерживаемый алгоритм: %s", algorithm),
		})
	}

	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка создания шифра: %v", err),
		})
	}

	// Генерируем ключ
	keyBytes, err := cipherInstance.GenerateKey(keySize)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка генерации ключа: %v", err),
		})
	}

	// Кодирование ключа в Base64
	keyBase64 := base64.StdEncoding.EncodeToString(keyBytes)

	return js.ValueOf(map[string]interface{}{
		"success": true,
		"key":     keyBase64,
	})
}

// generateIV генерирует вектор инициализации для шифра
func generateIV(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.ValueOf(map[string]interface{}{
			"error": "Требуется 1 аргумент: JSON с параметрами шифра",
		})
	}

	// Получаем параметры
	paramsJson := args[0].String()

	// Создаем шифр на основе параметров
	result := createCipher(this, []js.Value{js.ValueOf(paramsJson)})
	resultMap, ok := result.(js.Value)
	if !ok || !resultMap.Get("success").Bool() {
		return result // Возвращаем ошибку создания шифра
	}

	// Получаем параметры шифра
	cipherInfo := resultMap.Get("cipher")
	algorithm := cipherInfo.Get("algorithm").String()

	// Создаем конфигурацию шифра
	cipherConfig := cipher.CipherConfig{
		Algorithm:     algorithm,
		Mode:          cipher.BlockCipherMode(cipherInfo.Get("mode").String()),
		PaddingMethod: cipher.PaddingMethod(cipherInfo.Get("paddingMethod").String()),
		KeySize:       cipherInfo.Get("keySize").Int(),
	}

	// Создаем экземпляр шифра
	var cipherInstance cipher.SymmetricCipher
	var err error

	switch algorithm {
	case "camellia":
		cipherInstance, err = camelliaFactory.CreateCipher(cipherConfig)
	case "magenta":
		cipherInstance, err = magentaFactory.CreateCipher(cipherConfig)
	default:
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Неподдерживаемый алгоритм: %s", algorithm),
		})
	}

	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка создания шифра: %v", err),
		})
	}

	// Генерируем IV
	ivBytes, err := cipherInstance.GenerateIV()
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка генерации IV: %v", err),
		})
	}

	// Кодирование IV в Base64
	ivBase64 := base64.StdEncoding.EncodeToString(ivBytes)

	return js.ValueOf(map[string]interface{}{
		"success": true,
		"iv":      ivBase64,
	})
}

// getAvailableCiphers возвращает информацию о доступных алгоритмах шифрования
func getAvailableCiphers(this js.Value, args []js.Value) interface{} {
	// Создаем информацию о доступных алгоритмах

	// Для Camellia
	camelliaConfig := cipher.CipherConfig{
		Algorithm:     "camellia",
		Mode:          cipher.ModeCBC,
		PaddingMethod: cipher.PaddingPKCS7,
		KeySize:       256,
	}

	camelliaInstance, err := camelliaFactory.CreateCipher(camelliaConfig)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка создания Camellia: %v", err),
		})
	}

	// Для Magenta
	magentaConfig := cipher.CipherConfig{
		Algorithm:     "magenta",
		Mode:          cipher.ModeCBC,
		PaddingMethod: cipher.PaddingPKCS7,
		KeySize:       128,
	}

	magentaInstance, err := magentaFactory.CreateCipher(magentaConfig)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("Ошибка создания Magenta: %v", err),
		})
	}

	// Доступные режимы шифрования
	modes := []string{
		string(cipher.ModeECB),
		string(cipher.ModeCBC),
		string(cipher.ModePCBC),
		string(cipher.ModeCFB),
		string(cipher.ModeOFB),
		string(cipher.ModeCTR),
		string(cipher.ModeRandomDelta),
	}

	// Доступные методы набивки
	paddingMethods := []string{
		string(cipher.PaddingPKCS7),
		string(cipher.PaddingISO10126),
		string(cipher.PaddingZeros),
		string(cipher.PaddingANSIX923),
	}

	algorithmInfo := map[string]interface{}{
		"algorithms": []string{"camellia", "magenta"},
		"modes":      modes,
		"paddings":   paddingMethods,
		"camellia": map[string]interface{}{
			"name":      camelliaInstance.Name(),
			"keySizes":  camelliaInstance.KeySizes(),
			"blockSize": camelliaInstance.BlockSize(),
			"ivSize":    camelliaInstance.IVSize(),
		},
		"magenta": map[string]interface{}{
			"name":      magentaInstance.Name(),
			"keySizes":  magentaInstance.KeySizes(),
			"blockSize": magentaInstance.BlockSize(),
			"ivSize":    magentaInstance.IVSize(),
		},
	}

	return js.ValueOf(map[string]interface{}{
		"success": true,
		"info":    algorithmInfo,
	})
}

func main() {
	c := make(chan struct{}, 0)

	// Регистрируем функции в глобальном объекте JS
	js.Global().Set("EnveloupCipher", js.ValueOf(map[string]interface{}{
		"createCipher":        js.FuncOf(createCipher),
		"encrypt":             js.FuncOf(encrypt),
		"decrypt":             js.FuncOf(decrypt),
		"encryptWithIV":       js.FuncOf(encryptWithIV),
		"decryptWithIV":       js.FuncOf(decryptWithIV),
		"generateKey":         js.FuncOf(generateKey),
		"generateIV":          js.FuncOf(generateIV),
		"getAvailableCiphers": js.FuncOf(getAvailableCiphers),
	}))

	fmt.Println("WASM модуль для шифрования инициализирован!")
	<-c // Держим программу запущенной
}
