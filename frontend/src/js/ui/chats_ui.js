import { getChats, connectToChat, startChat, chat, stopChat, createChat } from "../api/chat";

let currentChat = null;
let lastDisplayedDate = null;

function loadChats() {
    const chatList = document.getElementById('chat-list');
    chatList.innerHTML = '';

    getChats((err, chats) => {
        if (err) {
            console.error("Loading chats error:", err);
            alert("You are not authenticated");
            window.location.href = 'index.html';
            return;
        }

        const urlParams = new URLSearchParams(window.location.search);
        const chatParam = urlParams.get('chat');

        chats.forEach(username => {
            const chatElement = document.createElement('div');
            chatElement.classList.add('chat-item');
            chatElement.textContent = username;
            chatElement.dataset.username = username;

            if (username === chatParam) {
                chatElement.classList.add('active');
            }

            chatList.appendChild(chatElement);
        });

        if (chatParam && chats.includes(chatParam)) {
            connectToChatHandler(chatParam);
        }
    });
}

function handleIncomingMessage(message) {
    const chatMessages = document.getElementById("chat-messages");

    const timestamp = message.timestamp || Math.floor(Date.now() / 1000);
    const messageDate = new Date(timestamp * 1000);
    const currentDate = formatMessageDate(messageDate);

    if (currentDate !== lastDisplayedDate) {
        const dateDivider = createDateDivider(currentDate);
        chatMessages.appendChild(dateDivider);
        lastDisplayedDate = currentDate;
    }

    const messageElement = document.createElement("div");
    messageElement.classList.add("message");

    const urlParams = new URLSearchParams(window.location.search);
    const currentCompanion = urlParams.get('chat');

    const escapeHtml = (str) => str.replace(/[&<>"']/g, tag => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    })[tag] || tag);

    const isFromCompanion = message.senderUsername === currentCompanion;
    messageElement.classList.add(isFromCompanion ? "received" : "sent");

    const formattedTime = messageDate.toLocaleString('ru-RU', {
        hour: '2-digit',
        minute: '2-digit'
    })

    messageElement.innerHTML = `
    <div class="message-content">
        <div class="message-text">${escapeHtml(message.content)}</div>
        <div class="message-time">${formattedTime}</div>
    </div>
    `

    chatMessages.appendChild(messageElement);

    chatMessages.scrollTo({
        top: chatMessages.scrollHeight,
        behavior: "smooth"
    });
}

function formatMessageDate(date) {
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    if (date.toDateString() === today.toDateString()) {
        return 'Сегодня';
    } else if (date.toDateString() === yesterday.toDateString()) {
        return 'Вчера';
    } else {
        return date.toLocaleString('ru-RU', {
            day: 'numeric',
            month: 'long',
            year: 'numeric'
        });
    }
}

function createDateDivider(dateText) {
    const divider = document.createElement('div');
    divider.className = 'date-divider';
    divider.innerHTML = `
        <span class="date-divider-text">${dateText}</span>
    `;
    return divider;
}

function connectToChatHandler(receiverUsername) {
    stopChat();

    const chatMessages = document.getElementById("chat-messages");
    chatMessages.innerHTML = '';
    lastDisplayedDate = null;

    connectToChat(receiverUsername, (err, success) => {
        if (err || !success) {
            alert("Не удалось подключиться к чату");
            resetChatView();
            return;
        }

        currentChat = receiverUsername;
        document.getElementById("chat-title").textContent = `Чат с ${receiverUsername}`;

        window.history.pushState(
            { chat: receiverUsername },
            '',
            `?chat=${encodeURIComponent(receiverUsername)}`
        );

        document.querySelectorAll('.chat-item').forEach(item => {
            item.classList.toggle('active', item.dataset.username === receiverUsername);
        });

        const footer = document.querySelector('.chat-footer');
        footer.classList.add('active');

        document.getElementById('message-input').focus();

        startChat(handleIncomingMessage);
    });
}

function onChatClick(event) {
    const chatItem = event.target.closest('.chat-item');
    if (!chatItem) return;

    const receiverUsername = chatItem.dataset.username;
    if (!receiverUsername) return;

    connectToChatHandler(receiverUsername);
}

function onSendMessage() {
    const messageInput = document.getElementById("message-input");
    const message = messageInput.value.trim();

    if (!message) {
        return;
    }

    chat(message);
    messageInput.value = "";
}

function resetChatView() {
    const footer = document.querySelector('.chat-footer');
    footer.classList.remove('active');
    document.getElementById('chat-title').textContent = 'Выберите чат';
    document.getElementById('message-input').value = '';

    document.getElementById('chat-messages').innerHTML = '';
    lastDisplayedDate = null;
}

function handleCreateChat() {
    const username = document.getElementById('new-chat-username').value.trim();

    if (!username) {
        showError('Введите имя пользователя');
        return;
    }

    createChat(username, (err, createdUsername) => {
        if (err) {
            showError(`Ошибка создания чата: ${err.message}`);
            return;
        }

        closeModal();
        loadChats();
        showSuccess(`Чат с ${createdUsername} создан`);
    });
}

function initChatCreationModal() {
    const modal = document.getElementById('create-chat-modal');
    const newChatBtn = document.getElementById('new-chat-btn');

    newChatBtn.addEventListener('click', () => {
        resetModal();
        modal.style.display = 'block';
    });

    // initAlgorithmButtons();
    // initModesButtons();
    // initPaddingButtons();
    initOptionButtons()

    document.getElementById('create-chat-submit').addEventListener('click', handleCreateChat);
}

function initOptionButtons() {
    initOptionGroup('algorithm-select'); // Для алгоритмов
    initOptionGroup('mode-select');     // Для режимов
    initOptionGroup('padding-select');  // Для набивки
}

function initAlgorithmButtons() {
    initOptionGroup('algorithm-select');
}

function initModesButtons() {
    initOptionGroup('mode-select');
}

function initPaddingButtons() {
    initOptionGroup('padding-select');
}

function initOptionGroup(groupId) {
    const group = document.getElementById(groupId);

    group.querySelectorAll(".option-btn").forEach(btn => {
        btn.addEventListener('click', function() {
            // Снимаем выделение со всех кнопок в группе
            group.querySelectorAll('.option-btn').forEach(b => b.classList.remove('active'));
            
            // Добавляем выделение текущей кнопке
            this.classList.add('active'); // <-- Добавлена эта строка
            
            // Можно добавить анимацию
            this.style.transform = "scale(0.95)";
            setTimeout(() => {
                this.style.transform = "scale(1)";
            }, 100);
        });
    });
}

function showError(message) {
    alert('Ошибка: ' + message)
}

function showSuccess(message) {
    alert("Успех: " + message)
}



function resetModal() {
    document.getElementById('new-chat-username').value = '';
    document.querySelectorAll('.option-btn.active').forEach(btn => {
        btn.classList.remove('active');
    });
}

function closeModal() {
    document.getElementById('create-chat-modal').style.display = 'none';
}

window.addEventListener('popstate', (event) => {
    const urlParams = new URLSearchParams(window.location.search);
    const chatParam = urlParams.get('chat');

    if (chatParam && chatParam !== currentChat) {
        connectToChatHandler(chatParam);
    } else if (!chatParam) {
        stopChat();
        resetChatView();
        document.getElementById("chat-title").textContent = "Выберите чат";
        currentChat = null;
    }
});

document.addEventListener("DOMContentLoaded", () => {
    document.getElementById("chat-list").addEventListener('click', onChatClick);
    document.getElementById("send-button").addEventListener('click', onSendMessage);
    window.addEventListener("beforeunload", stopChat);

    initChatCreationModal();
    document.querySelectorAll('.close, #create-chat-cancel').forEach(btn => {
        btn.addEventListener('click', closeModal)
    })

    loadChats();
})

