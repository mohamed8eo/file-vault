.PHONY: build run test clean deps fmt lint sqlc migrate

BINARY_NAME=file-vault
BINARY_PATH=tmp/$(BINARY_NAME)
MAIN_PATH=cmd/api/main.go
dev:
	air

build:
	go build -o $(BINARY_PATH) $(MAIN_PATH)

run: build
	./$(BINARY_PATH)

test:
	go test ./...

clean:
	rm -f $(BINARY_PATH)

deps:
	go mod download
	go mod tidy

fmt:
	go fmt ./...

lint:
	go vet ./...

sqlc:
	sqlc generate

migrate:
	goose -dir sql/schema postgres "$(DB_URL)" up

migrate-down:
	goose -dir sql/schema postgres "$(DB_URL)" down

.DEFAULT_GOAL := build
