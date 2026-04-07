package megamcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTestManifest creates a ToolsManifest for testing.
func makeTestManifest(apiName string) *ToolsManifest {
	return &ToolsManifest{
		APIName:     apiName,
		BaseURL:     "https://api.example.com",
		Description: apiName + " API",
		MCPReady:    "full",
		Auth: ManifestAuth{
			Type:    "api_key",
			Header:  "Authorization",
			Format:  "Bearer {TOKEN}",
			In:      "header",
			EnvVars: []string{"TOKEN"},
		},
		RequiredHeaders: []ManifestHeader{},
		Tools: []ManifestTool{
			{
				Name:        "items_list",
				Description: "List items",
				Method:      "GET",
				Path:        "/items",
				Params: []ManifestParam{
					{Name: "limit", Type: "integer", Location: "query"},
				},
			},
		},
	}
}

// marshalManifest marshals a manifest to JSON bytes.
func marshalManifest(t *testing.T, m *ToolsManifest) []byte {
	t.Helper()
	data, err := json.MarshalIndent(m, "", "  ")
	require.NoError(t, err)
	return data
}

// makeManifestServer creates an httptest server that serves manifests by slug.
func makeManifestServer(t *testing.T, manifests map[string][]byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for slug, data := range manifests {
			if r.URL.Path == "/library/"+slug+"/tools-manifest.json" {
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestLoadManifestsHappyPath(t *testing.T) {
	espnManifest := makeTestManifest("ESPN")
	dubManifest := makeTestManifest("Dub")
	espnData := marshalManifest(t, espnManifest)
	dubData := marshalManifest(t, dubManifest)
	espnChecksum := ComputeChecksum(espnData)
	dubChecksum := ComputeChecksum(dubData)

	server := makeManifestServer(t, map[string][]byte{
		"espn": espnData,
		"dub":  dubData,
	})
	defer server.Close()

	entries := []RegistryEntry{
		{
			API: "espn",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: espnChecksum,
				ManifestURL:      "library/espn/tools-manifest.json",
			},
		},
		{
			API: "dub",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: dubChecksum,
				ManifestURL:      "library/dub/tools-manifest.json",
			},
		},
	}

	cacheDir := t.TempDir()
	results, warnings := LoadManifests(entries, cacheDir, server.URL)

	assert.Empty(t, warnings)
	assert.Len(t, results, 2)

	// Both manifests should be cached.
	for _, entry := range results {
		cachePath := filepath.Join(cacheDir, "manifests", entry.Slug, "tools-manifest.json")
		assert.FileExists(t, cachePath)
	}
}

func TestLoadManifestsCachedWithMatchingChecksum(t *testing.T) {
	manifest := makeTestManifest("ESPN")
	data := marshalManifest(t, manifest)
	checksum := ComputeChecksum(data)

	// Pre-populate cache.
	cacheDir := t.TempDir()
	cacheSubDir := filepath.Join(cacheDir, "manifests", "espn")
	require.NoError(t, os.MkdirAll(cacheSubDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(cacheSubDir, "tools-manifest.json"), data, 0o600))

	// Server that should NOT be called.
	fetchCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetchCalled = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	entries := []RegistryEntry{
		{
			API: "espn",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: checksum,
				ManifestURL:      "library/espn/tools-manifest.json",
			},
		},
	}

	results, warnings := LoadManifests(entries, cacheDir, server.URL)

	assert.Empty(t, warnings)
	assert.Len(t, results, 1)
	assert.Equal(t, "espn", results[0].Slug)
	assert.Equal(t, "ESPN", results[0].Manifest.APIName)
	assert.False(t, fetchCalled, "server should not have been called when cache is valid")
}

func TestLoadManifestsCachedWithMismatchedChecksum(t *testing.T) {
	oldManifest := makeTestManifest("ESPN Old")
	oldData := marshalManifest(t, oldManifest)

	newManifest := makeTestManifest("ESPN New")
	newData := marshalManifest(t, newManifest)
	newChecksum := ComputeChecksum(newData)

	// Pre-populate cache with old data.
	cacheDir := t.TempDir()
	cacheSubDir := filepath.Join(cacheDir, "manifests", "espn")
	require.NoError(t, os.MkdirAll(cacheSubDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(cacheSubDir, "tools-manifest.json"), oldData, 0o600))

	// Server returns new manifest.
	server := makeManifestServer(t, map[string][]byte{
		"espn": newData,
	})
	defer server.Close()

	entries := []RegistryEntry{
		{
			API: "espn",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: newChecksum,
				ManifestURL:      "library/espn/tools-manifest.json",
			},
		},
	}

	results, warnings := LoadManifests(entries, cacheDir, server.URL)

	assert.Empty(t, warnings)
	require.Len(t, results, 1)
	assert.Equal(t, "ESPN New", results[0].Manifest.APIName, "should have fetched new manifest")

	// Verify cache was updated.
	cachedData, err := os.ReadFile(filepath.Join(cacheSubDir, "tools-manifest.json"))
	require.NoError(t, err)
	assert.NoError(t, VerifyChecksum(cachedData, newChecksum), "cache should contain new data")
}

