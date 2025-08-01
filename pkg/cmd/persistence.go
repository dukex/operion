package cmd

import (
	"context"
	"log/slog"
	"strings"

	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/dukex/operion/pkg/persistence/postgresql"
)

type Provider int

const (
	FileProvider Provider = iota
	PostgresqlProvider
)

// NewPersistence creates a new persistence layer based on the provided database URL.
func NewPersistence(ctx context.Context, logger *slog.Logger, databaseURL string) persistence.Persistence {
	provider := parsePersistenceProvider(databaseURL)

	logger.InfoContext(ctx, "Using persistence provider", "provider", provider)

	switch provider {
	case PostgresqlProvider:
		persistence, err := postgresql.NewPersistence(ctx, logger, databaseURL)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to create PostgreSQL persistence", "error", err)
			panic("Failed to create PostgreSQL persistence")
		}

		return persistence
	default:
		return file.NewFilePersistence(databaseURL)
	}
}

func parsePersistenceProvider(databaseURL string) Provider {
	parts := strings.Split(databaseURL, "://")

	provider := parts[0]

	switch provider {
	case "postgres", "postgresql":
		return PostgresqlProvider
	default:
		return FileProvider
	}
}
