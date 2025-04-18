.DEFAULT_GOAL := all

# Build the application
all: fmt vet build

build:
	@echo "Building..."
	@go build -o main cmd/bot/main.go

# Run the application
run:
	@go run cmd/bot/main.go

fmt:
	@echo "Formatting..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main