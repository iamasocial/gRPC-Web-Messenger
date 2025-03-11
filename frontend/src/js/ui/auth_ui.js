import { login as apiLogin, register as apiRegister } from "../api/auth";
import '../../styles/style.css';

class AuthManager {
    constructor() {
        this.initElements();
        this.initEventListeners();
    }

    initElements() {
        this.loginForm = document.getElementById('login-form');
        this.registerForm = document.getElementById('register-form');
        this.loginBtn = document.getElementById('login-btn');
        this.registerBtn = document.getElementById('register-btn');
        this.showRegisterLink = document.getElementById('show-register');
        this.showLoginLink = document.getElementById('show-login');
    }

    initEventListeners() {
        this.showRegisterLink.addEventListener('click', (e) => this.toggleForms(e, 'register'));
        this.showLoginLink.addEventListener('click', e => this.toggleForms(e, 'login'));
        this.loginBtn.addEventListener('click', () => this.handleLogin());
        this.registerBtn.addEventListener('click', () => this.handleRegister());
        document.querySelectorAll('.input-field').forEach(input => {
            input.addEventListener('keypress', (e) => this.handleEnterPress(event));
        });
    }

    toggleForms(event, formType) {
        event.preventDefault();
        if (formType === 'register') {
            this.loginForm.classList.add('hidden');
            this.registerForm.classList.remove('hidden');
        } else {
            this.registerForm.classList.add('hidden');
            this.loginForm.classList.remove('hidden');
        }
    }

    handleLogin() {
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;

        if (!username || !password) {
            alert("Введите имя пользователя и пароль");
            return;
        }

        apiLogin(username, password, (err, token) => {
            if (err) {
                alert("Ошибка входа: " + err.message);
            } else {
                localStorage.setItem('token', token);
                window.location.href = "chats.html";
            }
        });
    }

    handleRegister() {
        const username = document.getElementById('register-username').value;
        const password = document.getElementById('register-password').value;
        const confirmPassword = document.getElementById('register-confirm-password').value;

        if (password !== confirmPassword) {
            alert('Пароли не совпадают');
            return;
        }

        apiRegister(username, password, confirmPassword, (err, token) => {
            if (err) {
                alert('Ошибка регистрации: ' + err.message)
            } else {
                localStorage.setItem('token', token);
                window.location.href = "chats.html";
            }
        })
    }

    handleEnterPress(event) {
        if (event.key === 'Enter') {
            if (this.registerForm.classList.contains('hidden')) {
                this.handleLogin();
            } else {
                this.handleRegister();
            }
        }
    }
}

document.addEventListener('DOMContentLoaded', () => new AuthManager)
// window.login = function () {
//     const username = document.getElementById("login-username").value;
//     const password = document.getElementById("login-password").value;

//     if (!username || !password) {
//         alert("Username and password are required");
//         return;
//     }

//     apiLogin(username, password, (err, token) => {
//         if (err) {
//             alert("Authorization failed");
//         } else {
//             localStorage.setItem("token", token);
//             window.location.href = "chats.html";
//         }
//     });
// };

// window.register = function () {
//     const username = document.getElementById("register-username").value;
//     const password = document.getElementById("register-password").value;
//     const confirmPassword = document.getElementById("register-confirm-password").value;

//     if (!username || !password || !confirmPassword) {
//         alert("Username and passwords are required!");
//         return;
//     }

//     if (password !== confirmPassword) {
//         alert("Passwords must match!");
//         return;
//     }

//     apiRegister(username, password, confirmPassword, (err, token) => {
//         if (err) {
//             alert("Registration failed");
//         } else {
//             localStorage.setItem("token", token);
//             window.location.href = "chats.html";
//         }
//     });
// };