package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/neodoomer/crypto-price-alert-service/internal/coingecko"
	"github.com/neodoomer/crypto-price-alert-service/internal/db"
	"github.com/neodoomer/crypto-price-alert-service/internal/handler"
	"github.com/neodoomer/crypto-price-alert-service/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://crypto:crypto@localhost:5432/crypto_alerts?sslmode=disable"
		slog.Info("DATABASE_URL not set, using default (localhost)")
	} else {
		slog.Info("using DATABASE_URL from environment")
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to database")

	queries := db.New(sqlDB)

	alertSvc := service.NewAlertService(queries)
	cgClient := coingecko.NewClient()
	webhookSender := service.NewWebhookSender()
	priceChecker := service.NewPriceChecker(queries, cgClient, webhookSender)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	alertHandler := handler.NewAlertHandler(alertSvc)
	alertHandler.Register(e)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	priceChecker.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	go func() {
		slog.Info("server starting", "port", port)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("server stopped")
}
