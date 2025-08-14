package cmd

import (
	"context"
	"log/slog"
	"strings"

	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/file"
)

var supportedPersistenceProviders = []string{"file", "mysql", "postgresql", "mongodb", "redis"}

// NewPersistence creates a new persistence layer based on the provided database URL.
func NewPersistence(ctx context.Context, logger *slog.Logger, databaseURL string) persistence.Persistence {
	provider := parsePersistenceProvider(databaseURL)

	logger.InfoContext(ctx, "Using persistence provider", "provider", provider)

	switch provider {
	// case "mysql":
	// 	return persistence.NewMySQLPersistence(databaseURL)
	// case "postgresql":
	// 	return persistence.NewPostgreSQLPersistence(databaseURL)
	// case "mongodb":
	// 	return persistence.NewMongoDBPersistence(databaseURL)
	// case "redis":
	// 	return persistence.NewRedisPersistence(databaseURL)
	default:
		return file.NewPersistence(databaseURL)
	}
}

func parsePersistenceProvider(databaseURL string) string {
	parts := strings.Split(databaseURL, "://")

	provider := parts[0]
	for _, supported := range supportedPersistenceProviders {
		if provider == supported {
			return provider
		}
	}

	return "file"
}
