package persistence

import (
	"github.com/dukex/operion/pkg/providers/webhook/models"
)

// WebhookPersistence defines the persistence interface for the webhook provider.
// This interface is specific to webhook source needs and isolated from core persistence.
type WebhookPersistence interface {
	// WebhookSource operations
	SaveWebhookSource(source *models.WebhookSource) error
	WebhookSourceByID(id string) (*models.WebhookSource, error)
	WebhookSourceByExternalID(externalID string) (*models.WebhookSource, error)
	WebhookSources() ([]*models.WebhookSource, error)
	ActiveWebhookSources() ([]*models.WebhookSource, error)
	DeleteWebhookSource(id string) error

	// Health and lifecycle
	HealthCheck() error
	Close() error
}
