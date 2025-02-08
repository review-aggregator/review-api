# Load environment variables from .env file
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Go parameters
APP_NAME=myapp
BUILD_DIR=bin
SRC_DIR=.
GO_FILES=$(SRC_DIR)/*.go

# Database migration settings
MIGRATIONS_DIR=migrations
MIGRATE=migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)"

.PHONY: build run test clean db-up db-down db-down-all db-force db-create db-drop db-version help

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	go build -o $(BUILD_DIR)/$(APP_NAME) $(GO_FILES)

# Run the application
run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# Run tests
test:
	@echo "Running tests..."
	go test ./... -v

# Remove built binaries
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)

# Database migration commands
db-up:
	docker run --name postgres -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -p 5432:5432 -d postgres

db-down:
	docker exec -it postgres psql -U root -p 5432 -d postgres -exec "DROP DATABASE reviews;"

db-connect:
	docker exec -it postgres psql -U root -p 5432 -d postgres

db-db:
	docker exec -it postgres psql -U root -p 5432 -d postgres -exec "CREATE DATABASE reviews;"

db-migrate-up:
	$(MIGRATE) up

db-migrate-down:
	$(MIGRATE) down 1

db-down-all:
	$(MIGRATE) down

db-force:
	$(MIGRATE) force $(VERSION)

db-create:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)

db-drop:
	$(MIGRATE) drop

db-version:
	$(MIGRATE) version

db-sqlpad:
	docker run -d \
	--name sqlpad \
	-p 3000:3000 \
	-v sqlpad_data:/var/lib/sqlpad \
	-e SQLPAD_ADMIN="admin" \
	-e SQLPAD_ADMIN_PASSWORD="admin123" \
	sqlpad/sqlpad:latest


# Help command
help:
	@echo "Available commands:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Build and run the application"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Remove built binaries"
	@echo ""
	@echo "Database migration commands:"
	@echo "  make db-up        - Apply all migrations"
	@echo "  make db-down      - Rollback the last migration"
	@echo "  make db-down-all  - Rollback all migrations"
	@echo "  make db-force VERSION=<version> - Force set migration version"
	@echo "  make db-create NAME=<migration_name> - Create a new migration file"
	@echo "  make db-drop      - Drop the database (Caution: Destructive)"
	@echo "  make db-version   - Show the current migration version"
