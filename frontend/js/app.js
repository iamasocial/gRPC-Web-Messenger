// Подключаем необходимые зависимости
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');

// Указываем путь к файлу .proto
const PROTO_PATH = __dirname + '/../../proto/messenger.proto';

// Загружаем описание .proto файла
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true
});

// Создаем клиент с использованием загруженной спецификации
const messengerProto = grpc.loadPackageDefinition(packageDefinition).messenger;

// Подключаемся к серверу
const client = new messengerProto.Messenger('localhost', grpc.credentials.createInsecure());

const registerForm = document.getElementById('registation-form');
const usernameInput = document.getElementById('username')
const registerBtn = document.getElementById('registration-btn')
const chatContainer = document.getElementById('chat-container')
const sendBtn = document.getElementById('send-btn');
const messageInput = document.getElementById('message')
const chatBox = document.getElementById('chat-box')

registerBtn.addEventListener('click', () => {
    const username = usernameInput.value.trim();
    if (username) {
        registerUser(username);
    } else {
        alert('Please enter a username');
    }
});

function registerUser(username) {
    client.Register({ username: username }, (error, response) => {
    if (error) {
        console.error('Error registering user:', error);
    } else {
        const usedId = response.usedId;
        console.log('Registered successfully with User ID:', usedId)
        
        registerForm.style.display = 'none';
        chatContainer.style.display = 'block';
        messageInput.disabled = false;
        sendBtn.disabled = false;

        addMessageToChat('Welcome to the chat, ' + username);
    }
    });
}

sendBtn.addEventListener('click', () => {
    const message = messageInput.value;
    if (message.trim() !== "") {
        sendMessage(message);
        messageInput.value = '';
    }
});

function sendMessage(message) {
    client.sendMessage({ chatName: 'default', message }, (error, response) => {
        if (!error) {
            addMessageToChat('Me: ' + message);
        } else {
            console.error(error)
        }
    });
}

function addMessageToChat(message) {
    const messageElement = document.createElement('div');
    messageElement.textContent = message;
    chatBox.appendChild(messageElement);
    chatBox.scrollTop = chatBox.scrollHeight;
}

// Создаем новый запрос
const request = {
  message: 'Hello from client!'
};

// Отправляем запрос на сервер
client.sendMessage(request, (error, response) => {
  if (error) {
    console.error('Error:', error);
  } else {
    console.log('Server response:', response);
  }
});
