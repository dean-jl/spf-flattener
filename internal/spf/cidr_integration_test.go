package spf

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TestFlattenSPF_WithAggregation tests the integration of CIDR aggregation with SPF flattening
func TestFlattenSPF_WithAggregation(t *testing.T) {
	tests := []struct {
		name               string
		mockRecords        map[string][]string
		mockIPs            map[string][]net.IP
		domain             string
		aggregate          bool
		expectedMechanisms []string // Mechanisms that should be present in the result
		shouldAggregate    bool     // Whether aggregation should occur
	}{
		{
			name: "IPv4 aggregation with contiguous IPs",
			mockRecords: map[string][]string{
				"example.com": {"v=spf1 a:mail1.example.com a:mail2.example.com a:mail3.example.com a:mail4.example.com ~all"},
			},
			mockIPs: map[string][]net.IP{
				"mail1.example.com": {net.ParseIP("192.168.1.0")},
				"mail2.example.com": {net.ParseIP("192.168.1.1")},
				"mail3.example.com": {net.ParseIP("192.168.1.2")},
				"mail4.example.com": {net.ParseIP("192.168.1.3")},
			},
			domain:             "example.com",
			aggregate:          true,
			expectedMechanisms: []string{"ip4:192.168.1.0/30"},
			shouldAggregate:    true,
		},
		{
			name: "IPv4 no aggregation when disabled",
			mockRecords: map[string][]string{
				"example.com": {"v=spf1 a:mail1.example.com a:mail2.example.com a:mail3.example.com a:mail4.example.com ~all"},
			},
			mockIPs: map[string][]net.IP{
				"mail1.example.com": {net.ParseIP("192.168.1.0")},
				"mail2.example.com": {net.ParseIP("192.168.1.1")},
				"mail3.example.com": {net.ParseIP("192.168.1.2")},
				"mail4.example.com": {net.ParseIP("192.168.1.3")},
			},
			domain:             "example.com",
			aggregate:          false,
			expectedMechanisms: []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.2", "ip4:192.168.1.3"},
			shouldAggregate:    false,
		},
		{
			name: "IPv6 aggregation with contiguous IPs",
			mockRecords: map[string][]string{
				"example.com": {"v=spf1 a:mail1.example.com a:mail2.example.com ~all"},
			},
			mockIPs: map[string][]net.IP{
				"mail1.example.com": {net.ParseIP("2001:db8::0")},
				"mail2.example.com": {net.ParseIP("2001:db8::1")},
			},
			domain:             "example.com",
			aggregate:          true,
			expectedMechanisms: []string{"ip6:2001:db8::/127"},
			shouldAggregate:    true,
		},
		{
			name: "Mixed aggregation with non-contiguous IPs",
			mockRecords: map[string][]string{
				"example.com": {"v=spf1 a:mail1.example.com a:mail2.example.com a:mail3.example.com include:google.com ~all"},
				"google.com":  {"v=spf1 ip4:10.0.0.1 ~all"},
			},
			mockIPs: map[string][]net.IP{
				"mail1.example.com": {net.ParseIP("192.168.1.1")}, // Non-contiguous
				"mail2.example.com": {net.ParseIP("192.168.1.3")}, // Gap at .2
				"mail3.example.com": {net.ParseIP("192.168.2.1")}, // Different subnet
			},
			domain:             "example.com",
			aggregate:          true,
			expectedMechanisms: []string{"ip4:192.168.1.1", "ip4:192.168.1.3", "ip4:192.168.2.1", "ip4:10.0.0.1"},
			shouldAggregate:    false, // No aggregation should occur due to gaps
		},
		{
			name: "Mixed IPv4 and IPv6 with partial aggregation",
			mockRecords: map[string][]string{
				"example.com": {"v=spf1 a:mail1.example.com a:mail2.example.com a:mail3.example.com a:mail4.example.com ~all"},
			},
			mockIPs: map[string][]net.IP{
				"mail1.example.com": {net.ParseIP("192.168.1.0")}, // Contiguous IPv4
				"mail2.example.com": {net.ParseIP("192.168.1.1")}, // Contiguous IPv4
				"mail3.example.com": {net.ParseIP("2001:db8::0")}, // Contiguous IPv6
				"mail4.example.com": {net.ParseIP("2001:db8::1")}, // Contiguous IPv6
			},
			domain:             "example.com",
			aggregate:          true,
			expectedMechanisms: []string{"ip4:192.168.1.0/31", "ip6:2001:db8::/127"},
			shouldAggregate:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock DNS provider
			provider := &mockDNSProvider{
				Records: tt.mockRecords,
				IPs:     tt.mockIPs,
			}

			// Call FlattenSPF with aggregation flag
			_, flattened, err := FlattenSPF(context.Background(), tt.domain, provider, tt.aggregate)
			if err != nil {
				t.Fatalf("FlattenSPF failed: %v", err)
			}

			// Extract mechanisms from the flattened SPF record
			// Remove "v=spf1" and "~all" to get just the IP mechanisms
			parts := strings.Fields(flattened)
			var mechanisms []string
			for _, part := range parts {
				if part != "v=spf1" && part != "~all" {
					mechanisms = append(mechanisms, part)
				}
			}

			// Check that all expected mechanisms are present
			for _, expected := range tt.expectedMechanisms {
				found := false
				for _, mechanism := range mechanisms {
					if mechanism == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected mechanism %s not found in flattened SPF. Got: %v", expected, mechanisms)
				}
			}

			// Check that we don't have unexpected mechanisms (if aggregation should occur)
			if tt.shouldAggregate && len(mechanisms) != len(tt.expectedMechanisms) {
				t.Errorf("Expected %d mechanisms after aggregation, got %d. Mechanisms: %v",
					len(tt.expectedMechanisms), len(mechanisms), mechanisms)
			}

			t.Logf("Flattened SPF: %s", flattened)
			t.Logf("Mechanisms: %v", mechanisms)
		})
	}
}

