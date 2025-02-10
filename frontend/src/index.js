import { login, register } from './client';
import './styles/style.css';

function toggleForms() {
    document.getElementById("login-form").classList.toggle("hidden")
    document.getElementById("register-form").classList.toggle("hidden")
}

// function loginOnClick() {
//     const username = document.getElementById("login-username").value;
//     const password = document.getElementById("login-password").value;

//     console.log("Вход:", { username, password });
//     alert("Попытка входа...");
//     // loginUser(username, password);
// }

// function register() {
//     const username = document.getElementById("register-username").value;
//     const password = document.getElementById("register-password").value;

//     console.log("Регистрация:", { username, password });
//     alert("Попытка регистрации...");
// }

window.login = login;
window.register = register;
window.toggleForms = toggleForms;