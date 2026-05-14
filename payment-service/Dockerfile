FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
ENV GONOSUMDB=*
ENV GOFLAGS=-mod=mod
COPY . .
RUN go build -o payment-service ./cmd/payment-service

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/payment-service .
EXPOSE 8081
CMD ["./payment-service"]
