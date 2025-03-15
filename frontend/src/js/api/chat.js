import { getChatClient, getMetadata } from "./client"; 
import socket from "./websocket.js";
import { GetChatsRequst, ConnectRequest, CreateChatRequest } from "../../proto/chat_service_pb";

export function getChats(callback) {
    try {
        const metadata = getMetadata();
        const client = getChatClient();
        const request = new GetChatsRequst();

        client.getChats(request, metadata, (err, response) => {
            if (err) {
                callback(err, null);
                return;
            }

            callback(null, response.getUsernamesList());
        });
    } catch (error) {
        callback(error, null);
    }
}

export function connectToChat(receiverUsername, callback) {
    try {
        const metadata = getMetadata();
        const client = getChatClient();
        const request = new ConnectRequest();
        request.setReceiverusername(receiverUsername);
        
        client.connectToChat(request, metadata, (err, response) => {
            if (err) {
                console.error("Error connecting to chat:", err)
                callback(err, null);
                return;
            }

            console.log(`Connected to ${receiverUsername}`)
            callback(null, response.getSuccess());
        });
    } catch (error) {
        callback(error, null);
    }
}

let currentHandler = null;
let isConnected = false;

export function startChat(onMessageReceived) {
    const token = localStorage.getItem("token");
    if (!token) {
        console.error("Пользователь не авторизован");
        window.location.href = "index.html";
        return;
    }

    // Если уже подключены, только обновляем обработчик
    if (isConnected) {
        // Заменяем текущий обработчик сообщений
        if (currentHandler) {
            socket.removeMessageHandler(currentHandler);
        }
        currentHandler = onMessageReceived;
        socket.addMessageHandler(onMessageReceived);
        return;
    }

    // Подключаемся к сокету
    socket.connect().catch(error => {
        console.error("Ошибка при подключении к WebSocket:", error);
        if (error.message.includes('Токен не найден')) {
            window.location.href = "index.html";
        }
    });
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
        type: "text",
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
    const fileMessage = {
        content: comment,
        fileId: fileId,
        fileName: fileName,
        fileSize: fileSize,
        messageType: 'file'
    };
    
    socket.sendMessage(fileMessage);
}

export function stopChat() {
    if (currentHandler) {
        socket.removeMessageHandler(currentHandler);
        currentHandler = null;
    }
    isConnected = false;
    socket.close();
}

export function createChat(username, callback) {
    try {
        const metadata = getMetadata();
        const client = getChatClient();
        const request = new CreateChatRequest();
        request.setUsername(username);

        client.createChat(request, metadata, (err, response) => {
            if (err) {
                callback(err, null)
                return;
            }
            callback(null, response.getUsername());
        });
    } catch (error) {
        callback(error, null);
    }
}