# Step 1: Build the Go application (builder stage)
FROM golang:1.23-alpine AS builder  # Updated to Go 1.23

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./

# Download and cache dependencies
RUN go mod download

# Copy the source code to the container
COPY . .

# Build the Go application
RUN go build -o server .

# Step 2: Create a lightweight image to run the application
FROM alpine:latest

# Set working directory for the app
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server .

# Expose the port the app will run on
EXPOSE 8080

# Command to run the application
CMD ["./server"]
