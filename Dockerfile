# ---- Build Stage ----
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build deps (git needed for private Go modules sometimes)
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

# ---- Runtime Stage ----
FROM alpine:3.19

# Install CA certificates (REQUIRED for AWS + Supabase HTTPS)
RUN apk add --no-cache ca-certificates tzdata \
    && update-ca-certificates

# Create unprivileged user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /home/appuser

# Copy binary
COPY --from=builder /app/server ./server

# Make sure it's executable
RUN chmod +x ./server

# Expose port (can be overridden by env var at runtime)
EXPOSE 8080

# Optional: Health check (depends on Gin server having /health)
# HEALTHCHECK --interval=30s --timeout=3s \
#   CMD wget -qO- http://localhost:8080/health || exit 1

# Run the service
ENTRYPOINT ["./server"]
