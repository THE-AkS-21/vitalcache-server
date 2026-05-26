# Build stage
FROM golang:1.22-alpine AS builder

# Install git and certificates
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 ensures a static binary
# -ldflags "-s -w" removes debug symbols for a smaller binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o vitalcache-server ./cmd/api

# Final minimal image
FROM scratch

# Import certs from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the compiled binary from builder
COPY --from=builder /app/vitalcache-server /vitalcache-server

# Create a non-root user (optional but recommended)
# We can't use adduser in scratch, so we just specify a numeric UID
USER 1000

# Expose the API port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/vitalcache-server"]