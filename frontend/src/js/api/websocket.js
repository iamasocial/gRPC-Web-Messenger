class WebSocketClient {
    constructor() {
        this.socket = null;
        this.messageHandlers = [];
        this.fileHandlers = new Map(); // Для обработчиков файловых сообщений
    }

    connect(token) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            console.warn("WebSocket is already connected.");
            return;
        }

        token = localStorage.getItem("token");
        this.socket = new WebSocket("ws://localhost:8888/ws", [token]);

        this.socket.onopen = () => {
            console.log("WebSocket connected");
        };

        this.socket.onmessage = (event) => {
            console.log("Raw WebSocket data:", event.data);

            try {
                const data = JSON.parse(event.data);
                console.log("WebSocket received:", data);
                
                // Проверяем, является ли сообщение служебным сообщением о файлах
                const isFileServiceMessage = data.type && (
                    data.type.startsWith('file_upload_') || 
                    data.type.startsWith('file_chunk') || 
                    data.type.startsWith('file_download_') || 
                    data.type === 'file_info'
                );
                
                // Обрабатываем файловые сообщения через внешние обработчики
                if (isFileServiceMessage) {
                    // Пробрасываем сырое событие для обработчиков файлов
                    this.fileHandlers.forEach(handler => {
                        try {
                            handler(event);
                        } catch (e) {
                            console.error("Error in file handler:", e);
                        }
                    });
                } else {
                    // Вызываем общие обработчики только для обычных сообщений чата
                    this.messageHandlers.forEach((handler) => {
                        try {
                            handler(data);
                        } catch (e) {
                            console.error("Error in message handler:", e);
                        }
                    });
                }
            } catch (error) {
                console.error("Error parsing WebSocket message:", error);
            }
        };

        this.socket.onerror = (error) => {
            console.error("WebSocket error:", error);
        };

        this.socket.onclose = (event) => {
            console.log("WebSocket closed", event.reason);
            this.socket = null;
        };
    }

    sendMessage(message) {
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            console.error("WebSocket is not connected.");
            return;
        }

        try {
            this.socket.send(JSON.stringify(message));
            console.log("WebSocket sent:", message);

            // Не проксируем файловые сообщения в обработчики чата
            if (!message.type || 
                (message.type !== 'file_upload_init' && 
                 message.type !== 'file_chunk' && 
                 message.type !== 'file_download_request')) {
                this.messageHandlers.forEach((handler) => handler(message));
            }
        } catch (error) {
            console.error("Error sending WebSocket message:", error);
        }
    }

    addMessageHandler(handler) {
        this.messageHandlers.push(handler);
    }
    
    removeSpecificMessageHandler(handler) {
        const index = this.messageHandlers.indexOf(handler);
        if (index !== -1) {
            this.messageHandlers.splice(index, 1);
        }
    }

    removeMessageHandler() {
        this.messageHandlers = [];
    }
    
    // Для файловых обработчиков
    addFileHandler(handlerId, handler) {
        this.fileHandlers.set(handlerId, handler);
    }
    
    removeFileHandler(handlerId) {
        this.fileHandlers.delete(handlerId);
    }

    close() {
        if (this.socket) {
            this.socket.close();
            this.socket = null;
        }
        this.removeMessageHandler();
        this.fileHandlers.clear();
    }

    // Получение доступа к объекту сокета
    getSocket() {
        return this.socket;
    }
}

const socket = new WebSocketClient();
export default socket;