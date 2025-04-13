/**
 * Модуль для шифрования и дешифрования данных
 * Реализация соответствует интерфейсу SymmetricCipher из Go-модуля
 */

// Доступные алгоритмы шифрования
export const ALGORITHMS = {
    CAMELLIA: "camellia",
    MAGENTA: "magenta"
};

// Доступные режимы шифрования
export const MODES = {
    ECB: "ecb",
    CBC: "cbc",
    CFB: "cfb",
    OFB: "ofb",
    CTR: "ctr",
    RANDOM_DELTA: "randomdelta"
};

// Доступные методы набивки
export const PADDING = {
    PKCS7: "pkcs7",
    ISO10126: "iso10126",
    ZEROS: "zeros",
    ANSIX923: "ansix923"
};

// Импортируем WASM файл для webpack
// Для современного webpack примерный синтаксис
// import wasmModule from '../../wasm/build/main.wasm';

/**
 * Класс для взаимодействия с WASM модулем шифрования
 */
class CryptographyService {
    constructor() {
        this.wasmInstance = null;
        this.initialized = false;
    }

    /**
     * Инициализирует WASM модуль
     * @returns {Promise} Промис, который разрешится, когда модуль будет загружен
     */
    async init() {
        if (this.initialized) return;

        try {
            console.log('[WASM] Начало инициализации WASM модуля');
            
            // Загружаем WASM файл
            console.log('[WASM] Загрузка WASM файла из ./wasm/main.wasm');
            const response = await fetch('./wasm/main.wasm');
            if (!response.ok) {
                console.error(`[WASM] Ошибка загрузки файла: ${response.status} ${response.statusText}`);
                throw new Error(`Failed to load WASM file: ${response.status} ${response.statusText}`);
            }
            
            console.log('[WASM] WASM файл успешно загружен, получение ArrayBuffer');
            const wasmBuffer = await response.arrayBuffer();
            console.log(`[WASM] Получен ArrayBuffer размером ${wasmBuffer.byteLength} байт`);
            
            // Инициализируем WASM модуль
            console.log('[WASM] Инициализация Go объекта');
            const go = new Go();
            console.log('[WASM] Go объект создан, инстанцирование WASM модуля');
            
            const result = await WebAssembly.instantiate(wasmBuffer, go.importObject);
            console.log('[WASM] WASM модуль успешно инстанцирован');
            
            this.wasmInstance = result.instance;
            console.log('[WASM] Запуск WASM модуля');
            go.run(this.wasmInstance);
            console.log('[WASM] WASM модуль запущен');
            
            // Проверяем, что объект EnveloupCipher доступен
            console.log('[WASM] Проверка наличия объекта EnveloupCipher в window');
            if (!window.EnveloupCipher) {
                console.error('[WASM] Объект EnveloupCipher не найден в window');
                throw new Error('WASM module did not initialize EnveloupCipher object');
            }
            
            console.log('[WASM] Объект EnveloupCipher найден:', Object.keys(window.EnveloupCipher));
            this.initialized = true;
            console.log('[WASM] WASM модуль инициализирован успешно');
        } catch (error) {
            console.error('[WASM] Ошибка инициализации WASM модуля:', error);
            throw error;
        }
    }

    /**
     * Получает информацию о доступных алгоритмах шифрования
     * @returns {Promise<Object>} Информация о доступных алгоритмах
     */
    async getAvailableCiphers() {
        console.log('[WASM] Вызов getAvailableCiphers');
        await this.init();
        try {
            console.log('[WASM] Вызов EnveloupCipher.getAvailableCiphers');
            const result = window.EnveloupCipher.getAvailableCiphers();
            console.log('[WASM] Результат getAvailableCiphers:', result);
            return this._wrapResult(result);
        } catch (error) {
            console.error('[WASM] Ошибка в getAvailableCiphers:', error);
            throw error;
        }
    }

