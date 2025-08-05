package models

// Source is an instance of a Provider, directly related to its
// owner and having any configuration needed in order to receive
// the Provider's notification
type Source struct {
	ID         string `json:"id"          validate:"required"`
	ProviderID string `json:"provider_id" validate:"required,min=3"`
	OwnerID    string `json:"owner_id"    validate:"required"`
	// TODO: not sure if we're gonna need it
	Configuration map[string]any `json:"configuration"`
}
