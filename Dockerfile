# --- Stage 1: Build binary ---
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Set Go module download environment variables
ENV GONOSUMDB=*
ENV GOPROXY=https://proxy.golang.org,direct

# Copy dependency definitions
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build lightweight static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o telegram-git-pusher ./cmd/bot

# --- Stage 2: Runtime image ---
FROM alpine:latest

# Install runtime dependencies (git, ssh client, ca-certificates, tzdata)
RUN apk add --no-cache git openssh-client ca-certificates tzdata

WORKDIR /app

# Create directories for persistent data and temporary workspace
RUN mkdir -p /app/data /app/tmp_repos

# Copy compiled binary from builder stage
COPY --from=builder /app/telegram-git-pusher /app/telegram-git-pusher

# Expose environment variables defaults
ENV WORK_DIR=/app/tmp_repos \
    DB_PATH=/app/data/bot.db \
    DRY_RUN=false

# Run binary
ENTRYPOINT ["/app/telegram-git-pusher"]
