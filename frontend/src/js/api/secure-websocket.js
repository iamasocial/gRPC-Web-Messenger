/**
 * Защищенный WebSocket клиент с шифрованием на основе WASM-модуля
 */

import socket from './websocket';
import { encryptMessage, decryptMessage, encryptFile, decryptFile } from './crypto';

/**
 * Класс-обертка вокруг WebSocketClient для автоматического шифрования/дешифрования сообщений
 */
class SecureWebSocketClient {
    constructor() {
        this.socketClient = socket;
        this.secureMessageHandlers = [];
        this.encryptionEnabled = true; // Флаг для включения/отключения шифрования
    }

    /**
     * Подключение к серверу
     * @param {string} token - Токен авторизации
     */
    connect(token) {
        this.socketClient.connect(token);
        
        // Добавляем обработчик сообщений для автоматической расшифровки
        this.socketClient.addMessageHandler(this._handleEncryptedMessage.bind(this));
    }

    /**
     * Включить/выключить шифрование
     * @param {boolean} enabled - Флаг включения/выключения
     */
    setEncryptionEnabled(enabled) {
        this.encryptionEnabled = !!enabled;
    }

    /**
     * Получить состояние шифрования
     * @returns {boolean}
     */
    isEncryptionEnabled() {
        return this.encryptionEnabled;
    }

    /**
     * Отправка зашифрованного сообщения
     * @param {Object} message - Сообщение для отправки
     * @param {string} recipientId - ID получателя
     * @param {Object} encryptionParams - Параметры шифрования (опционально)
     */
    async sendSecureMessage(message, recipientId, encryptionParams = {}) {
        if (!this.encryptionEnabled) {
            this.socketClient.sendMessage(message);
            return;
        }

        try {
            // Если сообщение - объект, превращаем его в строку
            const messageContent = typeof message === 'object' ? 
                JSON.stringify(message) : 
                message.toString();
            
            // Шифруем сообщение
            const encryptedData = await encryptMessage(messageContent, recipientId, encryptionParams);
            
            // Формируем сообщение для отправки
            const secureMessage = {
                type: 'secure_message',
                recipientId: recipientId,
                encrypted: true,
                data: encryptedData
            };
            
            // Отправляем через WebSocket
            this.socketClient.sendMessage(secureMessage);
        } catch (error) {
            console.error('Ошибка при отправке зашифрованного сообщения:', error);
            throw error;
        }
    }

    /**
     * Отправка зашифрованного файла
     * @param {File} file - Файл для отправки
     * @param {string} recipientId - ID получателя
     * @param {Object} encryptionParams - Параметры шифрования (опционально)
     */
    async sendSecureFile(file, recipientId, encryptionParams = {}) {
        if (!this.encryptionEnabled) {
            // Если шифрование отключено, используйте обычную логику отправки файлов
            console.warn('Шифрование отключено, файл будет отправлен без шифрования');
            // Здесь должен быть ваш код для отправки файлов без шифрования
            return;
        }

        try {
            // Шифруем файл
            const encryptedFile = await encryptFile(file, recipientId, encryptionParams);
            
            // Формируем сообщение для отправки
            const fileMessage = {
                type: 'secure_file',
                recipientId: recipientId,
                encrypted: true,
                data: encryptedFile
            };
            
            // Отправляем через WebSocket
            this.socketClient.sendMessage(fileMessage);
        } catch (error) {
            console.error('Ошибка при отправке зашифрованного файла:', error);
            throw error;
        }
    }

    /**
     * Обработчик входящих зашифрованных сообщений
     * @param {Object} message - Полученное сообщение
     * @private
     */
    async _handleEncryptedMessage(message) {
        // Проверяем, является ли сообщение зашифрованным
        if (!message || !message.encrypted) {
            // Пропускаем незашифрованные сообщения
            this._notifySecureHandlers(message);
            return;
        }

        try {
            const senderId = message.senderId;
            
            if (message.type === 'secure_message') {
                // Расшифровываем сообщение
                const decryptedContent = await decryptMessage(message.data, senderId);
                
                // Пытаемся распарсить JSON, если это объект
                let decryptedMessage;
                try {
                    decryptedMessage = JSON.parse(decryptedContent);
                } catch (e) {
                    // Если не удалось распарсить как JSON, оставляем как строку
                    decryptedMessage = decryptedContent;
                }
                
                // Создаем объект расшифрованного сообщения
                const secureMessage = {
                    ...message,
                    data: decryptedMessage,
                    decrypted: true
                };
                
                // Оповещаем обработчики
                this._notifySecureHandlers(secureMessage);
            } 
            else if (message.type === 'secure_file') {
                // Расшифровываем файл
                const decryptedFile = await decryptFile(message.data, senderId);
                
                // Создаем объект с расшифрованным файлом
                const secureFile = {
                    ...message,
                    file: decryptedFile,
                    decrypted: true
                };
                
                // Оповещаем обработчики
                this._notifySecureHandlers(secureFile);
            }
        } catch (error) {
            console.error('Ошибка при обработке зашифрованного сообщения:', error);
            
            // Даже при ошибке расшифровки оповещаем обработчики с флагом ошибки
            this._notifySecureHandlers({
                ...message,
                decryptError: true,
                error: error.message
            });
        }
    }

    /**
     * Оповещает все обработчики безопасных сообщений
     * @param {Object} message - Сообщение для обработки
     * @private
     */
    _notifySecureHandlers(message) {
        this.secureMessageHandlers.forEach(handler => {
            try {
                handler(message);
            } catch (e) {
                console.error('Ошибка в обработчике защищенных сообщений:', e);
            }
        });
    }

    /**
     * Добавить обработчик защищенных сообщений
     * @param {Function} handler - Функция-обработчик
     */
    addSecureMessageHandler(handler) {
        this.secureMessageHandlers.push(handler);
    }

    /**
     * Удалить конкретный обработчик защищенных сообщений
     * @param {Function} handler - Функция-обработчик для удаления
     */
    removeSecureMessageHandler(handler) {
        const index = this.secureMessageHandlers.indexOf(handler);
        if (index !== -1) {
            this.secureMessageHandlers.splice(index, 1);
        }
    }

    /**
     * Удалить все обработчики защищенных сообщений
     */
    removeAllSecureMessageHandlers() {
        this.secureMessageHandlers = [];
    }

    /**
     * Закрыть соединение
     */
    close() {
        this.socketClient.close();
        this.removeAllSecureMessageHandlers();
    }

    /**
     * Проксирование методов из основного WebSocketClient
     */
    getSocket() {
        return this.socketClient.getSocket();
    }

    /**
     * Добавить обработчик файлов из основного WebSocketClient
     */
    addFileHandler(handlerId, handler) {
        this.socketClient.addFileHandler(handlerId, handler);
    }

    /**
     * Удалить обработчик файлов из основного WebSocketClient
     */
    removeFileHandler(handlerId) {
        this.socketClient.removeFileHandler(handlerId);
    }
}

// Создаем и экспортируем экземпляр защищенного WebSocket клиента
const secureSocket = new SecureWebSocketClient();
export default secureSocket; 