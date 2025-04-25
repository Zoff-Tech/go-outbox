# Use the official Go image with version 1.23 as the base image for building the application
FROM golang:1.24 as builder

# Set the working directory inside the container
WORKDIR /app

# Set GOPROXY environment variable
ENV GOPROXY=https://proxy.golang.org,direct



COPY ./sidecart/go.mod  ./sidecart/go.mod 
COPY ./sidecart/go.sum  ./sidecart/go.sum 

COPY ./schema/go.mod ./schema/go.mod 

RUN go work init ./sidecart

RUN go work use ./schema

RUN go mod download -x

# Copy the application source code
COPY . .


# Download Go module dependencies
#RUN go mod download


# Build the Go application
RUN go build -o outbox-sidecar ./sidecart/service

# Use a newer base image with GLIBC 2.32 or higher
FROM debian:stable-slim

# Set the working directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/outbox-sidecar .

# Copy the configuration file (if needed for runtime)
COPY ./sidecart/service/config/sidecar.yaml ./config/sidecar.yaml

# Expose the port your application listens on (if applicable)
EXPOSE 8080

# Set environment variables (optional, can also be passed at runtime)
ENV CONFIG_PATH=./config/sidecar.yaml

# Command to run the application
CMD ["./outbox-sidecar"]