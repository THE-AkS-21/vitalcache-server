# ---- Build Stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build deps
RUN apk add --no-cache git

# Download dependencies first (cached if go.mod/go.sum don't change)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary with optimizations
# -ldflags="-s -w" strips debug info for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/api

# ---- Runtime Stage ----
FROM alpine:3.19

# Install CA certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Create unprivileged user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /home/appuser

# Copy binary from builder
COPY --from=builder /app/server ./server

# Expose port
EXPOSE 8080

# Run the service
ENTRYPOINT ["./server"]
