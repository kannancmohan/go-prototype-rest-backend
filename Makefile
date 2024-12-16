MIGRATIONS_PATH = ./cmd/migrate/migrations

.PHONY: gofmt
gofmt:
	@find . -type f -name '*.go' -not -path './vendor/*' -not -path './pkg/mod/*' -exec gofmt -s -w {} +

.PHONY: build
build:
	@go build -o bin/rest-api cmd/api/main.go

.PHONY: test
test:
	@go test -v ./...

.PHONY: run
run: build
	@./bin/rest-api

.PHONY: migrate-create
migration-create: 
	@migrate create -seq -ext sql -dir $(MIGRATIONS_PATH) $(filter-out $@,$(MAKECMDGOALS))

.PHONY: migrate-up
migration-up:
	@migrate -path=$(MIGRATIONS_PATH) -database=$(DB_ADDR) up

.PHONY: migrate-down
migration-down:
	@migrate -path=$(MIGRATIONS_PATH) -database=$(DB_ADDR) down $(filter-out $@,$(MAKECMDGOALS))