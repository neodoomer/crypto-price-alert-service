# Crypto Price Alert Service

A Go service where users create price alerts for crypto tokens. A background job checks live prices every 30 seconds via CoinGecko. When a token hits the target price, the service sends an HMAC-signed webhook callback.

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)

## Setup

### 1. Start PostgreSQL

```bash
docker compose up -d
```

This starts a PostgreSQL 16 container and pgAdmin. The `alerts` table is created automatically from `sql/schema.sql`. pgAdmin is available at **http://localhost:5050**. Log in with **Email address** `admin@localhost` and **Password** `admin` (pgAdmin requires a full email in the first field). To connect to the DB, add a server with host `postgres`, port `5432`, database `crypto_alerts`, user `crypto`, password `crypto`.

### 2. Generate SQLC Code (optional -- already committed)

```bash
sqlc generate
```

### 3. Run the API Server

```bash
go run cmd/server/main.go
```

The server starts on **port 8080**. By default it connects to `postgres://crypto:crypto@localhost:5432/crypto_alerts?sslmode=disable`. Override with the `DATABASE_URL` environment variable.

### 4. Run the Test Webhook Server

In a separate terminal:

```bash
go run cmd/testserver/main.go
```

This starts a tiny HTTP server on **port 9090** that prints incoming webhook requests. Optionally set `SECRET` to enable HMAC verification:

```bash
SECRET=my-secret-key go run cmd/testserver/main.go
```

## API Endpoints

You can call the API with **curl** (examples below) or with the [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) extension using the included `api.http` file.

### Using REST Client (`api.http`)

1. Install the **REST Client** extension in VS Code/Cursor (e.g. "REST Client" by Huachao Mao).
2. Open `api.http` in the project root.
3. Click **Send Request** above any request (or use the keyboard shortcut) to execute it.
4. The file uses `@baseUrl = http://localhost:8080`. For **Delete alert**, run **Create alert** first so `{{alertId}}` is set from the response, or replace it with a real alert UUID.

### Create Alert

```bash
curl -s -X POST http://localhost:8080/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "token": "bitcoin",
    "target_price": 100000,
    "direction": "above",
    "callback_url": "http://localhost:9090/webhook",
    "callback_secret": "my-secret-key"
  }' | jq
```

**Response (201 Created):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "token": "bitcoin",
  "target_price": "100000",
  "direction": "above",
  "callback_url": "http://localhost:9090/webhook",
  "triggered": false,
  "created_at": "2026-03-02T12:00:00Z",
  "updated_at": "2026-03-02T12:00:00Z"
}
```

### List Alerts

```bash
# All alerts
curl -s http://localhost:8080/alerts | jq

# Only active alerts
curl -s "http://localhost:8080/alerts?triggered=false" | jq

# Only triggered alerts
curl -s "http://localhost:8080/alerts?triggered=true" | jq
```

### Delete Alert

```bash
curl -s -X DELETE http://localhost:8080/alerts/<alert-id>
```

Returns **204 No Content** on success, **404 Not Found** if the alert doesn't exist or was already triggered.

### Health Check

```bash
curl -s http://localhost:8080/health | jq
```

## How to Test End-to-End

1. Start PostgreSQL: `docker compose up -d`
2. Start the test webhook server: `SECRET=my-secret-key go run cmd/testserver/main.go`
3. Start the API server: `go run cmd/server/main.go`
4. Look up a token's current price on [CoinGecko](https://www.coingecko.com/) (e.g. "bitcoin")
5. Create an alert with a target price close to the current price:

```bash
curl -s -X POST http://localhost:8080/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "token": "bitcoin",
    "target_price": 99999,
    "direction": "below",
    "callback_url": "http://localhost:9090/webhook",
    "callback_secret": "my-secret-key"
  }' | jq
```

6. Watch the test server terminal -- within 30 seconds the background job will check prices and fire the webhook if the condition is met.

## Project Structure

```
├── cmd/
│   ├── server/main.go          # Main API server (port 8080)
│   └── testserver/main.go      # Test webhook receiver (port 9090)
├── internal/
│   ├── handler/alert.go        # Echo HTTP handlers
│   ├── service/
│   │   ├── alert.go            # Business logic (CRUD)
│   │   ├── pricechecker.go     # Background price-checking goroutine
│   │   └── webhook.go          # Webhook delivery + HMAC signing
│   ├── db/                     # SQLC-generated database code
│   └── coingecko/client.go     # CoinGecko API client
├── sql/
│   ├── schema.sql              # Database DDL
│   └── queries.sql             # SQL queries for SQLC
├── api.http                 # REST Client requests (VS Code/Cursor)
├── docker-compose.yml
├── sqlc.yaml
└── README.md
```

## Webhook Payload

When an alert triggers, the service POSTs to the callback URL:

```json
{
  "alert_id": "550e8400-e29b-41d4-a716-446655440000",
  "token": "bitcoin",
  "target_price": "100000",
  "current_price": 100150.50,
  "direction": "above",
  "triggered_at": "2026-03-02T12:30:00Z"
}
```

The payload body is signed with HMAC-SHA256 using the alert's callback secret. The signature is sent in the `X-Signature-256` header as `sha256=<hex>`.
