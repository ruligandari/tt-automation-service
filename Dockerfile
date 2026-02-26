# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod and sum files
COPY go.mod ./

# Download all dependencies.
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
# Use ARG to allow multi-arch build
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -installsuffix cgo -o main cmd/server/main.go || \
    CGO_ENABLED=0 go build -a -installsuffix cgo -o main cmd/server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .
# Copy ca-certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose port (can be overridden by docker-compose)
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
