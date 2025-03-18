/**
 * Модуль для шифрования и расшифровки сообщений в чате
 */

import { getEncryptionKey, saveEncryptionKey } from '../api/chat.js';

/**
 * Генерирует криптографически стойкую случайную строку заданной длины
 * @param {number} length - Длина строки
 * @returns {string} - Случайная строка в шестнадцатеричном формате
 */
function generateRandomString(length) {
  const array = new Uint8Array(length);
  window.crypto.getRandomValues(array);
  return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
}

/**
 * Генерирует пару ключей Диффи-Хеллмана
 * @returns {CryptoKeyPair} - Пара ключей (публичный и приватный)
 */
export async function generateDHKeyPair() {
  // Используем WebCrypto API для генерации ключей
  const keyPair = await window.crypto.subtle.generateKey(
    {
      name: "ECDH",
      namedCurve: "P-256", // Используем эллиптическую кривую P-256
    },
    true, // extractable
    ["deriveKey", "deriveBits"] // разрешенные операции
  );
  
  return keyPair;
}

/**
 * Экспортирует публичный ключ в формат для передачи
 * @param {CryptoKey} publicKey - Публичный ключ
 * @returns {string} - Экспортированный ключ в формате base64
 */
export async function exportPublicKey(publicKey) {
  const exported = await window.crypto.subtle.exportKey(
    "spki", // формат для публичных ключей
    publicKey
  );
  
  // Конвертируем в base64 для передачи
  return btoa(String.fromCharCode.apply(null, new Uint8Array(exported)));
}

/**
 * Импортирует публичный ключ из формата base64
 * @param {string} publicKeyBase64 - Публичный ключ в формате base64
 * @returns {CryptoKey} - Импортированный публичный ключ
 */
export async function importPublicKey(publicKeyBase64) {
  const binaryString = atob(publicKeyBase64);
  const len = binaryString.length;
  const bytes = new Uint8Array(len);
  
  for (let i = 0; i < len; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  
  return await window.crypto.subtle.importKey(
    "spki", // формат для публичных ключей
    bytes,
    {
      name: "ECDH",
      namedCurve: "P-256",
    },
    true,
    [] // у импортированного публичного ключа нет разрешенных операций
  );
}

/**
 * Вычисляет общий секретный ключ по протоколу Диффи-Хеллмана
 * @param {CryptoKey} privateKey - Приватный ключ текущего пользователя
 * @param {CryptoKey} otherPublicKey - Публичный ключ собеседника
 * @returns {string} - Общий секретный ключ в формате base64
 */
export async function deriveSharedSecret(privateKey, otherPublicKey) {
  // Вычисляем общий секрет
  const sharedSecret = await window.crypto.subtle.deriveBits(
    {
      name: "ECDH",
      public: otherPublicKey,
    },
    privateKey,
    256 // 256 бит
  );
  
  // Преобразуем бинарный секрет в строку base64
  return btoa(String.fromCharCode.apply(null, new Uint8Array(sharedSecret)));
}

/**
 * Шифрует сообщение с использованием ключа собеседника
 * @param {string} message - Сообщение для шифрования
 * @param {string} otherUsername - Имя собеседника
 * @returns {string} - Зашифрованное сообщение в формате base64
 */
export async function encryptMessage(message, otherUsername) {
  const currentUsername = localStorage.getItem('username');
  
  // Получаем ключ шифрования
  const encryptionKey = getEncryptionKey(currentUsername, otherUsername);
  if (!encryptionKey) {
    throw new Error("Ключ шифрования не найден для этого собеседника");
  }
  
  // Создаем вектор инициализации (IV)
  const iv = window.crypto.getRandomValues(new Uint8Array(12));
  
  // Подготавливаем ключ для использования в AES-GCM
  const key = await window.crypto.subtle.importKey(
    "raw",
    stringToArrayBuffer(atob(encryptionKey)),
    {
      name: "AES-GCM",
    },
    false,
    ["encrypt"]
  );
  
  // Шифруем сообщение
  const encodedMessage = new TextEncoder().encode(message);
  const ciphertext = await window.crypto.subtle.encrypt(
    {
      name: "AES-GCM",
      iv: iv
    },
    key,
    encodedMessage
  );
  
  // Объединяем IV и зашифрованный текст
  const result = new Uint8Array(iv.length + ciphertext.byteLength);
  result.set(iv, 0);
  result.set(new Uint8Array(ciphertext), iv.length);
  
  // Возвращаем в формате base64
  return btoa(String.fromCharCode.apply(null, result));
}

/**
 * Расшифровывает сообщение с использованием ключа собеседника
 * @param {string} encryptedMessage - Зашифрованное сообщение в формате base64
 * @param {string} otherUsername - Имя собеседника
 * @returns {string} - Расшифрованное сообщение
 */
export async function decryptMessage(encryptedMessage, otherUsername) {
  const currentUsername = localStorage.getItem('username');
  
  // Получаем ключ шифрования
  const encryptionKey = getEncryptionKey(currentUsername, otherUsername);
  if (!encryptionKey) {
    throw new Error("Ключ шифрования не найден для этого собеседника");
  }
  
  // Декодируем сообщение из base64
  const encryptedData = new Uint8Array(
    atob(encryptedMessage).split('').map(char => char.charCodeAt(0))
  );
  
  // Извлекаем IV (первые 12 байт)
  const iv = encryptedData.slice(0, 12);
  // Извлекаем зашифрованный текст
  const ciphertext = encryptedData.slice(12);
  
  // Подготавливаем ключ для использования в AES-GCM
  const key = await window.crypto.subtle.importKey(
    "raw",
    stringToArrayBuffer(atob(encryptionKey)),
    {
      name: "AES-GCM",
    },
    false,
    ["decrypt"]
  );
  
  // Расшифровываем
  try {
    const decryptedData = await window.crypto.subtle.decrypt(
      {
        name: "AES-GCM",
        iv: iv
      },
      key,
      ciphertext
    );
    
    // Преобразуем расшифрованные данные обратно в текст
    return new TextDecoder().decode(decryptedData);
  } catch (error) {
    throw new Error("Ошибка расшифровки сообщения: " + error.message);
  }
}

/**
 * Вспомогательная функция для преобразования строки в ArrayBuffer
 * @param {string} str - Входная строка
 * @returns {ArrayBuffer} - Результат преобразования
 */
function stringToArrayBuffer(str) {
  const buf = new ArrayBuffer(str.length);
  const bufView = new Uint8Array(buf);
  for (let i = 0, strLen = str.length; i < strLen; i++) {
    bufView[i] = str.charCodeAt(i);
  }
  return buf;
} 