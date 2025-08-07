package persistence

import (
	"github.com/dukex/operion/pkg/sources/webhook/models"
)

// WebhookPersistence defines the persistence interface for the webhook provider.
// This interface is specific to webhook source needs and isolated from core persistence.
type WebhookPersistence interface {
	// WebhookSource operations
	SaveWebhookSource(source *models.WebhookSource) error
	WebhookSourceByID(id string) (*models.WebhookSource, error)
	WebhookSourceByUUID(uuid string) (*models.WebhookSource, error)
	WebhookSourceBySourceID(sourceID string) (*models.WebhookSource, error)
	WebhookSources() ([]*models.WebhookSource, error)
	ActiveWebhookSources() ([]*models.WebhookSource, error)
	DeleteWebhookSource(id string) error
	DeleteWebhookSourceBySourceID(sourceID string) error

	// Health and lifecycle
	HealthCheck() error
	Close() error
}