// TestFlattenSPF_AggregationWithIPMechanisms tests aggregation with IP-resolving mechanisms
func TestFlattenSPF_AggregationWithIPMechanisms(t *testing.T) {
	provider := &mockDNSProvider{
		Records: map[string][]string{
			"example.com": {"v=spf1 a:mail.example.com ip4:192.168.1.3 ~all"},
		},
		IPs: map[string][]net.IP{
			"mail.example.com": {net.ParseIP("192.168.1.1"), net.ParseIP("192.168.1.2")},
		},
	}

	_, flattened, err := FlattenSPF(context.Background(), "example.com", provider, true)
	if err != nil {
		t.Fatalf("FlattenSPF failed: %v", err)
	}

	// The A record resolves to 192.168.1.1 and 192.168.1.2, and there's also ip4:192.168.1.3
	// Let's check if 1,2,3 can be aggregated:
	// 192.168.1.1 = ...00000001
	// 192.168.1.2 = ...00000010
	// 192.168.1.3 = ...00000011
	// 2 and 3 are contiguous and should form ip4:192.168.1.2/31
	// 1 stands alone as ip4:192.168.1.1
	expectedMechanisms := []string{"ip4:192.168.1.1", "ip4:192.168.1.2/31"}

	parts := strings.Fields(flattened)
	var mechanisms []string
	for _, part := range parts {
		if part != "v=spf1" && part != "~all" {
			mechanisms = append(mechanisms, part)
		}
	}

	// Check that all expected mechanisms are present
	for _, expected := range expectedMechanisms {
		found := false
		for _, mechanism := range mechanisms {
			if mechanism == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected mechanism %s not found in flattened SPF. Got: %v", expected, mechanisms)
		}
	}

	t.Logf("Flattened SPF: %s", flattened)
}

// TestFlattenSPF_AggregationPerformance_BestCase tests artificial best-case aggregation scenario
// NOTE: This represents the theoretical maximum capability (up to 97% reduction)
// but does not reflect typical real-world SPF record performance
func TestFlattenSPF_AggregationPerformance_BestCase(t *testing.T) {
	// Create a large set of contiguous IPs that should aggregate well
	// This is an ARTIFICIAL scenario - real SPF records rarely have this pattern
	mockIPs := make(map[string][]net.IP)
	var domains []string

	// Create 100 contiguous IPs (should aggregate to 192.168.0.0/25 and 192.168.0.128/26)
	for i := 0; i < 100; i++ {
		domain := fmt.Sprintf("mail%d.example.com", i)
		domains = append(domains, domain)
		mockIPs[domain] = []net.IP{net.ParseIP(fmt.Sprintf("192.168.0.%d", i))}
	}

	// Build SPF record with all A record lookups
	var spfParts []string
	spfParts = append(spfParts, "v=spf1")
	for _, domain := range domains {
		spfParts = append(spfParts, "a:"+domain)
	}
	spfParts = append(spfParts, "~all")

	provider := &mockDNSProvider{
		Records: map[string][]string{
			"example.com": {strings.Join(spfParts, " ")},
		},
		IPs: mockIPs,
	}

	// Test without aggregation
	start := time.Now()
	_, flattenedNormal, err := FlattenSPF(context.Background(), "example.com", provider, false)
	normalDuration := time.Since(start)
	if err != nil {
		t.Fatalf("FlattenSPF without aggregation failed: %v", err)
	}

	// Test with aggregation
	start = time.Now()
	_, flattenedAggregated, err := FlattenSPF(context.Background(), "example.com", provider, true)
	aggregatedDuration := time.Since(start)
	if err != nil {
		t.Fatalf("FlattenSPF with aggregation failed: %v", err)
	}

	// Check that aggregation significantly reduces the number of mechanisms
	normalMechanisms := len(strings.Fields(flattenedNormal)) - 2 // Remove "v=spf1" and "~all"
	aggregatedMechanisms := len(strings.Fields(flattenedAggregated)) - 2

	if aggregatedMechanisms >= normalMechanisms {
		t.Errorf("Aggregation should reduce mechanisms. Normal: %d, Aggregated: %d",
			normalMechanisms, aggregatedMechanisms)
	}

	// Performance should not be significantly worse (allow 5x slowdown)
	if aggregatedDuration > normalDuration*5 {
		t.Errorf("Aggregation is too slow. Normal: %v, Aggregated: %v",
			normalDuration, aggregatedDuration)
	}

	t.Logf("BEST-CASE ARTIFICIAL SCENARIO:")
	t.Logf("Normal: %d mechanisms in %v", normalMechanisms, normalDuration)
	t.Logf("Aggregated: %d mechanisms in %v", aggregatedMechanisms, aggregatedDuration)
	t.Logf("Reduction: %.1f%% (THIS IS NOT TYPICAL FOR REAL SPF RECORDS)", float64(normalMechanisms-aggregatedMechanisms)/float64(normalMechanisms)*100)
}

