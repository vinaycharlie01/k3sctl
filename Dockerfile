FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG GIT_COMMIT=unknown

ENV CGO_ENABLED=0

RUN go build -trimpath \
      -ldflags "-s -w \
        -X github.com/vinaycharlie01/k3sctl/version.Version=${VERSION} \
        -X github.com/vinaycharlie01/k3sctl/version.GitCommit=${GIT_COMMIT}" \
      -o /out/k3sctl .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates openssh-client tini

COPY --from=builder /out/k3sctl /usr/local/bin/k3sctl

ENTRYPOINT ["/sbin/tini", "--", "k3sctl"]
CMD ["--help"]
