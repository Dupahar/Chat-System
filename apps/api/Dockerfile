# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy shared packages
COPY pkg ./pkg
COPY go.mod go.sum ./

# Copy service code
COPY apps/api ./apps/api

# Build
RUN go build -o api ./apps/api

# Run Stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/api .

EXPOSE 8081

CMD ["./api"]
