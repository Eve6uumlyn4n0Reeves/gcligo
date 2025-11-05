package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWarnLegacyOverridesTracksFields(t *testing.T) {
	resetLegacyWarnings()
	cfg := &Config{}
	cfg.OpenAIPort = "9999"
	cfg.Server.OpenAIPort = ""

	cfg.warnLegacyOverrides("test")

	_, warned := legacyWarned.Load("OpenAIPort")
	require.True(t, warned, "expected legacy warning for OpenAIPort")
}
