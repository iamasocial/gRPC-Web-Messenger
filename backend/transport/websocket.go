package transport

import (
	"context"
	"encoding/json"
	pb "gRPCWebServer/backend/generated"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type WebSocketHandler struct {
	upgrader    websocket.Upgrader
	connections sync.Map
	client      pb.ChatServiceClient
}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Sec-WebSocket-Protocol")

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	grpcConn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed connection client to server")
	}

	h.client = pb.NewChatServiceClient(grpcConn)

	h.connections.Store(1, conn)
	defer h.connections.Delete(1)

	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "Authorization", "Bearer "+token)

	stream, err := h.client.Chat(ctx)
	if err != nil {
		log.Println("Error starting Chat stream:", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Errror starting chat stream"))
		return
	}

	// Логируем успешное соединение
	log.Println("WebSocket connection established")

	// sending messages
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading WebSocket message:", err)
				return
			}

			var req pb.ChatMessage
			if err := json.Unmarshal(msg, &req); err != nil {
				log.Println("Error parsing JSON:", err)
				continue
			}

			if err := stream.Send(&req); err != nil {
				log.Println("Error sending message to gRPC stream:", err)
				return
			}
		}
	}()

	// receiving messages
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				log.Println("Error receiving message from gRPC stream", err)
				return
			}

			messageJSON, err := json.Marshal(map[string]interface{}{
				"senderUsername": resp.Senderusername,
				"content":        resp.Content,
				"timestamp":      resp.Timestamp,
			})
			if err != nil {
				log.Println("Error marshalling JSON:", err)
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
				log.Println("Error sending WebSocket message:", err)
				return
			}
		}
	}()

	select {}
}
