package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mvanhorn/cli-printing-press/internal/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratedREADMEHasNoPlaceholderMarkers asserts that no <!-- *_OUTPUT -->
// HTML-comment markers ship in the rendered README. These markers were left
// over from an abandoned post-generate augmentation flow; the machine never
// populated them, so they leaked into every printed CLI as visible artifacts.
// Regression guard: if anyone re-introduces a marker without wiring up a
// fill path, this test fails.
func TestGeneratedREADMEHasNoPlaceholderMarkers(t *testing.T) {
	t.Parallel()

	apiSpec := &spec.APISpec{
		Name:    "markerless",
		Version: "0.1.0",
		BaseURL: "https://api.example.com",
		Auth: spec.AuthConfig{
			Type:    "api_key",
			Header:  "Authorization",
			Format:  "Bearer {token}",
			EnvVars: []string{"MARKERLESS_API_KEY"},
		},
		Config: spec.ConfigSpec{
			Format: "toml",
			Path:   "~/.config/markerless-pp-cli/config.toml",
		},
		Resources: map[string]spec.Resource{
			"items": {
				Description: "Manage items",
				Endpoints: map[string]spec.Endpoint{
					"list": {
						Method:      "GET",
						Path:        "/items",
						Description: "List items",
					},
				},
			},
		},
	}

	outputDir := filepath.Join(t.TempDir(), "markerless-pp-cli")
	gen := New(apiSpec, outputDir)
	require.NoError(t, gen.Generate())

	readme, err := os.ReadFile(filepath.Join(outputDir, "README.md"))
	require.NoError(t, err)
	content := string(readme)

	for _, marker := range []string{
		"<!-- HELP_OUTPUT -->",
		"<!-- DOCTOR_OUTPUT -->",
		"<!-- VERSION_OUTPUT -->",
	} {
		assert.False(t, strings.Contains(content, marker),
			"rendered README still contains placeholder marker %q — no machine code replaces it", marker)
	}
}
