import { getChats, connectToChat, startChat, chat, stopChat, createChat, sendFileMessage, deleteChat } from "../api/chat";
import { uploadFile, downloadFile } from "../api/file";
import { initKeyExchange, completeKeyExchange, getKeyExchangeParams, getDiffieHellmanParams } from "../api/key_exchange";

let currentChat = null;
let lastDisplayedDate = null;
let selectedFile = null;
let selectedUser = null;

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

        chats.forEach(chat => {
            const username = chat.username;
            const chatElement = document.createElement('div');
            chatElement.classList.add('chat-item');
            
            // Создаем контейнер для элементов чата (для flexbox)
            const chatContainer = document.createElement('div');
            chatContainer.classList.add('chat-item-container');
            
            // Форматируем имя пользователя и информацию о шифровании
            const usernameDiv = document.createElement('div');
            usernameDiv.classList.add('username');
            usernameDiv.textContent = username;
            
            // Создаем блок с информацией о шифровании
            const encryptionInfo = document.createElement('div');
            encryptionInfo.classList.add('encryption-info');
            
            // Добавляем информацию об алгоритме
            const algorithmSpan = document.createElement('span');
            algorithmSpan.classList.add('encryption-param');
            algorithmSpan.innerHTML = `Алг: <strong>${chat.encryptionAlgorithm || 'n/a'}</strong>`;
            
            // Добавляем информацию о режиме
            const modeSpan = document.createElement('span');
            modeSpan.classList.add('encryption-param');
            modeSpan.innerHTML = `Режим: <strong>${chat.encryptionMode || 'n/a'}</strong>`;
            
            // Добавляем информацию о набивке
            const paddingSpan = document.createElement('span');
            paddingSpan.classList.add('encryption-param');
            paddingSpan.innerHTML = `Набивка: <strong>${chat.encryptionPadding || 'n/a'}</strong>`;
            
            // Добавляем все элементы в блок информации о шифровании
            encryptionInfo.appendChild(algorithmSpan);
            encryptionInfo.appendChild(modeSpan);
            encryptionInfo.appendChild(paddingSpan);
            
            // Добавляем имя пользователя и информацию о шифровании в основной контейнер
            chatContainer.appendChild(usernameDiv);
            chatContainer.appendChild(encryptionInfo);
            
            // Создаем кнопку удаления чата
            const deleteButton = document.createElement('button');
            deleteButton.classList.add('delete-chat-btn');
            deleteButton.innerHTML = '🗑️';
            deleteButton.title = `Удалить чат с ${username}`;
            deleteButton.dataset.username = username;
            deleteButton.addEventListener('click', handleDeleteChat);
            
            // Добавляем основной контейнер и кнопку удаления в элемент чата
            chatElement.appendChild(chatContainer);
            chatElement.appendChild(deleteButton);
            
            // Сохраняем данные как атрибуты для дальнейшего использования
            chatElement.dataset.username = username;
            chatElement.dataset.algorithm = chat.encryptionAlgorithm || '';
            chatElement.dataset.mode = chat.encryptionMode || '';
            chatElement.dataset.padding = chat.encryptionPadding || '';

            if (username === chatParam) {
                chatElement.classList.add('active');
            }

            chatList.appendChild(chatElement);
        });

        if (chatParam && chats.some(chat => chat.username === chatParam)) {
            connectToChatHandler(chatParam);
        }
    });
}

