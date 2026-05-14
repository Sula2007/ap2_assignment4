# AP2 Assignment 4 – Performance Optimization & External Integrations

## Architecture Overview

```
                         ┌─────────────────────────────────────────────────────────┐
                         │                     Redis                               │
                         │  • Order cache (cache-aside, TTL 5m)                   │
                         │  • Rate limiter counters (per IP)                       │
                         │  • Idempotency keys (processed events, TTL 72h)         │
                         └────────────┬──────────────────────────┬─────────────────┘
                                      │                          │
        [Client]                      │                          │
           │                          │                          │
           ▼                          │                          │
   [Order Service :8082] ─────────────┘      ┌─────────────────[Notification Service]
           │                                  │                          ▲
           │ HTTP                             │   RabbitMQ               │
           ▼                                  │  exchange: payments       │
   [Payment Service :8081] ───────────────────┘  queue: payment.completed │
           │                                                              │
        order-db                                                    notification-db
      payment-db
```

## How to Run

```bash
docker-compose up --build
```

- **Order Service:** http://localhost:8082
- **Payment Service:** http://localhost:8081
- **RabbitMQ Management UI:** http://localhost:15672 (guest / guest)

### Example Requests

```bash
curl -X POST http://localhost:8082/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"user-1","item_name":"Book","amount":4999}'

curl http://localhost:8082/api/v1/orders/<order_id>

curl -X PATCH http://localhost:8082/api/v1/orders/<order_id>/cancel
```

---

## Caching Strategy (Cache-Aside)

**Read path** (`GET /orders/:id`):
1. Check Redis key `order:<id>`
2. On hit → return cached value (no DB query)
3. On miss → query PostgreSQL, write result to Redis with TTL, return

**Write path** (status change via `CreateOrder` / `CancelOrder`):
- After every `UpdateStatus` call the corresponding Redis key is **deleted immediately** (atomic invalidation)
- Next read will repopulate the cache from the fresh DB value

**TTL:** Configurable via `CACHE_TTL_SECONDS` (default 300 seconds / 5 minutes).

---

## Retry & Exponential Backoff

The Notification Service retries failed email sends with exponential backoff:

| Attempt | Delay before retry |
|---|---|
| 1 (initial) | — |
| 2 | 2 s |
| 3 | 4 s |
| 4 | 8 s |

Formula: `delay = 2^attempt seconds`. Max retries configured via `MAX_RETRIES` env var (default 3).

If all attempts fail, the RabbitMQ message is NACKed and either requeued (< 3 total deliveries) or routed to the Dead Letter Queue.

---

## Idempotency Strategy

Every `PaymentEvent` carries a unique `event_id`. Before sending a notification:

1. Check Redis key `processed:<event_id>`
2. If **exists** → skip (duplicate), ACK the message
3. If **not found** → send email via provider, then `SET processed:<event_id> 1 EX 259200` (72 h TTL)

Redis is used (not Postgres) for fast O(1) lookup and automatic expiry.

---

## External Provider Adapter

The `EmailSender` interface decouples business logic from the email provider:

```go
type EmailSender interface {
    Send(to, subject, body string) error
}
```

| `PROVIDER_MODE` | Implementation | Behavior |
|---|---|---|
| `SIMULATED` (default) | `MockEmailSender` | Random 100–500 ms latency, 25% random failure rate |
| `REAL` | `MailjetEmailSender` | HTTP call to Mailjet API v3.1 |

Switch via `.env`:
```
PROVIDER_MODE=REAL
MAILJET_API_KEY=...
MAILJET_SECRET_KEY=...
MAILJET_SENDER_EMAIL=no-reply@example.com
```

---

## Rate Limiter (Bonus)

Redis-backed middleware on the Order Service limits each client IP to `RATE_LIMIT_REQUESTS` requests per `RATE_LIMIT_WINDOW_SECONDS` seconds (default: 10 req / 60 s).

Returns `HTTP 429 Too Many Requests` when the limit is exceeded.

Implementation uses a Redis `INCR` + `EXPIRE` pipeline per IP key for atomic, low-latency counting.

---

## Configuration (.env)

| Variable | Default | Description |
|---|---|---|
| `CACHE_TTL_SECONDS` | `300` | Order cache TTL |
| `RATE_LIMIT_REQUESTS` | `10` | Max requests per window |
| `RATE_LIMIT_WINDOW_SECONDS` | `60` | Rate limit window |
| `PROVIDER_MODE` | `SIMULATED` | Email provider (`SIMULATED`/`REAL`) |
| `MAILJET_API_KEY` | — | Mailjet API key (REAL mode) |
| `MAILJET_SECRET_KEY` | — | Mailjet secret key (REAL mode) |
| `MAILJET_SENDER_EMAIL` | — | Sender email (REAL mode) |
| `MAX_RETRIES` | `3` | Max email send retries |

---

## Project Structure

```
ap2_assignment4/
├── .env
├── docker-compose.yml
├── order-service/
│   ├── cmd/order-service/main.go
│   └── internal/
│       ├── domain/
│       ├── repository/
│       │   ├── order_repository.go        ← DB layer
│       │   ├── order_cache.go             ← Redis cache-aside
│       │   └── payment_client.go
│       ├── transport/
│       │   ├── handler/order_handler.go
│       │   └── middleware/rate_limiter.go ← Redis rate limiter (bonus)
│       └── usecase/order_usecase.go       ← cache read/write/invalidate
├── payment-service/
│   └── ...
└── notification-service/
    ├── cmd/notification-service/main.go
    └── internal/
        ├── domain/event.go
        ├── provider/
        │   ├── email_sender.go            ← interface (adapter port)
        │   ├── mock_email_sender.go       ← simulated provider
        │   └── mailjet_email_sender.go    ← real Mailjet adapter
        ├── repository/idempotency_store.go ← Redis idempotency
        ├── usecase/notification_usecase.go ← backoff retry logic
        └── messaging/consumer.go           ← RabbitMQ worker
```
