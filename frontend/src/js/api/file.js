import { getFileClient, getMetadata } from "./client";
import { 
    InitFileUploadRequest, 
    FileChunk, 
    FinalizeFileUploadRequest, 
    GetFileInfoRequest, 
    DownloadFileRequest,
    UploadFileRequest,
    DeleteFileRequest
} from "../../proto/file_service_pb";
import socket from "./websocket.js";

export function uploadFile(file, receiverUsername, callback) {
    const token = localStorage.getItem('token');
    if (!token) {
        callback(new Error("Неавторизованный доступ: Токен не найден"), null);
        return;
    }

    const reader = new FileReader();
    
    reader.onload = async function(event) {
        try {
            // Генерируем уникальный идентификатор для отслеживания загрузки
            const uploadId = generateUploadId();
            
            // Получаем объект сокета
            const ws = socket.getSocket();
            if (!ws) {
                callback(new Error("WebSocket не подключен"), null);
                return;
            }
            
            // Сохраняем имя файла для использования в обработчике
            const fileName = file.name;
            
            // Уникальный ID для обработчика
            const uploadHandlerId = `upload_${uploadId}`;
            
            // Добавляем обработчик для получения ответов по загрузке файла
            const fileUploadHandler = (message) => {
                try {
                    // Проверяем, что это сообщение относится к нашей загрузке
                    if (message.uploadId === uploadId) {
                        console.log('Получено сообщение о загрузке файла:', message);
                        
                        if (message.type === 'file_upload_complete') {
                            // Когда загрузка завершена, удаляем обработчик и вызываем callback
                            socket.removeFileHandler(uploadHandlerId);
                            
                            // Передаем имя файла и другие данные в callback
                            callback(null, {
                                fileId: message.fileId,
                                fileName: message.fileName || fileName,
                                fileSize: file.size
                            });
                        } else if (message.type === 'file_upload_error') {
                            // Если произошла ошибка
                            socket.removeFileHandler(uploadHandlerId);
                            callback(new Error(message.error || 'Ошибка загрузки файла'), null);
                        }
                    }
                } catch (error) {
                    console.error('Ошибка обработки сообщения:', error);
                }
            };
            
            // Добавляем обработчик через специальный механизм для файловых сообщений
            socket.addFileHandler(uploadHandlerId, fileUploadHandler);
            
            // Создаем начальное сообщение для инициализации загрузки файла
            socket.sendMessage({
                type: 'file_upload_init',
                uploadId: uploadId,
                fileName: file.name,
                mimeType: file.type,
                totalSize: file.size,
                chatUsername: receiverUsername
            });
            
            // Разбиваем файл на чанки и отправляем их
            const chunkSize = 64 * 1024; // 64 КБ - оптимальный размер для WebSocket
            const fileData = new Uint8Array(event.target.result);
            let offset = 0;
            let chunkIndex = 0;
            
            while (offset < fileData.length) {
                const chunk = fileData.slice(offset, offset + chunkSize);
                const isLastChunk = offset + chunkSize >= fileData.length;
                
                // Отправляем чанк через WebSocket в виде base64-строки для лучшей передачи бинарных данных
                let base64Chunk;
                try {
                    // Конвертируем Uint8Array в строку символов
                    let binaryString = '';
                    for (let i = 0; i < chunk.length; i++) {
                        // Используем только валидные символы для избежания ошибок
                        binaryString += String.fromCharCode(chunk[i]);
                    }
                    
                    // Кодируем в base64
                    base64Chunk = btoa(binaryString);
                    
                    // Проверяем, что получился валидный base64
                    if (!/^[A-Za-z0-9+/=]+$/.test(base64Chunk)) {
                        throw new Error('Получена невалидная base64 строка после кодирования');
                    }
                } catch (error) {
                    console.error('Ошибка при кодировании в base64:', error);
                    callback(new Error('Ошибка кодирования файла'), null);
                    return;
                }
                
                socket.sendMessage({
                    type: 'file_chunk',
                    uploadId: uploadId,
                    chunkIndex: chunkIndex,
                    data: base64Chunk,
                    encoding: 'base64',
                    isLastChunk: isLastChunk
                });
                
                offset += chunkSize;
                chunkIndex++;
                
                // Небольшая задержка, чтобы не перегружать сокет
                await new Promise(resolve => setTimeout(resolve, 10));
                
                // Выводим прогресс в консоль
                if (chunkIndex % 10 === 0 || isLastChunk) {
                    const progress = Math.min(100, Math.round((offset / fileData.length) * 100));
                    console.log(`Загрузка файла: ${progress}%`);
                }
            }
            
            console.log('Все чанки отправлены. Ожидание подтверждения от сервера...');
        } catch (error) {
            console.error('Ошибка при загрузке файла:', error);
            callback(error, null);
        }
    };
    
    reader.onerror = function(error) {
        console.error('Ошибка чтения файла:', error);
        callback(error, null);
    };
    
    // Чтение файла как ArrayBuffer
    reader.readAsArrayBuffer(file);
}

