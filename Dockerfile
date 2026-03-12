# 1. Build stage (Updated to 1.25)
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git
WORKDIR /app

# Copy dependency files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /main .

# 2. Run stage
FROM alpine:3.21

RUN apk update && apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /main .

# Fix: Since you are already in /app, just run ./main
CMD ["./main"]