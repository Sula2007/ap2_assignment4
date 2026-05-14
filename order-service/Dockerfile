FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod tidy
COPY . .
RUN go build -o order-service ./cmd/order-service

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/order-service .
EXPOSE 8082
CMD ["./order-service"]
