import { setupAuthUI } from './ui/auth_ui.js';
import { setupChatsUI } from './ui/chats_ui.js';

document.addEventListener("DOMContentLoaded", () => {
    if (window.location.pathname.endsWith("index.html")) {
        setupAuthUI();
    } else if (window.location.pathname.endsWith("chats.html")) {
        setupChatsUI();
    }
});