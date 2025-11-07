.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down deps

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $1, $2}'

build: ## Build the application
	go build -o bin/app cmd/main.go

run: ## Run the application
	go run cmd/main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build files
	rm -rf bin/

docker-up: ## Start Docker containers
	docker compose up -d || docker-compose up -d

docker-down: ## Stop Docker containers
	docker compose down || docker-compose down

migrate-up: ## Run database migrations
	migrate -path migrations -database "postgresql://postgres:password@localhost:5432/shipment_quality?sslmode=disable" up

migrate-down: ## Rollback database migrations
	migrate -path migrations -database "postgresql://postgres:password@localhost:5432/shipment_quality?sslmode=disable" down

deps: ## Download dependencies
	go mod download
	go mod tidy
