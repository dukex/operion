package models

// Provider is technology or vendor that provides notifications
// that can be used as triggers. Examples providers: Google Drive,
// Scheduler, Webhook, Jira, Caju, etc...
type Provider struct {
	ID          string `json:"id" validate:"required"`
	Description string `json:"description"` // optional
}
