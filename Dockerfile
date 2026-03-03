FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install CA certificates for HTTPS (CoinGecko)
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static binary for Linux/amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /server /app/server

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/server"]

