import { LoginRequest, RegisterRequest }  from '../../proto/user_service_pb';
import { getUserClient } from './client';

export function login(username, password, callback) {
    const client = getUserClient();
    const request = new LoginRequest();
    request.setUsername(username);
    request.setPassword(password);

    client.login(request, {}, (err, response) => {
        if (err) {
            console.error('Login error:', err);
            callback(err, null);
        } else {
            const token = response.getToken();
            if (!token) {
                callback(new Error('Токен не получен от сервера'), null);
                return;
            }
            console.log('Получен токен:', token);
            localStorage.setItem('token', token);
            localStorage.setItem('username', username);
            callback(null, token);
        }
    });
}

export function register(username, password, confirmPassword, callback) {
    const client = getUserClient();
    const request = new RegisterRequest();
    request.setUsername(username);
    request.setPassword(password);
    request.setConfirmpassword(confirmPassword);
    console.log(password, confirmPassword);

    client.register(request, {}, (err, response) => {
        if (err) {
            console.error("Registration error:", err);
            callback(err, null);
        } else {
            const token = response.getToken();
            if (!token) {
                callback(new Error('Токен не получен от сервера'), null);
                return;
            }
            console.log('Получен токен:', token);
            localStorage.setItem('token', token);
            localStorage.setItem('username', username);
            callback(null, token);
        }
    });
}

