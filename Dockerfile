FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod ./
# Copy all source code (including locustfile.py)
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o key-value-cache .

# Use a smaller image for the final container
FROM alpine:latest

WORKDIR /root/

# Copy the binary and locustfile.py from the builder stage
COPY --from=builder /app/key-value-cache .


# Expose the application port and the Locust Web UI port
EXPOSE 7171

CMD sh -c "./key-value-cache"
