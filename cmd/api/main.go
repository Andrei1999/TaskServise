package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	infrastructurepostgres "example.com/taskservice/internal/infrastructure/postgres"
	postgresrepo "example.com/taskservice/internal/repository/postgres"
	transporthttp "example.com/taskservice/internal/transport/http"
	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
	taskusecase "example.com/taskservice/internal/usecase/task"
	templateusecase "example.com/taskservice/internal/usecase/template"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := infrastructurepostgres.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("open postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	taskRepo := postgresrepo.New(pool)
	templateRepo := postgresrepo.NewTemplateRepository(pool)

	taskUsecase := taskusecase.NewService(taskRepo)
	templateUsecase := templateusecase.NewService(templateRepo, taskRepo)

	taskHandler := httphandlers.NewTaskHandler(taskUsecase)
	templateHandler := httphandlers.NewTemplateHandler(templateUsecase)
	docsHandler := swaggerdocs.NewHandler()

	router := transporthttp.NewRouter(taskHandler, templateHandler, docsHandler)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Cron: каждый месяц 1-го числа в 00:00 МСК
	moscowLoc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		logger.Error("failed to load Moscow timezone", "error", err)
		// Используем UTC как fallback
		moscowLoc = time.UTC
	}
	
	cronSched := cron.New(cron.WithLocation(moscowLoc))
	if _, err := cronSched.AddFunc("0 0 1 * *", func() {
		logger.Info("starting monthly template generation")
		if err := templateUsecase.GenerateMissingInstances(context.Background()); err != nil {
			logger.Error("monthly generation failed", "error", err)
		} else {
			logger.Info("monthly generation completed")
		}
	}); err != nil {
		logger.Error("failed to schedule cron", "error", err)
	}
	cronSched.Start()
	defer cronSched.Stop()

	go func() {
		<-ctx.Done()
		
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown http server", "error", err)
		}
	}()

	logger.Info("http server started", "addr", cfg.HTTPAddr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("listen and serve", "error", err)
		os.Exit(1)
	}
}

type config struct {
	HTTPAddr    string
	DatabaseDSN string
}

func loadConfig() config {
	cfg := config{
		HTTPAddr:    envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseDSN: envOrDefault("DATABASE_DSN", "postgres://postgres:postgres@postgres:5432/taskservice?sslmode=disable"),
	}

	if cfg.DatabaseDSN == "" {
		panic(fmt.Errorf("DATABASE_DSN is required"))
	}

	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
