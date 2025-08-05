package httprequest_test

import (
	"testing"

	"github.com/dukex/operion/pkg/actions/httprequest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPRequestActionFactory_ID(t *testing.T) {
	t.Parallel()

	factory := httprequest.NewActionFactory()
	assert.Equal(t, "http_request", factory.ID())
}

func TestHTTPRequestActionFactory_Create(t *testing.T) {
	t.Parallel()

	factory := httprequest.NewActionFactory()

	config := map[string]any{
		"protocol": "https",
		"host":     "api.example.com",
		"path":     "/test",
		"method":   "GET",
	}

	action, err := factory.Create(t.Context(), config)
	require.NoError(t, err)
	assert.NotNil(t, action)

	httpAction, isHTTPAction := action.(*httprequest.Action)
	assert.True(t, isHTTPAction)
	assert.Equal(t, "https", httpAction.Protocol)
	assert.Equal(t, "api.example.com", httpAction.Host)
	assert.Equal(t, "/test", httpAction.Path)
	assert.Equal(t, "GET", httpAction.Method)
}

func TestHTTPRequestActionFactory_Create_MissingHost(t *testing.T) {
	t.Parallel()

	factory := httprequest.NewActionFactory()

	config := map[string]any{
		"method": "GET",
	}

	_, err := factory.Create(t.Context(), config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid 'host'")
}

func TestHTTPRequestActionFactory_Create_EmptyConfig(t *testing.T) {
	t.Parallel()

	factory := httprequest.NewActionFactory()

	// Test with empty configuration - should fail due to missing host
	config := map[string]any{}

	_, err := factory.Create(t.Context(), config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid 'host'")
}
