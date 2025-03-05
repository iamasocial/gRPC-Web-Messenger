import { UserServiceClient } from "../../proto/user_service_grpc_web_pb";
import { ChatServiceClient } from "../../proto/chat_service_grpc_web_pb";

const SERVER_URL = "http://localhost:8888";

export const userClient = new UserServiceClient(SERVER_URL);
export const chatClient = new ChatServiceClient(SERVER_URL);