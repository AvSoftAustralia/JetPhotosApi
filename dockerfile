# Start from the official Go image
FROM golang:1.24 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go app
RUN go build -o /app/app ./cmd/api

# Start a fresh container to keep it lean
FROM debian:bookworm-slim

# Install certificates (needed for HTTPS)
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /root/

# Copy the built binary from the builder
COPY --from=builder /app/app .

# Expose the port the app runs on
EXPOSE 4000

# Command to run the executable
CMD ["./app", "-addr=:4000"]