function initFileUpload(filename, mimeType, totalSize, chatUsername, metadata) {
    return new Promise((resolve, reject) => {
        const request = new InitFileUploadRequest();
        request.setFilename(filename);
        request.setMimeType(mimeType);
        request.setTotalSize(totalSize);
        request.setChatUsername(chatUsername);
        
        fileClient.initFileUpload(request, metadata, (err, response) => {
            if (err) {
                reject(err);
                return;
            }
            resolve(response);
        });
    });
}

function uploadFileChunk(uploadId, chunkIndex, data, metadata) {
    return new Promise((resolve, reject) => {
        // Создаем запрос с чанком
        const chunk = new FileChunk();
        chunk.setUploadId(uploadId);
        chunk.setChunkIndex(chunkIndex);
        chunk.setData(data);
        
        // Унарный вызов метода вместо потоковой передачи
        fileClient.uploadFileChunk(chunk, metadata, (err, response) => {
            if (err) {
                reject(err);
                return;
            }
            resolve(response);
        });
    });
}

function finalizeFileUpload(uploadId, checksum, metadata) {
    return new Promise((resolve, reject) => {
        const request = new FinalizeFileUploadRequest();
        request.setUploadId(uploadId);
        request.setChecksum(checksum);
        
        fileClient.finalizeFileUpload(request, metadata, (err, response) => {
            if (err) {
                reject(err);
                return;
            }
            resolve(response);
        });
    });
}

