.PHONY: build run test test-race lint fmt vet tidy clean

BINARY := bot
PKG := ./...

build: ## Build the bot binary
	go build -o $(BINARY) ./cmd/bot

run: ## Run the bot
	go run ./cmd/bot

test: ## Run tests
	go test $(PKG)

test-race: ## Run tests with the race detector
	go test -race $(PKG)

lint: ## Run golangci-lint (install: https://golangci-lint.run/welcome/install/)
	golangci-lint run $(PKG)

fmt: ## Format the code
	gofmt -w cmd internal pkg

vet: ## Run go vet
	go vet $(PKG)

tidy: ## Tidy go.mod / go.sum
	go mod tidy

clean: ## Remove build artifacts
	rm -f $(BINARY)
