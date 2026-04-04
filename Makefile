.PHONY: build build-all clean test init

APP_NAME = wikitnow
BUILD_DIR = bin
CMD_PATH = ./cmd/wikitnow

VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILDDATE := $(shell date -u +%Y-%m-%d)
LDFLAGS   := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILDDATE)"

build:
	@echo "Building $(APP_NAME) $(VERSION)..."
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

build-all: clean
	@echo "Cross-compiling $(APP_NAME) $(VERSION)..."
	@GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64       $(CMD_PATH)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_PATH)
	@GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64      $(CMD_PATH)
	@GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64      $(CMD_PATH)
	@echo "Cross-compilation complete."

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

init:
	@go mod tidy
