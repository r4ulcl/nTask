# STEP 1: Compile a static Go binary
FROM golang:1.23-alpine AS builder

# Install git for fetching dependencies, and ca-certificates for HTTPS
RUN apk add --no-cache git ca-certificates

WORKDIR /go/src/github.com/r4ulcl/nTask

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy only necessary files (dockerignore excludes extras)
COPY . .

# Build static binary
RUN CGO_ENABLED=0 \
    GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /nTask ./main.go


# STEP 2 build a small image
# start from kali for worker
FROM kalilinux/kali-rolling:latest
#GOPATH doesn-t exists in scratch
ENV GOPATH='/go' 

RUN apt-get update && apt-get install -y procps net-tools curl nmap \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

# Add root certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary and assets
COPY --from=builder /nTask /nTask
COPY --from=builder /go/src/github.com/r4ulcl/nTask/worker/modules /config/modules
COPY --from=builder /go/src/github.com/r4ulcl/nTask/docs /config/docs


# Create config directory and set workdir
WORKDIR /config

ENTRYPOINT ["/nTask"]