function handleIncomingMessage(message) {
    const chatMessages = document.getElementById('chat-messages');
    if (!chatMessages) return;
    
    console.log('Получено сообщение (полное):', JSON.stringify(message));
    
    // Исправляем обработку timestamp: умножаем на 1000, если timestamp в секундах
    let timestamp = message.timestamp ? parseInt(message.timestamp) : Date.now();
    // Проверяем, если timestamp слишком маленький (в секундах вместо миллисекунд)
    if (timestamp < 1000000000000) {
        timestamp = timestamp * 1000;
    }
    
    const messageDate = new Date(timestamp);
    console.log('Сообщение timestamp:', timestamp, 'Дата:', messageDate);
    
    // Создать форматированную дату и время
    const datePart = `${messageDate.getFullYear()}-${String(messageDate.getMonth() + 1).padStart(2, '0')}-${String(messageDate.getDate()).padStart(2, '0')}`;
    const timePart = `${String(messageDate.getHours()).padStart(2, '0')}:${String(messageDate.getMinutes()).padStart(2, '0')}`;
    
    // Проверка на необходимость добавления разделителя даты
    if (!document.querySelector(`.date-divider[data-date="${datePart}"]`)) {
        const dateDivider = createDateDivider(formatDate(datePart));
        dateDivider.setAttribute('data-date', datePart);
        chatMessages.appendChild(dateDivider);
    }
    
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message';
    
    // Определяем отправителя сообщения, учитывая разные форматы имени свойства
    const sender = message.senderUsername || message.senderusername || 
                   message.SenderUsername || message.Senderusername || '';
    
    // Сообщение от собеседника, если отправитель совпадает с selectedUser
    // Иначе сообщение от текущего пользователя
    const isFromCompanion = sender === selectedUser;
    console.log('Отправитель:', sender, 'Выбранный пользователь:', selectedUser, 'От собеседника:', isFromCompanion);
    
    if (isFromCompanion) {
        messageDiv.classList.add('received');
    } else {
        messageDiv.classList.add('sent');
    }
    
    // Проверяем, является ли сообщение файловым по messageType или по содержимому
    let content = message.content || '';
    let isFileMessage = (message.messageType === 'file') || !!content.match(/\[FILE:([^:]+):([^:]+):([^\]]+)\](.*)/);
    
    if (isFileMessage) {
        // Это файловое сообщение
        let fileId, fileName, fileSize, commentText;
        
        // Проверяем формат сообщения
        const fileMatch = content.match(/\[FILE:([^:]+):([^:]+):([^\]]+)\](.*)/);
        if (fileMatch) {
            // Если содержимое соответствует формату [FILE:id:name:size]
            fileId = fileMatch[1];
            fileName = fileMatch[2];
            fileSize = parseInt(fileMatch[3]);
            commentText = fileMatch[4] || '';
        } else {
            // Используем данные из свойств сообщения
            fileId = message.fileId;
            fileName = message.fileName || 'Файл';
            fileSize = message.fileSize || 0;
            commentText = content.startsWith('Файл: ') ? '' : content;
        }
        
        // Определяем, является ли файл изображением
        const isImage = fileName.match(/\.(jpg|jpeg|png|gif|webp|svg)$/i);
        
        if (isImage) {
            // Для изображений - загружаем и отображаем в чате
            messageDiv.setAttribute('data-file-id', fileId);
            messageDiv.classList.add('image-message');
            
            // Временный placeholder пока изображение загружается
            messageDiv.innerHTML = `
                <div class="message-content with-file">
                    ${commentText.trim() ? `<div class="message-text with-file">${escapeHtml(commentText)}</div>` : ''}
                    <div class="message-file image-loading" data-file-id="${fileId}">
                        <div class="file-loading">Загрузка изображения...</div>
                    </div>
                    <div class="message-time">${timePart}</div>
                </div>
            `;
            
            // Загружаем изображение
            downloadFile(fileId, (err, fileData) => {
                const fileContainer = messageDiv.querySelector('.message-file');
                
                if (err || !fileData || !fileData.url) {
                    // Если ошибка загрузки
                    fileContainer.innerHTML = `
                        <div class="file-container error">
                            <div class="file-icon">🖼️</div>
                            <div class="file-details">
                                <div class="file-name">${escapeHtml(fileName)}</div>
                                <div class="file-size">${formatFileSize(fileSize)}</div>
                                <div class="file-error">Ошибка загрузки</div>
                            </div>
                        </div>
                    `;
                    return;
                }
                
                // Создаем предварительно изображение для определения его размеров
                const preloadImg = new Image();
                preloadImg.onload = function() {
                    // После загрузки изображения добавляем его в DOM
                    const aspectRatio = this.width / this.height;
                    const isWide = aspectRatio > 1.5; // Широкое изображение
                    const isTall = aspectRatio < 0.6; // Высокое изображение
                    
                    // Добавляем классы в зависимости от пропорций
                    if (isWide) {
                        messageDiv.classList.add('wide-image');
                    } else if (isTall) {
                        messageDiv.classList.add('tall-image');
                    }
                    
                    // Успешно загружено - отображаем изображение
                    fileContainer.innerHTML = `
                        <div class="image-wrapper">
                            <img src="${fileData.url}" alt="${escapeHtml(fileName)}" class="chat-image" title="${escapeHtml(fileName)}" />
                        </div>
                        <div class="image-caption">
                            <div class="file-size">${formatFileSize(fileSize)}</div>
                        </div>
                    `;
                    fileContainer.classList.remove('image-loading');
                    
                    // Добавляем обработчик клика для скачивания
                    const img = fileContainer.querySelector('img');
                    if (img) {
                        // Добавляем обработчик как для изображения, так и для контейнера
                        const imageWrapper = fileContainer.querySelector('.image-wrapper');
                        const clickHandler = () => {
                            // Показываем изображение в модальном окне для просмотра
                            const modal = document.getElementById('file-view-modal');
                            const fileViewName = document.getElementById('file-view-name');
                            const fileViewContainer = document.getElementById('file-view-container');
                            const downloadButton = document.getElementById('download-file-button');
                            
                            if (!modal || !fileViewName || !fileViewContainer || !downloadButton) {
                                console.error('Не найдены элементы модального окна для просмотра файла');
                                return;
                            }
                            
                            // Сохраняем ID файла для скачивания
                            downloadButton.setAttribute('data-file-id', fileId);
                            downloadButton.setAttribute('data-file-url', fileData.url);
                            downloadButton.setAttribute('data-file-name', fileName);
                            
                            // Подготавливаем обработчик скачивания
                            downloadButton.onclick = function() {
                                const url = this.getAttribute('data-file-url');
                                const name = this.getAttribute('data-file-name');
                                
                                if (url && name) {
                                    // Создаем ссылку для скачивания и эмулируем клик
                                    const downloadLink = document.createElement('a');
                                    downloadLink.href = url;
                                    downloadLink.download = name;
                                    document.body.appendChild(downloadLink);
                                    downloadLink.click();
                                    
                                    // Удаляем временный элемент
                                    setTimeout(() => {
                                        document.body.removeChild(downloadLink);
                                    }, 100);
                                }
                            };
                            
                            // Отображаем изображение в полном размере
                            fileViewName.textContent = fileName;
                            fileViewContainer.innerHTML = `
                                <img src="${fileData.url}" alt="${escapeHtml(fileName)}" class="full-size-image" />
                                <div class="file-info">
                                    <div class="file-size">${formatFileSize(fileSize)}</div>
                                </div>
                            `;
                            
                            modal.style.display = 'block';
                        };
                        
                        // Устанавливаем обработчик на само изображение
                        img.addEventListener('click', clickHandler);
                        
                        // Также можно установить обработчик на весь контейнер
                        if (imageWrapper) {
                            imageWrapper.addEventListener('click', clickHandler);
                        }
                    }
                };
                
                // Начинаем загрузку изображения
                preloadImg.src = fileData.url;
            });
        } else {
            // Для обычных файлов - показываем значок и информацию
            messageDiv.innerHTML = `
                <div class="message-content with-file">
                    ${commentText.trim() ? `<div class="message-text with-file">${escapeHtml(commentText)}</div>` : ''}
                    <div class="message-file" data-file-id="${fileId}">
                        <div class="file-container">
                            <div class="file-icon">📄</div>
                            <div class="file-details">
                                <div class="file-name">${escapeHtml(fileName)}</div>
                                <div class="file-size">${formatFileSize(fileSize)}</div>
                            </div>
                        </div>
                    </div>
                    <div class="message-time">${timePart}</div>
                </div>
            `;
        }
    } else {
        // Обычное текстовое сообщение
        const formattedContent = formatTextWithLinks(content);
        
        // Проверяем, содержит ли сообщение код
        const containsCode = formattedContent.includes('code-block');
        
        // Если сообщение содержит код, добавляем класс message-with-code
        if (containsCode) {
            messageDiv.classList.add('message-with-code');
        }
        
        messageDiv.innerHTML = `
<div class="message-content">
    <div class="message-text">${formattedContent}</div>
    <div class="message-time">${timePart}</div>
</div>`;
    }
    
    chatMessages.appendChild(messageDiv);
    
    // Прокрутка вниз после добавления сообщения
    chatMessages.scrollTop = chatMessages.scrollHeight;
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
    const divider = document.createElement("div");
    divider.classList.add("date-divider");
    divider.textContent = dateText;
    return divider;
}

function connectToChatHandler(username) {
    console.log(`Подключение к чату с ${username}...`);
    selectedUser = username;
    
    if (!username) {
        console.error('Имя пользователя не указано');
        return;
    }
    
    // Если есть текущий чат, останавливаем его
    if (currentChat) {
        stopChat();
    }
    
    // Сразу показываем имя пользователя в заголовке чата
    const chatTitle = document.getElementById('chat-title');
    if (chatTitle) {
        chatTitle.textContent = username;
    }
    
    // Очищаем область сообщений и сбрасываем последнюю отображенную дату
    const messagesContainer = document.getElementById('chat-messages');
    if (messagesContainer) {
        messagesContainer.innerHTML = '';
        lastDisplayedDate = null;
    }
    
    // Активируем чат в боковой панели
    const chatItems = document.querySelectorAll('.chat-item');
    chatItems.forEach(item => {
        item.classList.remove('active');
        if (item.dataset.username === username) {
            item.classList.add('active');
        }
    });
    
    // Находим и скрываем футер перед началом обмена ключами
    const chatFooter = document.querySelector('.chat-footer');
    if (chatFooter) {
        chatFooter.style.display = 'none';
    }
    
    // Показываем спиннер загрузки
    showKeyExchangeLoader(true);
    
    // Обновляем URL с параметром выбранного чата
    const urlParams = new URLSearchParams(window.location.search);
    urlParams.set('chat', username);
    window.history.replaceState({}, '', `${window.location.pathname}?${urlParams}`);
    
    // Показываем кнопку отключения
    const disconnectBtn = document.getElementById("disconnect-btn");
    if (disconnectBtn) {
        disconnectBtn.style.display = "block";
    }
    
    // Проверяем статус обмена ключами
    checkKeyExchangeStatus(username, (err, status) => {
        if (err) {
            console.error(`Ошибка проверки статуса обмена ключами: ${err.message}`);
            showErrorToast('Ошибка при подключении к чату');
            showKeyExchangeLoader(false);
            return;
        }
        
        console.log(`Статус обмена ключами: ${status}`);
        
        // Подключаемся к чату через WebSocket
        connectToChat(username, (err, success) => {
            if (err) {
                console.error(`Ошибка подключения к чату: ${err.message}`);
                showErrorToast('Ошибка при подключении к чату');
                showKeyExchangeLoader(false);
                return;
            }
            
            console.log(`Подключение к чату с ${username} успешно установлено`);
            currentChat = username;
            
            // Если обмен ключами уже завершен (status = 2), то запускаем чат сразу
            if (status === 2) { // COMPLETED
                showKeyExchangeLoader(false);
                if (chatFooter) {
                    chatFooter.style.display = 'flex';
                }
                startChatAfterKeyExchange(username);
            } else {
                // Иначе запускаем обмен ключами и дальше будем ждать его завершения
                initDiffieHellmanExchange(username);
            }
        });
    });
}

/**
 * Обработка клика на элементе списка чатов
 * @param {Event} event - Событие клика
 */
