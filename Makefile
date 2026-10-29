.PHONY: run test test-integration lint seed migrate-up migrate-down swagger cover

run:
	go run ./cmd/api

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

lint:
	gofmt -l .
	go vet ./...
	golangci-lint run

seed:
	go run ./cmd/seed

migrate-up:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/gin_boilerplate?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/gin_boilerplate?sslmode=disable" down

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go -o swagger --parseInternal

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
