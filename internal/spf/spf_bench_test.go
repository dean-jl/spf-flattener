package spf

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
)

// Benchmark for SPF normalization
func BenchmarkNormalizeSPF(b *testing.B) {
	testCases := []string{
		"v=spf1 ip4:192.0.2.1 ~all",
		"v=spf1 include:_spf.google.com include:_spf.salesforce.com ip4:1.2.3.4 mx ~all",
		"v=spf1 a mx include:spf.example.com ip4:203.0.113.0/24 ip6:2001:db8::/32 ~all",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_, err := NormalizeSPF(tc)
			if err != nil {
				b.Fatalf("NormalizeSPF failed: %v", err)
			}
		}
	}
}

// Benchmark for SPF flattening with simple record
func BenchmarkFlattenSPF_Simple(b *testing.B) {
	provider := &mockDNSProvider{
		Records: map[string][]string{
			"example.com": {"v=spf1 ip4:192.0.2.1 ip4:203.0.113.1 ~all"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := FlattenSPF(context.Background(), "example.com", provider, false)
		if err != nil {
			b.Fatalf("FlattenSPF failed: %v", err)
		}
	}
}

// Benchmark for SPF flattening with includes
func BenchmarkFlattenSPF_WithIncludes(b *testing.B) {
	provider := &mockDNSProvider{
		Records: map[string][]string{
			"example.com":    {"v=spf1 include:google.com include:salesforce.com ~all"},
			"google.com":     {"v=spf1 ip4:74.125.0.0/16 ip6:2001:4860::/32 ~all"},
			"salesforce.com": {"v=spf1 ip4:136.146.0.0/16 ip4:198.2.177.0/24 ~all"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := FlattenSPF(context.Background(), "example.com", provider, false)
		if err != nil {
			b.Fatalf("FlattenSPF failed: %v", err)
		}
	}
}

// Benchmark for SPF flattening with A record lookups
func BenchmarkFlattenSPF_WithARecords(b *testing.B) {
	provider := &mockDNSProvider{
		Records: map[string][]string{
			"example.com": {"v=spf1 a:mail.example.com a:smtp.example.com ~all"},
		},
		IPs: map[string][]net.IP{
			"mail.example.com": {net.ParseIP("192.0.2.10")},
			"smtp.example.com": {net.ParseIP("192.0.2.20"), net.ParseIP("2001:db8::1")},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := FlattenSPF(context.Background(), "example.com", provider, false)
		if err != nil {
			b.Fatalf("FlattenSPF failed: %v", err)
		}
	}
}

// Benchmark for deep recursion (worst case scenario)
func BenchmarkFlattenSPF_DeepRecursion(b *testing.B) {
	records := make(map[string][]string)

	// Create a chain of includes 5 levels deep
	records["example.com"] = []string{"v=spf1 include:level1.com ~all"}
	records["level1.com"] = []string{"v=spf1 include:level2.com ip4:1.1.1.1 ~all"}
	records["level2.com"] = []string{"v=spf1 include:level3.com ip4:2.2.2.2 ~all"}
	records["level3.com"] = []string{"v=spf1 include:level4.com ip4:3.3.3.3 ~all"}
	records["level4.com"] = []string{"v=spf1 ip4:4.4.4.4 ip4:5.5.5.5 ~all"}

	provider := &mockDNSProvider{Records: records}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := FlattenSPF(context.Background(), "example.com", provider, false)
		if err != nil {
			b.Fatalf("FlattenSPF failed: %v", err)
		}
	}
}

// Benchmark for SPF splitting
func BenchmarkSplitAndChainSPF(b *testing.B) {
	// Create a very long SPF record that will require splitting
	var ips []string
	for i := 1; i <= 50; i++ {
		ips = append(ips, fmt.Sprintf("ip4:192.0.2.%d", i))
	}
	longSPF := "v=spf1 " + strings.Join(ips, " ") + " ~all"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := SplitAndChainSPF(longSPF, "example.com")
		if len(result) == 0 {
			b.Fatalf("SplitAndChainSPF returned empty result")
		}
	}
}

// Benchmark for ExtractMechanisms
func BenchmarkExtractMechanisms(b *testing.B) {
	spfRecord := "v=spf1 include:_spf.google.com a mx ip4:203.0.113.0/24 ip6:2001:db8::/32 ~all"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ExtractMechanisms(spfRecord)
		if err != nil {
			b.Fatalf("ExtractMechanisms failed: %v", err)
		}
	}
}

// Benchmark for ExtractMechanismSet
func BenchmarkExtractMechanismSet(b *testing.B) {
	spfRecord := "v=spf1 include:_spf.google.com a mx ip4:203.0.113.0/24 ip6:2001:db8::/32 ~all"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := ExtractMechanismSet(spfRecord)
		if len(result) == 0 {
			b.Fatalf("ExtractMechanismSet returned empty result")
		}
	}
}

// Benchmark for isSPFMechanism helper function
func BenchmarkIsSPFMechanism(b *testing.B) {
	testParts := []string{
		"include:_spf.google.com",
		"a:mail.example.com",
		"mx",
		"ip4:192.0.2.1",
		"ip6:2001:db8::1",
		"ptr:example.com",
		"exists:example.com",
		"redirect=example.com",
		"exp=explain.example.com",
		"unknown:mechanism",
		"~all",
		"v=spf1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, part := range testParts {
			_ = isSPFMechanism(part)
		}
	}
}

// Benchmark comparison: Old vs New string building approach
func BenchmarkStringBuilding_Sprintf(b *testing.B) {
	domain := "example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("\n===== Error processing domain: %s \n\n", domain)
	}
}

func BenchmarkStringBuilding_Direct(b *testing.B) {
	domain := "example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.WriteString("\n===== Error processing domain: ")
		sb.WriteString(domain)
		sb.WriteString(" \n\n")
		_ = sb.String()
	}
}