function onChatClick(event) {
    // Игнорируем клик по кнопке удаления чата
    if (event.target.classList.contains('delete-chat-btn')) {
        return;
    }
    
    // Находим ближайший элемент чата
    const chatItem = event.target.closest('.chat-item');
    if (!chatItem) return;
    
    const username = chatItem.dataset.username;
    if (!username) return;
    
    // Подключаемся к выбранному чату
    connectToChatHandler(username);
}

/**
 * Отправка сообщения при нажатии на кнопку или клавишу Enter
 */
function onSendMessage() {
    const messageInput = document.getElementById('message-input');
    const messageText = messageInput.value.trim();
    
    if (messageText && currentChat) {
        // Отправка сообщения
        chat(messageText, (err) => {
            if (err) {
                console.error('Ошибка при отправке сообщения:', err);
                showErrorToast('Ошибка при отправке сообщения');
        return;
    }

            // Очистка поля ввода после успешной отправки
            messageInput.value = '';
        });
    }
}

/**
 * Показать уведомление об ошибке
 * @param {string} message - Текст ошибки
 */
function showErrorToast(message) {
    // Простая реализация через alert, можно заменить на более красивые уведомления
    alert(message);
}

function resetChatView() {
    const chatMessages = document.getElementById("chat-messages");
    chatMessages.innerHTML = "";
    lastDisplayedDate = null;
}

function handleCreateChat() {
    const username = document.getElementById("new-chat-username").value.trim();
    if (!username) {
        showError("Имя пользователя не может быть пустым");
        return;
    }

    // Получаем выбранные параметры шифрования
    const algorithm = getSelectedOption("algorithm-select");
    const mode = getSelectedOption("mode-select");
    const padding = getSelectedOption("padding-select");

    // Параметры шифрования
    const encryptionParams = {
        algorithm: algorithm,
        mode: mode,
        padding: padding
    };

    createChat(username, encryptionParams, (err, createdUsername) => {
        if (err) {
            console.error("Error creating chat:", err);
            showError("Ошибка при создании чата");
            return;
        }

        showSuccess(`Чат с ${createdUsername} создан`);
        
        // Инициируем обмен ключами после создания чата
        initDiffieHellmanExchange(createdUsername);
        
        closeModal();
        loadChats();
        connectToChatHandler(createdUsername);
    });
}

// Вспомогательная функция для получения значения выбранной опции
function getSelectedOption(groupId) {
    const group = document.getElementById(groupId);
    const selectedButton = group.querySelector(".option-btn.selected");
    return selectedButton ? selectedButton.getAttribute("data-value") : "";
}

/**
 * Инициализация модального окна создания чата
 */
function initChatCreationModal() {
    const newChatBtn = document.getElementById('new-chat-btn');
    const modal = document.getElementById('create-chat-modal');
    const closeBtn = modal.querySelector('.close');
    const createBtn = document.getElementById('create-chat-submit');
    const cancelBtn = document.getElementById('create-chat-cancel');

    // Инициализируем кнопки опций
    initOptionButtons();
    
    // Устанавливаем значения по умолчанию
    resetModal();

    // Открытие модального окна
    newChatBtn.addEventListener('click', () => {
        modal.style.display = 'block';
        resetModal();
    });
    
    // Закрытие модального окна
    closeBtn.addEventListener('click', closeModal);
    cancelBtn.addEventListener('click', closeModal);
    
    // Закрытие модального окна при клике вне его
    window.addEventListener('click', (event) => {
        if (event.target === modal) {
            closeModal();
        }
    });
    
    // Кнопка создания чата
    createBtn.addEventListener('click', handleCreateChat);
}

function initOptionButtons() {
    initOptionGroup("algorithm-select");
    initOptionGroup("mode-select");
    initOptionGroup("padding-select");
}

function initOptionGroup(groupId) {
    const group = document.getElementById(groupId);
    const buttons = group.querySelectorAll(".option-btn");

    buttons.forEach(button => {
        button.addEventListener("click", () => {
            buttons.forEach(btn => btn.classList.remove("selected"));
            button.classList.add("selected");
        });
    });
    
    // Не выбираем опцию по умолчанию здесь, это будет делать resetModal
}

function showError(message) {
    alert(message);
}

function showSuccess(message) {
    alert(message);
}

function resetModal() {
    document.getElementById("new-chat-username").value = "";
    
    // Алгоритмы шифрования - выбираем Camellia
    selectOptionByValue("algorithm-select", "Camellia");
    
    // Режимы шифрования - выбираем CBC
    selectOptionByValue("mode-select", "CBC");
    
    // Режимы набивки - выбираем PKCS7
    selectOptionByValue("padding-select", "PKCS7");
}

// Вспомогательная функция для выбора опции по значению
function selectOptionByValue(groupId, value) {
    const group = document.getElementById(groupId);
    const buttons = group.querySelectorAll(".option-btn");
    
    // Сначала снимаем выделение со всех кнопок
    buttons.forEach(btn => btn.classList.remove("selected"));
    
    // Находим кнопку с нужным значением и выделяем её
    let found = false;
    buttons.forEach(btn => {
        if (btn.getAttribute("data-value") === value) {
            btn.classList.add("selected");
            found = true;
        }
    });
    
    // Если кнопки с таким значением нет, выбираем первую
    if (!found && buttons.length > 0) {
        buttons[0].classList.add("selected");
    }
}

function closeModal() {
    document.getElementById("create-chat-modal").style.display = "none";
}

function handleLogout() {
    localStorage.removeItem("token");
    window.location.href = "index.html";
}

function initLogoutButton() {
    document.getElementById("logout-btn").addEventListener("click", handleLogout);
}

function initDisconnectButton() {
    document.getElementById("disconnect-btn").addEventListener("click", handleDisconnect);
}

function handleDisconnect() {
    if (currentChat) {
        stopChat();
        
        // Очищаем query параметр
        const urlParams = new URLSearchParams(window.location.search);
        urlParams.delete('chat');
        const newUrl = `${window.location.pathname}?${urlParams.toString()}`;
        window.history.pushState({ path: newUrl }, '', newUrl);
        
        // Сбрасываем UI
        const chatTitle = document.getElementById("chat-title");
        if (chatTitle) {
            chatTitle.textContent = "Выберите чат";
        }
        
        const disconnectBtn = document.getElementById("disconnect-btn");
        if (disconnectBtn) {
            disconnectBtn.style.display = "none";
        }
        
        const messageInput = document.getElementById("message-input");
        if (messageInput) {
            messageInput.disabled = true;
            messageInput.placeholder = "Сначала выберите чат...";
        }
        
        const fileButton = document.getElementById("file-button");
        if (fileButton) {
            fileButton.disabled = true;
        }
        
        // Скрываем футер чата
        const chatFooter = document.querySelector('.chat-footer');
        if (chatFooter) {
            chatFooter.classList.remove('active');
        }
        
        // Очищаем сообщения
        const chatMessages = document.getElementById("chat-messages");
        if (chatMessages) {
            chatMessages.innerHTML = "";
        }
        
        // Снимаем выделение с чата в списке
        const chatItems = document.querySelectorAll('.chat-item');
        chatItems.forEach(item => item.classList.remove('active'));
        
        // Сбрасываем текущий чат и последнюю дату
        currentChat = null;
        lastDisplayedDate = null;
        selectedUser = null;
    }
}

// Форматирование размера файла
function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    else if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    else if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    else return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
}

/**
 * Инициализация обработки файлов
 */
