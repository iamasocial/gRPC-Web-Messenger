import { encryptionService } from "./encryption.js";

class WebSocketService {
    constructor() {
        this.socket = null;
        this.messageHandlers = new Set();
        this.fileHandlers = new Map();
        this.encryptionService = encryptionService;
    }

    async connect() {
        try {
            console.log("Инициализация шифрования для WebSocket...");
            await this.encryptionService.initialize();
            console.log("Шифрование инициализировано успешно");

            const token = localStorage.getItem('token');
            if (!token) {
                throw new Error('Токен не найден');
            }

            // Создаем WebSocket соединение с токеном в Sec-WebSocket-Protocol
            this.socket = new WebSocket('ws://localhost:8888/ws', [token]);
            console.log("WebSocket соединение создано");

            return new Promise((resolve, reject) => {
                this.socket.onopen = () => {
                    console.log("WebSocket соединение установлено");
                    resolve();
                };

                this.socket.onmessage = (event) => {
                    try {
                        const message = JSON.parse(event.data);
                        console.log("Получено сообщение:", message);

                        // Если это сообщение о шифровании, обрабатываем его
                        if (message.type === 'encryption') {
                            this.handleEncryptionMessage(message);
                            return;
                        }

                        // Если это файловое сообщение, используем специальные обработчики
                        if (message.type === 'file_info' || 
                            message.type === 'file_chunk' || 
                            message.type === 'file_download_error' ||
                            message.type === 'file_upload_initialized' ||
                            message.type === 'file_chunk_received' ||
                            message.type === 'file_upload_complete' ||
                            message.type === 'file_upload_error') {
                            try {
                                this.handleFileMessage(message);
                            } catch (error) {
                                console.error("Ошибка при обработке файлового сообщения:", error);
                            }
                            return;
                        }

                        // Для обычных сообщений используем стандартные обработчики
                        this.messageHandlers.forEach(handler => {
                            try {
                                handler(message);
                            } catch (error) {
                                console.error("Ошибка в обработчике сообщений:", error);
                            }
                        });
                    } catch (error) {
                        console.error("Ошибка обработки сообщения:", error);
                    }
                };

                this.socket.onclose = () => {
                    console.log("WebSocket соединение закрыто");
                };

                this.socket.onerror = (error) => {
                    console.error("Ошибка WebSocket:", error);
                    reject(error);
                };
            });
        } catch (error) {
            console.error("Ошибка при подключении к WebSocket:", error);
            throw error;
        }
    }

    // Обработка сообщений о шифровании
    handleEncryptionMessage(message) {
        console.log("Получено сообщение о шифровании:", message);
        // Проверяем тип сообщения о шифровании
        if (message.type === 'encryption' && message.action === 'key_exchange') {
            // Инициируем обмен ключами
            this.encryptionService.initialize(message.params);
            console.log("Инициирован обмен ключами");
        }
    }

    // Обработка файловых сообщений
    handleFileMessage(message) {
        console.log("Получено файловое сообщение:", message);
        
        // Проверяем что сообщение корректное
        if (!message || typeof message !== 'object') {
            console.error("Получено некорректное файловое сообщение");
            return;
        }
        
        // Особая обработка для сообщений file_chunk
        if (message.type === 'file_chunk') {
            // Выводим дополнительную информацию для диагностики
            if (message.fileId) {
                // Обнаружили сообщение без индекса чанка
                if (message.chunkIndex === undefined) {
                    console.warn(`WebSocket: чанк без индекса для файла ${message.fileId}. Будет обработан как чанк с индексом 0.`);
                }
                
                // Для скачивания файла
                const chunkIndex = message.chunkIndex !== undefined ? message.chunkIndex : 0;
                console.log(`WebSocket: получен чанк для скачивания файла ${message.fileId}, индекс: ${chunkIndex}, размер данных: ${message.data ? message.data.length : 'неизвестен'}`);
                
                // Проверка на наличие данных
                if (!message.data && !message.dataString) {
                    console.error(`WebSocket: чанк ${chunkIndex} не содержит данных`);
                } else {
                    console.log(`WebSocket: чанк ${chunkIndex} содержит данные, перенаправляем обработчикам`);
                }
            } else if (message.uploadId) {
                // Для загрузки файла
                const chunkIndex = message.chunkIndex !== undefined ? message.chunkIndex : 0;
                console.log(`WebSocket: получен чанк для загрузки файла ${message.uploadId}, индекс: ${chunkIndex}`);
            } else {
                console.error("WebSocket: В сообщении file_chunk отсутствуют и fileId, и uploadId:", message);
                return;
            }
        }
        
        // Передаем сообщение всем зарегистрированным обработчикам файлов
        let handlersCount = 0;
        this.fileHandlers.forEach((handler, handlerId) => {
            try {
                // Добавляем счетчик вызванных обработчиков
                handlersCount++;
                
                // Передаем сообщение обработчику
                handler(message);
            } catch (error) {
                console.error(`Ошибка в обработчике файлов ${handlerId}:`, error);
            }
        });
        
        // Проверяем, были ли вызваны обработчики
        if (handlersCount === 0 && message.type === 'file_chunk') {
            console.warn(`WebSocket: для чанка ${message.chunkIndex} не найдено обработчиков`);
        } else {
            console.log(`WebSocket: сообщение передано ${handlersCount} обработчикам`);
        }
    }

    sendMessage(message) {
        if (!this.socket) {
            throw new Error('WebSocket не подключен');
        }

        console.log('WebSocket sent:', message);
        this.socket.send(JSON.stringify(message));
    }

    addMessageHandler(handler) {
        this.messageHandlers.add(handler);
    }
    
    removeMessageHandler(handler) {
        this.messageHandlers.delete(handler);
    }
    
    // Для файловых обработчиков
    addFileHandler(handlerId, handler) {
        console.log(`WebSocket: регистрируем обработчик файлов '${handlerId}'`);
        
        // Проверяем, что handler является функцией
        if (typeof handler !== 'function') {
            console.error(`WebSocket: handler для '${handlerId}' не является функцией`);
            return false;
        }
        
        // Проверяем, не существует ли уже обработчик с таким ID
        if (this.fileHandlers.has(handlerId)) {
            console.warn(`WebSocket: обработчик '${handlerId}' уже зарегистрирован, обновляем его`);
            this.fileHandlers.delete(handlerId);
        }
        
        this.fileHandlers.set(handlerId, handler);
        console.log(`WebSocket: обработчик '${handlerId}' успешно зарегистрирован. Всего обработчиков: ${this.fileHandlers.size}`);
        return true;
    }
    
    removeFileHandler(handlerId) {
        console.log(`WebSocket: удаляем обработчик файлов '${handlerId}'`);
        
        if (!this.fileHandlers.has(handlerId)) {
            console.warn(`WebSocket: обработчик '${handlerId}' не найден при попытке удаления`);
            return false;
        }
        
        const success = this.fileHandlers.delete(handlerId);
        console.log(`WebSocket: обработчик '${handlerId}' ${success ? 'успешно удален' : 'не удалось удалить'}. Осталось обработчиков: ${this.fileHandlers.size}`);
        return success;
    }

    close() {
        if (this.socket) {
            this.socket.close();
            this.socket = null;
        }
        this.messageHandlers.clear();
        this.fileHandlers.clear();
    }

    // Получение доступа к объекту сокета
    getSocket() {
        return this.socket;
    }
}

const socket = new WebSocketService();
export default socket;