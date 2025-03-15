import { ChatServiceClient } from "../../proto/chat_service_grpc_web_pb";
import { UserServiceClient } from "../../proto/user_service_grpc_web_pb";
import { FileServiceClient } from "../../proto/file_service_grpc_web_pb";

const SERVER_URL = "http://localhost:8888";

// Функция для получения метаданных с токеном
function getMetadata() {
    const token = localStorage.getItem('token');
    if (!token) {
        throw new Error('Токен не найден');
    }
    return {
        'Authorization': `Bearer ${token}`
    };
}

// Функции для создания клиентов с актуальным токеном
export function getChatClient() {
    return new ChatServiceClient(SERVER_URL);
}

export function getUserClient() {
    return new UserServiceClient(SERVER_URL);
}

export function getFileClient() {
    return new FileServiceClient(SERVER_URL);
}

export { getMetadata };