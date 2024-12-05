gofmt:
	@find . -type f -name '*.go' -not -path './vendor/*' -not -path './pkg/mod/*' -exec gofmt -s -w {} +

build:
	@go build -o bin/rest-api cmd/api/main.go

test:
	@go test -v ./...
	
run: build
	@./bin/rest-api

migration: 
	@migrate create -ext sql -dir cmd/migrate/migrations $(filter-out $@,$(MAKECMDGOALS))

migration-up:
	@go run cmd/migrate/main.go up

migration-down:
	@go run cmd/migrate/main.go down