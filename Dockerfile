FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w \
    -X github.com/vinaycharlie01/k3sctl/version.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev) \
    -X github.com/vinaycharlie01/k3sctl/version.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" \
    -o k3sctl .

FROM alpine:3.20

RUN apk add --no-cache openssh-client

COPY --from=builder /build/k3sctl /usr/local/bin/k3sctl

ENTRYPOINT ["k3sctl"]
