# Multi-stage build for archfit.
# Produces a minimal scratch-based image with only the static binary.
# No CGO, no shell, no OS — just the binary.

FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags "-s -w -X github.com/shibuiwilliam/archfit/internal/version.Version=${VERSION}" \
    -o /archfit ./cmd/archfit

# ---

FROM scratch
COPY --from=builder /archfit /archfit
ENTRYPOINT ["/archfit"]
