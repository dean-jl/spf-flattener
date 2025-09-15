// Package config provides configuration management for the SPF Flattener application.
//
// This package handles loading and validating configuration from YAML files,
// with support for environment variable overrides for sensitive data like API keys.
// The configuration supports multiple domains, different DNS providers, and
// various operational settings.
//
// Configuration Structure:
//   - Global provider setting with per-domain overrides
//   - Multiple domain configurations with individual API credentials
//   - Optional DNS server specifications for custom resolution
//   - Logging and dry-run mode settings
//
// Environment Variables:
//   - SPF_FLATTENER_API_KEY: Default API key for domains without explicit keys
//   - SPF_FLATTENER_SECRET_KEY: Default secret key for domains without explicit keys
//
// Example configuration:
//
//	provider: porkbun
//	domains:
//	  - name: example.com
//	    api_key: "pk1_..."
//	    secret_key: "sk1_..."
//	  - name: another.com
//	    api_key: "pk2_..."
//	    secret_key: "sk2_..."
//	logging: true
//	dry_run: false
//	dns:
//	  - name: "Cloudflare"
//	    ip: "1.1.1.1"
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

const Version = "1.0.0"

const (
	EnvAPIKey    = "SPF_FLATTENER_API_KEY"
	EnvSecretKey = "SPF_FLATTENER_SECRET_KEY"
)

// Domain name validation regex - matches valid DNS domain names
var domainNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

// isValidDomainName validates a domain name according to DNS standards
func isValidDomainName(domain string) bool {
	// Basic checks
	if domain == "" || len(domain) > 253 {
		return false
	}

	// Remove trailing dot if present (for FQDN)
	domain = strings.TrimSuffix(domain, ".")

	// Check against regex pattern
	if !domainNameRegex.MatchString(domain) {
		return false
	}

	// Additional checks
	parts := strings.Split(domain, ".")
	for _, part := range parts {
		// Each label must be 1-63 characters
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		// Cannot start or end with hyphen
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return false
		}
	}

	return true
}

type DNSServer struct {
	Name string `yaml:"name"`
	IP   string `yaml:"ip"`
}

type Config struct {
	Provider   string      `yaml:"provider" validate:"required"`
	Domains    []Domain    `yaml:"domains" validate:"required,dive"`
	Logging    bool        `yaml:"logging"`
	DryRun     bool        `yaml:"dry_run"`
	DNSServers []DNSServer `yaml:"dns"`
}

type Domain struct {
	Name              string            `yaml:"name" validate:"required"`
	Provider          string            `yaml:"provider"`          // Per-domain provider override
	Options           map[string]string `yaml:"options,omitempty"` // Provider-specific options
	ApiKey            string            `yaml:"api_key,omitempty" validate:"required"`
	SecretKey         string            `yaml:"secret_key,omitempty" validate:"required"`
	OldRecordsLogFile string            `yaml:"old_records_log_file,omitempty"`
	TTL               int               `yaml:"ttl,omitempty"`
	Logging           *bool             `yaml:"logging,omitempty"`
	DryRun            *bool             `yaml:"dry_run,omitempty"`
}

// Validate validates the configuration using struct tags
func (c *Config) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	if len(c.Domains) == 0 {
		return fmt.Errorf("at least one domain is required")
	}

	for i, domain := range c.Domains {
		if err := domain.validate(); err != nil {
			return fmt.Errorf("domain[%d]: %w", i, err)
		}
	}

	return nil
}

// validate validates a single domain configuration
func (d *Domain) validate() error {
	if d.Name == "" {
		return fmt.Errorf("domain name is required")
	}
	if !isValidDomainName(d.Name) {
		return fmt.Errorf("invalid domain name format: %s", d.Name)
	}
	if d.ApiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if d.SecretKey == "" {
		return fmt.Errorf("secret key is required")
	}
	return nil
}

// LoadConfig loads and validates a configuration file from the specified path.
//
// This function performs the following operations:
//  1. Reads the YAML configuration file
//  2. Parses the YAML into the Config structure
//  3. Applies environment variable overrides for API keys/secrets
//  4. Sets default values for optional fields
//  5. Validates the complete configuration
//
// Environment variable processing:
//   - If a domain's api_key is empty, uses SPF_FLATTENER_API_KEY
//   - If a domain's secret_key is empty, uses SPF_FLATTENER_SECRET_KEY
//   - If a domain's provider is empty, uses the global provider setting
//
// Parameters:
//   - path: File path to the YAML configuration file
//
// Returns:
//   - *Config: Validated configuration object
//   - error: Any error encountered during loading, parsing, or validation
//
// Example:
//
//	cfg, err := LoadConfig("config.yaml")
//	if err != nil {
//		log.Fatalf("Failed to load config: %v", err)
//	}
//	fmt.Printf("Loaded config for %d domains\n", len(cfg.Domains))
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config file %s: %w", path, err)
	}

	// Load API keys/secrets from env if not set in YAML
	for i := range config.Domains {
		if config.Domains[i].Provider == "" {
			config.Domains[i].Provider = config.Provider // Use top-level provider if not set
		}
		if config.Domains[i].ApiKey == "" {
			config.Domains[i].ApiKey = os.Getenv(EnvAPIKey)
		}
		if config.Domains[i].SecretKey == "" {
			config.Domains[i].SecretKey = os.Getenv(EnvSecretKey)
		}
		// Set per-domain logging/dry-run overrides if present
		if config.Domains[i].Logging == nil {
			config.Domains[i].Logging = &config.Logging
		}
		if config.Domains[i].DryRun == nil {
			config.Domains[i].DryRun = &config.DryRun
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// SetDefaultTTL Set default TTL if not specified
func (c *Config) SetDefaultTTL() {
	for i := range c.Domains {
		if c.Domains[i].TTL == 0 {
			c.Domains[i].TTL = 600
		}
	}
}
