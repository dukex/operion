package persistence

import (
	"github.com/dukex/operion/pkg/providers/kafka/models"
)

// KafkaPersistence defines the persistence interface for the Kafka provider.
// This interface is specific to Kafka source needs and isolated from core persistence.
type KafkaPersistence interface {
	// KafkaSource operations
	SaveKafkaSource(source *models.KafkaSource) error
	KafkaSourceByID(id string) (*models.KafkaSource, error)
	KafkaSourceByConnectionDetailsID(connectionDetailsID string) ([]*models.KafkaSource, error)
	KafkaSources() ([]*models.KafkaSource, error)
	ActiveKafkaSources() ([]*models.KafkaSource, error)
	DeleteKafkaSource(id string) error

	// Health and lifecycle
	HealthCheck() error
	Close() error
}
