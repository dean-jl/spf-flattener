package config

import (
	"os"
	"testing"
)

// Benchmark for loading and parsing config files
func BenchmarkLoadConfig_Simple(b *testing.B) {
	configContent := `
provider: porkbun
domains:
  - name: example.com
    api_key: "test-key"
    secret_key: "test-secret"
logging: false
dry_run: false
`

	configFile := "bench_config_simple.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configFile)
		if err != nil {
			b.Fatalf("LoadConfig failed: %v", err)
		}
	}
}

// Benchmark for loading complex config with many domains
func BenchmarkLoadConfig_ManyDomains(b *testing.B) {
	configContent := `
provider: porkbun
domains:
  - name: example1.com
    api_key: "test-key-1"
    secret_key: "test-secret-1"
  - name: example2.com
    api_key: "test-key-2"
    secret_key: "test-secret-2"
  - name: example3.com
    api_key: "test-key-3"
    secret_key: "test-secret-3"
  - name: example4.com
    api_key: "test-key-4"
    secret_key: "test-secret-4"
  - name: example5.com
    api_key: "test-key-5"
    secret_key: "test-secret-5"
logging: true
dry_run: false
dns:
  - name: "Cloudflare"
    ip: "1.1.1.1"
  - name: "Google"
    ip: "8.8.8.8"
`

	configFile := "bench_config_complex.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configFile)
		if err != nil {
			b.Fatalf("LoadConfig failed: %v", err)
		}
	}
}

// Benchmark for config validation
func BenchmarkConfigValidation(b *testing.B) {
	config := &Config{
		Provider: "porkbun",
		Domains: []Domain{
			{
				Name:      "example1.com",
				ApiKey:    "test-key-1",
				SecretKey: "test-secret-1",
				Provider:  "porkbun",
			},
			{
				Name:      "example2.com",
				ApiKey:    "test-key-2",
				SecretKey: "test-secret-2",
				Provider:  "porkbun",
			},
			{
				Name:      "example3.com",
				ApiKey:    "test-key-3",
				SecretKey: "test-secret-3",
				Provider:  "porkbun",
			},
		},
		Logging: true,
		DryRun:  false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Fatalf("Config validation failed: %v", err)
		}
	}
}

// Benchmark for domain validation
func BenchmarkDomainValidation(b *testing.B) {
	domain := &Domain{
		Name:      "example.com",
		ApiKey:    "test-api-key",
		SecretKey: "test-secret-key",
		Provider:  "porkbun",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := domain.validate()
		if err != nil {
			b.Fatalf("Domain validation failed: %v", err)
		}
	}
}

// Benchmark for environment variable loading
func BenchmarkEnvVarLoading(b *testing.B) {
	// Set up environment variables
	os.Setenv("SPF_FLATTENER_API_KEY", "env-api-key")
	os.Setenv("SPF_FLATTENER_SECRET_KEY", "env-secret-key")
	defer func() {
		os.Unsetenv("SPF_FLATTENER_API_KEY")
		os.Unsetenv("SPF_FLATTENER_SECRET_KEY")
	}()

	configContent := `
provider: porkbun
domains:
  - name: example.com
    api_key: ""
    secret_key: ""
`

	configFile := "bench_config_env.yaml"
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Remove(configFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configFile)
		if err != nil {
			b.Fatalf("LoadConfig with env vars failed: %v", err)
		}
	}
}
