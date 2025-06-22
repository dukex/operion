package models

type ActionItem struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Configuration map[string]interface{} `json:"configuration"`
}
