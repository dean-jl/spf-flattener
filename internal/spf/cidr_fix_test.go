package spf

import (
	"testing"
)

// TestCIDRPreservation tests that existing CIDR blocks are preserved and not expanded
func TestCIDRPreservation(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "Preserve existing CIDR blocks",
			input: []string{
				"ip4:209.85.128.0/17", // Should be preserved as-is
				"ip4:64.233.160.0/19", // Should be preserved as-is
				"ip4:203.0.113.100",   // Individual IP should get /32
			},
			expected: []string{
				"ip4:203.0.113.100",   // Individual IP stays as bare address
				"ip4:209.85.128.0/17", // CIDR block preserved
				"ip4:64.233.160.0/19", // CIDR block preserved
			},
		},
		{
			name: "Mix CIDR blocks with contiguous individual IPs",
			input: []string{
				"ip4:209.61.151.0/24", // Existing CIDR - preserve
				"ip4:192.168.1.0",     // Individual IP - can aggregate (aligned)
				"ip4:192.168.1.1",     // Individual IP - can aggregate (aligned)
				"ip4:198.61.254.0/24", // Existing CIDR - preserve
			},
			expected: []string{
				"ip4:192.168.1.0/31",  // Aggregated individual IPs
				"ip4:209.61.151.0/24", // Preserved CIDR
				"ip4:198.61.254.0/24", // Preserved CIDR
			},
		},
		{
			name: "IPv6 CIDR preservation",
			input: []string{
				"ip6:2001:db8::/64", // Existing IPv6 CIDR - preserve
				"ip6:2001:db8:1::0", // Individual IPv6 - can aggregate (aligned)
				"ip6:2001:db8:1::1", // Individual IPv6 - can aggregate (aligned)
			},
			expected: []string{
				"ip6:2001:db8:1::/127", // Aggregated individual IPs
				"ip6:2001:db8::/64",    // Preserved CIDR
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregateCIDRs(tt.input)

			// Convert to maps for easier comparison (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, r := range result {
				resultMap[r] = true
			}

			expectedMap := make(map[string]bool)
			for _, e := range tt.expected {
				expectedMap[e] = true
			}

			if len(result) != len(tt.expected) {
				t.Errorf("AggregateCIDRs() returned %d items, expected %d. Got: %v, Expected: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("AggregateCIDRs() missing expected result: %s. Got: %v",
						expected, result)
				}
			}

			for _, actual := range result {
				if !expectedMap[actual] {
					t.Errorf("AggregateCIDRs() contains unexpected result: %s. Expected: %v",
						actual, tt.expected)
				}
			}
		})
	}
}

// TestNoCIDRExpansion specifically tests that CIDR blocks don't get expanded into individual IPs
func TestNoCIDRExpansion(t *testing.T) {
	// This test should complete quickly - if CIDR blocks are expanded, it would timeout
	input := []string{
		"ip4:209.85.128.0/17", // This is 32,768 IP addresses - should NOT be expanded
	}

	result := AggregateCIDRs(input)

	// Should return exactly one result - the original CIDR block
	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d: %v", len(result), result)
	}

	if result[0] != "ip4:209.85.128.0/17" {
		t.Errorf("Expected preserved CIDR block, got: %s", result[0])
	}
}
