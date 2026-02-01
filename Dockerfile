# Build stage
FROM golang:1.25-alpine3.23 AS builder
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your application code
COPY . .

# Build your Go app
RUN go build -o main .

# Final stage: minimal runtime image
FROM alpine:latest
WORKDIR /app

# Update and install any needed packages
RUN apk --no-cache upgrade

COPY --from=builder /app/main .

# Expose your app's port
EXPOSE 8081

# Command to run your app
CMD ["./main"]