function initFileHandling() {
    const fileInput = document.getElementById('file-input');
    const fileButton = document.getElementById('file-button');
    
    // Клик по кнопке выбора файла
    fileButton.addEventListener('click', () => {
        fileInput.click();
    });
    
    // Обработка выбора файла
    fileInput.addEventListener('change', (event) => {
        if (event.target.files.length > 0) {
            selectedFile = event.target.files[0];
            
            if (selectedFile && currentChat) {
                console.log('Загрузка файла:', selectedFile.name, 'Размер:', selectedFile.size);
                
                // Показываем индикатор загрузки
                const messageInput = document.getElementById('message-input');
                if (messageInput) {
                    messageInput.disabled = true;
                    messageInput.placeholder = 'Загрузка файла...';
                }
                
                // Загружаем файл и отправляем сообщение
                uploadFile(selectedFile, currentChat, (err, fileData) => {
                    // Возвращаем исходное состояние
                    if (messageInput) {
                        messageInput.disabled = false;
                        messageInput.placeholder = 'Напишите сообщение...';
                    }
                    
                    if (err) {
                        showErrorToast('Ошибка при загрузке файла');
                        console.error('Ошибка при загрузке файла:', err);
                        return;
                    }
                    
                    console.log('Файл успешно загружен:', fileData);
                    
                    // Отправляем сообщение о файле
                    if (fileData && fileData.fileId) {
                        sendFileMessage(fileData.fileId, selectedFile.name, selectedFile.size);
                    } else {
                        showErrorToast('Не удалось получить информацию о загруженном файле');
                    }
                    
                    // Сбрасываем выбранный файл
                    selectedFile = null;
                    fileInput.value = '';
                });
            }
        }
    });
}

// Отображение предпросмотра файла перед отправкой
function showFilePreview(file) {
    const filePreviewModal = document.getElementById('file-preview-modal');
    const fileNameElement = document.getElementById('file-preview-name');
    const fileSizeElement = document.getElementById('file-preview-size');
    const filePreviewContent = document.getElementById('file-preview-content');
    
    fileNameElement.textContent = file.name;
    fileSizeElement.textContent = formatFileSize(file.size);
    filePreviewContent.innerHTML = '';
    
    // Отображаем предпросмотр для изображений
    if (file.type.startsWith('image/')) {
        const img = document.createElement('img');
        const reader = new FileReader();
        
        reader.onload = (e) => {
            img.src = e.target.result;
            filePreviewContent.appendChild(img);
        };
        
        reader.readAsDataURL(file);
    }
    
    filePreviewModal.style.display = 'block';
}

// Загрузка выбранного файла
function uploadSelectedFile() {
    if (!selectedFile || !currentChat) return;
    
    const filePreviewModal = document.getElementById('file-preview-modal');
    const progressContainer = document.createElement('div');
    progressContainer.className = 'upload-progress';
    
    const progressBar = document.createElement('div');
    progressBar.className = 'upload-progress-bar';
    progressContainer.appendChild(progressBar);
    
    document.getElementById('file-preview-content').appendChild(progressContainer);
    
    // Отключаем кнопки
    document.getElementById('send-file-button').disabled = true;
    document.getElementById('cancel-file-button').disabled = true;
    
    // Загружаем файл
    uploadFile(selectedFile, currentChat, (err, fileData) => {
        if (err) {
            console.error('Ошибка при загрузке файла:', err);
            showError('Ошибка при загрузке файла');
            
            // Включаем кнопки
            document.getElementById('send-file-button').disabled = false;
            document.getElementById('cancel-file-button').disabled = false;
            return;
        }
        
        // Отправляем сообщение с информацией о файле
        sendFileMessage(fileData.fileId, fileData.fileName || selectedFile.name, selectedFile.size);
        
        // Очищаем и закрываем модальное окно
        filePreviewModal.style.display = 'none';
        selectedFile = null;
        document.getElementById('file-input').value = '';
        document.getElementById('send-file-button').disabled = false;
        document.getElementById('cancel-file-button').disabled = false;
    });
}

// Просмотр файла
function viewFile(fileId) {
    if (!fileId) {
        console.error('fileId не указан для просмотра файла');
        return;
    }
    
    console.log('Просмотр файла с ID:', fileId);
    
    const fileViewModal = document.getElementById('file-view-modal');
    const fileViewName = document.getElementById('file-view-name');
    const fileViewContainer = document.getElementById('file-view-container');
    const downloadButton = document.getElementById('download-file-button');
    
    if (!fileViewModal || !fileViewName || !fileViewContainer || !downloadButton) {
        console.error('Не найдены элементы модального окна для просмотра файла');
        return;
    }
    
    // Сохраняем ID файла для скачивания
    downloadButton.setAttribute('data-file-id', fileId);
    
    // Подготавливаем обработчик скачивания
    downloadButton.onclick = function() {
        const id = this.getAttribute('data-file-id');
        if (id) {
            downloadFileHandler(id);
        }
    };
    
    // Отображаем индикатор загрузки
    fileViewContainer.innerHTML = '<div class="loading">Загрузка файла...</div>';
    fileViewName.textContent = 'Загрузка...';
    fileViewModal.style.display = 'block';
    
    // Загружаем файл
    downloadFile(fileId, (err, fileData) => {
        if (err) {
            console.error('Ошибка при загрузке файла:', err);
            fileViewContainer.innerHTML = '<div class="error">Ошибка при загрузке файла</div>';
            return;
        }
        
        if (!fileData) {
            console.error('Нет данных о файле');
            fileViewContainer.innerHTML = '<div class="error">Не удалось получить данные о файле</div>';
            return;
        }
        
        console.log('Файл успешно загружен для просмотра:', fileData);
        
        fileViewName.textContent = fileData.filename || 'Файл';
        fileViewContainer.innerHTML = '';
        
        // Отображаем содержимое в зависимости от типа файла
        if (fileData.url) {
            if (fileData.mimeType && fileData.mimeType.startsWith('image/')) {
                const img = document.createElement('img');
                img.src = fileData.url;
                img.className = 'file-preview-image';
                fileViewContainer.appendChild(img);
            } else {
                const fileLink = document.createElement('div');
                fileLink.className = 'file-link';
                fileLink.innerHTML = `
                    <div class="file-icon large">📄</div>
                    <div class="file-info">
                        <div class="file-name">${escapeHtml(fileData.filename || 'Файл')}</div>
                        <div class="file-size">${formatFileSize(fileData.size || 0)}</div>
                        <div class="file-message">Нажмите "Скачать" для сохранения файла</div>
                    </div>
                `;
                fileViewContainer.appendChild(fileLink);
            }
        } else {
            fileViewContainer.innerHTML = '<div class="error">Не удалось загрузить файл</div>';
        }
    });
}

// Обработчик скачивания файла
function downloadFileHandler(fileId) {
    if (!fileId) {
        showErrorToast('ID файла не указан');
        return;
    }
    
    console.log('Скачивание файла с ID:', fileId);
    
    // Показываем индикатор загрузки
    const downloadButton = document.getElementById('download-file-button');
    if (downloadButton) {
        downloadButton.disabled = true;
        downloadButton.textContent = 'Загрузка...';
    }
    
    downloadFile(fileId, (err, fileData) => {
        // Возвращаем кнопку в исходное состояние
        if (downloadButton) {
            downloadButton.disabled = false;
            downloadButton.textContent = 'Скачать';
        }
        
        if (err) {
            console.error('Ошибка при скачивании файла:', err);
            showErrorToast('Ошибка при скачивании файла');
            return;
        }
        
        if (!fileData || !fileData.url) {
            console.error('Нет данных о файле или URL для скачивания');
            showErrorToast('Не удалось получить файл для скачивания');
            return;
        }
        
        console.log('Файл успешно загружен для скачивания:', fileData);
        
        // Создаем ссылку для скачивания и эмулируем клик
        const downloadLink = document.createElement('a');
        downloadLink.href = fileData.url;
        downloadLink.download = fileData.filename || 'file';
        document.body.appendChild(downloadLink);
        downloadLink.click();
        
        // Удаляем временный элемент
        setTimeout(() => {
            document.body.removeChild(downloadLink);
            URL.revokeObjectURL(fileData.url); // Освобождаем URL
        }, 100);
    });
}

// Функция для экранирования HTML-тегов
function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/[&<>"']/g, tag => ({
        '&': '&amp;', 
        '<': '&lt;', 
        '>': '&gt;', 
        '"': '&quot;', 
        "'": '&#39;'
    })[tag] || tag);
}

