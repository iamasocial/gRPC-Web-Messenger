import { 
    InitKeyExchangeRequest, 
    CompleteKeyExchangeRequest, 
    GetKeyExchangeParamsRequest
} from '../../proto/key_exchange_service_pb';
import { keyExchangeClient } from './client';

// Фиксированные значения p и g для всех пользователей
// Используем большие простые числа для безопасности
const DH_P = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF";
const DH_G = "2";

// Получение токена из localStorage
function getToken() {
    return localStorage.getItem("token");
}

/**
 * Инициирует процесс обмена ключами.
 * @param {string} username - Имя собеседника
 * @param {string} publicKey - Публичный ключ A = g^a mod p
 * @param {function} callback - Функция обратного вызова (err, success)
 */
export function initKeyExchange(username, publicKey, callback) {
    const request = new InitKeyExchangeRequest();
    request.setUsername(username);
    request.setDhG(DH_G);
    request.setDhP(DH_P);
    request.setDhAPublic(publicKey);
    
    const token = getToken();
    const metadata = { 'Authorization': `Bearer ${token}` };
    
    keyExchangeClient.initKeyExchange(request, metadata, (err, response) => {
        if (err) {
            console.error('Init key exchange error:', err);
            callback(err, null);
        } else {
            callback(null, response.getSuccess());
        }
    });
}

/**
 * Завершает процесс обмена ключами.
 * @param {string} username - Имя собеседника
 * @param {string} publicKey - Публичный ключ B = g^b mod p
 * @param {function} callback - Функция обратного вызова (err, success)
 */
export function completeKeyExchange(username, publicKey, callback) {
    const request = new CompleteKeyExchangeRequest();
    request.setUsername(username);
    request.setDhBPublic(publicKey);
    
    const token = getToken();
    const metadata = { 'Authorization': `Bearer ${token}` };
    
    keyExchangeClient.completeKeyExchange(request, metadata, (err, response) => {
        if (err) {
            console.error('Complete key exchange error:', err);
            callback(err, null);
        } else {
            callback(null, response.getSuccess());
        }
    });
}

/**
 * Получает параметры обмена ключами для конкретного собеседника.
 * @param {string} username - Имя собеседника
 * @param {function} callback - Функция обратного вызова (err, params)
 */
export function getKeyExchangeParams(username, callback) {
    const request = new GetKeyExchangeParamsRequest();
    request.setUsername(username);
    
    const token = getToken();
    const metadata = { 'Authorization': `Bearer ${token}` };
    
    keyExchangeClient.getKeyExchangeParams(request, metadata, (err, response) => {
        if (err) {
            console.error('Get key exchange params error:', err);
            callback(err, null);
        } else {
            if (response.getSuccess()) {
                const params = {
                    status: response.getStatus(),
                    dhG: response.getDhG(),
                    dhP: response.getDhP(),
                    dhAPublic: response.getDhAPublic(),
                    dhBPublic: response.getDhBPublic()
                };
                callback(null, params);
            } else {
                callback(new Error(response.getErrorMessage()), null);
            }
        }
    });
}

/**
 * Получает фиксированные параметры Диффи-Хеллмана (p и g)
 * @returns {Object} {p, g} - Параметры Диффи-Хеллмана
 */
export function getDiffieHellmanParams() {
    return {
        p: DH_P,
        g: DH_G,
        isHex: true // Флаг, указывающий, что значения в шестнадцатеричном формате
    };
} 