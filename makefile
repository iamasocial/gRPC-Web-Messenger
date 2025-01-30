# migrations
MIGRATE=migrate
DB_URL=postgres://admin:topsecret@localhost:5432/messenger_db?sslmode=disable
MIGRATION_PATH=backend/migrations

migrate-up:
	$(MIGRATE) -database "$(DB_URL)" -path $(MIGRATION_PATH) up

migrate-down:
	$(MIGRATE) -database "$(DB_URL)" -path $(MIGRATION_PATH) down

migrate-version:
	$(MIGRATE) -database "$(DB_URL)" -path $(MIGRATION_PATH) version

migrate-create:
	$(MIGRATE) create -ext sql -dir . -seq $(name)

# protobuf generation
PROTO_DIR = proto
GEN_DIR = backend

generate-proto:
	@protoc --go_out=$(GEN_DIR) --go-grpc_out=$(GEN_DIR) $(PROTO_DIR)/*.proto

# docker
DOCKER_COMPOSE_FILE = backend/docker/docker-compose.yml
CONTAINER_NAME = messenger-backend-container

docker-up:
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up --build -d && docker logs -f $(CONTAINER_NAME)

docker-down:
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down

logs:
	@docker logs -f $(CONTAINER_NAME)

