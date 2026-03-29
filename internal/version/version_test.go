package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionReturnsHardcodedDefault(t *testing.T) {
	assert.NotEmpty(t, Version)
	assert.Equal(t, "0.4.0", Version)
}

func TestGetReturnsVersion(t *testing.T) {
	assert.Equal(t, Version, Get())
}
