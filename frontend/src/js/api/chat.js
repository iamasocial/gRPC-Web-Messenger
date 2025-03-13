import { chatClient } from "./client"; 
import socket from "./websocket.js";
import { GetChatsRequst, ConnectRequest, CreateChatRequest } from "../../proto/chat_service_pb";

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

        callback(null, response.getUsernamesList());
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

export function createChat(username, callback) {
    const token = localStorage.getItem("token");
    if (!token) {
        callback(new Error("Unathorized: No token found"), null)
        return;
    }

    const request = new CreateChatRequest();
    request.setUsername(username);
    const metadata = { 'Authorization': `Bearer ${token}` };

    chatClient.createChat(request, metadata, (err, response) => {
        if (err) {
            callback(err, null)
            return;
        }
        callback(null, response.getUsername());
    });
}