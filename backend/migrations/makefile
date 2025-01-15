MIGRATE=migrate
DB_URL=postgres://admin:topsecret@localhost:5432/messenger_db?sslmode=disable
MIGRATION_PATH=.

migrate-up:
	$(MIGRATE) -database "$(DB_URL)" -path $(MIGRATION_PATH) up

migrate-down:
	$(MIGRATE) -database "$(DB_URL)" -path $(MIGRATION_PATH) down

migrate-version:
	$(MIGRATE) -database "$(DB_URL)" -path $(MIGRATION_PATH) version

migrate-create:
	$(MIGRATE) create -ext sql -dir . -seq $(name)