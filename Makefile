.PHONY: build build-all clean test init

APP_NAME = wikitnow
BUILD_DIR = bin
CMD_PATH = ./cmd/wikitnow

build:
	@echo "Building $(APP_NAME)..."
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

build-all: clean
	@echo "Cross-compiling $(APP_NAME)..."
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_PATH)
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_PATH)
	@GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_PATH)
	@echo "Cross-compilation complete."

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

init:
	@go mod tidy
