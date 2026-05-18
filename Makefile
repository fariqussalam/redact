.PHONY: build test test-race test-cover vet lint fmt tidy clean

BINARY_NAME := redact
BUILD_DIR := dist

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/redact

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -cover ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR) coverage.out