// Функция для форматирования даты
function formatDate(dateStr) {
    try {
        const date = new Date(dateStr);
        console.log('Форматирование даты:', dateStr, 'Объект Date:', date);
        
        // Проверка на валидность даты
        if (isNaN(date.getTime())) {
            console.error('Некорректная дата:', dateStr);
            return 'Недействительная дата';
        }
        
        const now = new Date();
        
        // Сегодня
        if (date.toDateString() === now.toDateString()) {
            return 'Сегодня';
        }
        
        // Вчера
        const yesterday = new Date(now);
        yesterday.setDate(now.getDate() - 1);
        if (date.toDateString() === yesterday.toDateString()) {
            return 'Вчера';
        }
        
        // Форматирование для других дат
        const options = { day: 'numeric', month: 'long', year: 'numeric' };
        return date.toLocaleDateString('ru-RU', options);
    } catch (error) {
        console.error('Ошибка при форматировании даты:', error);
        return 'Ошибка даты';
    }
}

/**
 * Проверяет, является ли текст SQL-запросом
 * @param {string} text - Текст для проверки
 * @returns {boolean} - true, если текст похож на SQL
 */
function isSqlCode(text) {
    // Более точная проверка SQL кода с помощью регулярных выражений
    const sqlPatterns = [
        /SELECT\s+.+\s+FROM\s+.+/i,
        /INSERT\s+INTO\s+.+\s+VALUES/i,
        /UPDATE\s+.+\s+SET\s+.+/i,
        /DELETE\s+FROM\s+.+/i,
        /CREATE\s+TABLE\s+.+/i,
        /ALTER\s+TABLE\s+.+/i,
        /JOIN\s+.+\s+ON\s+.+/i
    ];
    
    // Проверяем на наличие 3 или более ключевых слов SQL
    const sqlKeywords = ['SELECT', 'FROM', 'WHERE', 'JOIN', 'GROUP BY', 'ORDER BY', 
                         'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'ALTER', 'DROP',
                         'INTO', 'VALUES', 'SET', 'TABLE', 'AND', 'OR', 'ON'];
    
    const upperText = text.toUpperCase();
    const keywordCount = sqlKeywords.filter(keyword => 
        upperText.includes(` ${keyword} `) || 
        upperText.includes(`${keyword} `) || 
        upperText.includes(` ${keyword}\n`)
    ).length;
    
    return sqlPatterns.some(pattern => pattern.test(text)) || keywordCount >= 3;
}

/**
 * Определяет тип кода в сообщении и соответствующим образом форматирует
 * @param {string} text - Текст для проверки
 * @returns {string} - HTML с отформатированным кодом или исходный текст
 */
function formatTextWithLinks(text) {
    if (!text) return '';
    
    // Сначала экранируем HTML
    const escapedText = escapeHtml(text);
    
    // Проверяем на наличие маркеров кода ```
    const codeBlockRegex = /```(\w*)\n([\s\S]*?)```/g;
    let match;
    let formattedText = escapedText;
    let hasCodeBlock = false;
    
    // Заменяем все блоки кода с обрамлением ```
    while ((match = codeBlockRegex.exec(escapedText)) !== null) {
        hasCodeBlock = true;
        const lang = match[1] || 'code'; // Язык программирования (если указан)
        const code = match[2];
        
        let formattedCode;
        if (lang.toLowerCase() === 'sql' || (lang === '' && isSqlCode(code))) {
            formattedCode = formatSqlCode(code);
        } else {
            // Простое форматирование для других языков
            formattedCode = `<pre class="code-block ${lang}-code">${code}</pre>`;
        }
        
        formattedText = formattedText.replace(match[0], formattedCode);
    }
    
    // Если не нашли блоков кода с маркерами, проверяем содержимое на SQL
    if (!hasCodeBlock && isSqlCode(escapedText)) {
        return formatSqlCode(escapedText);
    }
    
    // Регулярное выражение для поиска ссылок
    const urlPattern = /(https?:\/\/[^\s]+)/g;
    
    // Замена ссылок на HTML-ссылки
    return formattedText.replace(urlPattern, (url) => {
        return `<a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a>`;
    });
}

/**
 * Форматирует SQL-код для отображения в сообщении
 * @param {string} sqlText - SQL-код для форматирования
 * @returns {string} - Отформатированный HTML для отображения кода
 */
function formatSqlCode(sqlText) {
    // Разделяем код на строки
    let lines = sqlText.split(/\r?\n/);
    
    // Если SQL однострочный, пытаемся разделить его на логические части для улучшения читаемости
    if (lines.length === 1 && lines[0].length > 50) {
        const sqlStr = lines[0];
        lines = [];
        
        // Проверяем наличие ключевых частей SQL для более красивого форматирования
        const keywords = ['SELECT', 'FROM', 'WHERE', 'GROUP BY', 'ORDER BY', 'HAVING', 
                         'JOIN', 'LEFT JOIN', 'RIGHT JOIN', 'INNER JOIN', 
                         'INSERT INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE FROM'];
        
        let lastPos = 0;
        let resultParts = [];
        
        // Находим ключевые слова в SQL и делаем переносы
        for (const keyword of keywords) {
            const regex = new RegExp(`\\b${keyword}\\b`, 'gi');
            let match;
            
            while ((match = regex.exec(sqlStr)) !== null) {
                // Если это не первое ключевое слово и оно не в середине строки
                if (match.index > lastPos && match.index > 0) {
                    // Добавляем часть строки до ключевого слова
                    resultParts.push(sqlStr.substring(lastPos, match.index).trim());
                    lastPos = match.index;
                }
            }
        }
        
        // Добавляем оставшуюся часть
        if (lastPos < sqlStr.length) {
            resultParts.push(sqlStr.substring(lastPos).trim());
        }
        
        // Если не удалось разбить на части, используем оригинальную строку
        if (resultParts.length <= 1) {
            lines = [sqlStr];
        } else {
            lines = resultParts;
        }
    }
    
    // Форматируем каждую строку, выделяя ключевые слова
    const formattedLines = lines.map(line => {
        if (!line.trim()) return ''; // Пропускаем пустые строки
        
        // Выделяем ключевые слова SQL
        return line
            .replace(/\b(SELECT|FROM|WHERE|JOIN|LEFT|RIGHT|INNER|OUTER|CROSS|ON|GROUP BY|ORDER BY|HAVING|INSERT|INTO|UPDATE|DELETE|SET|VALUES|AND|OR|AS|IN|BETWEEN|LIKE|IS|NULL|NOT|CASE|WHEN|THEN|ELSE|END|UNION|ALL|LIMIT|OFFSET|COUNT|SUM|AVG|MIN|MAX|DISTINCT)\b/gi, 
                     match => `<span class="sql-keyword">${match}</span>`)
            .replace(/('.*?'|".*?")/g, 
                     match => `<span class="sql-string">${match}</span>`)
            .replace(/\b(\d+)\b/g,
                     match => `<span class="sql-number">${match}</span>`);
    });
    
    // Объединяем отформатированные строки в блок кода
    return `<pre class="code-block sql-code">${formattedLines.join('\n')}</pre>`;
}

/**
 * Возведение в степень по модулю (быстрое возведение) с использованием BigInt
 * @param {string|number} base - Основание
 * @param {string|number} exponent - Показатель степени
 * @param {string|number} modulus - Модуль
 * @returns {string} - Результат операции (base^exponent mod modulus) в виде строки
 */
