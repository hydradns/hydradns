// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"fmt"
	"strings"
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
		{"ON", true, true, "on, case-insensitive"},
		{"false", false, true, "canonical false"},
		{"0", false, true, "numeric false"},
		{"no", false, true, "no"},
		{"off", false, true, "off"},
		{" off ", false, true, "off, trims whitespace"},
		{"", false, false, "empty means unset"},
		{"maybe", false, false, "garbage means unset"},
		{"true ", true, true, "trailing space (the exact near-miss H1 flagged for HYDRA_DEMO_MODE)"},
	}
	for _, tt := range tests {
		got, ok := ParseBoolEnvValue(tt.raw)
		if got != tt.want || ok != tt.wantOk {
			t.Errorf("ParseBoolEnvValue(%q) = (%v, %v), want (%v, %v) [%s]", tt.raw, got, ok, tt.want, tt.wantOk, tt.comment)
		}
	}
}

func TestMustParseBoolEnv_UnsetUsesDefault(t *testing.T) {
	t.Setenv("HYDRA_TEST_BOOL", "")
	if got := MustParseBoolEnv("HYDRA_TEST_BOOL", true); !got {
		t.Errorf("expected default true when unset, got %v", got)
	}
	if got := MustParseBoolEnv("HYDRA_TEST_BOOL", false); got {
		t.Errorf("expected default false when unset, got %v", got)
	}
}

func TestMustParseBoolEnv_RecognizedValues(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"1", true}, {"true", true}, {"yes", true}, {"on", true},
		{"0", false}, {"false", false}, {"no", false}, {"off", false},
	}
	for _, tc := range cases {
		t.Setenv("HYDRA_TEST_BOOL", tc.raw)
		if got := MustParseBoolEnv("HYDRA_TEST_BOOL", !tc.want); got != tc.want {
			t.Errorf("MustParseBoolEnv(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

// TestMustParseBoolEnv_UnrecognizedValueCallsFatalFunc proves the fatal
// path fires exactly on an unrecognized value, without exiting the test
// binary — see H1: a near-miss value must fail loudly, not silently take
// the default.
func TestMustParseBoolEnv_UnrecognizedValueCallsFatalFunc(t *testing.T) {
	orig := FatalFunc
	defer func() { FatalFunc = orig }()

	var gotFormat string
	var gotArgs []interface{}
	called := false
	FatalFunc = func(format string, args ...interface{}) {
		called = true
		gotFormat = format
		gotArgs = args
	}

	t.Setenv("HYDRA_DEMO_MODE", "tru") // near-miss typo: not a recognized token
	MustParseBoolEnv("HYDRA_DEMO_MODE", false)

	if !called {
		t.Fatal("expected FatalFunc to be called for an unrecognized value")
	}
	msg := fmt.Sprintf(gotFormat, gotArgs...)
	if !strings.Contains(msg, "HYDRA_DEMO_MODE") {
		t.Errorf("expected the fatal message to name the variable, got %q", msg)
	}
	if !strings.Contains(msg, "tru") {
		t.Errorf("expected the fatal message to include the offending value, got %q", msg)
	}
}

func TestMustParseBoolEnv_RecognizedValueDoesNotCallFatalFunc(t *testing.T) {
	orig := FatalFunc
	defer func() { FatalFunc = orig }()
	FatalFunc = func(format string, args ...interface{}) {
		t.Fatalf("FatalFunc must not be called for a recognized value: "+format, args...)
	}

	t.Setenv("HYDRA_DEMO_MODE", "true")
	if !MustParseBoolEnv("HYDRA_DEMO_MODE", false) {
		t.Error("expected true")
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
