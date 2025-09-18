package spf

import (
	"net"
	"testing"
)

// TestAggregateCIDRsWithConfig tests the advanced CIDR aggregation with configuration
func TestAggregateCIDRsWithConfig(t *testing.T) {
	tests := []struct {
		name       string
		mechanisms []string
		config     *AggregationConfig
		expected   []string
	}{
		{
			name:       "Basic aggregation with default config",
			mechanisms: []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.2", "ip4:192.168.1.3"},
			config:     nil, // Should use defaults
			expected:   []string{"ip4:192.168.1.0/30"},
		},
		{
			name:       "IPv4 min prefix restriction prevents aggregation",
			mechanisms: []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.2", "ip4:192.168.1.3"},
			config: &AggregationConfig{
				IPv4MaxPrefix:      32, // Only allow individual IPs (/32)
				IPv6MaxPrefix:      64,
				PreserveIndividual: []string{},
			},
			expected: []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.2", "ip4:192.168.1.3"},
		},
		{
			name:       "IPv4 min prefix allows aggregation",
			mechanisms: []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.2", "ip4:192.168.1.3"},
			config: &AggregationConfig{
				IPv4MaxPrefix:      20, // Less restrictive than /30
				IPv6MaxPrefix:      64,
				PreserveIndividual: []string{},
			},
			expected: []string{"ip4:192.168.1.0/30"},
		},
		{
			name:       "Preserve individual IPv4 addresses",
			mechanisms: []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.2", "ip4:192.168.1.3"},
			config: &AggregationConfig{
				IPv4MaxPrefix:      24,
				IPv6MaxPrefix:      64,
				PreserveIndividual: []string{"192.168.1.1", "192.168.1.3"},
			},
			expected: []string{"ip4:192.168.1.0", "ip4:192.168.1.2", "ip4:192.168.1.1", "ip4:192.168.1.3"},
		},
		{
			name:       "IPv6 aggregation with min prefix",
			mechanisms: []string{"ip6:2001:db8::0", "ip6:2001:db8::1"},
			config: &AggregationConfig{
				IPv4MaxPrefix:      24,
				IPv6MaxPrefix:      64,
				PreserveIndividual: []string{},
			},
			expected: []string{"ip6:2001:db8::/127"},
		},
		{
			name:       "IPv6 min prefix prevents aggregation",
			mechanisms: []string{"ip6:2001:db8::0", "ip6:2001:db8::1"},
			config: &AggregationConfig{
				IPv4MaxPrefix:      24,
				IPv6MaxPrefix:      128, // More restrictive than /127
				PreserveIndividual: []string{},
			},
			expected: []string{"ip6:2001:db8::", "ip6:2001:db8::1"},
		},
		{
			name:       "Mixed IPv4/IPv6 with preserve",
			mechanisms: []string{"ip4:192.168.1.1", "ip4:192.168.1.2", "ip6:2001:db8::1", "include:example.com"},
			config: &AggregationConfig{
				IPv4MaxPrefix:      24,
				IPv6MaxPrefix:      64,
				PreserveIndividual: []string{"192.168.1.1"},
			},
			expected: []string{"ip4:192.168.1.2", "ip6:2001:db8::1", "include:example.com", "ip4:192.168.1.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregateCIDRsWithConfig(tt.mechanisms, tt.config)

			// Convert to map for easier comparison (order doesn't matter for most tests)
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}

			expectedMap := make(map[string]bool)
			for _, e := range tt.expected {
				expectedMap[e] = true
			}

			if len(result) != len(tt.expected) {
				t.Errorf("AggregateCIDRsWithConfig() returned %d items, expected %d. Got: %v, Expected: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("AggregateCIDRsWithConfig() missing expected result: %s. Got: %v",
						expected, result)
				}
			}

			for _, actual := range result {
				if !expectedMap[actual] {
					t.Errorf("AggregateCIDRsWithConfig() contains unexpected result: %s. Expected: %v",
						actual, tt.expected)
				}
			}
		})
	}
}

// TestAggregateIPv4WithConfig tests IPv4-specific advanced aggregation
func TestAggregateIPv4WithConfig(t *testing.T) {
	tests := []struct {
		name     string
		ips      []string
		config   *AggregationConfig
		expected []string
	}{
		{
			name: "Preserve specific IPs",
			ips:  []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"},
			config: &AggregationConfig{
				IPv4MaxPrefix:      24,
				IPv6MaxPrefix:      64,
				PreserveIndividual: []string{"192.168.1.2", "192.168.1.4"},
			},
			expected: []string{"ip4:192.168.1.1", "ip4:192.168.1.3", "ip4:192.168.1.2", "ip4:192.168.1.4"},
		},
		{
			name: "Min prefix prevents aggregation",
			ips:  []string{"192.168.1.1", "192.168.1.2"},
			config: &AggregationConfig{
				IPv4MaxPrefix:      32, // Single hosts only
				IPv6MaxPrefix:      64,
				PreserveIndividual: []string{},
			},
			expected: []string{"ip4:192.168.1.1", "ip4:192.168.1.2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert string IPs to net.IP
			var ips []net.IP
			for _, ipStr := range tt.ips {
				ip := net.ParseIP(ipStr)
				if ip != nil {
					ips = append(ips, ip)
				}
			}

			result := aggregateIPv4WithConfig(ips, tt.config)

			if len(result) != len(tt.expected) {
				t.Errorf("aggregateIPv4WithConfig() returned %d items, expected %d. Got: %v, Expected: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			// Convert to maps for comparison
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("aggregateIPv4WithConfig() missing expected result: %s. Got: %v",
						expected, result)
				}
			}
		})
	}
}

// BenchmarkAggregateCIDRsWithConfig benchmarks the advanced aggregation
func BenchmarkAggregateCIDRsWithConfig(b *testing.B) {
	mechanisms := []string{
		"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.2", "ip4:192.168.1.3",
		"ip6:2001:db8::0", "ip6:2001:db8::1",
		"include:example.com",
	}

	config := &AggregationConfig{
		IPv4MaxPrefix:      24,
		IPv6MaxPrefix:      64,
		PreserveIndividual: []string{"192.168.1.1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AggregateCIDRsWithConfig(mechanisms, config)
	}
}
