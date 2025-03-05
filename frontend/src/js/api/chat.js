import { chatClient } from "./client"; 
import socket from "./websocket.js";
import { GetChatsRequst, ConnectRequest } from "../../proto/chat_service_pb";

export function getChats(callback) {
    const token = localStorage.getItem('token');
    if (!token) {
        callback(new Error("Unathorized: No token found"), null);
        return;
    }

    const request = new GetChatsRequst();
    const metadata = { 'Authorization': `Bearer ${token}` };

    chatClient.getChats(request, metadata, (err, response) => {
        if (err) {
            callback(err, null);
            return;
        }

        callback(null, response.getUsernamesList());
    });
}

export function connectToChat(receiverUsername, callback) {
    const token = localStorage.getItem("token");
    if (!token) {
        callback(new Error("User is not authenticated"), null);
        return;
    }

    const request = new ConnectRequest();
    request.setReceiverusername(receiverUsername);
    const metadata = { 'Authorization': `Bearer ${token}` };
    
    chatClient.connectToChat(request, metadata, (err, response) => {
        if (err) {
            console.error("Error connecting to chat:", err)
            callback(err, null);
            return;
        }

        console.log(`Connected to ${receiverUsername}`)
        callback(null, response.getSuccess());
    });
}

export function startChat(onMessageReceived) {
    const token = localStorage.getItem("token");
    if (!token) {
        console.error("User is not authenticated");
        window.location.href = "index.html";
        return;
    }

    socket.connect(token);

    socket.addMessageHandler(onMessageReceived);
}

export function chat(content) {
    if (!content.trim()) {
        console.error("Message content cannot be empty");
        return;
    }

    const message = {
        content: content,
    }

    socket.sendMessage(message)
}

export function stopChat() {
    socket.close();
}