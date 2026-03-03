package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/neodoomer/crypto-price-alert-service/internal/db"
	"github.com/neodoomer/crypto-price-alert-service/internal/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AlertHandler struct {
	svc *service.AlertService
}

func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

func (h *AlertHandler) Register(e *echo.Echo) {
	e.POST("/alerts", h.CreateAlert)
	e.GET("/alerts", h.ListAlerts)
	e.DELETE("/alerts/:id", h.DeleteAlert)
}

type createAlertRequest struct {
	Token          string  `json:"token"`
	TargetPrice    float64 `json:"target_price"`
	Direction      string  `json:"direction"`
	CallbackURL    string  `json:"callback_url"`
	CallbackSecret string  `json:"callback_secret"`
}

func (h *AlertHandler) CreateAlert(c echo.Context) error {
	var req createAlertRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse("invalid request body", err.Error()))
	}

	params := db.CreateAlertParams{
		Token:          req.Token,
		TargetPrice:    strconv.FormatFloat(req.TargetPrice, 'f', -1, 64),
		Direction:      req.Direction,
		CallbackUrl:    req.CallbackURL,
		CallbackSecret: req.CallbackSecret,
	}

	alert, err := h.svc.Create(c.Request().Context(), params)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			return c.JSON(http.StatusBadRequest, errorResponse("validation failed", err.Error()))
		}
		return c.JSON(http.StatusInternalServerError, errorResponse("internal error", ""))
	}

	return c.JSON(http.StatusCreated, alertToResponse(alert))
}

func (h *AlertHandler) ListAlerts(c echo.Context) error {
	var triggered *bool
	if q := c.QueryParam("triggered"); q != "" {
		v, err := strconv.ParseBool(q)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errorResponse("invalid query param", "triggered must be true or false"))
		}
		triggered = &v
	}

	alerts, err := h.svc.List(c.Request().Context(), triggered)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResponse("internal error", ""))
	}

	resp := make([]map[string]interface{}, 0, len(alerts))
	for _, a := range alerts {
		resp = append(resp, map[string]interface{}{
			"id":           a.ID,
			"token":        a.Token,
			"target_price": a.TargetPrice,
			"direction":    a.Direction,
			"callback_url": a.CallbackUrl,
			"triggered":    a.Triggered,
			"created_at":   a.CreatedAt,
			"updated_at":   a.UpdatedAt,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *AlertHandler) DeleteAlert(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse("invalid id", "id must be a valid UUID"))
	}

	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, service.ErrAlertNotFound) {
			return c.JSON(http.StatusNotFound, errorResponse("not found", "alert not found or already triggered"))
		}
		return c.JSON(http.StatusInternalServerError, errorResponse("internal error", ""))
	}

	return c.NoContent(http.StatusNoContent)
}

func alertToResponse(a db.Alert) map[string]interface{} {
	return map[string]interface{}{
		"id":           a.ID,
		"token":        a.Token,
		"target_price": a.TargetPrice,
		"direction":    a.Direction,
		"callback_url": a.CallbackUrl,
		"triggered":    a.Triggered,
		"created_at":   a.CreatedAt,
		"updated_at":   a.UpdatedAt,
	}
}

type apiError struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func errorResponse(msg, details string) apiError {
	return apiError{Error: msg, Details: details}
}
