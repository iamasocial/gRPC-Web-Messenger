import { fileClient } from "./client";
import { 
    InitFileUploadRequest, 
    FileChunk, 
    FinalizeFileUploadRequest, 
    GetFileInfoRequest, 
    DownloadFileRequest 
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
            const fileUploadHandler = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    
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
                const base64Chunk = btoa(
                    Array.from(chunk)
                        .map(byte => String.fromCharCode(byte))
                        .join('')
                );
                
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
    const token = localStorage.getItem('token');
    if (!token) {
        callback(new Error("Неавторизованный доступ: Токен не найден"), null);
        return;
    }
    
    // Получаем объект сокета
    const ws = socket.getSocket();
    if (!ws) {
        callback(new Error("WebSocket не подключен"), null);
        return;
    }
    
    // Генерируем уникальный ID для этой операции скачивания
    const downloadHandlerId = `download_${fileId}_${generateUploadId()}`;
    
    let fileInfo = null;
    const chunks = [];
    
    // Устанавливаем обработчик для получения чанков файла
    const fileDownloadHandler = (event) => {
        try {
            const message = JSON.parse(event.data);
            // Проверяем, что это сообщение относится к нашему файлу
            if (message.fileId === fileId) {
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
                    
                    console.log(`Получен чанк ${message.chunkIndex}, encoding: ${message.encoding}`);
                    
                    // Обрабатываем данные в зависимости от формата кодирования
                    if (message.encoding === 'base64' && message.data) {
                        // Декодируем данные из base64
                        try {
                            const binaryString = atob(message.data);
                            chunkData = new Uint8Array(binaryString.length);
                            for (let i = 0; i < binaryString.length; i++) {
                                chunkData[i] = binaryString.charCodeAt(i);
                            }
                            console.log(`Чанк ${message.chunkIndex} успешно декодирован из base64, размер: ${chunkData.length} байт`);
                        } catch (e) {
                            console.error("Ошибка декодирования base64:", e);
                            chunkData = new Uint8Array(0);
                        }
                    } else if (Array.isArray(message.data)) {
                        // Если это массив (старый формат), преобразуем его в Uint8Array
                        chunkData = new Uint8Array(message.data);
                    } else {
                        console.error("Неизвестный формат данных:", typeof message.data);
                        // Если формат неизвестен, создаем пустой массив
                        chunkData = new Uint8Array(0);
                    }
                    
                    chunks.push(chunkData);
                    
                    // Если это последний чанк, собираем файл
                    if (message.isLastChunk) {
                        // Объединяем все чанки в один файл
                        const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
                        console.log(`Получено ${chunks.length} чанков, общий размер: ${totalLength} байт`);
                        
                        const fileData = new Uint8Array(totalLength);
                        
                        let offset = 0;
                        for (let i = 0; i < chunks.length; i++) {
                            const chunk = chunks[i];
                            console.log(`Обработка чанка ${i}, размер: ${chunk.length} байт`);
                            fileData.set(chunk, offset);
                            offset += chunk.length;
                        }
                        
                        console.log(`Файл собран, итоговый размер: ${fileData.length} байт`);
                        
                        // Создаем Blob и ссылку для скачивания
                        const blob = new Blob([fileData], { type: fileInfo?.mimeType || 'application/octet-stream' });
                        console.log(`Blob создан, размер: ${blob.size} байт, тип: ${fileInfo?.mimeType || 'application/octet-stream'}`);
                        
                        const url = URL.createObjectURL(blob);
                        
                        // Удаляем обработчик
                        socket.removeFileHandler(downloadHandlerId);
                        
                        callback(null, { 
                            filename: fileInfo?.filename || 'file', 
                            url, 
                            size: fileInfo?.size || blob.size,
                            mimeType: fileInfo?.mimeType || 'application/octet-stream'
                        });
                    }
                } else if (message.type === 'file_download_error') {
                    // Если произошла ошибка
                    socket.removeFileHandler(downloadHandlerId);
                    callback(new Error(message.error || 'Ошибка скачивания файла'), null);
                }
            }
        } catch (error) {
            console.error('Ошибка обработки сообщения:', error);
        }
    };
    
    // Добавляем обработчик через специальный механизм для файловых сообщений
    socket.addFileHandler(downloadHandlerId, fileDownloadHandler);
    
    // Отправляем запрос на начало скачивания файла
    socket.sendMessage({
        type: 'file_download_request',
        fileId: fileId
    });
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