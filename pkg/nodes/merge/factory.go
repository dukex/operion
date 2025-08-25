// Package merge provides merge node factory for registry integration.
package merge

import (
	"context"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
)

// MergeNodeFactory creates MergeNode instances.
type MergeNodeFactory struct{}

// Create creates a new MergeNode instance.
func (f *MergeNodeFactory) Create(ctx context.Context, id string, config map[string]any) (models.Node, error) {
	return NewMergeNode(id, config)
}

// ID returns the factory ID.
func (f *MergeNodeFactory) ID() string {
	return "merge"
}

// Name returns the factory name.
func (f *MergeNodeFactory) Name() string {
	return "Merge"
}

// Description returns the factory description.
func (f *MergeNodeFactory) Description() string {
	return "Merges multiple execution paths into a single output, combining data from different workflow branches"
}

// Schema returns the JSON schema for Merge node configuration.
func (f *MergeNodeFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input_ports": map[string]any{
				"type":        "array",
				"description": "List of input port names that this merge node will wait for",
				"items": map[string]any{
					"type": "string",
				},
				"minItems": 2,
				"examples": [][]string{
					{"path_a", "path_b"},
					{"api_call", "db_query", "cache_lookup"},
					{"validation", "enrichment"},
				},
			},
			"merge_mode": map[string]any{
				"type":        "string",
				"description": "How to handle merging: 'all' waits for all inputs, 'any' proceeds with first input, 'first' uses only first input received",
				"enum":        []string{"all", "any", "first"},
				"default":     "all",
				"examples":    []string{"all", "any", "first"},
			},
		},
		"required": []string{"input_ports"},
		"examples": []map[string]any{
			{
				"input_ports": []string{"validation_result", "enrichment_data"},
				"merge_mode":  "all",
			},
			{
				"input_ports": []string{"primary_api", "backup_api", "cache_fallback"},
				"merge_mode":  "first",
			},
			{
				"input_ports": []string{"user_data", "permissions", "preferences"},
				"merge_mode":  "all",
			},
		},
	}
}

// NewMergeNodeFactory creates a new factory instance.
func NewMergeNodeFactory() protocol.NodeFactory {
	return &MergeNodeFactory{}
}