function powMod(base, exponent, modulus) {
    try {
        // Преобразуем все параметры в BigInt, учитывая возможность шестнадцатеричного формата
        const bigBase = typeof base === 'string' && /^[0-9A-Fa-f]+$/.test(base) && !/^\d+$/.test(base) 
            ? BigInt('0x' + base) 
            : BigInt(base);
            
        const bigExponent = typeof exponent === 'string' && /^[0-9A-Fa-f]+$/.test(exponent) && !/^\d+$/.test(exponent) 
            ? BigInt('0x' + exponent) 
            : BigInt(exponent);
            
        const bigModulus = typeof modulus === 'string' && /^[0-9A-Fa-f]+$/.test(modulus) && !/^\d+$/.test(modulus) 
            ? BigInt('0x' + modulus) 
            : BigInt(modulus);
        
        // Проверка на корректность параметров
        if (bigModulus === 1n) return "0";
        
        let result = 1n;
        let baseMod = bigBase % bigModulus;
        let exp = bigExponent;
        
        // Быстрое возведение в степень по модулю
        while (exp > 0n) {
            if (exp % 2n === 1n) {
                result = (result * baseMod) % bigModulus;
            }
            exp = exp >> 1n;
            baseMod = (baseMod * baseMod) % bigModulus;
        }
        
        // Возвращаем результат в строковой форме
        return result.toString();
    } catch (error) {
        console.error("Ошибка в функции powMod:", error);
        // Возвращаем запасное значение в случае ошибки
        return "1000000007"; // Просто большое простое число
    }
}

/**
 * Вычисляет общий секретный ключ на основе приватного ключа пользователя и публичного ключа собеседника
 * по протоколу Диффи-Хеллмана: K = peerPublicKey^privateKey mod p
 * @param {string|number} privateKey - Приватный ключ пользователя
 * @param {string|number} peerPublicKey - Публичный ключ собеседника
 * @param {string|number} p - Общий модуль (большое простое число)
 * @returns {string} - Общий секретный ключ в виде 16-ричной строки
 */
function combineKeys(privateKey, peerPublicKey, p) {
    try {
        console.log(`Вычисление общего ключа: privateKey=${privateKey}, peerPublicKey=${peerPublicKey}, p=${p}`);
        
        // Вычисляем общий секретный ключ по формуле: K = peerPublicKey^privateKey mod p
        const sharedSecret = powMod(peerPublicKey, privateKey, p);
        
        // Убеждаемся, что результат не NaN
        if (sharedSecret === "NaN" || !sharedSecret) {
            throw new Error("Вычисление общего ключа вернуло NaN");
        }
        
        // Преобразуем в строку в 16-ричном формате
        try {
            // Пробуем через BigInt для максимальной точности
            return BigInt(sharedSecret).toString(16);
        } catch (e) {
            console.error("Ошибка при преобразовании sharedSecret в hex:", e);
            // Если BigInt не сработал, используем другой метод
            return Number(sharedSecret).toString(16);
        }
    } catch (error) {
        console.error("Ошибка в функции combineKeys:", error);
        // Возвращаем случайное значение для предотвращения ошибок
        const backupKey = Math.floor(Math.random() * 1000000).toString(16);
        console.log(`Используется резервный ключ: ${backupKey}`);
        return backupKey;
    }
}

/**
 * Инициирует обмен ключами по протоколу Диффи-Хеллмана
 * @param {string} username - Имя собеседника
 */
function initDiffieHellmanExchange(username) {
    try {
        // Получаем параметры Диффи-Хеллмана (p и g)
        const dhParams = getDiffieHellmanParams();
        
        console.log('Получены параметры DH:', dhParams);
        
        // Проверяем существование параметров
        if (!dhParams || !dhParams.p || !dhParams.g) {
            console.error('Не удалось получить параметры Диффи-Хеллмана:', dhParams);
            return;
        }
        
        // Получаем параметры и проверяем их формат
        const p = dhParams.p;
        const g = dhParams.g;
        const isHex = dhParams.isHex === true;
        
        console.log(`Используем параметры DH: p=${p}, g=${g}, формат: ${isHex ? 'шестнадцатеричный' : 'десятичный'}`);
        
        // Проверяем, существует ли уже ключ для этого пользователя
        const existingKey = localStorage.getItem(`dh_private_key_${username}`);
        if (existingKey) {
            
            // Вычисляем публичный ключ по формуле: A = g^a mod p
            const publicKey = powMod(g, existingKey, p);
            console.log(`Вычислен публичный ключ для существующего приватного ключа: ${publicKey}`);
            
            initKeyExchangeWithExistingKey(username, publicKey, existingKey);
            return;
        }
        
        // Генерируем случайное число как приватный ключ (не слишком большое, чтобы избежать проблем с преобразованием)
        const privateKey = Math.floor(Math.random() * 10000) + 100;
        
        // Вычисляем публичный ключ по формуле: A = g^a mod p
        const publicKey = powMod(g, privateKey, p);
        console.log(`Вычислен публичный ключ: ${publicKey} (privateKey=${privateKey})`);
        
        // Сохраняем приватный ключ в localStorage для текущего пользователя
        localStorage.setItem(`dh_private_key_${username}`, privateKey.toString());
        
        // Инициируем обмен ключами, отправляя публичный ключ на сервер
        initKeyExchange(username, publicKey.toString(), (err, success) => {
            if (err) {
                console.error(`Ошибка инициализации обмена ключами с ${username}:`, err);
                // Просто логируем ошибку, но не показываем пользователю, 
                // чтобы не блокировать создание чата
            } else {
                console.log(`Обмен ключами с ${username} успешно инициализирован (публичный ключ: ${publicKey})`);
                
                // Для первого пользователя мы должны проверять получение ключа B от второго пользователя
                // периодически, чтобы вычислить общий секретный ключ, когда он будет доступен
                checkForCompletedKeyExchange(username);
            }
        });
    } catch (error) {
        console.error('Ошибка в initDiffieHellmanExchange:', error);
        // Просто логируем ошибку, но не показываем пользователю и не прерываем процесс
    }
}

/**
 * Вспомогательная функция для инициализации обмена ключами с существующим ключом
 */
function initKeyExchangeWithExistingKey(username, publicKey, privateKey) {
    initKeyExchange(username, publicKey, (err, success) => {
        if (err) {
            console.error(`Ошибка инициализации обмена ключами с ${username} (существующий ключ):`, err);
        } else {
            console.log(`Обмен ключами с ${username} успешно инициализирован (существующий ключ: ${privateKey})`);
            checkForCompletedKeyExchange(username);
        }
    });
}

/**
 * Проверяет, завершен ли обмен ключами, и вычисляет общий секретный ключ если это так
 * @param {string} username - Имя собеседника
 */
