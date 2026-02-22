package main

import (
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	config "gitgub.com/religiosa1/auth_server/internal/config"
	handlers "gitgub.com/religiosa1/auth_server/internal/http/handlers"
	middleware "gitgub.com/religiosa1/auth_server/internal/http/middleware"
)

//go:embed static
var staticFiles embed.FS

func main() {
	config, err := config.Load("")
	if err != nil {
		log.Fatal(err)
	}

	logger := setupLogger(config.LogType, config.LogLevel)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	address := fmt.Sprintf("%s:%s", config.Host, config.Port)
	go func() {
		mux := http.NewServeMux()
		middlewares := middleware.Chain(
			middleware.WithLogger(logger),
		)

		http.Handle("/static", http.FileServer(http.FS(staticFiles)))

		mux.HandleFunc("/", handlers.Healthcheck)

		mux.Handle("/verify", middlewares(handlers.Verify{}))
		mux.Handle("GET /login", middlewares(handlers.Login{}))
		mux.Handle("POST /login", middlewares(handlers.LoginSubmit{}))

		// mux.Handle("GET /users", middlewares(handlers.UserList))
		// mux.Handle("GET /users/new", middlewares(handlers.UserCreate))
		// mux.Handle("POST /users/new", middlewares(handlers.UserCreateSubmit))
		// mux.Handle("GET /users/{id}", middlewares(handlers.UserEdit))
		// mux.Handle("POST /users/{id}", middlewares(handlers.UserEditSubmit))
		// mux.Handle("DELETE /users/{id}", middlewares(handlers.UserDeleteSubmit))

		if err := http.ListenAndServe(address, mux); err != nil {
			logger.Error("Error starting the server", slog.Any("error", err))
			errCh <- err
		}
	}()
	logger.Info("Running bot http server", slog.String("address", address))

	select {
	case <-done:
		logger.Info("Server closed")
	case err := <-errCh:
		log.Fatal(err)
	}
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
