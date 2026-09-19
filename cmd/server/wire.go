//go:build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"log/slog"

	"github.com/bogeia/echo-layout/internal/biz"
	"github.com/bogeia/echo-layout/internal/data"
	"github.com/bogeia/echo-layout/internal/server"
	"github.com/bogeia/echo-layout/internal/service"
	"github.com/google/wire"
	"github.com/labstack/echo/v5"
)

// wireRouter assembles data, biz, service and HTTP routes.
func wireRouter(*slog.Logger) (*echo.Echo, func(), error) {
	panic(wire.Build(data.ProviderSet, biz.ProviderSet, service.ProviderSet, server.ProviderSet))
}
