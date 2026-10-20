BINARY_NAME := k3sctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w \
	-X github.com/vinaycharlie01/k3sctl/version.Version=$(VERSION) \
	-X github.com/vinaycharlie01/k3sctl/version.GitCommit=$(GIT_COMMIT)

.PHONY: build clean test vet fmt check

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check-fmt:
	@test -z "$$(gofmt -l .)" || { echo "Files need formatting:"; gofmt -l .; exit 1; }

check: check-fmt vet test

lint:
	golangci-lint run ./...
