package processor

import (
	"strings"

	"github.com/dean-jl/spf-flattener/internal/config"
	"golang.org/x/time/rate"
)

// ProviderGroup represents domains grouped by DNS provider
type ProviderGroup struct {
	Provider    string
	Domains     []config.Domain
	RateLimiter *rate.Limiter
}

// GroupDomainsByProvider groups domains by their DNS provider
// Each provider group gets its own rate limiter to prevent conflicts
func GroupDomainsByProvider(domains []config.Domain) map[string]*ProviderGroup {
	groups := make(map[string]*ProviderGroup)

	for _, domain := range domains {
		provider := strings.ToLower(domain.Provider)
		if groups[provider] == nil {
			groups[provider] = &ProviderGroup{
				Provider:    provider,
				Domains:     make([]config.Domain, 0),
				RateLimiter: rate.NewLimiter(rate.Limit(2.0), 1), // 2 req/sec, burst 1
			}
		}
		groups[provider].Domains = append(groups[provider].Domains, domain)
	}

	return groups
}
