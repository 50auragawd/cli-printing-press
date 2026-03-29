package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/mvanhorn/cli-printing-press/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGoreleaserLdflagsTargetMatchesVersionVar(t *testing.T) {
	// The goreleaser config injects the version via ldflags into
	// internal/version.Version. If the variable is renamed or moved,
	// goreleaser silently injects into nothing and the binary
	// reports the hardcoded fallback. This test catches that drift.

	// 1. Verify the version variable exists and is settable.
	assert.IsType(t, "", version.Version)

	// 2. Verify the goreleaser config references the correct ldflags path.
	data, err := os.ReadFile("../../.goreleaser.yaml")
	require.NoError(t, err)

	var config struct {
		Builds []struct {
			Ldflags []string `yaml:"ldflags"`
		} `yaml:"builds"`
	}
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.NotEmpty(t, config.Builds)

	ldflags := strings.Join(config.Builds[0].Ldflags, " ")
	assert.Contains(t, ldflags,
		"github.com/mvanhorn/cli-printing-press/internal/version.Version",
		"goreleaser ldflags must target internal/version.Version")
}

func TestReleasePleaseAnnotationExists(t *testing.T) {
	// release-please uses the x-release-please-version annotation
	// to find and bump the hardcoded version. If the annotation is
	// removed, release-please silently stops updating it.
	data, err := os.ReadFile("../version/version.go")
	require.NoError(t, err)

	assert.Contains(t, string(data), "x-release-please-version",
		"version.go must have x-release-please-version annotation for automated version bumps")
}

func TestVersionConsistencyAcrossFiles(t *testing.T) {
	// All version surfaces should match. release-please keeps them
	// in sync, but this catches manual edits that drift.

	// Read plugin.json version
	pluginData, err := os.ReadFile("../../.claude-plugin/plugin.json")
	require.NoError(t, err)

	var plugin struct {
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(pluginData, &plugin))

	// Read marketplace.json version
	marketData, err := os.ReadFile("../../.claude-plugin/marketplace.json")
	require.NoError(t, err)

	var market struct {
		Plugins []struct {
			Version string `json:"version"`
		} `json:"plugins"`
	}
	require.NoError(t, json.Unmarshal(marketData, &market))
	require.NotEmpty(t, market.Plugins)

	// All three should match
	assert.Equal(t, plugin.Version, market.Plugins[0].Version,
		"plugin.json and marketplace.json versions must match")
	assert.Equal(t, plugin.Version, version.Version,
		"plugin.json and version.go hardcoded version must match")
}