    /**
     * Создает экземпляр шифра с указанными параметрами
     * @param {Object} params Параметры шифра
     * @returns {Promise<Object>} Информация о созданном шифре
     */
    async createCipher(params) {
        console.log('[WASM] Вызов createCipher с параметрами:', JSON.stringify(params));
        await this.init();
        try {
            const paramsString = JSON.stringify(params);
            console.log('[WASM] Вызов EnveloupCipher.createCipher с JSON:', paramsString);
            const result = window.EnveloupCipher.createCipher(paramsString);
            console.log('[WASM] Результат createCipher:', result);
            return this._wrapResult(result);
        } catch (error) {
            console.error('[WASM] Ошибка в createCipher:', error);
            throw error;
        }
    }

    /**
     * Шифрует данные
     * @param {Object} cipherParams Параметры шифра
     * @param {string} plaintext Данные в формате Base64 для шифрования
     * @param {string} key Ключ в формате Base64
     * @returns {Promise<Object>} Результат шифрования
     */
    async encrypt(cipherParams, plaintext, key) {
        await this.init();
        return this._wrapResult(window.EnveloupCipher.encrypt(
            JSON.stringify(cipherParams),
            plaintext,
            key
        ));
    }

    /**
     * Шифрует данные с вектором инициализации
     * @param {Object} cipherParams Параметры шифра
     * @param {string} plaintext Данные в формате Base64 для шифрования
     * @param {string} key Ключ в формате Base64
     * @param {string} iv Вектор инициализации в формате Base64
     * @returns {Promise<Object>} Результат шифрования
     */
    async encryptWithIV(cipherParams, plaintext, key, iv) {
        console.log('[WASM] Вызов encryptWithIV');
        console.log('[WASM] Параметры шифра:', JSON.stringify(cipherParams));
        console.log('[WASM] Длина plaintext:', plaintext ? plaintext.length : 'null/undefined');
        console.log('[WASM] Длина ключа:', key ? key.length : 'null/undefined');
        console.log('[WASM] Длина IV:', iv ? iv.length : 'null/undefined');
        
        await this.init();
        try {
            console.log('[WASM] Вызов EnveloupCipher.encryptWithIV');
            const result = window.EnveloupCipher.encryptWithIV(
                JSON.stringify(cipherParams),
                plaintext,
                key,
                iv
            );
            console.log('[WASM] Результат encryptWithIV:', result);
            return this._wrapResult(result);
        } catch (error) {
            console.error('[WASM] Ошибка в encryptWithIV:', error);
            throw error;
        }
    }

    /**
     * Дешифрует данные
     * @param {Object} cipherParams Параметры шифра
     * @param {string} ciphertext Шифротекст в формате Base64
     * @param {string} key Ключ в формате Base64
     * @returns {Promise<Object>} Результат дешифрования
     */
    async decrypt(cipherParams, ciphertext, key) {
        await this.init();
        return this._wrapResult(window.EnveloupCipher.decrypt(
            JSON.stringify(cipherParams),
            ciphertext,
            key
        ));
    }

    /**
     * Дешифрует данные с вектором инициализации
     * @param {Object} cipherParams Параметры шифра
     * @param {string} ciphertext Шифротекст в формате Base64
     * @param {string} key Ключ в формате Base64
     * @param {string} iv Вектор инициализации в формате Base64
     * @returns {Promise<Object>} Результат дешифрования
     */
    async decryptWithIV(cipherParams, ciphertext, key, iv) {
        await this.init();
        return this._wrapResult(window.EnveloupCipher.decryptWithIV(
            JSON.stringify(cipherParams),
            ciphertext,
            key,
            iv
        ));
    }

    /**
     * Генерирует ключ для шифра
     * @param {Object} cipherParams Параметры шифра
     * @param {number} keySize Размер ключа в битах
     * @returns {Promise<Object>} Сгенерированный ключ в формате Base64
     */
    async generateKey(cipherParams, keySize) {
        await this.init();
        return this._wrapResult(window.EnveloupCipher.generateKey(
            JSON.stringify(cipherParams),
            keySize
        ));
    }

    /**
     * Генерирует вектор инициализации для шифра
     * @param {Object} cipherParams Параметры шифра
     * @returns {Promise<Object>} Сгенерированный вектор инициализации в формате Base64
     */
    async generateIV(cipherParams) {
        await this.init();
        return this._wrapResult(window.EnveloupCipher.generateIV(
            JSON.stringify(cipherParams)
        ));
    }

