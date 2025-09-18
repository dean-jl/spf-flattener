package spf

import (
	"testing"
)

// TestSPFSemanticallyDifferent tests the semantic comparison functionality
func TestSPFSemanticallyDifferent(t *testing.T) {
	tests := []struct {
		name     string
		oldSPF   string
		newSPF   string
		expected bool // true if they are different
	}{
		{
			name:     "Identical SPF records",
			oldSPF:   "v=spf1 ip4:192.168.1.1 ip4:192.168.1.2 ~all",
			newSPF:   "v=spf1 ip4:192.168.1.1 ip4:192.168.1.2 ~all",
			expected: false, // Not different
		},
		{
			name:     "Same IPs different order",
			oldSPF:   "v=spf1 ip4:192.168.1.2 ip4:192.168.1.1 ~all",
			newSPF:   "v=spf1 ip4:192.168.1.1 ip4:192.168.1.2 ~all",
			expected: false, // Not different
		},
		{
			name:     "Aggregated vs individual IPs (contiguous)",
			oldSPF:   "v=spf1 ip4:192.168.1.0 ip4:192.168.1.1 ip4:192.168.1.2 ip4:192.168.1.3 ~all",
			newSPF:   "v=spf1 ip4:192.168.1.0/30 ~all",
			expected: false, // Not different - same IP set
		},
		{
			name:     "Different IP sets",
			oldSPF:   "v=spf1 ip4:192.168.1.1 ip4:192.168.1.2 ~all",
			newSPF:   "v=spf1 ip4:192.168.1.3 ip4:192.168.1.4 ~all",
			expected: true, // Different
		},
		{
			name:     "IPv6 aggregation equivalence",
			oldSPF:   "v=spf1 ip6:2001:db8::0 ip6:2001:db8::1 ~all",
			newSPF:   "v=spf1 ip6:2001:db8::/127 ~all",
			expected: false, // Not different - same IP set
		},
		{
			name:     "Mixed IPv4 and IPv6",
			oldSPF:   "v=spf1 ip4:192.168.1.1 ip6:2001:db8::1 ~all",
			newSPF:   "v=spf1 ip6:2001:db8::1 ip4:192.168.1.1 ~all",
			expected: false, // Not different - same IPs different order
		},
		{
			name:     "Non-IP mechanisms ignored",
			oldSPF:   "v=spf1 ip4:192.168.1.1 include:example.com ~all",
			newSPF:   "v=spf1 ip4:192.168.1.1 include:different.com ~all",
			expected: false, // Not different - only IP mechanisms compared
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SPFSemanticallyDifferent(tt.oldSPF, tt.newSPF)
			if result != tt.expected {
				t.Errorf("SPFSemanticallyDifferent(%q, %q) = %v, expected %v",
					tt.oldSPF, tt.newSPF, result, tt.expected)
			}
		})
	}
}

// TestExpandSPFToIPSet tests the IP set expansion functionality
func TestExpandSPFToIPSet(t *testing.T) {
	tests := []struct {
		name      string
		spfRecord string
		expected  map[string]bool
	}{
		{
			name:      "Single IPv4",
			spfRecord: "v=spf1 ip4:192.168.1.1 ~all",
			expected: map[string]bool{
				"192.168.1.1": true,
			},
		},
		{
			name:      "IPv4 CIDR /31",
			spfRecord: "v=spf1 ip4:192.168.1.0/31 ~all",
			expected: map[string]bool{
				"192.168.1.0": true,
				"192.168.1.1": true,
			},
		},
		{
			name:      "Mixed IPv4 and IPv6",
			spfRecord: "v=spf1 ip4:192.168.1.1 ip6:2001:db8::1 ~all",
			expected: map[string]bool{
				"192.168.1.1": true,
				"2001:db8::1": true,
			},
		},
		{
			name:      "Non-IP mechanisms ignored",
			spfRecord: "v=spf1 include:example.com ip4:192.168.1.1 a mx ~all",
			expected: map[string]bool{
				"192.168.1.1": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandSPFToIPSet(tt.spfRecord)

			if len(result) != len(tt.expected) {
				t.Errorf("expandSPFToIPSet(%q) returned %d IPs, expected %d",
					tt.spfRecord, len(result), len(tt.expected))
			}

			for expectedIP := range tt.expected {
				if !result[expectedIP] {
					t.Errorf("expandSPFToIPSet(%q) missing expected IP %s",
						tt.spfRecord, expectedIP)
				}
			}

			for actualIP := range result {
				if !tt.expected[actualIP] {
					t.Errorf("expandSPFToIPSet(%q) contains unexpected IP %s",
						tt.spfRecord, actualIP)
				}
			}
		})
	}
}

// TestExpandCIDRToIPs tests the CIDR expansion functionality
func TestExpandCIDRToIPs(t *testing.T) {
	tests := []struct {
		name      string
		mechanism string
		expected  []string
	}{
		{
			name:      "Single IPv4",
			mechanism: "ip4:192.168.1.1",
			expected:  []string{"192.168.1.1"},
		},
		{
			name:      "IPv4 /31 CIDR",
			mechanism: "ip4:192.168.1.0/31",
			expected:  []string{"192.168.1.0", "192.168.1.1"},
		},
		{
			name:      "Single IPv6",
			mechanism: "ip6:2001:db8::1",
			expected:  []string{"2001:db8::1"},
		},
		{
			name:      "IPv6 /127 CIDR",
			mechanism: "ip6:2001:db8::/127",
			expected:  []string{"2001:db8::", "2001:db8::1"},
		},
		{
			name:      "Invalid mechanism",
			mechanism: "include:example.com",
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandCIDRToIPs(tt.mechanism)

			if len(result) != len(tt.expected) {
				t.Errorf("expandCIDRToIPs(%q) returned %d IPs, expected %d",
					tt.mechanism, len(result), len(tt.expected))
				t.Errorf("Got: %v, Expected: %v", result, tt.expected)
			}

			// Convert result to map for easier comparison
			resultMap := make(map[string]bool)
			for _, ip := range result {
				resultMap[ip] = true
			}

			for _, expectedIP := range tt.expected {
				if !resultMap[expectedIP] {
					t.Errorf("expandCIDRToIPs(%q) missing expected IP %s",
						tt.mechanism, expectedIP)
				}
			}
		})
	}
}

// BenchmarkSPFSemanticallyDifferent benchmarks the semantic comparison performance
func BenchmarkSPFSemanticallyDifferent(b *testing.B) {
	oldSPF := "v=spf1 ip4:192.168.1.0 ip4:192.168.1.1 ip4:192.168.1.2 ip4:192.168.1.3 ~all"
	newSPF := "v=spf1 ip4:192.168.1.0/30 ~all"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SPFSemanticallyDifferent(oldSPF, newSPF)
	}
}

// BenchmarkExpandSPFToIPSet benchmarks the IP set expansion performance
func BenchmarkExpandSPFToIPSet(b *testing.B) {
	spfRecord := "v=spf1 ip4:192.168.1.0/24 ip6:2001:db8::/64 ~all"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = expandSPFToIPSet(spfRecord)
	}
}