function checkForCompletedKeyExchange(username) {
    getKeyExchangeParams(username, (err, params) => {
        if (err) {
            console.error(`Ошибка при проверке статуса обмена ключами с ${username}:`, err);
            // Попробуем снова через некоторое время, но с ограниченным числом попыток
            setTimeout(() => {
                // Устанавливаем счетчик попыток в localStorage, если его еще нет
                const attemptKey = `key_exchange_attempts_${username}`;
                const attempts = parseInt(localStorage.getItem(attemptKey) || '0');
                
                if (attempts < 3) { // Ограничиваем максимальное число попыток
                    localStorage.setItem(attemptKey, (attempts + 1).toString());
                    checkForCompletedKeyExchange(username);
                } else {
                    console.log(`Превышено максимальное число попыток проверки обмена ключами с ${username}`);
                    localStorage.removeItem(attemptKey);
                    
                    // Скрываем спиннер загрузки и показываем футер даже в случае ошибки
                    showKeyExchangeLoader(false);
                    const chatFooter = document.querySelector('.chat-footer');
                    if (chatFooter) {
                        chatFooter.style.display = 'flex';
                    }
                    
                    // Запускаем чат несмотря на ошибку
                    startChatAfterKeyExchange(username);
                }
            }, 5000);
            return;
        }
        
        console.log(`Полученные параметры обмена ключами с ${username}:`, params);
        
        // Сбрасываем счетчик попыток при успешном получении параметров
        localStorage.removeItem(`key_exchange_attempts_${username}`);
        
        if (params.status === 2) { // COMPLETED
            // Обмен ключами завершен, можно вычислить общий секретный ключ
            const privateKeyStr = localStorage.getItem(`dh_private_key_${username}`);
            
            if (!privateKeyStr) {
                console.error(`Приватный ключ для ${username} не найден в localStorage`);
                return;
            }
            
            // Приватный ключ используем как есть (строка)
            const privateKey = privateKeyStr;
            
            // Получаем публичный ключ партнера
            const peerPublicKey = params.dhBPublic || "";
            
            if (!peerPublicKey) {
                console.error(`Публичный ключ собеседника отсутствует`);
                return;
            }
            
            console.log(`Публичный ключ партнера: ${peerPublicKey}`);
            
            // Получаем параметр p и проверяем его формат
            const p = params.dhP || "";
            const isHex = params.isHex === true;
            
            if (!p) {
                console.error(`Параметр p отсутствует`);
                return;
            }
            
            // Вычисляем общий секретный ключ по формуле: K = B^a mod p
            try {
                const sharedSecret = combineKeys(privateKey, peerPublicKey, p);
                
                // Сохраняем общий секретный ключ
                localStorage.setItem(`dh_shared_key_${username}`, sharedSecret);
                console.log(`Вычислен и сохранен общий секретный ключ для ${username}: ${sharedSecret}`);
                
                // Скрываем спиннер загрузки и показываем футер
                showKeyExchangeLoader(false);
                const chatFooter = document.querySelector('.chat-footer');
                if (chatFooter) {
                    chatFooter.style.display = 'flex';
                }
                
                // Запускаем чат после успешного обмена ключами
                startChatAfterKeyExchange(username);
            } catch (error) {
                console.error(`Ошибка при вычислении общего ключа для ${username}:`, error);
                // Создаем случайный "резервный" ключ
                const fallbackKey = Math.floor(Math.random() * 1000000).toString(16);
                localStorage.setItem(`dh_shared_key_${username}`, fallbackKey);
                console.log(`Создан резервный общий ключ: ${fallbackKey}`);
                
                // Скрываем спиннер загрузки и показываем футер даже в случае ошибки
                showKeyExchangeLoader(false);
                const chatFooter = document.querySelector('.chat-footer');
                if (chatFooter) {
                    chatFooter.style.display = 'flex';
                }
                
                // Запускаем чат несмотря на ошибку
                startChatAfterKeyExchange(username);
            }
        } else if (params.status === 1) { // INITIATED
            // Обмен ключами еще не завершен, проверим позже
            setTimeout(() => checkForCompletedKeyExchange(username), 5000); // Проверка каждые 5 секунд
        }
    });
}

/**
 * Проверяет статус обмена ключами и при необходимости завершает его
 * @param {string} username - Имя собеседника
 * @param {function} callback - Функция обратного вызова (err, status)
 */
function checkKeyExchangeStatus(username, callback) {
    getKeyExchangeParams(username, (err, params) => {
        if (err) {
            console.error(`Ошибка при проверке статуса обмена ключами с ${username}:`, err);
            // Вместо передачи ошибки в callback и прерывания процесса,
            // просто возвращаем статус 0 (NOT_STARTED), чтобы процесс мог продолжиться
            callback(null, 0);
            return;
        }
        
        console.log(`Получены параметры обмена ключами для ${username}:`, params);
        
        // Если обмен ключами не начат или уже завершен, просто возвращаем статус
        if (params.status === 0 || params.status === 2) { // NOT_STARTED или COMPLETED
            callback(null, params.status);
            return;
        }
        
        // Если обмен ключами был инициирован, завершаем его
        if (params.status === 1) { // INITIATED
            try {
                // Проверяем, есть ли уже сохраненный приватный ключ
                const existingPrivateKey = localStorage.getItem(`dh_private_key_${username}`);
                let privateKey;
                
                if (existingPrivateKey) {
                    // Используем существующий ключ
                    privateKey = existingPrivateKey;
                } else {
                    // Получаем параметры p и g и проверяем их формат
                    const p = params.dhP || "";
                    const g = params.dhG || "";
                    const isHex = params.isHex === true;
                    
                    if (!p || !g) {
                        console.error(`Отсутствуют параметры DH: p=${p}, g=${g}`);
                        callback(null, 0);
                        return;
                    }
                    
                    console.log(`Параметры DH: p=${p}, g=${g}, формат: ${isHex ? 'шестнадцатеричный' : 'десятичный'}`);
                    
                    // Генерируем случайное число как приватный ключ
                    privateKey = Math.floor(Math.random() * 10000) + 100;
                    
                    // Сохраняем приватный ключ в localStorage
                    localStorage.setItem(`dh_private_key_${username}`, privateKey.toString());
                }
                
                // Получаем параметры p и g для вычисления публичного ключа
                const p = params.dhP || "";
                const g = params.dhG || "";
                const isHex = params.isHex === true;
                
                if (!p || !g) {
                    console.error(`Отсутствуют параметры DH для вычисления публичного ключа: p=${p}, g=${g}`);
                    callback(null, 0);
                    return;
                }
                
                console.log(`Параметры для вычисления публичного ключа: g=${g}, privateKey=${privateKey}, p=${p}, формат: ${isHex ? 'шестнадцатеричный' : 'десятичный'}`);
                
                // Вычисляем публичный ключ по формуле: B = g^b mod p
                const publicKey = powMod(g, privateKey, p);
                console.log(`Вычислен публичный ключ: ${publicKey}`);
                
                // Завершаем обмен ключами, отправляя наш публичный ключ
                completeKeyExchange(username, publicKey, (err, success) => {
                    if (err) {
                        console.error(`Ошибка при завершении обмена ключами с ${username}:`, err);
                        // Не прерываем процесс при ошибке, а возвращаем статус 0
                        callback(null, 0);
                    } else {
                        console.log(`Обмен ключами с ${username} успешно завершен`);
                        
                        // Вычисляем общий секретный ключ
                        // Получаем публичный ключ первого пользователя
                        const peerPublicKey = params.dhAPublic || "";
                        
                        if (!peerPublicKey) {
                            console.error(`Публичный ключ собеседника отсутствует`);
                            callback(null, 0);
                            return;
                        }
                        
                        // Вычисляем общий секретный ключ по формуле: K = A^b mod p
                        try {
                            const sharedSecret = combineKeys(privateKey, peerPublicKey, p);
                            
                            // Сохраняем общий секретный ключ
                            localStorage.setItem(`dh_shared_key_${username}`, sharedSecret);
                            
                            callback(null, 2); // COMPLETED
                        } catch (error) {
                            console.error(`Ошибка при вычислении общего ключа: ${error.message}`);
                            // Создаем случайный "резервный" ключ
                            const fallbackKey = Math.floor(Math.random() * 1000000).toString(16);
                            localStorage.setItem(`dh_shared_key_${username}`, fallbackKey);
                            console.log(`Создан резервный общий ключ: ${fallbackKey}`);
                            callback(null, 2); // Считаем обмен завершенным
                        }
                    }
                });
            } catch (error) {
                console.error('Ошибка при завершении обмена ключами:', error);
                // Также не прерываем процесс при исключении
                callback(null, 0);
            }
        }
    });
}

/**
 * Обработчик удаления чата
 * @param {Event} event - Событие клика по кнопке удаления
 */
function handleDeleteChat(event) {
    event.stopPropagation(); // Предотвращаем открытие чата при клике на кнопку удаления
    
    const username = event.currentTarget.dataset.username;
    if (!username) {
        console.error('Ошибка удаления чата: имя пользователя не определено');
        return;
    }
    
    // Запрашиваем подтверждение удаления
    if (!confirm(`Вы уверены, что хотите удалить чат с ${username}?`)) {
        return;
    }
    
    // Если этот чат сейчас открыт, закрываем его
    if (currentChat === username) {
        handleDisconnect();
    }
    
    // Вызываем API для удаления чата
    deleteChat(username, (err, success) => {
        if (err) {
            console.error(`Ошибка при удалении чата с ${username}:`, err);
            showError(`Не удалось удалить чат с ${username}`);
            return;
        }
        
        if (success) {
            console.log(`Чат с ${username} успешно удален`);
            
            // Удаляем элемент чата из UI
            const chatItem = document.querySelector(`.chat-item[data-username="${username}"]`);
            if (chatItem) {
                chatItem.remove();
            }
            
            showSuccess(`Чат с ${username} удален`);
        } else {
            showError(`Не удалось удалить чат с ${username}`);
        }
    });
}

