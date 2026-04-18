# Multi-stage build for Pacer API

# Stage 1: Builder  
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pacer-api .

# Stage 2: Runtime
FROM debian:bookworm-slim

# Install ca-certificates for HTTPS
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/pacer-api .

# Copy migrations
COPY --from=builder /app/db/migrations ./db/migrations

# Expose port
EXPOSE 3001

# Run the binary
CMD ["./pacer-api"]
