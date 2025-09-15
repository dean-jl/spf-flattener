package processor

import (
	"testing"

	"github.com/dean-jl/spf-flattener/internal/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestGroupDomainsByProvider(t *testing.T) {
	t.Run("Groups domains by provider", func(t *testing.T) {
		domains := []config.Domain{
			{Name: "example.com", Provider: "porkbun"},
			{Name: "test.com", Provider: "porkbun"},
			{Name: "mydomain.org", Provider: "cloudflare"},
			{Name: "api.example.com", Provider: "Porkbun"}, // Test case insensitive
		}

		groups := GroupDomainsByProvider(domains)

		assert.Len(t, groups, 2)
		assert.Contains(t, groups, "porkbun")
		assert.Contains(t, groups, "cloudflare")

		porkbunGroup := groups["porkbun"]
		assert.Equal(t, "porkbun", porkbunGroup.Provider)
		assert.Len(t, porkbunGroup.Domains, 3)
		assert.NotNil(t, porkbunGroup.RateLimiter)

		cloudflareGroup := groups["cloudflare"]
		assert.Equal(t, "cloudflare", cloudflareGroup.Provider)
		assert.Len(t, cloudflareGroup.Domains, 1)
		assert.NotNil(t, cloudflareGroup.RateLimiter)
	})

	t.Run("Creates rate limiters with correct settings", func(t *testing.T) {
		domains := []config.Domain{
			{Name: "example.com", Provider: "porkbun"},
		}

		groups := GroupDomainsByProvider(domains)

		porkbunGroup := groups["porkbun"]

		// Check rate limiter settings (2 req/sec, burst 1)
		assert.NotNil(t, porkbunGroup.RateLimiter)
		assert.Equal(t, rate.Limit(2.0), porkbunGroup.RateLimiter.Limit())
		assert.Equal(t, 1, porkbunGroup.RateLimiter.Burst())
	})

	t.Run("Handles empty domain list", func(t *testing.T) {
		domains := []config.Domain{}

		groups := GroupDomainsByProvider(domains)

		assert.Empty(t, groups)
	})

	t.Run("Handles single domain", func(t *testing.T) {
		domains := []config.Domain{
			{Name: "example.com", Provider: "porkbun"},
		}

		groups := GroupDomainsByProvider(domains)

		assert.Len(t, groups, 1)
		assert.Contains(t, groups, "porkbun")
		assert.Len(t, groups["porkbun"].Domains, 1)
	})
}
