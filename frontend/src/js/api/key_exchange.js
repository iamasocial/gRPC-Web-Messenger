import { KeyExchangeServiceClient } from '../../proto/key_exchange_service_grpc_web_pb';
import { InitiateKeyExchangeRequest, CompleteKeyExchangeRequest } from '../../proto/key_exchange_service_pb';
import { client } from './client';
import BN from 'bn.js';

class KeyExchangeAPI {
    constructor() {
        this.client = new KeyExchangeServiceClient('http://localhost:8888', null);
        this.prime = null;
        this.generator = null;
        this.serverPublic = null;
        this.sessionId = null;
        this.privateKey = null;
    }

    initiateKeyExchange() {
        return new Promise((resolve, reject) => {
            const request = new InitiateKeyExchangeRequest();
            console.log('Начинаем обмен ключами...');
            this.client.initiateKeyExchange(request, {
                authorization: `Bearer ${localStorage.getItem('token')}`
            }, (err, response) => {
                if (err) {
                    console.error('Ошибка при инициировании обмена ключами:', err);
                    reject(err);
                    return;
                }

                // Сохраняем параметры
                this.prime = new BN(response.getPrime());
                this.generator = new BN(response.getGenerator());
                this.serverPublic = new BN(response.getServerPublic());
                this.sessionId = response.getSessionId();

                console.log('Получены параметры от сервера:', {
                    prime: this.prime.toString('hex'),
                    generator: this.generator.toString('hex'),
                    serverPublic: this.serverPublic.toString('hex'),
                    sessionId: this.sessionId
                });

                // Генерируем приватный ключ
                this.privateKey = this.generatePrivateKey();
                console.log('Сгенерирован приватный ключ:', this.privateKey.toString('hex'));

                // Вычисляем публичный ключ клиента
                const clientPublic = this.calculateClientPublic();
                console.log('Вычислен публичный ключ клиента:', clientPublic.toString('hex'));

                resolve({
                    clientPublic: new Uint8Array(clientPublic.toArray('be')),
                    sessionId: this.sessionId
                });
            });
        });
    }

    completeKeyExchange(clientPublic) {
        return new Promise((resolve, reject) => {
            const request = new CompleteKeyExchangeRequest();
            request.setClientPublic(new Uint8Array(clientPublic));
            request.setSessionId(this.sessionId);

            console.log('Отправляем публичный ключ клиента на сервер...');
            this.client.completeKeyExchange(request, {
                authorization: `Bearer ${localStorage.getItem('token')}`
            }, (err, response) => {
                if (err) {
                    console.error('Ошибка при завершении обмена ключами:', err);
                    reject(err);
                    return;
                }

                if (!response.getSuccess()) {
                    reject(new Error('Обмен ключами не удался'));
                    return;
                }

                // Вычисляем общий секрет
                const sharedSecret = this.calculateSharedSecret();
                console.log('Вычислен общий секрет:', sharedSecret.toString('hex'));

                resolve({
                    success: true,
                    sharedSecret: new Uint8Array(sharedSecret.toArray('be'))
                });
            });
        });
    }

    generatePrivateKey() {
        // Генерируем случайное число меньше prime
        const max = this.prime.sub(new BN(1));
        let privateKey;
        do {
            // Создаем массив случайных байтов
            const bytes = new Uint8Array(Math.ceil(this.prime.bitLength() / 8));
            crypto.getRandomValues(bytes);
            privateKey = new BN(bytes);
        } while (privateKey.isZero() || privateKey.gte(max));
        return privateKey;
    }

    calculateClientPublic() {
        // Вычисляем g^b mod p
        const result = this.generator.toRed(BN.mont(this.prime)).redPow(this.privateKey).fromRed();
        console.log('Вычисление публичного ключа клиента:', {
            generator: this.generator.toString('hex'),
            privateKey: this.privateKey.toString('hex'),
            prime: this.prime.toString('hex'),
            result: result.toString('hex')
        });
        return result;
    }

    calculateSharedSecret() {
        // Вычисляем A^b mod p
        const result = this.serverPublic.toRed(BN.mont(this.prime)).redPow(this.privateKey).fromRed();
        console.log('Вычисление общего секрета:', {
            serverPublic: this.serverPublic.toString('hex'),
            privateKey: this.privateKey.toString('hex'),
            prime: this.prime.toString('hex'),
            result: result.toString('hex')
        });
        return result;
    }
}

export const keyExchangeAPI = new KeyExchangeAPI(); 