// TestCIDRAggregation_RealWorldScenarios tests CIDR aggregation directly on mechanisms
// (bypassing complex SPF flattening to focus on aggregation behavior)
func TestCIDRAggregation_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name              string
		inputMechanisms   []string
		expectedReduction float64
		description       string
	}{
		{
			name: "Already Optimized Individual Provider IPs",
			inputMechanisms: []string{
				"ip4:74.125.224.26", // Google IP (would normally be in optimized block)
				"ip4:173.194.79.27", // Google IP (would normally be in optimized block)
				"ip4:203.0.113.100", // Single corporate IP
			},
			expectedReduction: 0.0, // No contiguous ranges - typical enterprise
			description:       "Individual IPs from providers (no contiguous ranges)",
		},
		{
			name: "Well-managed Enterprise with Regional Servers",
			inputMechanisms: []string{
				"ip4:203.0.113.10", // US East
				"ip4:198.51.100.5", // US West
				"ip4:192.0.2.20",   // Europe
				"ip4:172.16.1.30",  // Asia Pacific
			},
			expectedReduction: 0.0, // Scattered IPs across regions
			description:       "Regional mail servers - no contiguous ranges",
		},
		{
			name: "Legacy Setup with Manual IP Lists",
			inputMechanisms: []string{
				"ip4:192.168.1.10", // First contiguous block
				"ip4:192.168.1.11",
				"ip4:192.168.1.12",
				"ip4:192.168.1.13",
				"ip4:192.168.1.20", // Second contiguous block
				"ip4:192.168.1.21",
			},
			expectedReduction: 50.0, // 6 IPs → 3 CIDR blocks
			description:       "Unoptimized configuration with some contiguous ranges",
		},
		{
			name: "Mixed Cloud + Scattered On-Premises",
			inputMechanisms: []string{
				"ip4:209.61.151.0/24", // Mailgun - already optimized
				"ip4:198.61.254.0/24", // Mailgun - already optimized
				"ip4:192.168.1.100",   // On-prem server 1
				"ip4:192.168.1.102",   // On-prem server 2 (gap at .101)
				"ip4:203.0.113.50",    // Corporate gateway
			},
			expectedReduction: 0.0, // Provider blocks optimal, on-prem scattered
			description:       "Cloud provider + non-contiguous on-premises",
		},
		{
			name: "Corporate Network with Sequential Allocation",
			inputMechanisms: []string{
				"ip4:10.0.1.10", // Corporate block start
				"ip4:10.0.1.11",
				"ip4:10.0.1.12",
				"ip4:10.0.1.13", // Corporate block end
				"ip4:10.0.1.16", // Isolated server (gap at 14,15)
			},
			expectedReduction: 40.0, // 5 IPs → 3 entries (4 contiguous + 1 individual)
			description:       "Corporate network with mostly sequential IPs",
		},
		{
			name: "Small Business with Individual IPs",
			inputMechanisms: []string{
				"ip4:192.168.1.5", // Scattered individual IPs
				"ip4:10.0.0.100",
				"ip4:172.16.5.25",
			},
			expectedReduction: 0.0, // No contiguous ranges
			description:       "Small business with scattered mail servers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test aggregation directly on mechanisms
			result := AggregateCIDRs(tt.inputMechanisms)

			originalCount := len(tt.inputMechanisms)
			aggregatedCount := len(result)

			var actualReduction float64
			if originalCount > 0 {
				actualReduction = float64(originalCount-aggregatedCount) / float64(originalCount) * 100
			}

			// Allow some tolerance
			tolerance := 5.0
			if actualReduction < tt.expectedReduction-tolerance || actualReduction > tt.expectedReduction+tolerance {
				t.Errorf("Reduction outside expected range. Got %.1f%%, expected %.1f%% ±%.1f%%",
					actualReduction, tt.expectedReduction, tolerance)
			}

			t.Logf("REALISTIC SCENARIO: %s", tt.description)
			t.Logf("Original mechanisms: %d", originalCount)
			t.Logf("Aggregated mechanisms: %d", aggregatedCount)
			t.Logf("Actual reduction: %.1f%% (expected ~%.1f%%)", actualReduction, tt.expectedReduction)
			t.Logf("Input:  %v", tt.inputMechanisms)
			t.Logf("Output: %v", result)
		})
	}
}
