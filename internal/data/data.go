package data

import (
	"log/slog"

	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewUsers)

// Data holds the long-lived storage clients shared by repos, such as *sql.DB or Redis.
type Data struct{}

// NewData opens storage clients and returns a cleanup function that closes them.
func NewData(log *slog.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.Info("closing the data resources")
	}
	return &Data{}, cleanup, nil
}