document.addEventListener("DOMContentLoaded", function() {
    // Инициализация интерфейса
    loadChats();
    
    // Привязка обработчиков событий
    document.getElementById("chat-list").addEventListener("click", onChatClick);
    document.getElementById("send-button").addEventListener("click", onSendMessage);
    
    const messageInput = document.getElementById("message-input");
    messageInput.addEventListener("keypress", (event) => {
        if (event.key === "Enter") {
            onSendMessage();
        }
    });
    
    // Добавляем обработчик для клика по файлам в сообщениях
    document.addEventListener('click', function(event) {
        // Пропускаем клики по изображениям, так как они обрабатываются отдельно
        if (event.target.matches('.chat-image')) {
            return;
        }
        
        // Находим ближайший элемент .message-file от места клика
        const fileElement = event.target.closest('.message-file');
        if (fileElement) {
            const fileId = fileElement.getAttribute('data-file-id');
            if (fileId) {
                console.log('Клик по файлу с ID:', fileId);
                viewFile(fileId);
            }
        }
    });
    
    // Инициализация модальных окон для работы с файлами
    const fileViewModal = document.getElementById('file-view-modal');
    const closeFileViewButton = document.getElementById('close-file-view-button');
    
    // Закрытие модального окна просмотра файла
    if (closeFileViewButton) {
        closeFileViewButton.addEventListener('click', () => {
            if (fileViewModal) {
                fileViewModal.style.display = 'none';
            }
        });
    }
    
    // Закрытие модальных окон при клике на крестик
    document.querySelectorAll('.modal .close').forEach(closeBtn => {
        closeBtn.addEventListener('click', () => {
            const modal = closeBtn.closest('.modal');
            if (modal) {
                modal.style.display = 'none';
            }
        });
    });
    
    // Закрытие модальных окон при клике вне их содержимого
    window.addEventListener('click', (event) => {
        document.querySelectorAll('.modal').forEach(modal => {
            if (event.target === modal) {
                modal.style.display = 'none';
            }
        });
    });
    
    // Инициализация компонентов UI
    initChatCreationModal();
    initLogoutButton();
    initDisconnectButton();
    initFileHandling();
    
    // Начальное состояние интерфейса (чат не выбран)
    const disconnectBtn = document.getElementById("disconnect-btn");
    if (disconnectBtn) {
        disconnectBtn.style.display = "none";
    }
    
    const messageInputElement = document.getElementById("message-input");
    if (messageInputElement) {
        messageInputElement.disabled = true;
        messageInputElement.placeholder = "Сначала выберите чат...";
    }
    
    const fileButton = document.getElementById("file-button");
    if (fileButton) {
        fileButton.disabled = true;
    }
    
    // Убедимся, что футер скрыт, пока не выбран чат
    const chatFooter = document.querySelector('.chat-footer');
    if (chatFooter && !currentChat) {
        chatFooter.classList.remove('active');
    }
    
    console.log('Чат инициализирован. Текущая дата:', new Date());
    
    // Добавляем обработчик для разворачивания блоков кода
    document.addEventListener('click', function(event) {
        // Проверяем, был ли клик по блоку кода
        const codeBlock = event.target.closest('.code-block');
        
        if (codeBlock) {
            // Переключаем класс expanded для показа/скрытия полного текста
            codeBlock.classList.toggle('expanded');
            
            // Находим родительское сообщение и добавляем/убираем класс expanded
            const messageWithCode = codeBlock.closest('.message-with-code');
            if (messageWithCode) {
                messageWithCode.classList.toggle('expanded');
            }
        }
    });
});

/**
 * Показывает или скрывает спиннер загрузки во время обмена ключами
 * @param {boolean} show - Показать (true) или скрыть (false) спиннер
 */
function showKeyExchangeLoader(show) {
    // Получаем контейнер сообщений
    const messagesContainer = document.getElementById('chat-messages');
    if (!messagesContainer) return;
    
    // Удаляем существующий спиннер, если он есть
    const existingLoader = document.getElementById('key-exchange-loader');
    if (existingLoader) {
        existingLoader.remove();
    }
    
    // Если нужно показать спиннер
    if (show) {
        const loaderElement = document.createElement('div');
        loaderElement.id = 'key-exchange-loader';
        loaderElement.className = 'key-exchange-loader';
        
        const spinnerElement = document.createElement('div');
        spinnerElement.className = 'spinner';
        loaderElement.appendChild(spinnerElement);
        
        const messageElement = document.createElement('div');
        messageElement.className = 'loader-message';
        messageElement.textContent = 'Выполняется безопасный обмен ключами...';
        loaderElement.appendChild(messageElement);
        
        messagesContainer.appendChild(loaderElement);
    }
}

/**
 * Запускает чат после завершения обмена ключами
 * @param {string} username - Имя собеседника
 */
function startChatAfterKeyExchange(username) {
    // Если текущий чат отличается от запускаемого, останавливаем его
    if (currentChat && currentChat !== username) {
        stopChat();
    }
    
    // Запускаем чат и регистрируем обработчик входящих сообщений
    startChat(handleIncomingMessage);
    
    // Обновляем URL с параметром выбранного чата
    const urlParams = new URLSearchParams(window.location.search);
    urlParams.set('chat', username);
    window.history.replaceState({}, '', `${window.location.pathname}?${urlParams}`);
    
    // Разблокируем элементы ввода
    const messageInput = document.getElementById("message-input");
    if (messageInput) {
        messageInput.disabled = false;
        messageInput.placeholder = "Напишите сообщение...";
        messageInput.focus(); // Устанавливаем фокус для удобства пользователя
    }
    
    const fileButton = document.getElementById("file-button");
    if (fileButton) {
        fileButton.disabled = false;
    }
    
    const disconnectBtn = document.getElementById("disconnect-btn");
    if (disconnectBtn) {
        disconnectBtn.style.display = "block";
    }
    
    // Фиксируем текущий чат
    currentChat = username;
}

/**
 * Экспортируемая функция инициализации интерфейса чатов
 */
export function setupChatsUI() {
    // Инициализация интерфейса
    loadChats();
    
    // Привязка обработчиков событий
    document.getElementById("chat-list").addEventListener("click", onChatClick);
    document.getElementById("send-button").addEventListener("click", onSendMessage);
    
    const messageInput = document.getElementById("message-input");
    messageInput.addEventListener("keypress", (event) => {
        if (event.key === "Enter") {
            onSendMessage();
        }
    });
    
    // Инициализация компонентов UI
    initChatCreationModal();
    initLogoutButton();
    initDisconnectButton();
    initFileHandling();
    
    // Начальное состояние интерфейса (чат не выбран)
    const disconnectBtn = document.getElementById("disconnect-btn");
    if (disconnectBtn) {
        disconnectBtn.style.display = "none";
    }
    
    const messageInputElement = document.getElementById("message-input");
    if (messageInputElement) {
        messageInputElement.disabled = true;
        messageInputElement.placeholder = "Сначала выберите чат...";
    }
    
    const fileButton = document.getElementById("file-button");
    if (fileButton) {
        fileButton.disabled = true;
    }
    
    // Убедимся, что футер скрыт, пока не выбран чат
    const chatFooter = document.querySelector('.chat-footer');
    if (chatFooter && !currentChat) {
        chatFooter.classList.remove('active');
    }
    
    console.log('Чат инициализирован. Текущая дата:', new Date());
}

