MIGRATIONS_PATH = ./cmd/migrate/migrations

gofmt:
	@find . -type f -name '*.go' -not -path './vendor/*' -not -path './pkg/mod/*' -exec gofmt -s -w {} +

build:
	@go build -o bin/rest-api cmd/api/main.go

test:
	@go test -v ./...
	
run: build
	@./bin/rest-api

migration-create: 
	@migrate create -seq -ext sql -dir $(MIGRATIONS_PATH) $(filter-out $@,$(MAKECMDGOALS))

migration-up:
	@migrate -path=$(MIGRATIONS_PATH) -database=$(DB_ADDR) up

migration-down:
	@migrate -path=$(MIGRATIONS_PATH) -database=$(DB_ADDR) down $(filter-out $@,$(MAKECMDGOALS))