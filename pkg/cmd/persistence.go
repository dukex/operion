package cmd

import (
	"log/slog"
	"strings"

	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/dukex/operion/pkg/persistence/postgresql"
)

var supportedPersistenceProviders = []string{"file", "mysql", "postgres", "mongodb", "redis"}

func NewPersistence(logger *slog.Logger, databaseUrl string) persistence.Persistence {
	provider := parsePersistenceProvider(databaseUrl)

	println("Using persistence provider:", provider)

	switch provider {
	// case "mysql":
	// 	return persistence.NewMySQLPersistence(databaseUrl)
	case "postgres":
		pers, err := postgresql.NewPostgreSQLPersistence(logger, databaseUrl)
		if err != nil {
			logger.Error("Failed to create PostgreSQL persistence", "error", err)
			panic("Failed to create PostgreSQL persistence")
		}
		return pers
	// case "mongodb":
	// 	return persistence.NewMongoDBPersistence(databaseUrl)
	// case "redis":
	// 	return persistence.NewRedisPersistence(databaseUrl)
	default:
		return file.NewFilePersistence(databaseUrl)
	}

}

func parsePersistenceProvider(databaseUrl string) string {
	parts := strings.Split(databaseUrl, "://")

	provider := parts[0]
	for _, supported := range supportedPersistenceProviders {
		if provider == supported {
			return provider
		}
	}

	return "file"
}
