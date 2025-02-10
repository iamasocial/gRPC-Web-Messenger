import * as grpc from 'grpc-web';
import { LoginRequest, RegisterRequest }  from './proto/user_service_pb';
import { UserServiceClient  } from "./proto/user_service_grpc_web_pb";

const client = new UserServiceClient('http://localhost:8888');

export function login() {
    const username = document.getElementById("login-username").value;
    const password = document.getElementById("login-password").value;

    if (!username || !password) {
        alert("Username and password are required");
        return;
    }

    const request = new LoginRequest();
    request.setUsername(username);
    request.setPassword(password);

    client.login(request, {}, (err, response) => {
        if (err) {
            console.error('Login error:', err);
            alert('Error during login');
        } else {
            console.log('Login succes:', response.getToken());
            alert('Logged in successfully. Token: ', response.getToken());
            localStorage.setItem('token', response.getToken());
        }
    });
}

export function register() {
    const username = document.getElementById("register-username").value;
    const password = document.getElementById("register-password").value;
    const confirmPassword = document.getElementById("register-confirm-password").value;

    if (password !== confirmPassword) {
        alert("Пароли должны совпадать!")
        return
    }

    const request = new RegisterRequest();
    request.setUsername(username);
    request.setPassword(username);
    request.setConfirmpassword(confirmPassword);
    console.log(password, confirmPassword);

    client.register(request, {}, (err, response) => {
        if (err) {
            console.error("Registration error:", err);
            alert("Error during registration")
        } else {
            console.log("Registration successful", response);
            alert("Registered successfully");
            localStorage.setItem('token', response.getToken())
        }
    })
}

