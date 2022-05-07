BIN := "./bin/gomigrator"

GIT_HASH := $(shell git log --format="%h" -n 1)
LDFLAGS := -X main.release="develop" -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S) -X main.gitHash=$(GIT_HASH)

test:
	go test -race -count 100 ./pkg/...

test-integration:
	go test -count 1 -tags=integration ./pkg/...

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.45.2

lint: install-lint-deps
	golangci-lint run ./...

generate:
	go generate ./...

build:
	go build -v -o $(BIN) -ldflags "$(LDFLAGS)" ./cmd/gomigrator
