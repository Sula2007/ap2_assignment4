FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
ENV GONOSUMDB=*
ENV GOFLAGS=-mod=mod
COPY . .
RUN go build -o notification-service ./cmd/notification-service

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/notification-service .
CMD ["./notification-service"]