func TestLoadManifestsPrintingPressAPIsFilter(t *testing.T) {
	espnData := marshalManifest(t, makeTestManifest("ESPN"))
	dubData := marshalManifest(t, makeTestManifest("Dub"))
	espnChecksum := ComputeChecksum(espnData)
	dubChecksum := ComputeChecksum(dubData)

	server := makeManifestServer(t, map[string][]byte{
		"espn": espnData,
		"dub":  dubData,
	})
	defer server.Close()

	entries := []RegistryEntry{
		{
			API: "espn",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: espnChecksum,
				ManifestURL:      "library/espn/tools-manifest.json",
			},
		},
		{
			API: "dub",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: dubChecksum,
				ManifestURL:      "library/dub/tools-manifest.json",
			},
		},
	}

	t.Setenv("PRINTING_PRESS_APIS", "espn")
	cacheDir := t.TempDir()
	results, warnings := LoadManifests(entries, cacheDir, server.URL)

	assert.Empty(t, warnings)
	require.Len(t, results, 1)
	assert.Equal(t, "espn", results[0].Slug)
}

func TestLoadManifestsOneFetchFailsOthersSucceed(t *testing.T) {
	dubData := marshalManifest(t, makeTestManifest("Dub"))
	dubChecksum := ComputeChecksum(dubData)

	// Server returns 404 for espn, 200 for dub.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/library/dub/tools-manifest.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(dubData)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	entries := []RegistryEntry{
		{
			API: "espn",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: "sha256:abc",
				ManifestURL:      "library/espn/tools-manifest.json",
			},
		},
		{
			API: "dub",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: dubChecksum,
				ManifestURL:      "library/dub/tools-manifest.json",
			},
		},
	}

	cacheDir := t.TempDir()
	results, warnings := LoadManifests(entries, cacheDir, server.URL)

	require.Len(t, results, 1)
	assert.Equal(t, "dub", results[0].Slug)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "espn")
}

func TestLoadManifestsCLIOnlyFiltered(t *testing.T) {
	entries := []RegistryEntry{
		{
			API: "pagliacci",
			MCP: RegistryMCP{
				MCPReady:    "cli-only",
				ManifestURL: "library/pagliacci/tools-manifest.json",
			},
		},
	}

	cacheDir := t.TempDir()
	// No server needed since cli-only entries are filtered out before fetch.
	results, warnings := LoadManifests(entries, cacheDir, "http://unused")

	assert.Empty(t, results)
	assert.Empty(t, warnings)
}

func TestLoadManifestsSlugTraversalRejected(t *testing.T) {
	entries := []RegistryEntry{
		{
			API: "../etc",
			MCP: RegistryMCP{
				MCPReady:    "full",
				ManifestURL: "library/etc/tools-manifest.json",
			},
		},
	}

	cacheDir := t.TempDir()
	results, warnings := LoadManifests(entries, cacheDir, "http://unused")

	assert.Empty(t, results)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "path traversal")
}

func TestLoadManifestsCacheDirCreated(t *testing.T) {
	manifest := makeTestManifest("ESPN")
	data := marshalManifest(t, manifest)
	checksum := ComputeChecksum(data)

	server := makeManifestServer(t, map[string][]byte{
		"espn": data,
	})
	defer server.Close()

	entries := []RegistryEntry{
		{
			API: "espn",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: checksum,
				ManifestURL:      "library/espn/tools-manifest.json",
			},
		},
	}

	// Use a nested cache dir that doesn't exist yet.
	cacheDir := filepath.Join(t.TempDir(), "nested", "cache")
	results, warnings := LoadManifests(entries, cacheDir, server.URL)

	assert.Empty(t, warnings)
	require.Len(t, results, 1)

	// Verify the cache directory was created.
	cachePath := filepath.Join(cacheDir, "manifests", "espn", "tools-manifest.json")
	assert.FileExists(t, cachePath)
}

func TestLoadManifestsEmptyManifestURLFiltered(t *testing.T) {
	entries := []RegistryEntry{
		{
			API: "nourl",
			MCP: RegistryMCP{
				MCPReady:    "full",
				ManifestURL: "", // empty
			},
		},
	}

	cacheDir := t.TempDir()
	results, warnings := LoadManifests(entries, cacheDir, "http://unused")

	assert.Empty(t, results)
	assert.Empty(t, warnings)
}

func TestSlugToToolPrefix(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		want    string
		wantErr bool
	}{
		{
			name: "simple slug",
			slug: "dub",
			want: "dub",
		},
		{
			name: "hyphenated slug",
			slug: "steam-web",
			want: "steam_web",
		},
		{
			name: "consecutive hyphens collapsed",
			slug: "steam--web",
			want: "steam_web",
		},
		{
			name: "multiple hyphens",
			slug: "my-cool-api",
			want: "my_cool_api",
		},
		{
			name:    "empty slug errors",
			slug:    "",
			wantErr: true,
		},
		{
			name:    "single hyphen errors",
			slug:    "-",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := slugToToolPrefix(tt.slug)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestLoadManifestsNormalizedPrefix(t *testing.T) {
	data := marshalManifest(t, makeTestManifest("Steam Web"))
	checksum := ComputeChecksum(data)

	server := makeManifestServer(t, map[string][]byte{
		"steam-web": data,
	})
	defer server.Close()

	entries := []RegistryEntry{
		{
			API: "steam-web",
			MCP: RegistryMCP{
				MCPReady:         "full",
				ManifestChecksum: checksum,
				ManifestURL:      "library/steam-web/tools-manifest.json",
			},
		},
	}

	cacheDir := t.TempDir()
	results, warnings := LoadManifests(entries, cacheDir, server.URL)

	assert.Empty(t, warnings)
	require.Len(t, results, 1)
	assert.Equal(t, "steam_web", results[0].NormalizedPrefix)
}
