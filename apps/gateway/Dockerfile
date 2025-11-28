# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy shared packages
COPY pkg ./pkg
COPY go.mod go.sum ./

# Copy service code
COPY apps/gateway ./apps/gateway

# Build
RUN go build -o gateway ./apps/gateway

# Run Stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gateway .

EXPOSE 8080

CMD ["./gateway"]
