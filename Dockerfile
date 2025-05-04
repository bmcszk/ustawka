# Build stage
FROM golang:1.24.2-alpine AS builder

# Install git, SQLite and gcc for CGO
RUN apk add --no-cache git sqlite gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o ustawka

# Create data directory for SQLite database
RUN mkdir -p /app/data

# Final stage
FROM alpine:latest

# Install SQLite for runtime
RUN apk add --no-cache sqlite

# Set working directory
WORKDIR /app

# Copy binary and assets from builder
COPY --from=builder /app/ustawka .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/data ./data

# Set environment variables
ENV SEJM_API_TIMEOUT=15s
ENV SEJM_CACHE_TTL=24h
ENV SEJM_DB_PATH=/app/data/sejm.db
ENV USTAWKA_PORT=8080

# Expose default port
EXPOSE 8080

# Run the application
CMD ["./ustawka"] 
