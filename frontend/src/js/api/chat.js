import { chatClient } from "./client"; 
import socket from "./websocket.js";
import { GetChatsRequst, ConnectRequest, CreateChatRequest, DeleteChatRequest } from "../../proto/chat_service_pb";

export function getChats(callback) {
    const token = localStorage.getItem('token');
    if (!token) {
        callback(new Error("Unathorized: No token found"), null);
        return;
    }

    const request = new GetChatsRequst();
    const metadata = { 'Authorization': `Bearer ${token}` };

    chatClient.getChats(request, metadata, (err, response) => {
        if (err) {
            callback(err, null);
            return;
        }

        // Получаем список объектов ChatInfo
        const chatsList = response.getChatsList();
        
        // Преобразуем в массив объектов с информацией о чате
        const chats = chatsList.map(chatInfo => ({
            username: chatInfo.getUsername(),
            encryptionAlgorithm: chatInfo.getEncryptionAlgorithm(),
            encryptionMode: chatInfo.getEncryptionMode(),
            encryptionPadding: chatInfo.getEncryptionPadding()
        }));

        callback(null, chats);
    });
}

export function connectToChat(receiverUsername, callback) {
    const token = localStorage.getItem("token");
    if (!token) {
        callback(new Error("User is not authenticated"), null);
        return;
    }

    const request = new ConnectRequest();
    request.setReceiverusername(receiverUsername);
    const metadata = { 'Authorization': `Bearer ${token}` };
    
    chatClient.connectToChat(request, metadata, (err, response) => {
        if (err) {
            console.error("Error connecting to chat:", err)
            callback(err, null);
            return;
        }

        console.log(`Connected to ${receiverUsername}`)
        callback(null, response.getSuccess());
    });
}

let currentHandler = null;
let isConnected = false;

export function startChat(onMessageReceived) {
    const token = localStorage.getItem("token");
    if (!token) {
        console.error("User is not authenticated");
        window.location.href = "index.html";
        return;
    }

    // Если уже подключены, только обновляем обработчик
    if (isConnected) {
        // Заменяем текущий обработчик сообщений
        if (currentHandler) {
            socket.removeSpecificMessageHandler(currentHandler);
        }
        currentHandler = onMessageReceived;
        socket.addMessageHandler(onMessageReceived);
        return;
    }

    // Подключаемся к сокету
    socket.connect(token);
    isConnected = true;
    currentHandler = onMessageReceived;
    socket.addMessageHandler(onMessageReceived);
}

/**
 * Отправка текстового сообщения в чат
 * @param {string} content - Содержимое сообщения
 * @param {function} callback - Функция обратного вызова (err)
 */
export function chat(content, callback) {
    if (!content.trim()) {
        const error = new Error("Message content cannot be empty");
        if (callback) callback(error);
        return;
    }

    const message = {
        content: content,
    };

    try {
        socket.sendMessage(message);
        if (callback) callback(null);
    } catch (err) {
        console.error("Error sending message:", err);
        if (callback) callback(err);
    }
}

/**
 * Отправка файлового сообщения в чат
 * @param {string} fileId - Идентификатор файла
 * @param {string} fileName - Имя файла
 * @param {number} fileSize - Размер файла в байтах
 * @param {string} comment - Дополнительный комментарий к файлу (опционально)
 */
export function sendFileMessage(fileId, fileName, fileSize, comment = '') {
    // Создаем сообщение о файле с метаданными
    const fileMessage = {
        content: comment, // Сохраняем комментарий (если есть) в текстовом поле
        fileId: fileId,
        fileName: fileName,
        fileSize: fileSize,
        messageType: 'file'
    };
    
    // Отправляем сообщение
    socket.sendMessage(fileMessage);
}

export function stopChat() {
    if (currentHandler) {
        socket.removeSpecificMessageHandler(currentHandler);
        currentHandler = null;
    }
    isConnected = false;
    socket.close();
}

/**
 * Создает новый чат с пользователем
 * @param {string} username - Имя пользователя
 * @param {Object} encryptionParams - Параметры шифрования
 * @param {string} encryptionParams.algorithm - Алгоритм шифрования (например, "AES")
 * @param {string} encryptionParams.mode - Режим шифрования (например, "GCM")
 * @param {string} encryptionParams.padding - Тип набивки (например, "NoPadding")
 * @param {function} callback - Функция обратного вызова (err, username)
 */
export function createChat(username, encryptionParams = {}, callback) {
    // Если второй параметр - функция, значит encryptionParams не передан
    if (typeof encryptionParams === 'function') {
        callback = encryptionParams;
        encryptionParams = {};
    }

    const token = localStorage.getItem("token");
    if (!token) {
        callback(new Error("Unathorized: No token found"), null)
        return;
    }

    const request = new CreateChatRequest();
    request.setUsername(username);
    
    // Устанавливаем параметры шифрования, если указаны
    if (encryptionParams.algorithm) {
        request.setEncryptionAlgorithm(encryptionParams.algorithm);
    }
    if (encryptionParams.mode) {
        request.setEncryptionMode(encryptionParams.mode);
    }
    if (encryptionParams.padding) {
        request.setEncryptionPadding(encryptionParams.padding);
    }
    
    const metadata = { 'Authorization': `Bearer ${token}` };

    chatClient.createChat(request, metadata, (err, response) => {
        if (err) {
            callback(err, null)
            return;
        }
        callback(null, response.getUsername());
    });
}

/**
 * Инициализирует чат с пользователем
 * @param {string} username - Имя собеседника
 * @param {Object} encryptionParams - Параметры шифрования
 * @param {function} callback - Функция обратного вызова (err, chatInfo)
 */
export function initChat(username, encryptionParams = {}, callback) {
    // Если второй параметр - функция, значит encryptionParams не передан
    if (typeof encryptionParams === 'function') {
        callback = encryptionParams;
        encryptionParams = {};
    }
    
    // Устанавливаем значения по умолчанию
    const params = {
        algorithm: encryptionParams.algorithm || 'AES',
        mode: encryptionParams.mode || 'GCM',
        padding: encryptionParams.padding || 'NoPadding'
    };
    
    // Создание чата
    createChat(username, params, (err, otherUsername) => {
        if (err) {
            callback(err, null);
            return;
        }
        
        callback(null, { 
            username: otherUsername,
            encryptionParams: params 
        });
    });
}

/**
 * Удаляет чат с указанным пользователем
 * @param {string} username - Имя пользователя чата для удаления
 * @param {function} callback - Функция обратного вызова (err, success)
 */
export function deleteChat(username, callback) {
    const token = localStorage.getItem('token');
    if (!token) {
        callback(new Error("Unathorized: No token found"), null);
        return;
    }

    const request = new DeleteChatRequest();
    request.setUsername(username);
    
    const metadata = { 'Authorization': `Bearer ${token}` };

    chatClient.deleteChat(request, metadata, (err, response) => {
        if (err) {
            console.error(`Ошибка при удалении чата с ${username}:`, err);
            callback(err, null);
            return;
        }
        
        // Также удалим связанные ключи из localStorage
        localStorage.removeItem(`dh_private_key_${username}`);
        localStorage.removeItem(`dh_shared_key_${username}`);
        
        callback(null, response.getSuccess());
    });
}