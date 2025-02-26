# Step 1: Build the Go application (builder stage)
FROM golang:1.23-alpine AS builder

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

# Copy the static directory
COPY --from=builder /app/static ./static

COPY --from=builder /app/views/*.html ./views/

# Copy .env file
COPY --from=builder /app/.env ./

RUN mkdir -p /app/data

# Expose the port the app will run on
EXPOSE 8081

# Command to run the application
CMD ["./server"]
