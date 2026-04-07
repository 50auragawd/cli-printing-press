package megamcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRegistryHappyPath(t *testing.T) {
	registry := Registry{
		SchemaVersion: 1,
		Entries: []RegistryEntry{
			{
				Name:        "espn-pp-cli",
				Category:    "media-and-entertainment",
				API:         "espn",
				Description: "ESPN sports data",
				Path:        "library/media-and-entertainment/espn-pp-cli",
				MCP: RegistryMCP{
					Binary:           "espn-pp-mcp",
					Transport:        "stdio",
					ToolCount:        3,
					PublicToolCount:  3,
					AuthType:         "none",
					MCPReady:         "full",
					ManifestChecksum: "sha256:abc123",
					SpecFormat:       "openapi3",
					ManifestURL:      "library/media-and-entertainment/espn-pp-cli/tools-manifest.json",
				},
			},
			{
				Name:        "dub-pp-cli",
				Category:    "developer-tools",
				API:         "dub",
				Description: "Dub link management",
				Path:        "library/developer-tools/dub-pp-cli",
				MCP: RegistryMCP{
					Binary:           "dub-pp-mcp",
					Transport:        "stdio",
					ToolCount:        10,
					PublicToolCount:  10,
					AuthType:         "api_key",
					EnvVars:          []string{"DUB_TOKEN"},
					MCPReady:         "full",
					ManifestChecksum: "sha256:def456",
					SpecFormat:       "openapi3",
					ManifestURL:      "library/developer-tools/dub-pp-cli/tools-manifest.json",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/registry.json", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(registry)
	}))
	defer server.Close()

	result, err := FetchRegistry(server.URL)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.SchemaVersion)
	assert.Len(t, result.Entries, 2)
	assert.Equal(t, "espn", result.Entries[0].API)
	assert.Equal(t, "dub", result.Entries[1].API)
	assert.Equal(t, "sha256:abc123", result.Entries[0].MCP.ManifestChecksum)
	assert.Equal(t, []string{"DUB_TOKEN"}, result.Entries[1].MCP.EnvVars)
}

func TestFetchRegistry404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result, err := FetchRegistry(server.URL)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestFetchRegistryInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	result, err := FetchRegistry(server.URL)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parsing registry JSON")
}
