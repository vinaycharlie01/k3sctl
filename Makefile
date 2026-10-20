BINARY_NAME := k3sctl
DIST_DIR := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w \
	-X github.com/vinaycharlie01/k3sctl/version.Version=$(VERSION) \
	-X github.com/vinaycharlie01/k3sctl/version.GitCommit=$(GIT_COMMIT)

# k3s publishes amd64, arm64 and armhf builds.
PLATFORMS := linux/amd64 linux/arm64 linux/arm darwin/amd64 darwin/arm64

.PHONY: build build-all install clean test vet fmt check

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

build-all:
	@mkdir -p $(DIST_DIR)
	@set -e; \
	for platform in $(PLATFORMS); do \
		os=$${platform%%/*}; arch=$${platform##*/}; \
		output=$(DIST_DIR)/$(BINARY_NAME)-$${os}-$${arch}; \
		if [ "$$arch" = "arm" ]; then export GOARM=7; else unset GOARM; fi; \
		echo "building $$output"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o "$$output" .; \
	done

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -rf $(BINARY_NAME) $(DIST_DIR)

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
