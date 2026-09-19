// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseBoolEnvValue(t *testing.T) {
	tests := []struct {
		raw     string
		want    bool
		wantOk  bool
		comment string
	}{
		{"true", true, true, "canonical true"},
		{"1", true, true, "numeric true"},
		{"TRUE", true, true, "case-insensitive"},
		{" yes ", true, true, "trims whitespace"},
		{"on", true, true, "on"},
		{"false", false, true, "canonical false"},
		{"0", false, true, "numeric false"},
		{"no", false, true, "no"},
		{"off", false, true, "off"},
		{"", false, false, "empty means unset"},
		{"maybe", false, false, "garbage means unset"},
	}
	for _, tt := range tests {
		got, ok := parseBoolEnvValue(tt.raw)
		if got != tt.want || ok != tt.wantOk {
			t.Errorf("parseBoolEnvValue(%q) = (%v, %v), want (%v, %v) [%s]", tt.raw, got, ok, tt.want, tt.wantOk, tt.comment)
		}
	}
}

func TestAnonymizationConfig_DefaultsToDisabled(t *testing.T) {
	cfg := defaultConfig()
	if cfg.DataPlane.Anonymization.Enabled {
		t.Error("expected anonymization to default to disabled")
	}
}

func TestAnonymizationConfig_MissingKeyInYAMLDefaultsToDisabled(t *testing.T) {
	// A config.yaml written before this field existed (or one that simply
	// omits "anonymization.enabled") must still resolve to disabled, not
	// zero-value-panic or accidentally enable a behavior change for
	// existing installs upgrading in place.
	yamlNoAnonKey := []byte(`
dataplane:
  listen_addr: "0.0.0.0:1053"
  upstream_resolvers:
    - "8.8.8.8:53"
`)
	var cfg Config
	if err := yaml.Unmarshal(yamlNoAnonKey, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.DataPlane.Anonymization.Enabled {
		t.Error("expected Enabled=false when the yaml key is absent")
	}
	if cfg.DataPlane.Anonymization.Secret != "" {
		t.Errorf("expected empty Secret when the yaml key is absent, got %q", cfg.DataPlane.Anonymization.Secret)
	}
}
