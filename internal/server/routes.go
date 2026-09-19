package server

import (
	"log/slog"
	"net/http"

	"github.com/bogeia/echo-layout/internal/service"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type echoOption func(*echo.Echo)

func withMiddleware(middlewares ...echo.MiddlewareFunc) echoOption {
	return func(router *echo.Echo) {
		router.Use(middlewares...)
	}
}

// NewRouter creates Echo and registers all HTTP routes.
// Wrap a handler with logging.Handler to log its parsed input.
func NewRouter(log *slog.Logger, users *service.Users) *echo.Echo {
	// RequestLogger must wrap Recover so recovered panics are logged as 500s.
	options := []echoOption{
		withMiddleware(
			requestLogger(log),
			middleware.Recover(),
		),
	}
	router := newEcho(log, options...)
	config := huma.DefaultConfig("Echo Layout API", "1.0.0")
	config.Info.Description = "演示 CRUD：data 仅记录日志，不持久化；列表为空，单条查询返回 404。"
	api := humaecho.New(router, config)
	huma.Register(api, huma.Operation{OperationID: "create-user", Method: http.MethodPost, Path: "/users", Summary: "创建用户（演示）", DefaultStatus: http.StatusCreated}, users.Create)
	huma.Register(api, huma.Operation{OperationID: "list-users", Method: http.MethodGet, Path: "/users", Summary: "用户列表（空数组）"}, users.List)
	huma.Register(api, huma.Operation{OperationID: "get-user", Method: http.MethodGet, Path: "/users/{id}", Summary: "查询用户（返回 404）", Errors: []int{http.StatusNotFound}}, users.Get)
	huma.Register(api, huma.Operation{OperationID: "update-user", Method: http.MethodPut, Path: "/users/{id}", Summary: "更新用户（演示）"}, users.Update)
	huma.Register(api, huma.Operation{OperationID: "delete-user", Method: http.MethodDelete, Path: "/users/{id}", Summary: "删除用户（演示）", DefaultStatus: http.StatusNoContent}, users.Delete)
	return router
}
