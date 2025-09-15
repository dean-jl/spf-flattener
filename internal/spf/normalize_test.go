package spf

import (
	"testing"
)

func TestNormalizeSPF(t *testing.T) {
	testCases := []struct {
		name      string
		spfRecord string
		expected  string
		hasError  bool
	}{
		{
			name:      "Simple record, no change",
			spfRecord: "v=spf1 ip4:192.0.2.1 ~all",
			expected:  "v=spf1 ip4:192.0.2.1 ~all",
			hasError:  false,
		},
		{
			name:      "Order change, should normalize",
			spfRecord: "v=spf1 ip4:192.0.2.1 include:_spf.google.com ~all",
			expected:  "v=spf1 include:_spf.google.com ip4:192.0.2.1 ~all",
			hasError:  false,
		},
		{
			name:      "Multiple mechanisms, different order",
			spfRecord: "v=spf1 mx a ip4:1.2.3.4 include:example.com ~all",
			expected:  "v=spf1 a include:example.com ip4:1.2.3.4 mx ~all",
			hasError:  false,
		},
		{
			name:      "Invalid SPF record",
			spfRecord: "ip4:192.0.2.1 ~all",
			expected:  "",
			hasError:  true,
		},
		{
			name:      "Empty SPF record",
			spfRecord: "",
			expected:  "",
			hasError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized, err := NormalizeSPF(tc.spfRecord)
			if (err != nil) != tc.hasError {
				t.Fatalf("Expected error: %v, got: %v", tc.hasError, err)
			}
			if normalized != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, normalized)
			}
		})
	}
}