    /**
     * Обрабатывает результат выполнения WASM функции
     * @param {Object} result Результат выполнения функции
     * @returns {Object} Обработанный результат
     * @private
     */
    _wrapResult(result) {
        console.log('[WASM] Обработка результата:', result);
        if (!result) {
            console.error('[WASM] Результат отсутствует (undefined/null)');
            throw new Error('WASM функция вернула пустой результат');
        }
        
        if (result.error) {
            console.error('[WASM] Ошибка в результате:', result.error);
            throw new Error(result.error);
        }
        return result;
    }

    /**
     * Преобразует строку в Base64
     * @param {string} str Строка для кодирования
     * @returns {string} Строка в формате Base64
     */
    static stringToBase64(str) {
        return btoa(unescape(encodeURIComponent(str)));
    }

    /**
     * Преобразует Base64 в строку
     * @param {string} base64 Строка в формате Base64
     * @returns {string} Исходная строка
     */
    static base64ToString(base64) {
        return decodeURIComponent(escape(atob(base64)));
    }
}

// Экспортируем singleton для использования в приложении
export const cryptoService = new CryptographyService();

/**
 * Создает экземпляр шифра с заданными параметрами
 * @param {string} algorithm - Алгоритм шифрования
 * @param {string} mode - Режим шифрования
 * @param {string} paddingMethod - Метод набивки
 * @param {number} keySize - Размер ключа в битах
 * @returns {SymmetricCipher} - Экземпляр шифра
 */
export function createCipher(algorithm, mode, paddingMethod, keySize = 128) {
    return new SymmetricCipher(algorithm, mode, paddingMethod, keySize);
}

/**
 * Шифрует сообщение для отправки через WebSocket
 * @param {string} message - Текст сообщения для шифрования
 * @param {string} recipientId - ID получателя (для получения общего ключа)
 * @param {Object} params - Параметры шифрования (опционально)
 * @returns {Promise<Object>} - Объект с зашифрованным сообщением и метаданными
 */
export async function encryptMessage(message, recipientId, params = {}) {
    console.log('[CRYPTO] Начало шифрования сообщения для получателя:', recipientId);
    // Получаем общий ключ из localStorage по ID пользователя
    const sharedKey = localStorage.getItem(`dh_shared_key_${recipientId}`);
    if (!sharedKey) {
        console.error('[CRYPTO] Общий ключ не найден для пользователя:', recipientId);
        throw new Error(`Общий ключ для пользователя ${recipientId} не найден`);
    }
    console.log('[CRYPTO] Общий ключ получен, длина:', sharedKey.length);
    
    // Проверяем формат ключа
    try {
        // Проверяем, что ключ в формате Base64
        atob(sharedKey);
        console.log('[CRYPTO] Ключ в корректном Base64 формате');
    } catch (error) {
        console.error('[CRYPTO] Некорректный формат ключа:', error);
        throw new Error(`Некорректный формат ключа для пользователя ${recipientId}`);
    }
    
    // Используем только Camellia, так как MAGENTA не работает в WASM-модуле
    const algorithm = ALGORITHMS.CAMELLIA;
    console.log(`[CRYPTO] Используем алгоритм шифрования: ${algorithm}`);
    
    // Настраиваем параметры шифрования
    const cipherParams = {
        algorithm: algorithm.toLowerCase(),
        mode: (params.mode || MODES.CBC).toLowerCase(),
        padding: (params.padding || PADDING.PKCS7).toLowerCase(),
        keySize: params.keySize || 256,
        generateIV: true
    };
    
    // Логируем параметры
    console.log('[CRYPTO] Параметры шифрования:', JSON.stringify(cipherParams));
    
    try {
        // Получаем информацию о шифре и IV
        console.log('[CRYPTO] Вызов createCipher для получения IV');
        const cipherInfo = await cryptoService.createCipher(cipherParams);
        console.log('[CRYPTO] Результат createCipher:', JSON.stringify(cipherInfo));
        
        if (!cipherInfo.success) {
            console.error('[CRYPTO] Ошибка при создании шифра:', cipherInfo.error);
            throw new Error(`Ошибка создания шифра: ${cipherInfo.error || 'Неизвестная ошибка'}`);
        }
        
        // Проверяем наличие cipher в результате
        if (!cipherInfo.cipher) {
            console.error('[CRYPTO] Отсутствует cipher в результате createCipher');
            throw new Error('Неполный результат создания шифра: отсутствует cipher');
        }
        
        // Преобразуем сообщение в Base64
        console.log('[CRYPTO] Преобразование сообщения в Base64');
        const messageBase64 = CryptographyService.stringToBase64(message);
        console.log('[CRYPTO] Длина сообщения в Base64:', messageBase64.length);
        
        // Шифруем сообщение с IV
        const key = sharedKey;
        const iv = cipherInfo.cipher.iv;
        
        console.log('[CRYPTO] Проверка наличия IV');
        if (!iv) {
            console.error('[CRYPTO] IV отсутствует в результате createCipher');
            throw new Error('Не удалось получить IV для шифрования');
        }
        console.log('[CRYPTO] IV получен, длина:', iv.length);
        
        console.log('[CRYPTO] Вызов encryptWithIV');
        const encryptResult = await cryptoService.encryptWithIV(
            cipherParams,
            messageBase64,
            key, 
            iv
        );
        
        console.log('[CRYPTO] Результат encryptWithIV:', JSON.stringify(encryptResult));
        if (!encryptResult.success) {
            console.error('[CRYPTO] Ошибка шифрования:', encryptResult.error);
            throw new Error(`Ошибка шифрования: ${encryptResult.error || 'Неизвестная ошибка'}`);
        }
        
        // Возвращаем объект с зашифрованным сообщением и метаданными
        console.log('[CRYPTO] Шифрование успешно завершено');
        return {
            content: encryptResult.result,
            iv: iv,
            encryptionParams: {
                algorithm: cipherParams.algorithm,
                mode: cipherParams.mode,
                padding: cipherParams.padding,
                keySize: cipherParams.keySize
            }
        };
    } catch (error) {
        console.error('[CRYPTO] Ошибка при шифровании:', error);
        throw error;
    }
}

