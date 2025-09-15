package spf

import (
	"strings"
	"testing"
)

func TestSplitSPF(t *testing.T) {
	testCases := []struct {
		name               string
		spfRecord          string
		expectedNumRecords int
	}{
		{
			name:               "No split needed",
			spfRecord:          "v=spf1 ip4:192.0.2.1 ~all",
			expectedNumRecords: 1,
		},
		{
			name:               "Single split needed",
			spfRecord:          "v=spf1 " + strings.Repeat("ip4:192.0.2.1 ", 20) + "~all", // This will be > 255 chars
			expectedNumRecords: 2,
		},
		{
			name:               "Multiple splits needed",
			spfRecord:          "v=spf1 " + strings.Repeat("ip4:192.0.2.1 ", 40) + "~all", // This will be > 255 * 2 chars
			expectedNumRecords: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SplitSPF(tc.spfRecord)

			if len(result) != tc.expectedNumRecords {
				t.Fatalf("Expected %d records, got %d", tc.expectedNumRecords, len(result))
			}
			for i, rec := range result {
				if !strings.HasPrefix(rec, "v=spf1") || !strings.HasSuffix(rec, "~all") {
					t.Errorf("Record %d does not start with v=spf1 or end with ~all: %s", i, rec)
				}
				if len(rec) > 255 {
					t.Errorf("Record %d is too long: %d chars", i, len(rec))
				}
			}
		})
	}
}

func TestSplitAndChainSPF_Lengths(t *testing.T) {
	spfRecord := "v=spf1 " + strings.Repeat("ip4:192.0.2.1 ", 50) + "~all" // Large record
	domain := "example.com"
	result := SplitAndChainSPF(spfRecord, domain)

	for name, rec := range result {
		if len(rec) > 255 {
			t.Errorf("Record %s exceeds 255 chars: %d", name, len(rec))
		}
		if !strings.HasPrefix(rec, "v=spf1") {
			t.Errorf("Record %s does not start with v=spf1: %s", name, rec)
		}
		if !strings.HasSuffix(rec, "~all") {
			t.Errorf("Record %s does not end with ~all: %s", name, rec)
		}
	}
}
