package spf

import (
	"fmt"
	"math/big"
	"net"
	"reflect"
	"sort"
	"testing"
)

func TestAggregateCIDRs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single IPv4",
			input:    []string{"ip4:192.168.1.1"},
			expected: []string{"ip4:192.168.1.1"},
		},
		{
			name:     "Two contiguous IPv4",
			input:    []string{"ip4:192.168.1.0", "ip4:192.168.1.1"},
			expected: []string{"ip4:192.168.1.0/31"},
		},
		{
			name:     "Four contiguous IPv4",
			input:    []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.2", "ip4:192.168.1.3"},
			expected: []string{"ip4:192.168.1.0/30"},
		},
		{
			name:     "IPv4 with gap",
			input:    []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip4:192.168.1.3"},
			expected: []string{"ip4:192.168.1.0/31", "ip4:192.168.1.3"},
		},
		{
			name:     "Preserve existing IPv4 CIDR blocks",
			input:    []string{"ip4:192.168.0.0/24", "ip4:192.168.1.0/24"},
			expected: []string{"ip4:192.168.0.0/24", "ip4:192.168.1.0/24"},
		},
		{
			name:     "Single IPv6",
			input:    []string{"ip6:2001:db8::1"},
			expected: []string{"ip6:2001:db8::1"},
		},
		{
			name:     "Two contiguous IPv6",
			input:    []string{"ip6:2001:db8::0", "ip6:2001:db8::1"},
			expected: []string{"ip6:2001:db8::/127"},
		},
		{
			name:     "Four contiguous IPv6",
			input:    []string{"ip6:2001:db8::0", "ip6:2001:db8::1", "ip6:2001:db8::2", "ip6:2001:db8::3"},
			expected: []string{"ip6:2001:db8::/126"},
		},
		{
			name:     "Mixed IPv4 and IPv6",
			input:    []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "ip6:2001:db8::0", "ip6:2001:db8::1"},
			expected: []string{"ip4:192.168.1.0/31", "ip6:2001:db8::/127"},
		},
		{
			name:     "Mixed with non-IP mechanisms",
			input:    []string{"ip4:192.168.1.0", "ip4:192.168.1.1", "include:example.com", "ip6:2001:db8::1"},
			expected: []string{"ip4:192.168.1.0/31", "ip6:2001:db8::1", "include:example.com"},
		},
		{
			name:     "Non-contiguous IPs",
			input:    []string{"ip4:192.168.1.1", "ip4:10.0.0.1", "ip6:2001:db8::1"},
			expected: []string{"ip4:10.0.0.1", "ip4:192.168.1.1", "ip6:2001:db8::1"},
		},
		{
			name:     "Duplicate IPs",
			input:    []string{"ip4:192.168.1.1", "ip4:192.168.1.1", "ip4:192.168.1.2"},
			expected: []string{"ip4:192.168.1.1", "ip4:192.168.1.2"},
		},
		{
			name:     "Complex IPv4 ranges",
			input:    []string{"ip4:192.168.0.0", "ip4:192.168.0.1", "ip4:192.168.0.2", "ip4:192.168.0.3", "ip4:192.168.0.4", "ip4:192.168.0.5", "ip4:192.168.0.6", "ip4:192.168.0.7"},
			expected: []string{"ip4:192.168.0.0/29"},
		},
		{
			name:     "Unaligned IPv4 range",
			input:    []string{"ip4:192.168.0.1", "ip4:192.168.0.2", "ip4:192.168.0.3"},
			expected: []string{"ip4:192.168.0.1", "ip4:192.168.0.2/31"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregateCIDRs(tt.input)

			// Sort both slices for comparison since order may vary
			sort.Strings(result)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("AggregateCIDRs(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractIPv4Addresses(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string // IP strings for easier comparison
	}{
		{
			name:     "Single IPv4",
			input:    []string{"ip4:192.168.1.1"},
			expected: []string{"192.168.1.1"},
		},
		{
			name:     "Multiple IPv4",
			input:    []string{"ip4:192.168.1.1", "ip4:10.0.0.1"},
			expected: []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:     "IPv4 CIDR /30",
			input:    []string{"ip4:192.168.1.0/30"},
			expected: []string{"192.168.1.0", "192.168.1.1", "192.168.1.2", "192.168.1.3"},
		},
		{
			name:     "Mixed with non-IPv4",
			input:    []string{"ip4:192.168.1.1", "ip6:2001:db8::1", "include:example.com"},
			expected: []string{"192.168.1.1"},
		},
		{
			name:     "Invalid IPv4",
			input:    []string{"ip4:invalid", "ip4:192.168.1.1"},
			expected: []string{"192.168.1.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIPv4Addresses(tt.input)

			// Convert result to strings for comparison
			resultStrings := make([]string, len(result))
			for i, ip := range result {
				resultStrings[i] = ip.String()
			}

			sort.Strings(resultStrings)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(resultStrings, tt.expected) {
				t.Errorf("extractIPv4Addresses(%v) = %v, expected %v", tt.input, resultStrings, tt.expected)
			}
		})
	}
}

func TestExtractIPv6Addresses(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string // IP strings for easier comparison
	}{
		{
			name:     "Single IPv6",
			input:    []string{"ip6:2001:db8::1"},
			expected: []string{"2001:db8::1"},
		},
		{
			name:     "Multiple IPv6",
			input:    []string{"ip6:2001:db8::1", "ip6:2001:db8::2"},
			expected: []string{"2001:db8::1", "2001:db8::2"},
		},
		{
			name:     "IPv6 CIDR /126",
			input:    []string{"ip6:2001:db8::/126"},
			expected: []string{"2001:db8::", "2001:db8::1", "2001:db8::2", "2001:db8::3"},
		},
		{
			name:     "Mixed with non-IPv6",
			input:    []string{"ip6:2001:db8::1", "ip4:192.168.1.1", "include:example.com"},
			expected: []string{"2001:db8::1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIPv6Addresses(tt.input)

			// Convert result to strings for comparison
			resultStrings := make([]string, len(result))
			for i, ip := range result {
				resultStrings[i] = ip.String()
			}

			sort.Strings(resultStrings)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(resultStrings, tt.expected) {
				t.Errorf("extractIPv6Addresses(%v) = %v, expected %v", tt.input, resultStrings, tt.expected)
			}
		})
	}
}

