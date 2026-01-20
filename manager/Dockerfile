# STEP 1: Compile a static Go binary
FROM golang:1.23-alpine AS builder

# Install ca-certificates only
RUN apk add --no-cache ca-certificates

WORKDIR /go/src/github.com/r4ulcl/nTask

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source (dockerignore limits context)
COPY . .

# Build static binary with optimizations
RUN CGO_ENABLED=0 \
    GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /nTask ./main.go

# STEP 2: Minimal runtime image
FROM scratch

# Add CA certificates for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary
COPY --from=builder /nTask /nTask

# Copy configuration and assets
COPY --from=builder /go/src/github.com/r4ulcl/nTask/worker/modules /config/modules
COPY --from=builder /go/src/github.com/r4ulcl/nTask/docs /config/docs

# Set working directory
WORKDIR /config

ENTRYPOINT ["/nTask"]