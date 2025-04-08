# Use the official Go image with version 1.23 as the base image for building the application
FROM golang:1.23 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy the application source code
COPY . .

# Build the Go application
RUN go build -o outbox-sidecar ./cmd/outbox-sidecar

# Use a newer base image with GLIBC 2.32 or higher
FROM debian:bookworm-slim

# Set the working directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/outbox-sidecar .

# Copy the configuration file (if needed for runtime)
COPY ./cmd/outbox-sidecar/sidecar.yaml ./sidecar.yaml

# Expose the port your application listens on (if applicable)
EXPOSE 8080

# Set environment variables (optional, can also be passed at runtime)
ENV CONFIG_PATH=./sidecar.yaml

# Command to run the application
CMD ["./outbox-sidecar"]