func TestExtractNonIPMechanisms(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Only non-IP mechanisms",
			input:    []string{"include:example.com", "a:example.com", "mx:example.com"},
			expected: []string{"include:example.com", "a:example.com", "mx:example.com"},
		},
		{
			name:     "Mixed mechanisms",
			input:    []string{"ip4:192.168.1.1", "include:example.com", "ip6:2001:db8::1", "a:example.com"},
			expected: []string{"include:example.com", "a:example.com"},
		},
		{
			name:     "Only IP mechanisms",
			input:    []string{"ip4:192.168.1.1", "ip6:2001:db8::1"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNonIPMechanisms(tt.input)

			sort.Strings(result)
			sort.Strings(tt.expected)

			// Handle nil vs empty slice comparison
			if len(result) == 0 && len(tt.expected) == 0 {
				// Both are empty, consider them equal
			} else if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractNonIPMechanisms(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIPToUint32(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{
			name:     "192.168.1.1",
			ip:       "192.168.1.1",
			expected: 0xC0A80101, // 192*256^3 + 168*256^2 + 1*256 + 1
		},
		{
			name:     "10.0.0.1",
			ip:       "10.0.0.1",
			expected: 0x0A000001,
		},
		{
			name:     "0.0.0.0",
			ip:       "0.0.0.0",
			expected: 0x00000000,
		},
		{
			name:     "255.255.255.255",
			ip:       "255.255.255.255",
			expected: 0xFFFFFFFF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip).To4()
			result := ipToUint32(ip)

			if result != tt.expected {
				t.Errorf("ipToUint32(%s) = 0x%08X, expected 0x%08X", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestUint32ToIP(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected string
	}{
		{
			name:     "192.168.1.1",
			input:    0xC0A80101,
			expected: "192.168.1.1",
		},
		{
			name:     "10.0.0.1",
			input:    0x0A000001,
			expected: "10.0.0.1",
		},
		{
			name:     "0.0.0.0",
			input:    0x00000000,
			expected: "0.0.0.0",
		},
		{
			name:     "255.255.255.255",
			input:    0xFFFFFFFF,
			expected: "255.255.255.255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uint32ToIP(tt.input).String()

			if result != tt.expected {
				t.Errorf("uint32ToIP(0x%08X) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLargestPowerOfTwoLessOrEqual(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		expected uint32
	}{
		{
			name:     "Zero",
			input:    0,
			expected: 1,
		},
		{
			name:     "One",
			input:    1,
			expected: 1,
		},
		{
			name:     "Two",
			input:    2,
			expected: 2,
		},
		{
			name:     "Three",
			input:    3,
			expected: 2,
		},
		{
			name:     "Four",
			input:    4,
			expected: 4,
		},
		{
			name:     "Seven",
			input:    7,
			expected: 4,
		},
		{
			name:     "Eight",
			input:    8,
			expected: 8,
		},
		{
			name:     "Fifteen",
			input:    15,
			expected: 8,
		},
		{
			name:     "Sixteen",
			input:    16,
			expected: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := largestPowerOfTwoLessOrEqual(tt.input)

			if result != tt.expected {
				t.Errorf("largestPowerOfTwoLessOrEqual(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRangeToExactCIDRs(t *testing.T) {
	tests := []struct {
		name     string
		start    uint32
		end      uint32
		expected []string
	}{
		{
			name:     "Single IP",
			start:    ipToUint32(net.ParseIP("192.168.1.1").To4()),
			end:      ipToUint32(net.ParseIP("192.168.1.1").To4()),
			expected: []string{"ip4:192.168.1.1"},
		},
		{
			name:     "Two contiguous IPs",
			start:    ipToUint32(net.ParseIP("192.168.1.0").To4()),
			end:      ipToUint32(net.ParseIP("192.168.1.1").To4()),
			expected: []string{"ip4:192.168.1.0/31"},
		},
		{
			name:     "Four contiguous IPs",
			start:    ipToUint32(net.ParseIP("192.168.1.0").To4()),
			end:      ipToUint32(net.ParseIP("192.168.1.3").To4()),
			expected: []string{"ip4:192.168.1.0/30"},
		},
		{
			name:     "Unaligned range",
			start:    ipToUint32(net.ParseIP("192.168.1.1").To4()),
			end:      ipToUint32(net.ParseIP("192.168.1.3").To4()),
			expected: []string{"ip4:192.168.1.1", "ip4:192.168.1.2/31"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rangeToExactCIDRs(tt.start, tt.end)

			sort.Strings(result)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("rangeToExactCIDRs(%d, %d) = %v, expected %v", tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

func TestIPv6ToBigInt(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected string // big.Int string representation
	}{
		{
			name:     "IPv6 loopback",
			ip:       "::1",
			expected: "1",
		},
		{
			name:     "IPv6 zero",
			ip:       "::",
			expected: "0",
		},
		{
			name:     "Simple IPv6",
			ip:       "2001:db8::1",
			expected: "42540766411282592856903984951653826561", // 2001:db8::1 as decimal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := ipv6ToBigInt(ip)

			if result.String() != tt.expected {
				t.Errorf("ipv6ToBigInt(%s) = %s, expected %s", tt.ip, result.String(), tt.expected)
			}
		})
	}
}

func TestBigIntToIPv6(t *testing.T) {
	tests := []struct {
		name     string
		input    string // big.Int string representation
		expected string
	}{
		{
			name:     "IPv6 loopback",
			input:    "1",
			expected: "::1",
		},
		{
			name:     "IPv6 zero",
			input:    "0",
			expected: "::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bigInt := new(big.Int)
			bigInt.SetString(tt.input, 10)
			result := bigIntToIPv6(bigInt)

			if result.String() != tt.expected {
				t.Errorf("bigIntToIPv6(%s) = %s, expected %s", tt.input, result.String(), tt.expected)
			}
		})
	}
}

func TestAggregateIPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    []string // IP strings
		expected []string
	}{
		{
			name:     "Empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single IP",
			input:    []string{"192.168.1.1"},
			expected: []string{"ip4:192.168.1.1"},
		},
		{
			name:     "Two contiguous IPs",
			input:    []string{"192.168.1.0", "192.168.1.1"},
			expected: []string{"ip4:192.168.1.0/31"},
		},
		{
			name:     "Duplicates",
			input:    []string{"192.168.1.1", "192.168.1.1", "192.168.1.2"},
			expected: []string{"ip4:192.168.1.1", "ip4:192.168.1.2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert string IPs to net.IP
			ips := make([]net.IP, len(tt.input))
			for i, ipStr := range tt.input {
				ips[i] = net.ParseIP(ipStr).To4()
			}

			result := aggregateIPv4(ips)

			sort.Strings(result)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("aggregateIPv4(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAggregateIPv6(t *testing.T) {
	tests := []struct {
		name     string
		input    []string // IP strings
		expected []string
	}{
		{
			name:     "Empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single IP",
			input:    []string{"2001:db8::1"},
			expected: []string{"ip6:2001:db8::1"},
		},
		{
			name:     "Two contiguous IPs",
			input:    []string{"2001:db8::", "2001:db8::1"},
			expected: []string{"ip6:2001:db8::/127"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert string IPs to net.IP
			ips := make([]net.IP, len(tt.input))
			for i, ipStr := range tt.input {
				ips[i] = net.ParseIP(ipStr).To16()
			}

			result := aggregateIPv6(ips)

			sort.Strings(result)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("aggregateIPv6(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Benchmark tests

func BenchmarkAggregateCIDRs100IPs(b *testing.B) {
	// Generate 100 consecutive IPv4 addresses
	ips := make([]string, 100)
	for i := 0; i < 100; i++ {
		ips[i] = fmt.Sprintf("ip4:192.168.%d.%d", i/256, i%256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AggregateCIDRs(ips)
	}
}

func BenchmarkAggregateCIDRs1000IPs(b *testing.B) {
	// Generate 1000 consecutive IPv4 addresses
	ips := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		ips[i] = fmt.Sprintf("ip4:192.%d.%d.%d", i/65536, (i/256)%256, i%256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AggregateCIDRs(ips)
	}
}

func BenchmarkAggregateCIDRsRandom1000IPs(b *testing.B) {
	// Generate 1000 random IPv4 addresses (won't aggregate well)
	ips := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		ips[i] = fmt.Sprintf("ip4:%d.%d.%d.%d", i%255+1, (i*7)%255+1, (i*13)%255+1, (i*19)%255+1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AggregateCIDRs(ips)
	}
}
