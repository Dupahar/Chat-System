# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy shared packages
COPY pkg ./pkg
COPY go.mod go.sum ./

# Copy service code
COPY apps/messaging ./apps/messaging

# Build
RUN go build -o messaging ./apps/messaging

# Run Stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/messaging .

CMD ["./messaging"]
