package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/neodoomer/crypto-price-alert-service/internal/db"
	"github.com/google/uuid"
)

type WebhookPayload struct {
	AlertID      uuid.UUID `json:"alert_id"`
	Token        string    `json:"token"`
	TargetPrice  string    `json:"target_price"`
	CurrentPrice float64   `json:"current_price"`
	Direction    string    `json:"direction"`
	TriggeredAt  time.Time `json:"triggered_at"`
}

type WebhookSender struct {
	client *http.Client
}

func NewWebhookSender() *WebhookSender {
	return &WebhookSender{
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (w *WebhookSender) Send(alert db.Alert, currentPrice float64) {
	payload := WebhookPayload{
		AlertID:      alert.ID,
		Token:        alert.Token,
		TargetPrice:  alert.TargetPrice,
		CurrentPrice: currentPrice,
		Direction:    alert.Direction,
		TriggeredAt:  time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhook: marshal payload", "error", err, "alert_id", alert.ID)
		return
	}

	signature := signPayload(body, alert.CallbackSecret)

	req, err := http.NewRequest(http.MethodPost, alert.CallbackUrl, bytes.NewReader(body))
	if err != nil {
		slog.Error("webhook: build request", "error", err, "alert_id", alert.ID)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", fmt.Sprintf("sha256=%s", signature))

	resp, err := w.client.Do(req)
	if err != nil {
		slog.Error("webhook: send failed", "error", err, "alert_id", alert.ID, "url", alert.CallbackUrl)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("webhook: delivered", "alert_id", alert.ID, "status", resp.StatusCode)
	} else {
		slog.Warn("webhook: non-2xx response", "alert_id", alert.ID, "status", resp.StatusCode)
	}
}

func signPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
