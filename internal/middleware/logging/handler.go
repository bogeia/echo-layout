package logging

import (
	"context"
	"log/slog"
)

// Handler logs validated input before invoking the business handler.
// Input types can implement slog.LogValuer to redact sensitive fields.
func Handler[I, O any](logger *slog.Logger, operation string, next func(context.Context, *I) (*O, error)) func(context.Context, *I) (*O, error) {
	return func(ctx context.Context, input *I) (*O, error) {
		logger.DebugContext(ctx, "request args", "operation", operation, "args", input)
		return next(ctx, input)
	}
}
