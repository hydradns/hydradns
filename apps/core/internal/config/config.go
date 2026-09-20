package config

// SPDX-License-Identifier: GPL-3.0-or-later
import (
	"os"
	"strings"

	"github.com/hydradns/hydra-core/internal/logger"
	"gopkg.in/yaml.v3"
)

type Config struct {
	DataPlane    DataPlaneConfig    `yaml:"dataplane"`
	ControlPlane ControlPlaneConfig `yaml:"controlplane"`
}

type GRPCServerConfig struct {
	ListenAddr string `yaml:"listen_addr"`
	Port       int    `yaml:"port"`
}

type DataPlaneConfig struct {
	ListenAddr              string              `yaml:"listen_addr"`
	UpstreamResolvers       []string            `yaml:"upstream_resolvers"`
	GRPCServer              GRPCServerConfig    `yaml:"grpc_server"`
	BlocklistUpdateInterval string              `yaml:"blocklist_update_interval"`
	Anonymization           AnonymizationConfig `yaml:"anonymization"`
}

// AnonymizationConfig controls whether the dataplane hashes client IPs
// (via utils.AnonymizeIP) before they're written to the query log.
//
// Enabled defaults to false: per-device visibility in the query log is a
// core feature of a home/office DNS firewall, so turning this on is an
// explicit opt-in (HYDRA_ANONYMIZE_CLIENT_IPS=true/1, or this field), not
// the default. Secret is the raw config-file value; it is not the final
// secret used at runtime — see ResolveAnonymizationSecret, which applies
// the HYDRA_ANON_SECRET env override and, only when anonymization is
// enabled, generates+persists a per-install secret if this is left at the
// documented placeholder.
type AnonymizationConfig struct {
	Enabled bool   `yaml:"enabled"`
	Secret  string `yaml:"secret"`
}

// ParseBoolEnvValue interprets a raw env var value as a boolean override.
// ok is false for an empty or unrecognized value, meaning "leave the
// configured value alone." Recognizes the same truthy/falsy tokens
// (case-insensitive, whitespace-trimmed) on both sides.
//
// Exported so every boolean env var across the control plane parses the
// same way — see MustParseBoolEnv, and the launch-prep review (H1): before
// this, HYDRA_DEMO_MODE used a strict EqualFold("true") with no trim (so
// "1", "yes", or a trailing space from a Docker env_file line silently
// left demo mode OFF), and CORS_ALLOW_SAME_HOST treated anything except
// the literal "false" as "on". One parser, one set of accepted spellings.
func ParseBoolEnvValue(raw string) (value bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

// FatalFunc is invoked by MustParseBoolEnv when an environment variable is
// set to a value ParseBoolEnvValue does not recognize. It defaults to
// logger.Log.Fatalf (which exits the process) but is a variable
// specifically so tests can override it and assert the fatal path without
// killing the test binary.
var FatalFunc = func(format string, args ...interface{}) {
	logger.Log.Fatalf(format, args...)
}

// MustParseBoolEnv reads name from the OS environment and resolves it to a
// boolean, falling back to def when the variable is unset or empty. A
// non-empty value that ParseBoolEnvValue does not recognize is a
// misconfiguration, not something to silently paper over: it calls
// FatalFunc naming both the variable and the offending value, so a
// near-miss (a typo, a stray space, "1" where only "true" used to work)
// fails loudly at startup instead of quietly taking the default — see H1
// in the launch-prep review, where exactly this silently disabled
// HYDRA_DEMO_MODE and left a public demo's /auth/setup open to the first
// visitor.
func MustParseBoolEnv(name string, def bool) bool {
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}
	v, ok := ParseBoolEnvValue(raw)
	if !ok {
		FatalFunc("invalid %s=%q: expected one of true/false/1/0/yes/no/on/off (case-insensitive, whitespace trimmed)", name, raw)
		return def // unreachable when FatalFunc actually exits; keeps a test-overridden FatalFunc from continuing with a bogus value
	}
	return v
}

type ControlPlaneConfig struct {
	ListenAddr string `yaml:"listen_addr"`
}

func defaultConfig() *Config {
	return &Config{
		DataPlane: DataPlaneConfig{
			ListenAddr:              "0.0.0.0:1053",
			UpstreamResolvers:       []string{"8.8.8.8:53", "1.1.1.1:53"},
			BlocklistUpdateInterval: "6h",
			GRPCServer: GRPCServerConfig{
				Port:       50051,
				ListenAddr: "localhost:50051",
			},
		},
		ControlPlane: ControlPlaneConfig{
			ListenAddr: "0.0.0.0:8080",
		},
	}
}

func loadConfig(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Log.Warnf("Config file not found (%s), using defaults", path)
		return defaultConfig()
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Log.Errorf("Failed to unmarshal config: %v, using defaults", err)
		return defaultConfig()
	}

	return &cfg
}

var DefaultConfig = func() *Config {
	cfg := loadConfig(configPath())
	if addr := os.Getenv("DNS_LISTEN_ADDR"); addr != "" {
		cfg.DataPlane.ListenAddr = addr
	}
	if interval := os.Getenv("BLOCKLIST_UPDATE_INTERVAL"); interval != "" {
		cfg.DataPlane.BlocklistUpdateInterval = interval
	}
	// Not routed through MustParseBoolEnv: this runs inside DefaultConfig's
	// package-level initializer, which executes once at import time, before
	// any test has a chance to set the env var or override FatalFunc — a
	// Fatal here would not be testable without restructuring config loading
	// into a lazy call, which is a larger change than this fix warrants.
	// Warn-and-keep-configured-value is the existing, deliberate behavior.
	if raw := os.Getenv("HYDRA_ANONYMIZE_CLIENT_IPS"); raw != "" {
		if v, ok := ParseBoolEnvValue(raw); ok {
			cfg.DataPlane.Anonymization.Enabled = v
		} else {
			logger.Log.Warnf("invalid HYDRA_ANONYMIZE_CLIENT_IPS=%q, keeping configured value %v", raw, cfg.DataPlane.Anonymization.Enabled)
		}
	}
	return cfg
}()

func configPath() string {
	if p := os.Getenv("HYDRA_CONFIG"); p != "" {
		return p
	}
	return "/app/configs/config.yaml"
}
