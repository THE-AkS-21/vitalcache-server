# ---- Build Stage ----
# Use the official Golang image to build the application
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the application, creating a static binary.
# CGO_ENABLED=0 is important for creating a static binary for Alpine.
# -o /app/server creates the output binary named 'server' in the /app directory.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

# ---- Final Stage ----
# Use a minimal, non-root Alpine image for the final container
FROM alpine:latest

# Set a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /home/appuser

# Copy ONLY the compiled binary from the builder stage
COPY --from=builder /app/server .

# Expose the port the app runs on
EXPOSE 8080

# The command to run when the container starts
ENTRYPOINT ["./server"]