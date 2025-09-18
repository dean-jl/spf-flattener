package spf

import (
	"context"
	"testing"
)

func TestCountDNSLookups(t *testing.T) {
	tests := []struct {
		name          string
		mockRecords   map[string][]string
		domain        string
		expectedCount int
		expectError   bool
	}{
		{
			name: "Simple SPF with two includes",
			mockRecords: map[string][]string{
				"example.com":           {"v=spf1 include:_spf.google.com include:_spf.firebasemail.com ~all"},
				"_spf.google.com":       {"v=spf1 ip4:74.125.0.0/16 ~all"},
				"_spf.firebasemail.com": {"v=spf1 ip4:198.21.0.0/21 ~all"},
			},
			domain:        "example.com",
			expectedCount: 3, // example.com + _spf.google.com + _spf.firebasemail.com
			expectError:   false,
		},
		{
			name: "Complex SPF with nested includes and duplicates",
			mockRecords: map[string][]string{
				"example.com":            {"v=spf1 include:_spf.google.com include:_spf.firebasemail.com ~all"},
				"_spf.google.com":        {"v=spf1 include:_netblocks.google.com include:_netblocks2.google.com ~all"},
				"_netblocks.google.com":  {"v=spf1 ip4:74.125.0.0/16 ~all"},
				"_netblocks2.google.com": {"v=spf1 ip4:173.194.0.0/16 ~all"},
				"_spf.firebasemail.com":  {"v=spf1 include:sendgrid.net include:_spf.google.com ~all"}, // duplicate _spf.google.com
				"sendgrid.net":           {"v=spf1 ip4:167.89.0.0/17 ~all"},
			},
			domain:        "example.com",
			expectedCount: 9, // example.com + _spf.google.com + _netblocks.google.com + _netblocks2.google.com + _spf.firebasemail.com + sendgrid.net + _spf.google.com (duplicate) + _netblocks.google.com (from duplicate) + _netblocks2.google.com (from duplicate)
			expectError:   false,
		},
		{
			name: "SPF with A and MX mechanisms",
			mockRecords: map[string][]string{
				"example.com":     {"v=spf1 a mx include:_spf.google.com ~all"},
				"_spf.google.com": {"v=spf1 ip4:74.125.0.0/16 ~all"},
			},
			domain:        "example.com",
			expectedCount: 4, // example.com + a:example.com + mx:example.com + _spf.google.com
			expectError:   false,
		},
		{
			name: "SPF with only IP addresses (no lookups needed)",
			mockRecords: map[string][]string{
				"example.com": {"v=spf1 ip4:192.168.1.1 ip6:2001:db8::1 ~all"},
			},
			domain:        "example.com",
			expectedCount: 1, // Only the initial TXT lookup
			expectError:   false,
		},
		{
			name: "Domain with no SPF record",
			mockRecords: map[string][]string{
				"example.com": {"v=dkim1; k=rsa; p=..."}, // No SPF record
			},
			domain:        "example.com",
			expectedCount: 1, // Only the initial TXT lookup
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockDNSProvider{
				Records: tt.mockRecords,
			}

			count, err := CountDNSLookups(context.Background(), tt.domain, provider)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if count != tt.expectedCount {
				t.Errorf("Expected %d DNS lookups, got %d", tt.expectedCount, count)
			}
		})
	}
}

func TestFlattenSPFWithThreshold(t *testing.T) {
	tests := []struct {
		name            string
		mockRecords     map[string][]string
		domain          string
		forceFlatten    bool
		expectFlattened bool
		expectedLookups int
	}{
		{
			name: "Low lookup count - should not flatten",
			mockRecords: map[string][]string{
				"example.com":     {"v=spf1 include:_spf.google.com ~all"},
				"_spf.google.com": {"v=spf1 ip4:74.125.0.0/16 ~all"},
			},
			domain:          "example.com",
			forceFlatten:    false,
			expectFlattened: false,
			expectedLookups: 2,
		},
		{
			name: "Low lookup count - force flatten",
			mockRecords: map[string][]string{
				"example.com":     {"v=spf1 include:_spf.google.com ~all"},
				"_spf.google.com": {"v=spf1 ip4:74.125.0.0/16 ~all"},
			},
			domain:          "example.com",
			forceFlatten:    true,
			expectFlattened: true,
			expectedLookups: 2,
		},
		{
			name: "High lookup count - should flatten",
			mockRecords: map[string][]string{
				"example.com": {"v=spf1 include:spf1.com include:spf2.com include:spf3.com include:spf4.com include:spf5.com include:spf6.com include:spf7.com include:spf8.com include:spf9.com include:spf10.com ~all"},
				"spf1.com":    {"v=spf1 ip4:1.1.1.1 ~all"},
				"spf2.com":    {"v=spf1 ip4:2.2.2.2 ~all"},
				"spf3.com":    {"v=spf1 ip4:3.3.3.3 ~all"},
				"spf4.com":    {"v=spf1 ip4:4.4.4.4 ~all"},
				"spf5.com":    {"v=spf1 ip4:5.5.5.5 ~all"},
				"spf6.com":    {"v=spf1 ip4:6.6.6.6 ~all"},
				"spf7.com":    {"v=spf1 ip4:7.7.7.7 ~all"},
				"spf8.com":    {"v=spf1 ip4:8.8.8.8 ~all"},
				"spf9.com":    {"v=spf1 ip4:9.9.9.9 ~all"},
				"spf10.com":   {"v=spf1 ip4:10.10.10.10 ~all"},
			},
			domain:          "example.com",
			forceFlatten:    false,
			expectFlattened: true,
			expectedLookups: 11, // 1 + 10 includes = 11 total lookups
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockDNSProvider{
				Records: tt.mockRecords,
			}

			original, flattened, lookupCount, wasFlattened, err := FlattenSPFWithThreshold(context.Background(), tt.domain, provider, false, tt.forceFlatten)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if lookupCount != tt.expectedLookups {
				t.Errorf("Expected %d DNS lookups, got %d", tt.expectedLookups, lookupCount)
			}

			if wasFlattened != tt.expectFlattened {
				t.Errorf("Expected flattened=%v, got %v", tt.expectFlattened, wasFlattened)
			}

			if original == "" {
				t.Error("Original SPF should not be empty")
			}

			if tt.expectFlattened && original == flattened {
				t.Error("When flattened, the result should be different from original")
			}

			if !tt.expectFlattened && original != flattened {
				t.Error("When not flattened, the result should be the same as original")
			}
		})
	}
}
