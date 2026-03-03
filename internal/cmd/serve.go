package cmd

import (
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	handlers "github.com/religiosa1/doors_of_durin/internal/http/handlers"
	middleware "github.com/religiosa1/doors_of_durin/internal/http/middleware"
	"github.com/religiosa1/doors_of_durin/internal/ratelimit"
	"github.com/religiosa1/doors_of_durin/internal/service"
)

var StaticFiles embed.FS

type Serve struct {
	CommonArgs `embed:""`
	Port       string `short:"p" placeholder:"4000" help:"Server port"`
	Host       string `short:"H" placeholder:"localhost" help:"Server host"`
}

func (s *Serve) Run() error {
	cfg, db, err := loadConfigAndDB(s.Config)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	MergeValueInto(&cfg.Port, s.Port)
	MergeValueInto(&cfg.Host, s.Host)

	limiter := ratelimit.New(ratelimit.Config{
		MaxAttempts: cfg.RateLimit.MaxAttempts,
		Window:      cfg.RateLimit.Window,
		FailDelay:   cfg.RateLimit.FailDelay,
	})
	defer limiter.Stop()

	logger := setupLogger(cfg.LogType, cfg.LogLevel)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	address := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	go func() {
		mux := http.NewServeMux()
		middlewares := middleware.Chain(
			middleware.WithLogger(logger),
		)
		authService := service.AuthService{
			DB:         db,
			SessionTTL: cfg.SessionTTL,
		}

		mux.Handle("/static/", http.FileServer(http.FS(StaticFiles)))

		mux.HandleFunc("GET /{$}", handlers.Healthcheck)

		mux.Handle("/verify", middlewares(handlers.Verify{AuthService: authService}))
		mux.Handle("GET /login", middlewares(handlers.Login{URLPrefix: cfg.URLPrefix}))
		mux.Handle("POST /login", middlewares(handlers.LoginSubmit{AuthService: authService, Limiter: limiter, SessionTTL: cfg.SessionTTL, URLPrefix: cfg.URLPrefix}))
		mux.Handle("POST /logout", middlewares(handlers.Logout{AuthService: authService}))

		if err := http.ListenAndServe(address, mux); err != nil {
			logger.Error("Error starting the server", slog.Any("error", err))
			errCh <- err
		}
	}()
	logger.Info("Running auth server", slog.String("address", address))

	select {
	case <-done:
		logger.Info("Server closed")
	case err := <-errCh:
		return err
	}
	return nil
}

func setupLogger(logType string, logLevel string) *slog.Logger {
	var logger *slog.Logger
	programLevel := new(slog.LevelVar)
	programLevel.Set(strLogLevelToEnumValue(logLevel))
	handlerOpts := &slog.HandlerOptions{Level: programLevel}
	switch logType {
	case "text":
		logger = slog.New(slog.NewTextHandler(os.Stdout, handlerOpts))
	case "json":
		logger = slog.New((slog.NewJSONHandler(os.Stdout, handlerOpts)))
	default:
		log.Fatalf("Unknown logger type %s", logType)
	}
	return logger
}

func strLogLevelToEnumValue(logLevel string) slog.Level {
	switch logLevel {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		log.Fatalf("Unexpected log level %s", logLevel)
		return slog.LevelInfo
	}
}
