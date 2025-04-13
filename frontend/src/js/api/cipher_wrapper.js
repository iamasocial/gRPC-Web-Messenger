/**
 * Модуль-обертка для работы с WASM-модулем шифрования
 * Обеспечивает интеграцию JavaScript кода с алгоритмами Camellia и MAGENTA через WebAssembly
 */

import { ALGORITHMS, MODES, PADDING, cryptoService } from './crypto.js';

/**
 * Создает интерфейс для работы с шифрованием
 * @param {Object} config Конфигурация шифрования
 * @returns {Object} API для шифрования
 */
export function createCipherAPI(config = {}) {
    const defaultConfig = {
        algorithm: ALGORITHMS.CAMELLIA,
        mode: MODES.CBC,
        padding: PADDING.PKCS7,
        keySize: 256
    };

    const mergedConfig = { ...defaultConfig, ...config };
    
    return {
        /**
         * Шифрует сообщение
         * @param {string} plaintext Исходное сообщение
         * @param {string} username Имя пользователя для получения ключа
         * @returns {Promise<Object>} Зашифрованное сообщение и метаданные
         */
        async encryptMessage(plaintext, username) {
            try {
                // Получение общего ключа из localStorage
                const sharedKey = localStorage.getItem(`dh_shared_key_${username}`);
                if (!sharedKey) {
                    throw new Error(`Ключ для пользователя ${username} не найден`);
                }

                // Создаем параметры шифра для WASM-модуля
                const cipherParams = {
                    algorithm: mergedConfig.algorithm,
                    mode: mergedConfig.mode,
                    padding: mergedConfig.padding,
                    keySize: mergedConfig.keySize,
                    generateIV: true
                };
                
                // Преобразуем сообщение в Base64
                const messageBase64 = cryptoService.constructor.stringToBase64(plaintext);
                
                // Получаем информацию о шифре для создания IV
                const cipherInfo = await cryptoService.createCipher(cipherParams);
                if (!cipherInfo.success) {
                    throw new Error(`Ошибка создания шифра: ${cipherInfo.error || 'Неизвестная ошибка'}`);
                }
                
                // Шифруем сообщение с помощью WASM
                const encryptResult = await cryptoService.encryptWithIV(
                    cipherParams,
                    messageBase64,
                    sharedKey,
                    cipherInfo.cipher.iv
                );
                
                if (!encryptResult.success) {
                    throw new Error(`Ошибка шифрования: ${encryptResult.error || 'Неизвестная ошибка'}`);
                }
                
                return {
                    content: encryptResult.result,
                    iv: cipherInfo.cipher.iv,
                    encryptionParams: {
                        algorithm: mergedConfig.algorithm,
                        mode: mergedConfig.mode,
                        padding: mergedConfig.padding,
                        keySize: mergedConfig.keySize
                    }
                };
            } catch (error) {
                console.error('Ошибка шифрования:', error);
                throw error;
            }
        },

        /**
         * Дешифрует сообщение
         * @param {Object} encryptedData Зашифрованное сообщение и метаданные
         * @param {string} username Имя пользователя отправителя
         * @returns {Promise<string>} Расшифрованное сообщение
         */
        async decryptMessage(encryptedData, username) {
            try {
                // Получение общего ключа из localStorage
                const sharedKey = localStorage.getItem(`dh_shared_key_${username}`);
                if (!sharedKey) {
                    throw new Error(`Ключ для пользователя ${username} не найден`);
                }

                // Проверка наличия необходимых данных
                const { content, iv, encryptionParams } = encryptedData;
                
                if (!content || !iv) {
                    throw new Error('Неверный формат зашифрованных данных');
                }
                
                // Создаем параметры для дешифрования
                const cipherParams = {
                    algorithm: encryptionParams?.algorithm || mergedConfig.algorithm,
                    mode: encryptionParams?.mode || mergedConfig.mode,
                    padding: encryptionParams?.padding || mergedConfig.padding,
                    keySize: encryptionParams?.keySize || mergedConfig.keySize
                };
                
                // Дешифруем сообщение с помощью WASM
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
                return cryptoService.constructor.base64ToString(decryptResult.result);
            } catch (error) {
                console.error('Ошибка дешифрования:', error);
                throw error;
            }
        },

        /**
         * Возвращает информацию о текущей конфигурации шифрования
         * @returns {Object} Информация о конфигурации
         */
        getConfig() {
            return {
                algorithm: mergedConfig.algorithm,
                mode: mergedConfig.mode,
                padding: mergedConfig.padding,
                keySize: mergedConfig.keySize
            };
        }
    };
}

