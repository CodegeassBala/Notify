include .env

APP_NAME=notification-app
CMD_PATH=./cmd/main.go
IMAGE_NAME=notification-app
POSTGRES_USER=${DB_USER}
POSTGRES_PASSWORD=${DB_PASSWORD}
POSTGRES_DB=${DB_NAME}
POSTGRES_PORT=${DB_PORT}
DB_URL=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:${POSTGRES_PORT}/$(POSTGRES_DB)?sslmode=disable

.PHONY: build run docker-build docker-run clean

build:
	go build -o $(APP_NAME) $(CMD_PATH)

run: build
	 ./$(APP_NAME)

dev: 
	go run $(CMD_PATH)

docker-run:
	docker compose up -d

clean:
	rm -f $(APP_NAME)
	docker rm -f $(IMAGE_NAME)-container || true
	docker rmi -f $(IMAGE_NAME) || true

migrate-up:
	migrate -path ./database/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path ./database/migrations -database "$(DB_URL)" down

force-clean:
	migrate -path ./database/migrations -database "$(DB_URL)" drop -f

migrate-down-to:
	@read -p "Enter migration version to rollback to: " version; \
	migrate -path ./database/migrations -database "$(DB_URL)" down $$version

start: 
	make docker-run && make run

schema:
	docker exec -t postgres-container pg_dump -U postgres --schema-only --no-owner --no-comments --quote-all-identifiers notify > database/schema.sql
	sqlc generate --file database/sqlc.yaml

