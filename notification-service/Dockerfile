FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod tidy
COPY . .
RUN go build -o notification-service ./cmd/notification-service

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/notification-service .
CMD ["./notification-service"]
