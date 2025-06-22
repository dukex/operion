package cmd

import (
	"strings"

	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/persistence/file"
)

var supportedPersistenceProviders = []string{"file", "mysql", "postgresql", "mongodb", "redis"}

func NewPersistence(databaseUrl string) persistence.Persistence {
	provider := parsePersistenceProvider(databaseUrl)

	switch provider {
	// case "mysql":
	// 	return persistence.NewMySQLPersistence(databaseUrl)
	// case "postgresql":
	// 	return persistence.NewPostgreSQLPersistence(databaseUrl)
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
