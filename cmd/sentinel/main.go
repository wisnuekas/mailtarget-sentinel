package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	chrepo "github.com/wisnuekas/mailtarget-sentinel/internal/clickhouse"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/handler"
	"github.com/wisnuekas/mailtarget-sentinel/internal/mailtarget"
	"github.com/wisnuekas/mailtarget-sentinel/internal/middleware"
	pgrepo "github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
	"github.com/wisnuekas/mailtarget-sentinel/internal/sqlite"
	"github.com/wisnuekas/mailtarget-sentinel/internal/worker"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/robfig/cron/v3"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pgPool, err := pgrepo.Open(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("postgres connect failed", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	chConn, err := chrepo.NewClient(cfg.ClickHouse)
	if err != nil {
		slog.Error("clickhouse connect failed", "error", err)
		os.Exit(1)
	}
	defer chConn.Close()

	db, err := sqlite.Open(cfg.SQLitePath)
	if err != nil {
		slog.Error("sqlite open failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	store := redisstore.NewStore(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, cfg.KillSwitch.HMACSecret, cfg.KillSwitch.TokenTTL)
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		slog.Error("redis connect failed", "error", err)
		os.Exit(1)
	}

	events := chrepo.NewEventRepository(chConn)
	alerts := sqlite.NewAlertRepository(db)
	transmission := mailtarget.NewTransmissionClient(cfg.Mailtarget.TransmissionURL, cfg.Mailtarget.APIKey)

	companies := pgrepo.NewCompanyRepository(pgPool)
	subAccounts := pgrepo.NewSubAccountRepository(pgPool)
	domains := pgrepo.NewDomainRepository(pgPool)
	enricher := service.NewRiskEnricher(events, companies, subAccounts, domains)

	detector := worker.NewDetector(cfg, events, companies, store, alerts, transmission, slog.Default())

	c := cron.New(cron.WithSeconds())
	if _, err := c.AddFunc("0 */5 * * * *", func() {
		detector.Run(ctx)
	}); err != nil {
		slog.Error("cron registration failed", "error", err)
		os.Exit(1)
	}
	c.Start()
	defer c.Stop()

	go func() {
		detector.Run(ctx)
	}()

	app := fiber.New(fiber.Config{AppName: "Mailtarget Sentinel"})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(middleware.DashboardCORS(cfg.CORSOrigins))

	api := app.Group("/api/v1/sentinel")

	authHandler := handler.NewAuthHandler(cfg)
	metricsHandler := handler.NewMetricsHandler(cfg, events, store)
	settingsHandler := handler.NewSettingsHandler(cfg, store)
	killSwitchHandler := handler.NewKillSwitchHandler(cfg, store, alerts, subAccounts, companies, transmission)
	resumeSwitchHandler := handler.NewResumeSwitchHandler(cfg, store, alerts, subAccounts)
	manualHandler := handler.NewManualOverrideHandler(cfg, alerts, subAccounts)
	alertsHandler := handler.NewAlertsHandler(alerts)
	atRiskHandler := handler.NewAtRiskHandler(cfg, events, store, enricher)
	companiesHandler := handler.NewCompaniesHandler(cfg, companies, events, store, enricher)
	workerHandler := handler.NewWorkerHandler(cfg, detector)
	subAccountsHandler := handler.NewSubAccountsHandler(cfg, subAccounts, companies, store, transmission, events)
	killSwitchEmailHandler := handler.NewKillSwitchEmailHandler(cfg, detector)

	api.Post("/auth/login", authHandler.Login)
	api.Get("/metrics", metricsHandler.Get)
	api.Get("/settings", settingsHandler.Get)
	api.Post("/settings", settingsHandler.Update)
	api.Post("/manual-override", manualHandler.Execute)

	api.Get("/companies", companiesHandler.List)
	api.Get("/sub-accounts", subAccountsHandler.List)
	api.Get("/sub-accounts/:id", subAccountsHandler.Get)
	api.Post("/sub-accounts/warning-email", subAccountsHandler.SendWarning)
	api.Post("/sub-accounts/kill-switch-email", killSwitchEmailHandler.Send)
	api.Post("/alerts/kill-switch-email", killSwitchEmailHandler.Send)

	api.Get("/alerts", alertsHandler.List)
	api.Get("/alerts/overview", alertsHandler.Overview)
	api.Get("/alerts/:id", alertsHandler.Get)

	api.Get("/companies/at-risk", atRiskHandler.List)
	api.Get("/companies/at-risk/summary", atRiskHandler.Summary)
	api.Post("/worker/run", workerHandler.Run)

	killSwitch := api.Group("/kill-switch", middleware.AMPCORS())
	killSwitch.Options("/", killSwitchHandler.Execute)
	killSwitch.Post("/", killSwitchHandler.Execute)
	killSwitch.Get("/", killSwitchHandler.Execute)

	resumeSwitch := api.Group("/resume-switch", middleware.AMPCORS())
	resumeSwitch.Options("/", resumeSwitchHandler.Execute)
	resumeSwitch.Post("/", resumeSwitchHandler.Execute)
	resumeSwitch.Get("/", resumeSwitchHandler.Execute)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "mailtarget-sentinel"})
	})

	go func() {
		slog.Info("server listening", "port", cfg.AppPort)
		if err := app.Listen(":" + cfg.AppPort); err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("sentinel started",
		"company_scope", companyScopeLabel(cfg.CompanyID),
		"worker_interval", "5m",
	)
	<-ctx.Done()
	slog.Info("shutting down")
	_ = app.Shutdown()
}

func companyScopeLabel(companyID int32) string {
	if companyID > 0 {
		return fmt.Sprintf("company %d", companyID)
	}
	return "all companies"
}
