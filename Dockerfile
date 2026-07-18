# --- Build Stage ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy source file
COPY data-fetcher/main.go ./

# Initialize module and build a statically linked binary
RUN go mod init data-fetcher && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o data-fetcher main.go

# --- Final Stage ---
FROM alpine:3.19

WORKDIR /

# Copy statically compiled binary from builder
COPY --from=builder /app/data-fetcher /data-fetcher

# Expose API server port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/data-fetcher"]
