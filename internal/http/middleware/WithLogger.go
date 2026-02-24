package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

type LoggingContextKey string

const (
	loggingContextRequestID   = LoggingContextKey("logging_context.request_id")
	loggingContextLogger      = LoggingContextKey("logging_context.logger")
	loggingContextRequestInfo = LoggingContextKey("logging_context.request_info")
)

type LoggingContext struct {
	Logger      *slog.Logger
	RequestID   string
	RequestInfo RequestInfo
}

func WithLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := ulid.Make().String()
			newLogger := logger.With(slog.String("request_id", id))

			ctx := context.WithValue(r.Context(), loggingContextRequestID, id)
			ctx = context.WithValue(ctx, loggingContextLogger, newLogger)

			info := parseRequestInfo(r)
			ctx = context.WithValue(ctx, loggingContextRequestInfo, info)

			newLogger.Info("Incoming request",
				slog.String("method", info.Method),
				slog.String("scheme", info.Scheme),
				slog.String("path", info.Path),
				slog.String("host", info.Scheme),
				slog.String("remote_addr", info.IP),
			)
			t1 := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r.WithContext(ctx))
			newLogger.Info(
				"request completed",
				slog.String("duration", time.Since(t1).String()),
				slog.Int("status_code", rw.status),
			)
		})
	}
}

// Wrapper around the response writer, to capture the response code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func GetLogger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggingContextLogger).(*slog.Logger)
	if !ok {
		logger = slog.Default()
	}
	return logger
}

func GetRequestID(ctx context.Context) string {
	contextID, _ := ctx.Value(loggingContextRequestID).(string)
	return contextID
}

func GetRequestInfo(ctx context.Context) RequestInfo {
	contextID, _ := ctx.Value(loggingContextRequestInfo).(RequestInfo)
	return contextID
}