/**
 * Расширение для chat.js - модифицирует функции для шифрования сообщений
 * @param {Object} chat Объект с методами из chat.js
 * @returns {Object} Модифицированные методы для шифрования
 */
export function extendChatWithEncryption(chat) {
    const originalSend = chat.chat;
    const originalFileMessage = chat.sendFileMessage;
    
    // Создаем API для шифрования
    const cipherAPI = createCipherAPI();
    
    return {
        ...chat,
        
        /**
         * Отправляет зашифрованное сообщение
         * @param {string} content Содержимое сообщения
         * @param {string} username Имя получателя
         * @param {function} callback Функция обратного вызова
         */
        async chat(content, username, callback) {
            try {
                // Шифруем сообщение
                const encrypted = await cipherAPI.encryptMessage(content, username);
                
                // Формируем сообщение с зашифрованными данными
                const secureMessage = {
                    encrypted: true,
                    content: encrypted.content,
                    iv: encrypted.iv,
                    encryptionParams: encrypted.encryptionParams,
                    recipientUsername: username
                };
                
                // Отправляем зашифрованное сообщение
                originalSend(JSON.stringify(secureMessage), (err) => {
                    if (callback) callback(err);
                });
            } catch (err) {
                console.error('Ошибка отправки зашифрованного сообщения:', err);
                if (callback) callback(err);
            }
        },
        
        /**
         * Отправляет зашифрованное файловое сообщение
         * @param {string} fileId ID файла
         * @param {string} fileName Имя файла
         * @param {number} fileSize Размер файла
         * @param {string} comment Комментарий к файлу
         * @param {string} username Имя получателя
         */
        async sendFileMessage(fileId, fileName, fileSize, comment, username) {
            try {
                // Шифруем комментарий, если он есть
                let encryptedComment = '';
                let encryptionData = null;
                
                if (comment && comment.trim()) {
                    encryptionData = await cipherAPI.encryptMessage(comment, username);
                    encryptedComment = encryptionData.content;
                }
                
                // Формируем зашифрованное файловое сообщение
                const secureFileMessage = {
                    encrypted: true,
                    fileId: fileId,
                    fileName: fileName,
                    fileSize: fileSize,
                    content: encryptedComment,
                    messageType: 'file',
                    recipientUsername: username,
                    iv: encryptionData?.iv,
                    encryptionParams: encryptionData?.encryptionParams
                };
                
                // Отправляем зашифрованное файловое сообщение
                originalSend(secureFileMessage);
            } catch (err) {
                console.error('Ошибка отправки зашифрованного файлового сообщения:', err);
            }
        },
        
        /**
         * Обработчик для дешифрования входящих сообщений
         * @param {Object} message Входящее сообщение
         * @returns {Promise<Object>} Дешифрованное сообщение
         */
        async processIncomingMessage(message) {
            try {
                // Проверяем, является ли сообщение зашифрованным
                if (!message || !message.encrypted) {
                    return message;
                }
                
                const senderUsername = message.senderUsername || message.senderusername || 
                                       message.SenderUsername || message.Senderusername;
                
                if (!senderUsername) {
                    console.warn('Невозможно определить отправителя зашифрованного сообщения');
                    return message;
                }
                
                // Дешифруем сообщение с помощью WASM
                let decryptedContent = '';
                
                if (message.content) {
                    const encryptedData = {
                        content: message.content,
                        iv: message.iv,
                        encryptionParams: message.encryptionParams
                    };
                    
                    decryptedContent = await cipherAPI.decryptMessage(encryptedData, senderUsername);
                }
                
                // Проверяем, является ли сообщение файловым
                if (message.messageType === 'file') {
                    return {
                        ...message,
                        content: decryptedContent,
                        isDecrypted: true
                    };
                }
                
                // Пытаемся распарсить JSON, если это объект
                let decryptedData;
                try {
                    decryptedData = JSON.parse(decryptedContent);
                } catch (e) {
                    // Если не удалось распарсить как JSON, оставляем как строку
                    decryptedData = decryptedContent;
                }
                
                // Возвращаем обновленное сообщение с дешифрованными данными
                return {
                    ...message,
                    content: typeof decryptedData === 'object' ? decryptedData.content || decryptedContent : decryptedContent,
                    isDecrypted: true
                };
            } catch (err) {
                console.error('Ошибка дешифрования сообщения:', err);
                // Возвращаем исходное сообщение с пометкой об ошибке
                return {
                    ...message,
                    decryptError: err.message
                };
            }
        }
    };
} 