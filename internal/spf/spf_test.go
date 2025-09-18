package spf

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
)

// mockDNSProvider is a mock for the DNSProvider interface for testing.
type mockDNSProvider struct {
	Records map[string][]string
	IPs     map[string][]net.IP
	MXs     map[string][]*net.MX
}

func (m *mockDNSProvider) LookupTXT(ctx context.Context, domain string) ([]string, error) {
	if recs, ok := m.Records[domain]; ok {
		return recs, nil
	}
	return nil, fmt.Errorf("no TXT records found for %s", domain)
}

func (m *mockDNSProvider) LookupIP(ctx context.Context, domain string) ([]net.IP, error) {
	if ips, ok := m.IPs[domain]; ok {
		return ips, nil
	}
	return nil, fmt.Errorf("no A/AAAA records found for %s", domain)
}

func (m *mockDNSProvider) LookupMX(ctx context.Context, domain string) ([]*net.MX, error) {
	if mxs, ok := m.MXs[domain]; ok {
		return mxs, nil
	}
	return nil, fmt.Errorf("no MX records found for %s", domain)
}

func (m *mockDNSProvider) Close() error {
	return nil
}

func TestFlattenSPF(t *testing.T) {
	testCases := []struct {
		name     string
		domain   string
		provider *mockDNSProvider
		expected string
		hasError bool
	}{
		{
			name:   "Simple include",
			domain: "example.com",
			provider: &mockDNSProvider{
				Records: map[string][]string{
					"example.com":     {"v=spf1 include:_spf.google.com ~all"},
					"_spf.google.com": {"v=spf1 ip4:8.8.8.8 ~all"},
				},
			},
			expected: "v=spf1 ip4:8.8.8.8 ~all",
			hasError: false,
		},
		{
			name:   "A mechanism",
			domain: "example.com",
			provider: &mockDNSProvider{
				Records: map[string][]string{"example.com": {"v=spf1 a ~all"}},
				IPs:     map[string][]net.IP{"example.com": {net.ParseIP("1.2.3.4")}},
			},
			expected: "v=spf1 ip4:1.2.3.4 ~all",
			hasError: false,
		},
		{
			name:   "MX mechanism",
			domain: "example.com",
			provider: &mockDNSProvider{
				Records: map[string][]string{"example.com": {"v=spf1 mx ~all"}},
				MXs:     map[string][]*net.MX{"example.com": {{Host: "mx.google.com", Pref: 10}}},
				IPs:     map[string][]net.IP{"mx.google.com": {net.ParseIP("8.8.8.8")}},
			},
			expected: "v=spf1 ip4:8.8.8.8 ~all",
			hasError: false,
		},
		{
			name:   "PTR mechanism (is ignored)",
			domain: "example.com",
			provider: &mockDNSProvider{
				Records: map[string][]string{"example.com": {"v=spf1 ptr:other.com ip4:1.2.3.4 ~all"}},
			},
			expected: "v=spf1 ip4:1.2.3.4 ~all",
			hasError: false,
		},
		{
			name:   "No SPF record",
			domain: "example.com",
			provider: &mockDNSProvider{
				Records: map[string][]string{"example.com": {"some other record"}},
			},
			expected: "",
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, flattened, err := FlattenSPF(context.Background(), tc.domain, tc.provider, false)
			if (err != nil) != tc.hasError {
				t.Fatalf("Expected error: %v, got: %v", tc.hasError, err)
			}
			if !tc.hasError && flattened != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, flattened)
			}
		})
	}
}

// Additional comprehensive error path tests

func TestFlattenSPF_RecursionErrors(t *testing.T) {
	// Test infinite recursion detection
	provider := &mockDNSProvider{
		Records: map[string][]string{
			"example.com":   {"v=spf1 include:recursive.com ~all"},
			"recursive.com": {"v=spf1 include:example.com ~all"}, // Circular reference
		},
	}

	_, _, err := FlattenSPF(context.Background(), "example.com", provider, false)
	if err == nil {
		t.Errorf("Expected recursion error, got nil")
	}
}

