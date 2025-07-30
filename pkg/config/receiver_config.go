// Package config provides configuration loading for receivers
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"github.com/dukex/operion/pkg/protocol"
)

// ReceiverConfigFile represents the structure of the receivers.yaml file
type ReceiverConfigFile struct {
	TriggerTopic string                         `yaml:"trigger_topic"`
	Sources      []SourceConfigFile             `yaml:"sources"`
	Transformers []protocol.TransformerConfig   `yaml:"transformers"`
}

// SourceConfigFile represents a source configuration in the YAML file
type SourceConfigFile struct {
	Type          string                 `yaml:"type"`
	Name          string                 `yaml:"name"`
	Configuration map[string]interface{} `yaml:"configuration"`
	Schema        map[string]interface{} `yaml:"schema"`
}

// LoadReceiverConfig loads receiver configuration from a YAML file
func LoadReceiverConfig(filepath string) (protocol.ReceiverConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return protocol.ReceiverConfig{}, fmt.Errorf("failed to read config file %s: %w", filepath, err)
	}

	var configFile ReceiverConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return protocol.ReceiverConfig{}, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Convert to protocol.ReceiverConfig
	config := protocol.ReceiverConfig{
		TriggerTopic: configFile.TriggerTopic,
		Transformers: configFile.Transformers,
		Sources:      make([]protocol.SourceConfig, len(configFile.Sources)),
	}

	// Set default trigger topic if not specified
	if config.TriggerTopic == "" {
		config.TriggerTopic = "operion.trigger"
	}

	// Convert sources
	for i, source := range configFile.Sources {
		config.Sources[i] = protocol.SourceConfig{
			Type:          source.Type,
			Name:          source.Name,
			Configuration: source.Configuration,
			Schema:        source.Schema,
		}
	}

	return config, nil
}

// LoadReceiverConfigOrDefault attempts to load receiver config from file,
// falling back to a default configuration if the file doesn't exist
func LoadReceiverConfigOrDefault(filepath string) protocol.ReceiverConfig {
	config, err := LoadReceiverConfig(filepath)
	if err != nil {
		// Return a minimal default configuration
		return protocol.ReceiverConfig{
			TriggerTopic: "operion.trigger",
			Sources:      []protocol.SourceConfig{},
			Transformers: []protocol.TransformerConfig{},
		}
	}
	return config
}

// ValidateReceiverConfig validates the receiver configuration
func ValidateReceiverConfig(config protocol.ReceiverConfig) error {
	if config.TriggerTopic == "" {
		return fmt.Errorf("trigger_topic is required")
	}

	if len(config.Sources) == 0 {
		return fmt.Errorf("at least one source must be configured")
	}

	for i, source := range config.Sources {
		if source.Type == "" {
			return fmt.Errorf("source[%d]: type is required", i)
		}
		if source.Name == "" {
			return fmt.Errorf("source[%d]: name is required", i)
		}
		if source.Configuration == nil {
			return fmt.Errorf("source[%d]: configuration is required", i)
		}

		// Type-specific validation
		switch source.Type {
		case "kafka":
			if err := validateKafkaSource(source, i); err != nil {
				return err
			}
		case "webhook":
			if err := validateWebhookSource(source, i); err != nil {
				return err
			}
		case "schedule":
			if err := validateScheduleSource(source, i); err != nil {
				return err
			}
		default:
			return fmt.Errorf("source[%d]: unknown source type '%s'", i, source.Type)
		}
	}

	return nil
}

func validateKafkaSource(source protocol.SourceConfig, index int) error {
	topics, exists := source.Configuration["topics"]
	if !exists {
		return fmt.Errorf("source[%d]: kafka source requires 'topics' configuration", index)
	}

	switch v := topics.(type) {
	case []interface{}, []string:
		// Valid
	default:
		return fmt.Errorf("source[%d]: kafka 'topics' must be a list, got %T", index, v)
	}

	return nil
}

func validateWebhookSource(source protocol.SourceConfig, index int) error {
	endpoints, exists := source.Configuration["endpoints"]
	if !exists {
		return fmt.Errorf("source[%d]: webhook source requires 'endpoints' configuration", index)
	}

	endpointsList, ok := endpoints.([]interface{})
	if !ok {
		return fmt.Errorf("source[%d]: webhook 'endpoints' must be a list", index)
	}

	if len(endpointsList) == 0 {
		return fmt.Errorf("source[%d]: webhook source requires at least one endpoint", index)
	}

	for j, ep := range endpointsList {
		epMap, ok := ep.(map[string]interface{})
		if !ok {
			return fmt.Errorf("source[%d].endpoints[%d]: endpoint must be an object", index, j)
		}

		path, exists := epMap["path"]
		if !exists {
			return fmt.Errorf("source[%d].endpoints[%d]: 'path' is required", index, j)
		}

		pathStr, ok := path.(string)
		if !ok || pathStr == "" {
			return fmt.Errorf("source[%d].endpoints[%d]: 'path' must be a non-empty string", index, j)
		}

		if pathStr[0] != '/' {
			return fmt.Errorf("source[%d].endpoints[%d]: 'path' must start with '/'", index, j)
		}
	}

	return nil
}

func validateScheduleSource(source protocol.SourceConfig, index int) error {
	cronExpr, exists := source.Configuration["cron"]
	if !exists {
		return fmt.Errorf("source[%d]: schedule source requires 'cron' configuration", index)
	}

	cronStr, ok := cronExpr.(string)
	if !ok || cronStr == "" {
		return fmt.Errorf("source[%d]: schedule 'cron' must be a non-empty string", index)
	}

	// Basic validation - in a full implementation, we'd validate the cron expression format
	// For now, just ensure it's a non-empty string
	return nil
}