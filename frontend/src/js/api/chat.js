import { chatClient } from "./client"; 
import socket from "./websocket.js";
import { GetChatsRequst, ConnectRequest, CreateChatRequest } from "../../proto/chat_service_pb";
import { generateDHKeyPair, exportPublicKey, importPublicKey, deriveSharedSecret } from '../crypto/encryption.js';

/**
 * Функции для работы с ключами шифрования
 */

/**
 * Генерирует уникальный ключ для пары пользователей
 * @param {string} username1 - Имя первого пользователя
 * @param {string} username2 - Имя второго пользователя
 * @returns {string} - Уникальный ключ для пары пользователей
 */
function generateUserPairKey(username1, username2) {
  // Сортируем имена, чтобы ключ был одинаковым независимо от порядка
  const sortedUsernames = [username1, username2].sort();
  return `key_${sortedUsernames[0]}_${sortedUsernames[1]}`;
}

/**
 * Сохраняет ключ шифрования для пары пользователей
 * @param {string} username1 - Имя первого пользователя
 * @param {string} username2 - Имя второго пользователя
 * @param {string} key - Ключ шифрования
 */
export function saveEncryptionKey(username1, username2, key) {
  const pairKey = generateUserPairKey(username1, username2);
  const encryptionKeys = JSON.parse(localStorage.getItem('encryptionKeys') || '{}');
  encryptionKeys[pairKey] = key;
  localStorage.setItem('encryptionKeys', JSON.stringify(encryptionKeys));
}

/**
 * Получает ключ шифрования для пары пользователей
 * @param {string} username1 - Имя первого пользователя
 * @param {string} username2 - Имя второго пользователя
 * @returns {string|null} - Ключ шифрования или null, если ключ не найден
 */
export function getEncryptionKey(username1, username2) {
  const pairKey = generateUserPairKey(username1, username2);
  const encryptionKeys = JSON.parse(localStorage.getItem('encryptionKeys') || '{}');
  return encryptionKeys[pairKey] || null;
}

/**
 * Проверяет, существует ли ключ для пары пользователей
 * @param {string} username1 - Имя первого пользователя
 * @param {string} username2 - Имя второго пользователя
 * @returns {boolean} - true, если ключ существует, иначе false
 */
export function hasEncryptionKey(username1, username2) {
  return getEncryptionKey(username1, username2) !== null;
}

/**
 * Выполняет обмен ключами между пользователями (реализация Диффи-Хеллмана)
 * @param {string} otherUsername - Имя собеседника
 * @param {Function} callback - Функция обратного вызова (err, sharedKey)
 */
export async function exchangeKeys(otherUsername, callback) {
  try {
    const currentUsername = localStorage.getItem('username');
    
    // Проверяем, существует ли уже ключ для этого собеседника
    if (hasEncryptionKey(currentUsername, otherUsername)) {
      const existingKey = getEncryptionKey(currentUsername, otherUsername);
      callback(null, existingKey);
      return;
    }
    
    // 1. Генерируем пару ключей Диффи-Хеллмана
    const keyPair = await generateDHKeyPair();
    
    // 2. Экспортируем публичный ключ для отправки собеседнику
    const publicKeyBase64 = await exportPublicKey(keyPair.publicKey);
    
    // 3. Отправляем публичный ключ собеседнику
    // Здесь нужно реализовать API для обмена ключами
    // Пример:
    // await sendPublicKey(otherUsername, publicKeyBase64);
    
    // 4. Получаем публичный ключ от собеседника
    // const otherPublicKeyBase64 = await receivePublicKey(otherUsername);
    
    // Временное решение для демонстрации:
    const otherPublicKeyBase64 = publicKeyBase64; // В реальном приложении здесь должен быть ключ от собеседника
    
    // 5. Импортируем публичный ключ собеседника
    const otherPublicKey = await importPublicKey(otherPublicKeyBase64);
    
    // 6. Вычисляем общий секретный ключ
    const sharedSecret = await deriveSharedSecret(keyPair.privateKey, otherPublicKey);
    
    // 7. Сохраняем ключ в локальном хранилище
    saveEncryptionKey(currentUsername, otherUsername, sharedSecret);
    
    // 8. Возвращаем ключ через callback
    callback(null, sharedSecret);
  } catch (error) {
    callback(error, null);
  }
}

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
        // После создания чата ключа шифрования еще нет
        // Он будет создан позже с помощью функции exchangeKeys
        callback(null, response.getUsername());
    });
}

/**
 * Инициализирует защищенный чат с обменом ключами
 * @param {string} username - Имя собеседника
 * @param {Object} encryptionParams - Параметры шифрования
 * @param {function} callback - Функция обратного вызова (err, chatInfo)
 */
export function initSecureChat(username, encryptionParams = {}, callback) {
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
    
    // Шаг 1: Создание чата
    createChat(username, params, (err, otherUsername) => {
        if (err) {
            callback(err, null);
            return;
        }
        
        // Шаг 2: Обмен ключами
        exchangeKeys(otherUsername, (err, sharedKey) => {
            if (err) {
                console.error("Ошибка обмена ключами:", err);
                callback(null, { username: otherUsername, keyExchanged: false });
                return;
            }
            
            callback(null, { 
                username: otherUsername, 
                keyExchanged: true, 
                key: sharedKey,
                encryptionParams: params 
            });
        });
    });
}