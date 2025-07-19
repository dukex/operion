package httprequest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPRequestActionFactory_ID(t *testing.T) {
	factory := NewHTTPRequestActionFactory()
	assert.Equal(t, "http_request", factory.ID())
}

func TestHTTPRequestActionFactory_Create(t *testing.T) {
	factory := NewHTTPRequestActionFactory()

	config := map[string]interface{}{
		"protocol": "https",
		"host":     "api.example.com",
		"path":     "/test",
		"method":   "GET",
	}

	action, err := factory.Create(config)
	require.NoError(t, err)
	assert.NotNil(t, action)

	httpAction, ok := action.(*HTTPRequestAction)
	assert.True(t, ok)
	assert.Equal(t, "https", httpAction.Protocol)
	assert.Equal(t, "api.example.com", httpAction.Host)
	assert.Equal(t, "/test", httpAction.Path)
	assert.Equal(t, "GET", httpAction.Method)
}

func TestHTTPRequestActionFactory_Create_MissingHost(t *testing.T) {
	factory := NewHTTPRequestActionFactory()

	config := map[string]interface{}{
		"method": "GET",
	}

	_, err := factory.Create(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid 'host'")
}

func TestHTTPRequestActionFactory_Create_EmptyConfig(t *testing.T) {
	factory := NewHTTPRequestActionFactory()

	// Test with empty configuration - should fail due to missing host
	config := map[string]interface{}{}

	_, err := factory.Create(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid 'host'")
}
