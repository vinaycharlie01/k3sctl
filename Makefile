GO_SRCS := $(shell find . -type f -name '*.go' -a ! \( -name 'zz_generated*' -o -name '*_test.go' \))
GO_TESTS := $(shell find . -type f -name '*_test.go')
TAG_NAME = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_COMMIT = $(shell git rev-parse --short=7 HEAD)
PREFIX = /usr/local

LD_FLAGS = -s -w -X github.com/vinaycharlie01/k3sctl/version.Version=$(TAG_NAME) -X github.com/vinaycharlie01/k3sctl/version.GitCommit=$(GIT_COMMIT)
BUILD_FLAGS = -trimpath -a -tags "netgo,osusergo,static_build" -installsuffix netgo -ldflags "$(LD_FLAGS) -extldflags '-static'"

BINARY_NAME := k3sctl

.PHONY: build clean test vet fmt check-fmt check lint build-all install

build: $(GO_SRCS)
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY_NAME) .

bin/k3sctl-linux-amd64: $(GO_SRCS)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ .

bin/k3sctl-linux-arm64: $(GO_SRCS)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ .

bin/k3sctl-linux-arm: $(GO_SRCS)
	GOOS=linux GOARCH=arm CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ .

bin/k3sctl-darwin-amd64: $(GO_SRCS)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ .

bin/k3sctl-darwin-arm64: $(GO_SRCS)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ .

bin/k3sctl-win-amd64.exe: $(GO_SRCS)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ .

bins := k3sctl-linux-amd64 k3sctl-linux-arm64 k3sctl-linux-arm k3sctl-darwin-amd64 k3sctl-darwin-arm64 k3sctl-win-amd64.exe

bin/checksums.txt: $(addprefix bin/,$(bins))
	sha256sum -b $(addprefix bin/,$(bins)) | sed 's/bin\///' > $@

bin/checksums.md: bin/checksums.txt
	@echo "### SHA256 Checksums" > $@
	@echo >> $@
	@echo '```' >> $@
	@cat $< >> $@
	@echo '```' >> $@

build-all: $(addprefix bin/,$(bins)) bin/checksums.md

clean:
	rm -rf bin/ $(BINARY_NAME)

test: $(GO_SRCS) $(GO_TESTS)
	go test -v ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check-fmt:
	@test -z "$$(gofmt -l .)" || { echo "Files need formatting:"; gofmt -l .; exit 1; }

lint:
	golangci-lint run ./...

check: check-fmt vet test

install: build
	install -d $(DESTDIR)$(PREFIX)/bin/
	install -m 755 $(BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/
