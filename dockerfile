# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build evcc
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o evcc \
    ./cmd/evcc

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/evcc .

# Copy templates and assets
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/assets ./assets

# Create data directory
RUN mkdir -p /app/data

EXPOSE 7070

CMD ["./evcc"]
