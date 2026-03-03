package service

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	"github.com/neodoomer/crypto-price-alert-service/internal/coingecko"
	"github.com/neodoomer/crypto-price-alert-service/internal/db"
)

type PriceChecker struct {
	queries  *db.Queries
	cg       *coingecko.Client
	webhook  *WebhookSender
	interval time.Duration
}

func NewPriceChecker(queries *db.Queries, cg *coingecko.Client, webhook *WebhookSender) *PriceChecker {
	return &PriceChecker{
		queries:  queries,
		cg:       cg,
		webhook:  webhook,
		interval: 30 * time.Second,
	}
}

// Start launches the price-checking loop in a goroutine.
// It stops when ctx is cancelled.
func (pc *PriceChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(pc.interval)
	go func() {
		defer ticker.Stop()
		slog.Info("price checker started", "interval", pc.interval)

		// Run once immediately at startup, then on each tick.
		pc.check(ctx)

		for {
			select {
			case <-ctx.Done():
				slog.Info("price checker stopped")
				return
			case <-ticker.C:
				pc.check(ctx)
			}
		}
	}()
}

func (pc *PriceChecker) check(ctx context.Context) {
	alerts, err := pc.queries.GetActiveAlerts(ctx)
	if err != nil {
		slog.Error("price checker: fetch active alerts", "error", err)
		return
	}
	if len(alerts) == 0 {
		return
	}

	tokenSet := make(map[string]struct{})
	for _, a := range alerts {
		tokenSet[a.Token] = struct{}{}
	}
	tokens := make([]string, 0, len(tokenSet))
	for t := range tokenSet {
		tokens = append(tokens, t)
	}

	prices, err := pc.cg.GetPrices(ctx, tokens)
	if err != nil {
		slog.Error("price checker: fetch prices", "error", err)
		return
	}

	for _, alert := range alerts {
		currentPrice, ok := prices[alert.Token]
		if !ok {
			slog.Warn("price checker: no price data", "token", alert.Token)
			continue
		}

		if !isTriggered(alert.Direction, alert.TargetPrice, currentPrice) {
			continue
		}

		if err := pc.queries.MarkAlertTriggered(ctx, alert.ID); err != nil {
			slog.Error("price checker: mark triggered", "error", err, "alert_id", alert.ID)
			continue
		}

		slog.Info("alert triggered",
			"alert_id", alert.ID,
			"token", alert.Token,
			"direction", alert.Direction,
			"target_price", alert.TargetPrice,
			"current_price", currentPrice,
		)

		go pc.webhook.Send(alert, currentPrice)
	}
}

func isTriggered(direction string, targetPriceStr string, currentPrice float64) bool {
	target, _, err := new(big.Float).Parse(targetPriceStr, 10)
	if err != nil {
		slog.Error("price checker: parse target price", "error", err, "raw", targetPriceStr)
		return false
	}

	current := new(big.Float).SetFloat64(currentPrice)

	switch direction {
	case "below":
		return current.Cmp(target) <= 0
	case "above":
		return current.Cmp(target) >= 0
	default:
		return false
	}
}
