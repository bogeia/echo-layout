package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/wire"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/bogeia/echo-layout/internal/config"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewRouter)

func newEcho(log *slog.Logger, options ...echoOption) *echo.Echo {
	router := echo.New()
	router.Logger = log
	router.HTTPErrorHandler = httpErrorHandler
	for _, option := range options {
		option(router)
	}
	return router
}

func requestLogger(log *slog.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		HandleError: true,
		LogMethod:   true, LogURIPath: true, LogStatus: true, LogLatency: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			level := slog.LevelInfo
			attrs := []slog.Attr{
				slog.String("method", v.Method), slog.String("path", v.URIPath),
				slog.Int("status", v.Status), slog.Duration("latency", v.Latency),
			}
			if v.Status >= http.StatusInternalServerError || v.Error != nil {
				level = slog.LevelError
			}
			if v.Error != nil {
				attrs = append(attrs, slog.String("error", v.Error.Error()))
			}
			log.LogAttrs(c.Request().Context(), level, "http request", attrs...)
			return nil
		},
	})
}

func httpErrorHandler(c *echo.Context, err error) {
	response, status := echo.ResolveResponseStatus(c.Response(), err)
	if response != nil && response.Committed {
		return
	}
	problem := huma.ErrorModel{
		Type: "about:blank", Title: http.StatusText(status), Status: status,
		Detail: http.StatusText(status),
	}
	var httpError *echo.HTTPError
	if status < http.StatusInternalServerError && errors.As(err, &httpError) && httpError.Message != "" {
		problem.Detail = httpError.Message
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
	var writeErr error
	if c.Request().Method == http.MethodHead {
		writeErr = c.NoContent(status)
	} else {
		writeErr = c.JSON(status, &problem)
	}
	if writeErr != nil {
		c.Logger().ErrorContext(c.Request().Context(), "write error response", "error", writeErr)
	}
}

func Run(ctx context.Context, router *echo.Echo, config config.Server) error {
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler:           router,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(router.Logger.Handler(), slog.LevelError),
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	router.Logger.Info("server started", "address", listener.Addr().String())
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
		defer cancel()
		err := server.Shutdown(shutdown)
		if err != nil {
			err = errors.Join(err, server.Close())
		}
		if serveErr := <-done; !errors.Is(serveErr, http.ErrServerClosed) {
			err = errors.Join(err, serveErr)
		}
		router.Logger.Info("server stopped")
		return err
	}
}
