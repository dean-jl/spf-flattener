package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	configContent := `
provider: porkbun
domains:
  - name: example.com
    api_key: "pk1_..."
    secret_key: "sk1_..."
  - name: another-domain.com
    api_key: "pk2_..."
    secret_key: "sk2_..."
`
	configFile := "config.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Domains) != 2 {
		t.Fatalf("Expected 2 domains, got %d", len(cfg.Domains))
	}

	if cfg.Domains[0].Name != "example.com" {
		t.Errorf("Expected domain name 'example.com', got '%s'", cfg.Domains[0].Name)
	}

	if cfg.Domains[1].ApiKey != "pk2_..." {
		t.Errorf("Expected api key 'pk2_...', got '%s'", cfg.Domains[1].ApiKey)
	}
}

func TestLoadConfig_EnvVars(t *testing.T) {
	if err := os.Setenv("SPF_FLATTENER_API_KEY", "env_api_key"); err != nil {
		t.Fatalf("Failed to set env: %v", err)
	}
	if err := os.Setenv("SPF_FLATTENER_SECRET_KEY", "env_secret_key"); err != nil {
		t.Fatalf("Failed to set env: %v", err)
	}
	configContent := `
provider: porkbun
domains:
  - name: envdomain.com
    api_key: ""
    secret_key: ""
`
	configFile := "config_env.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.Domains[0].ApiKey != "env_api_key" {
		t.Errorf("Expected API key from env, got '%s'", cfg.Domains[0].ApiKey)
	}
	if cfg.Domains[0].SecretKey != "env_secret_key" {
		t.Errorf("Expected secret key from env, got '%s'", cfg.Domains[0].SecretKey)
	}
}

func TestLoadConfig_Validation(t *testing.T) {
	configContent := `
domains:
  - name: ""
    api_key: ""
    secret_key: ""
`
	configFile := "config_invalid.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	_, err = LoadConfig(configFile)
	if err == nil {
		t.Errorf("Expected validation error for missing required fields, got nil")
	}
}

func TestLoadConfig_Options(t *testing.T) {
	configContent := `
provider: porkbun
domains:
  - name: optdomain.com
    api_key: "optkey"
    secret_key: "optsecret"
logging: true
dry_run: true
`
	configFile := "config_options.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if !cfg.Logging {
		t.Errorf("Expected logging=true, got false")
	}
	if !cfg.DryRun {
		t.Errorf("Expected dry_run=true, got false")
	}
}

// Additional comprehensive error path tests

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent_file.yaml")
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	configContent := `
provider: porkbun
domains:
  - name: test.com
    api_key: "key"
    secret_key: "secret"
  invalid_yaml: [unclosed
`
	configFile := "config_invalid_yaml.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	_, err = LoadConfig(configFile)
	if err == nil {
		t.Errorf("Expected YAML parsing error, got nil")
	}
}

func TestLoadConfig_MissingProvider(t *testing.T) {
	configContent := `
domains:
  - name: test.com
    api_key: "key"
    secret_key: "secret"
`
	configFile := "config_no_provider.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	_, err = LoadConfig(configFile)
	if err == nil {
		t.Errorf("Expected validation error for missing provider, got nil")
	}
}

func TestLoadConfig_EmptyDomains(t *testing.T) {
	configContent := `
provider: porkbun
domains: []
`
	configFile := "config_empty_domains.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	_, err = LoadConfig(configFile)
	if err == nil {
		t.Errorf("Expected validation error for empty domains, got nil")
	}
}

func TestLoadConfig_MissingDomainName(t *testing.T) {
	configContent := `
provider: porkbun
domains:
  - api_key: "key"
    secret_key: "secret"
`
	configFile := "config_missing_name.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	_, err = LoadConfig(configFile)
	if err == nil {
		t.Errorf("Expected validation error for missing domain name, got nil")
	}
}

// Domain name validation tests
func TestIsValidDomainName(t *testing.T) {
	validDomains := []string{
		"example.com",
		"sub.example.com",
		"test-domain.com",
		"a.b.c.d.e.f",
		"123.com",
		"example-123.com",
		"x.co",
		"a.io",
		"example.com.", // FQDN with trailing dot
	}

	invalidDomains := []string{
		"",                        // Empty string
		"-example.com",            // Starts with hyphen
		"example-.com",            // Ends with hyphen
		"ex ample.com",            // Contains space
		"example..com",            // Double dot
		".example.com",            // Starts with dot
		"example.com-",            // Ends with hyphen
		string(make([]byte, 254)), // Domain > 253 chars
		"example.c_m",             // Underscore (invalid in hostnames)
	}

	for _, domain := range validDomains {
		if !isValidDomainName(domain) {
			t.Errorf("Expected '%s' to be valid, but it was rejected", domain)
		}
	}

	for _, domain := range invalidDomains {
		if isValidDomainName(domain) {
			t.Errorf("Expected '%s' to be invalid, but it was accepted", domain)
		}
	}
}

func TestLoadConfig_InvalidDomainNames(t *testing.T) {
	testCases := []struct {
		domain   string
		expected string // Expected error substring
	}{
		{"", "domain name is required"},
		{"-invalid.com", "invalid domain name format"},
		{"invalid-.com", "invalid domain name format"},
		{"thisisaverylongdomainnamethatexceedsthe63characterlimitforlabels.com", "invalid domain name format"},
	}

	for _, tc := range testCases {
		configContent := `
provider: porkbun
domains:
  - name: "` + tc.domain + `"
    api_key: "key"
    secret_key: "secret"
`
		configFile := "config_invalid_domain.yaml"
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
		defer os.Remove(configFile)

		_, err = LoadConfig(configFile)
		if err == nil {
			t.Errorf("Expected validation error for invalid domain '%s', got nil", tc.domain)
		}
	}
}
