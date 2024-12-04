build:
	@go build -o bin/goecom cmd/main.go

test:
	@go test -v ./...
	
run: build
	@./bin/goecom

migration: 
	@migrate create -ext sql -dir cmd/migrate/migrations $(filter-out $@,$(MAKECMDGOALS))

migration-up:
	@go run cmd/migrate/main.go up

migration-down:
	@go run cmd/migrate/main.go down