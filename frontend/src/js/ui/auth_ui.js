import { login as apiLogin, register as apiRegister } from "../api/auth";
import '../../styles/style.css';

window.login = function () {
    const username = document.getElementById("login-username").value;
    const password = document.getElementById("login-password").value;

    if (!username || !password) {
        alert("Username and password are required");
        return;
    }

    apiLogin(username, password, (err, token) => {
        if (err) {
            alert("Authorization failed");
        } else {
            localStorage.setItem("token", token);
            window.location.href = "chats.html";
        }
    });
};

window.register = function () {
    const username = document.getElementById("register-username").value;
    const password = document.getElementById("register-password").value;
    const confirmPassword = document.getElementById("register-confirm-password").value;

    if (!username || !password || !confirmPassword) {
        alert("Username and passwords are required!");
        return;
    }

    if (password !== confirmPassword) {
        alert("Passwords must match!");
        return;
    }

    apiRegister(username, password, confirmPassword, (err, token) => {
        if (err) {
            alert("Registration failed");
        } else {
            localStorage.setItem("token", token);
            window.location.href = "chats.html";
        }
    });
};