/**
 * Дешифрует полученное сообщение
 * @param {Object} encryptedMessage - Объект с зашифрованным сообщением и метаданными
 * @param {string} senderId - ID отправителя (для получения общего ключа)
 * @returns {Promise<string>} - Расшифрованное сообщение
 */
export async function decryptMessage(encryptedMessage, senderId) {
    // Получаем общий ключ из localStorage по ID пользователя
    const sharedKey = localStorage.getItem(`dh_shared_key_${senderId}`);
    if (!sharedKey) {
        throw new Error(`Общий ключ для пользователя ${senderId} не найден`);
    }
    
    // Получаем параметры шифрования и данные
    const {content, iv, encryptionParams} = encryptedMessage;
    
    // Настраиваем параметры шифра
    const cipherParams = {
        algorithm: (encryptionParams.algorithm && encryptionParams.algorithm.toLowerCase()) || ALGORITHMS.CAMELLIA.toLowerCase(),
        mode: (encryptionParams.mode && encryptionParams.mode.toLowerCase()) || MODES.CBC,
        padding: (encryptionParams.padding && encryptionParams.padding.toLowerCase()) || PADDING.PKCS7,
        keySize: encryptionParams.keySize || 256
    };
    
    // Дешифруем сообщение
    const decryptResult = await cryptoService.decryptWithIV(
        cipherParams,
        content,
        sharedKey,
        iv
    );
    
    if (!decryptResult.success) {
        throw new Error(`Ошибка дешифрования: ${decryptResult.error || 'Неизвестная ошибка'}`);
    }
    
    // Преобразуем Base64 обратно в строку
    return CryptographyService.base64ToString(decryptResult.result);
}

/**
 * Шифрует файл для отправки через WebSocket
 * @param {File} file - Файл для шифрования
 * @param {string} recipientId - ID получателя (для получения общего ключа)
 * @param {Object} params - Параметры шифрования (опционально)
 * @returns {Promise<Object>} - Объект с зашифрованным файлом и метаданными
 */
