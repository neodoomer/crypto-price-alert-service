package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/neodoomer/crypto-price-alert-service/internal/db"
	"github.com/google/uuid"
)

var (
	ErrAlertNotFound = errors.New("alert not found or already triggered")
	ErrValidation    = errors.New("validation error")
)

type AlertService struct {
	queries *db.Queries
}

func NewAlertService(queries *db.Queries) *AlertService {
	return &AlertService{queries: queries}
}

func (s *AlertService) Create(ctx context.Context, params db.CreateAlertParams) (db.Alert, error) {
	if params.Token == "" {
		return db.Alert{}, fmt.Errorf("%w: token is required", ErrValidation)
	}
	if params.TargetPrice == "" {
		return db.Alert{}, fmt.Errorf("%w: target_price is required", ErrValidation)
	}
	if params.Direction != "above" && params.Direction != "below" {
		return db.Alert{}, fmt.Errorf("%w: direction must be 'above' or 'below'", ErrValidation)
	}
	if params.CallbackUrl == "" {
		return db.Alert{}, fmt.Errorf("%w: callback_url is required", ErrValidation)
	}
	if params.CallbackSecret == "" {
		return db.Alert{}, fmt.Errorf("%w: callback_secret is required", ErrValidation)
	}

	alert, err := s.queries.CreateAlert(ctx, params)
	if err != nil {
		return db.Alert{}, fmt.Errorf("create alert: %w", err)
	}
	return alert, nil
}

func (s *AlertService) List(ctx context.Context, triggered *bool) ([]db.ListAlertsRow, error) {
	filter := sql.NullBool{}
	if triggered != nil {
		filter.Valid = true
		filter.Bool = *triggered
	}

	alerts, err := s.queries.ListAlerts(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	return alerts, nil
}

func (s *AlertService) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := s.queries.DeleteAlert(ctx, id)
	if err != nil {
		return fmt.Errorf("delete alert: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete alert: %w", err)
	}
	if rows == 0 {
		return ErrAlertNotFound
	}
	return nil
}
