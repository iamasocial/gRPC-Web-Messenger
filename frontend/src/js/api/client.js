import { ChatServiceClient } from "../../proto/chat_service_grpc_web_pb";
import { UserServiceClient } from "../../proto/user_service_grpc_web_pb";
import { FileServiceClient } from "../../proto/file_service_grpc_web_pb";
import { KeyExchangeServiceClient } from "../../proto/key_exchange_service_grpc_web_pb";

const SERVER_URL = "http://localhost:8888";

export const chatClient = new ChatServiceClient(SERVER_URL);
export const userClient = new UserServiceClient(SERVER_URL);
export const fileClient = new FileServiceClient(SERVER_URL);
export const keyExchangeClient = new KeyExchangeServiceClient(SERVER_URL);