export async function encryptFile(file, recipientId, params = {}) {
    // Получаем общий ключ из localStorage по ID пользователя
    const sharedKey = localStorage.getItem(`dh_shared_key_${recipientId}`);
    if (!sharedKey) {
        throw new Error(`Общий ключ для пользователя ${recipientId} не найден`);
    }
    
    // Используем только Camellia, так как MAGENTA не работает в WASM-модуле
    const algorithm = ALGORITHMS.CAMELLIA;
    console.log(`Используем алгоритм шифрования: ${algorithm}`);
    
    // Настраиваем параметры шифрования
    const cipherParams = {
        algorithm: algorithm.toLowerCase(), // Всегда "camellia" в нижнем регистре
        mode: (params.mode || MODES.CBC).toLowerCase(),
        padding: (params.padding || PADDING.PKCS7).toLowerCase(),
        keySize: params.keySize || 256,
        generateIV: true
    };
    
    // Логируем параметры
    console.log('Параметры шифрования:', JSON.stringify(cipherParams));
    
    // Получаем информацию о шифре и IV
    const cipherInfo = await cryptoService.createCipher(cipherParams);
    if (!cipherInfo.success) {
        console.error('Ошибка при создании шифра:', cipherInfo.error);
        throw new Error(`Ошибка создания шифра: ${cipherInfo.error || 'Неизвестная ошибка'}`);
    }
    
    // Читаем содержимое файла как ArrayBuffer
    const fileArrayBuffer = await file.arrayBuffer();
    const fileBytes = new Uint8Array(fileArrayBuffer);
    
    // Преобразуем содержимое файла в Base64
    const fileBase64 = btoa(
        Array.from(fileBytes)
            .map(byte => String.fromCharCode(byte))
            .join('')
    );
    
    // Шифруем файл с IV
    const key = sharedKey;
    const iv = cipherInfo.cipher.iv;
    
    const encryptResult = await cryptoService.encryptWithIV(
        cipherParams,
        fileBase64,
        key, 
        iv
    );
    
    if (!encryptResult.success) {
        throw new Error(`Ошибка шифрования файла: ${encryptResult.error || 'Неизвестная ошибка'}`);
    }
    
    // Возвращаем объект с зашифрованным файлом и метаданными
    return {
        content: encryptResult.result,
        iv: iv,
        filename: file.name,
        size: file.size,
        type: file.type,
        encryptionParams: {
            algorithm: cipherParams.algorithm,
            mode: cipherParams.mode,
            padding: cipherParams.padding,
            keySize: cipherParams.keySize
        }
    };
}

/**
 * Дешифрует полученный файл
 * @param {Object} encryptedFile - Объект с зашифрованным файлом и метаданными
 * @param {string} senderId - ID отправителя (для получения общего ключа)
 * @returns {Promise<File>} - Расшифрованный файл
 */
export async function decryptFile(encryptedFile, senderId) {
    // Получаем общий ключ из localStorage по ID пользователя
    const sharedKey = localStorage.getItem(`dh_shared_key_${senderId}`);
    if (!sharedKey) {
        throw new Error(`Общий ключ для пользователя ${senderId} не найден`);
    }
    
    // Получаем параметры шифрования и данные
    const {content, iv, filename, type, encryptionParams} = encryptedFile;
    
    // Настраиваем параметры шифра
    const cipherParams = {
        algorithm: (encryptionParams.algorithm && encryptionParams.algorithm.toLowerCase()) || ALGORITHMS.CAMELLIA.toLowerCase(),
        mode: (encryptionParams.mode && encryptionParams.mode.toLowerCase()) || MODES.CBC,
        padding: (encryptionParams.padding && encryptionParams.padding.toLowerCase()) || PADDING.PKCS7,
        keySize: encryptionParams.keySize || 256
    };
    
    // Дешифруем файл
    const decryptResult = await cryptoService.decryptWithIV(
        cipherParams,
        content,
        sharedKey,
        iv
    );
    
    if (!decryptResult.success) {
        throw new Error(`Ошибка дешифрования файла: ${decryptResult.error || 'Неизвестная ошибка'}`);
    }
    
    // Преобразуем Base64 обратно в ArrayBuffer
    const binaryString = atob(decryptResult.result);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
    }
    
    // Создаем новый File объект
    return new File([bytes], filename, {type});
} 