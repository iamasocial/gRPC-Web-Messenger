import { keyExchangeAPI } from './key_exchange';
import CryptoJS from 'crypto-js';

class EncryptionService {
    constructor() {
        this.sharedSecret = null;
        this.isInitialized = false;
    }

    async initialize() {
        try {
            console.log('Инициализация шифрования...');
            
            // Инициируем обмен ключами
            const { clientPublic, sessionId } = await keyExchangeAPI.initiateKeyExchange();
            console.log('Получены параметры для обмена ключами');

            // Завершаем обмен ключами
            const { success, sharedSecret } = await keyExchangeAPI.completeKeyExchange(clientPublic);
            if (!success) {
                throw new Error('Не удалось завершить обмен ключами');
            }

            // Сохраняем общий секрет
            this.sharedSecret = sharedSecret;
            this.isInitialized = true;
            console.log('Шифрование успешно инициализировано');
        } catch (error) {
            console.error('Ошибка при инициализации шифрования:', error);
            throw error;
        }
    }

    encrypt(message) {
        if (!this.isInitialized) {
            throw new Error('Шифрование не инициализировано');
        }

        // Преобразуем общий секрет в формат для CryptoJS
        const secretKey = CryptoJS.enc.Hex.parse(Array.from(this.sharedSecret)
            .map(b => b.toString(16).padStart(2, '0'))
            .join(''));
        
        console.log('Шифрование сообщения:', {
            исходноеСообщение: message,
            длинаКлюча: secretKey.words.length
        });

        // Генерируем случайный IV
        const iv = CryptoJS.lib.WordArray.random(16);
        
        // Шифруем сообщение
        const encrypted = CryptoJS.AES.encrypt(message, secretKey, {
            mode: CryptoJS.mode.CBC,
            padding: CryptoJS.pad.Pkcs7,
            iv: iv
        });

        // Объединяем IV и зашифрованное сообщение
        const result = {
            iv: iv.toString(),
            data: encrypted.toString()
        };

        console.log('Сообщение зашифровано');
        return JSON.stringify(result);
    }

    decrypt(encryptedMessage) {
        if (!this.isInitialized) {
            throw new Error('Шифрование не инициализировано');
        }

        // Преобразуем общий секрет в формат для CryptoJS
        const secretKey = CryptoJS.enc.Hex.parse(Array.from(this.sharedSecret)
            .map(b => b.toString(16).padStart(2, '0'))
            .join(''));
        
        console.log('Дешифрование сообщения:', {
            длинаКлюча: secretKey.words.length
        });

        try {
            // Парсим JSON с IV и данными
            const { iv, data } = JSON.parse(encryptedMessage);
            
            // Дешифруем сообщение
            const decrypted = CryptoJS.AES.decrypt(data, secretKey, {
                mode: CryptoJS.mode.CBC,
                padding: CryptoJS.pad.Pkcs7,
                iv: CryptoJS.enc.Hex.parse(iv)
            });

            const result = decrypted.toString(CryptoJS.enc.Utf8);
            console.log('Сообщение расшифровано:', result);
            return result;
        } catch (error) {
            console.error('Ошибка при расшифровке:', error);
            throw error;
        }
    }

    isReady() {
        return this.isInitialized;
    }
}

export const encryptionService = new EncryptionService(); 