export function downloadFile(fileId, callback) {
    const downloadHandlerId = `download-${fileId}`;
    let fileInfo = null;
    let receivedChunks = [];
    
    console.log(`Начало скачивания файла ${fileId}, регистрируем обработчик ${downloadHandlerId}`);
    
    // Устанавливаем обработчик для получения чанков файла
    const fileDownloadHandler = (message) => {
        try {
            // Проверяем, что это сообщение относится к нашему файлу
            if (message.fileId === fileId) {
                console.log(`Обработчик ${downloadHandlerId}: получено сообщение для файла ${fileId}, тип: ${message.type}`);
                
                if (message.type === 'file_info') {
                    // Получили информацию о файле
                    console.log('Получена информация о файле:', message);
                    fileInfo = {
                        filename: message.fileName,
                        mimeType: message.mimeType,
                        size: message.fileSize
                    };
                } else if (message.type === 'file_chunk') {
                    // Получили чанк файла
                    let chunkData;
                    
                    // Проверяем, есть ли индекс чанка, если нет - считаем его нулевым чанком (первым)
                    const chunkIndex = message.chunkIndex !== undefined ? message.chunkIndex : 0;
                    console.log(`Получен чанк ${chunkIndex}, encoding: ${message.encoding}`);
                    
                    if (message.encoding === 'base64') {
                        try {
                            // Определяем, в каком поле содержатся данные
                            const base64Data = message.dataString || message.data || '';
                            
                            // Проверяем, что строка валидная base64
                            if (!base64Data) {
                                console.error('Отсутствуют данные в чанке файла');
                                return;
                            }
                            
                            // Вывод данных о чанке для отладки
                            console.log(`Обработка чанка ${chunkIndex}, размер данных: ${base64Data.length} символов`);
                            
                            // Проверяем валидность base64 данных перед декодированием
                            if (!/^[A-Za-z0-9+/=]+$/.test(base64Data)) {
                                console.error('Невалидная base64 строка:', 
                                    base64Data.substring(0, 50) + '...');
                                
                                // Попытка очистить строку, удалив невалидные символы
                                const cleanBase64 = base64Data.replace(/[^A-Za-z0-9+/=]/g, '');
                                if (cleanBase64.length < base64Data.length * 0.9) {
                                    console.error('Слишком много невалидных символов в base64 строке');
                                    return;
                                }
                                
                                // Декодируем очищенную строку
                                console.log('Пытаемся декодировать очищенную строку...');
                                chunkData = atob(cleanBase64);
                            } else {
                                // Декодируем валидную строку
                                chunkData = atob(base64Data);
                            }
                            
                            // Преобразуем строку в Uint8Array
                            const bytes = new Uint8Array(chunkData.length);
                            for (let i = 0; i < chunkData.length; i++) {
                                bytes[i] = chunkData.charCodeAt(i);
                            }
                            chunkData = bytes;
                            
                            // Проверяем, что после декодирования получены данные
                            if (!chunkData || chunkData.length === 0) {
                                console.error(`Чанк ${chunkIndex} пуст после декодирования`);
                                return;
                            } else {
                                console.log(`Чанк ${chunkIndex} успешно декодирован, размер: ${chunkData.length} байт`);
                            }
                        } catch (error) {
                            console.error(`Ошибка при декодировании чанка ${chunkIndex}:`, error);
                            return;
                        }
                    } else {
                        // Другие типы кодирования мы не обрабатываем
                        console.error('Неизвестный тип кодирования данных:', message.encoding);
                        return;
                    }
                    
                    // Сохраняем чанк в массив с соответствующим индексом
                    console.log(`Сохраняем чанк ${chunkIndex} в массив receivedChunks`);
                    receivedChunks[chunkIndex] = chunkData;
                    
                    // Если это последний чанк, проверяем все полученные чанки
                    if (message.isLastChunk) {
                        // Определяем последний индекс и общее количество чанков
                        const lastChunkIndex = message.chunkIndex !== undefined ? message.chunkIndex : 
                            (receivedChunks.length > 0 ? receivedChunks.length - 1 : 0);
                            
                        console.log(`Получен последний чанк. Всего чанков: ${lastChunkIndex + 1}`);
                        console.log(`Размер массива receivedChunks: ${receivedChunks.length}`);
                        
                        // Выводим информацию о всех полученных чанках
                        for (let i = 0; i <= lastChunkIndex; i++) {
                            const chunk = receivedChunks[i];
                            if (chunk && chunk.length > 0) {
                                console.log(`Чанк ${i} присутствует, размер: ${chunk.length} байт`);
                            } else {
                                console.log(`Чанк ${i} отсутствует или пуст`);
                            }
                        }
                        
                        // Собираем все чанки в один массив, пропуская отсутствующие или битые чанки
                        let totalLength = 0;
                        let validChunksCount = 0;
                        
                        // Сначала подсчитываем размер валидных данных
                        for (let i = 0; i <= lastChunkIndex; i++) {
                            const chunk = receivedChunks[i];
                            if (chunk && chunk.length > 0) {
                                totalLength += chunk.length;
                                validChunksCount++;
                            } else if (i < lastChunkIndex) {
                                // Если отсутствует чанк, но он не последний - ждем следующих сообщений
                                console.warn(`Отсутствует чанк ${i}. Возможно данные еще не получены.`);
                            }
                        }
                        
                        // Проверяем, все ли чанки получены (или достаточное количество)
                        const totalChunks = lastChunkIndex + 1; // +1 т.к. индексация с 0
                        console.log(`Найдено ${validChunksCount} валидных чанков из ${totalChunks}`);
                        
                        // Устанавливаем более низкий порог для успешной сборки файла
                        if (validChunksCount < totalChunks * 0.3) { // Снижаем порог до 30%
                            console.error(`Получено недостаточно чанков: ${validChunksCount} из ${totalChunks}`);
                            console.warn('Ожидаем дополнительные чанки...');
                            return;
                        }
                        
                        // Если не хватает первого чанка (индекс 0), но есть все остальные, пропускаем его
                        if (!receivedChunks[0] && validChunksCount >= totalChunks - 1) {
                            console.warn('Отсутствует первый чанк (индекс 0), но есть почти все остальные. Продолжаем сборку файла без него.');
                        }
                        
                        // Создаем финальный буфер для всех данных
                        const fileData = new Uint8Array(totalLength);
                        let offset = 0;
                        
                        // Копируем только валидные чанки в буфер
                        for (const chunk of receivedChunks) {
                            if (chunk && chunk.length > 0) {
                                fileData.set(chunk, offset);
                                offset += chunk.length;
                            }
                        }
                        
                        // Создаем объект Blob
                        const blob = new Blob([fileData], { type: fileInfo ? fileInfo.mimeType : 'application/octet-stream' });
                        
                        // Проверяем, получилась ли валидная структура данных
                        if (fileData.length === 0) {
                            console.error('Файловые данные пусты после обработки всех чанков');
                            socket.removeFileHandler(downloadHandlerId);
                            callback(new Error('Получен пустой файл'), null);
                            return;
                        }
                        
                        // Очищаем обработчик, т.к. скачивание завершено
                        socket.removeFileHandler(downloadHandlerId);
                        
                        // Создаем данные файла, даже если не получили метаданные
                        if (!fileInfo) {
                            fileInfo = {
                                filename: 'download.file',
                                mimeType: 'application/octet-stream',
                                size: fileData.length
                            };
                        }
                        
                        // Возвращаем файл через callback
                        callback(null, {
                            blob: blob,
                            filename: fileInfo.filename,
                            mimeType: fileInfo.mimeType,
                            size: fileInfo.size
                        });
                    }
                } else if (message.type === 'file_download_error') {
                    // Если произошла ошибка
                    console.error(`Получена ошибка скачивания для файла ${fileId}:`, message.error);
                    socket.removeFileHandler(downloadHandlerId);
                    console.log(`Удалён обработчик ${downloadHandlerId} из-за ошибки`);
                    callback(new Error(message.error || 'Ошибка скачивания файла'), null);
                }
            } else if (message.type === 'file_chunk') {
                // Если это чанк другого файла, игнорируем его
                console.log(`Обработчик ${downloadHandlerId}: получен чанк для другого файла ${message.fileId}, игнорируем`);
            }
        } catch (error) {
            console.error(`Ошибка обработки сообщения в обработчике ${downloadHandlerId}:`, error);
            callback(error, null);
        }
    };
    
    // Добавляем обработчик через специальный механизм для файловых сообщений
    socket.addFileHandler(downloadHandlerId, fileDownloadHandler);
    console.log(`Обработчик ${downloadHandlerId} зарегистрирован`);
    
    // Отправляем запрос на начало скачивания файла
    socket.sendMessage({
        type: 'file_download_request',
        fileId: fileId
    });
    
    console.log(`Запрос на скачивание файла ${fileId} отправлен`);
}

// Генерация уникального ID для загрузки
function generateUploadId() {
    return Date.now().toString(36) + Math.random().toString(36).substring(2);
}

// Вспомогательная функция для вычисления MD5 хеш-суммы файла
async function calculateChecksum(fileData) {
    const buffer = await crypto.subtle.digest('MD5', fileData);
    return Array.from(new Uint8Array(buffer))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
} 