FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
ENV GONOSUMDB=*
ENV GOFLAGS=-mod=mod
COPY . .
RUN go build -o order-service ./cmd/order-service

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/order-service .
EXPOSE 8082
CMD ["./order-service"]