func TestFlattenSPF_MaxDepthExceeded(t *testing.T) {
	// Test maximum recursion depth (10 levels)
	provider := &mockDNSProvider{
		Records: map[string][]string{
			"level0.com":  {"v=spf1 include:level1.com ~all"},
			"level1.com":  {"v=spf1 include:level2.com ~all"},
			"level2.com":  {"v=spf1 include:level3.com ~all"},
			"level3.com":  {"v=spf1 include:level4.com ~all"},
			"level4.com":  {"v=spf1 include:level5.com ~all"},
			"level5.com":  {"v=spf1 include:level6.com ~all"},
			"level6.com":  {"v=spf1 include:level7.com ~all"},
			"level7.com":  {"v=spf1 include:level8.com ~all"},
			"level8.com":  {"v=spf1 include:level9.com ~all"},
			"level9.com":  {"v=spf1 include:level10.com ~all"},
			"level10.com": {"v=spf1 include:level11.com ~all"},
			"level11.com": {"v=spf1 ip4:1.2.3.4 ~all"}, // This should exceed max depth
		},
	}

	_, _, err := FlattenSPF(context.Background(), "level0.com", provider, false)
	if err == nil {
		t.Errorf("Expected depth exceeded error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "recursion depth exceeded") {
		t.Errorf("Expected 'recursion depth exceeded' error, got: %v", err)
	}
}

func TestNormalizeSPF_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "Very long SPF record",
			input:    "v=spf1 " + strings.Repeat("ip4:1.2.3.4 ", 50) + "~all",
			expected: "v=spf1 " + strings.Repeat("ip4:1.2.3.4 ", 50) + "~all",
			hasError: false,
		},
		{
			name:     "SPF with unknown mechanisms",
			input:    "v=spf1 unknown:test ip4:1.2.3.4 future-mech:value ~all",
			expected: "v=spf1 unknown:test future-mech:value ip4:1.2.3.4 ~all",
			hasError: false,
		},
		{
			name:     "SPF with malformed mechanisms",
			input:    "v=spf1 ip4 mx: include ~all",
			expected: "v=spf1 ip4 include mx: ~all", // Sorted alphabetically
			hasError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NormalizeSPF(tc.input)
			if (err != nil) != tc.hasError {
				t.Errorf("Expected error: %v, got: %v", tc.hasError, err)
			}
			if !tc.hasError && result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// DNS validation tests
func TestValidateTXTRecords(t *testing.T) {
	// Valid records
	validRecords := []string{
		"v=spf1 ip4:192.168.1.1 ~all",
		"simple text record",
		"",
	}
	result, err := validateTXTRecords(validRecords, "example.com")
	if err != nil {
		t.Errorf("Expected no error for valid records, got %v", err)
	}
	if len(result) != len(validRecords) {
		t.Errorf("Expected %d records, got %d", len(validRecords), len(result))
	}

	// Invalid records (too long, non-printable)
	invalidRecords := []string{
		strings.Repeat("a", 5000), // Too long
		"valid record",
		string([]byte{0x01, 0x02, 0x03}), // Non-printable
	}
	result, err = validateTXTRecords(invalidRecords, "example.com")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 valid record, got %d", len(result))
	}
}

func TestValidateIPAddresses(t *testing.T) {
	// Valid IPs
	validIPs := []net.IP{
		net.ParseIP("192.168.1.1"),
		net.ParseIP("2001:db8::1"),
	}
	result, err := validateIPAddresses(validIPs, "example.com")
	if err != nil {
		t.Errorf("Expected no error for valid IPs, got %v", err)
	}
	if len(result) != len(validIPs) {
		t.Errorf("Expected %d IPs, got %d", len(validIPs), len(result))
	}

	// Invalid IPs (unspecified addresses)
	invalidIPs := []net.IP{
		net.ParseIP("192.168.1.1"),
		net.ParseIP("0.0.0.0"), // Unspecified
		net.ParseIP("::"),      // Unspecified IPv6
		nil,                    // Nil IP
	}
	result, err = validateIPAddresses(invalidIPs, "example.com")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 valid IP, got %d", len(result))
	}
}

func TestValidateMXRecords(t *testing.T) {
	// Valid MX records
	validMXs := []*net.MX{
		{Host: "mail.example.com", Pref: 10},
		{Host: "backup.example.com.", Pref: 20}, // With trailing dot
	}
	result, err := validateMXRecords(validMXs, "example.com")
	if err != nil {
		t.Errorf("Expected no error for valid MX records, got %v", err)
	}
	if len(result) != len(validMXs) {
		t.Errorf("Expected %d MX records, got %d", len(validMXs), len(result))
	}

	// Invalid MX records
	invalidMXs := []*net.MX{
		{Host: "valid.example.com", Pref: 10},
		{Host: "", Pref: 20},                       // Empty hostname
		{Host: strings.Repeat("a", 300), Pref: 30}, // Too long
		nil, // Nil record
	}
	result, err = validateMXRecords(invalidMXs, "example.com")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 valid MX record, got %d", len(result))
	}
}

func TestIsPrintableASCII(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"hello world", true},
		{"v=spf1 ip4:192.168.1.1 ~all", true},
		{"", true},
		{"!@#$%^&*()", true},
		{string([]byte{0x01}), false}, // Control character
		{string([]byte{0x7F}), false}, // DEL character
		{"héllo", false},              // Non-ASCII
	}

	for _, tc := range testCases {
		result := isPrintableASCII(tc.input)
		if result != tc.expected {
			t.Errorf("isPrintableASCII(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestIsValidMXHostname(t *testing.T) {
	validHostnames := []string{
		"mail.example.com",
		"backup.example.com.",
		"mx-1.example.org",
		"123.example.net",
	}

	invalidHostnames := []string{
		"",
		strings.Repeat("a", 300),
		"invalid_hostname.com", // Underscore
		"space hostname.com",   // Space
	}

	for _, hostname := range validHostnames {
		if !isValidMXHostname(hostname) {
			t.Errorf("Expected '%s' to be valid MX hostname", hostname)
		}
	}

	for _, hostname := range invalidHostnames {
		if isValidMXHostname(hostname) {
			t.Errorf("Expected '%s' to be invalid MX hostname", hostname)
		}
	}
}
