import { LoginRequest, RegisterRequest }  from '../../proto/user_service_pb';
import { userClient } from './client';

export function login(username, password, callback) {

    const request = new LoginRequest();
    request.setUsername(username);
    request.setPassword(password);

    userClient.login(request, {}, (err, response) => {
        if (err) {
            console.error('Login error:', err);
            callback(err, null);
        } else {
            callback(null, response.getToken());
        }
    });
}

export function register(username, password, confirmPassword, callback) {
    const request = new RegisterRequest();
    request.setUsername(username);
    request.setPassword(password);
    request.setConfirmpassword(confirmPassword);
    console.log(password, confirmPassword);

    userClient.register(request, {}, (err, response) => {
        if (err) {
            console.error("Registration error:", err);
            callback(err, null);
        } else {
            callback(null, response.getToken());
        }
    })
}

