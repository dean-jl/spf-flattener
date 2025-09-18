// Package spf contains CIDR aggregation functionality for SPF record optimization.
// This module provides exact CIDR aggregation that merges contiguous IP ranges
// into efficient CIDR blocks while maintaining identical reachability.
package spf

import (
	"fmt"
	"math"
	"math/big"
	"net"
	"sort"
	"strings"
)

// AggregationConfig holds configuration for CIDR aggregation behavior
type AggregationConfig struct {
	IPv4MaxPrefix      int      // Maximum IPv4 CIDR prefix allowed - prevents overly broad aggregation (default: 24)
	IPv6MaxPrefix      int      // Maximum IPv6 CIDR prefix allowed - prevents overly broad aggregation (default: 64)
	PreserveIndividual []string // List of IPs that should never be aggregated
}

// AggregateCIDRs takes a slice of SPF IP mechanisms and returns aggregated CIDR blocks.
// It separates IPv4, IPv6, and non-IP mechanisms, performs exact aggregation on IP addresses,
// and returns the combined result. Only exact aggregations are performed - no unintended
// IP addresses are ever authorized.
//
// This function uses SPF-optimized defaults that allow aggressive aggregation to reduce
// the number of DNS lookup mechanisms in SPF records.
//
// Input format: []string{"ip4:192.168.1.1", "ip4:192.168.1.2", "ip6:2001:db8::1", "include:example.com"}
// Output format: []string{"ip4:192.168.1.0/31", "ip6:2001:db8::1/128", "include:example.com"}
func AggregateCIDRs(ipMechanisms []string) []string {
	// Use SPF-optimized configuration that allows any level of aggregation
	spfConfig := &AggregationConfig{
		IPv4MaxPrefix:      1, // Allow any aggregation for SPF optimization
		IPv6MaxPrefix:      1, // Allow any aggregation for SPF optimization
		PreserveIndividual: []string{},
	}
	return AggregateCIDRsWithConfig(ipMechanisms, spfConfig)
}

// AggregateCIDRsWithConfig performs CIDR aggregation with advanced configuration options.
// This allows per-domain customization of aggregation behavior including minimum prefix
// lengths and preservation of specific individual IP addresses.
func AggregateCIDRsWithConfig(ipMechanisms []string, config *AggregationConfig) []string {
	if len(ipMechanisms) == 0 {
		return []string{}
	}

	// Set default configuration if none provided
	if config == nil {
		config = &AggregationConfig{
			IPv4MaxPrefix:      24,
			IPv6MaxPrefix:      64,
			PreserveIndividual: []string{},
		}
	}

	// Separate individual IPs from existing CIDR blocks and other mechanisms
	ipv4IndividualIPs, ipv4ExistingCIDRs := separateIPv4IndividualFromCIDR(ipMechanisms)
	ipv6IndividualIPs, ipv6ExistingCIDRs := separateIPv6IndividualFromCIDR(ipMechanisms)
	otherMechanisms := extractNonIPMechanisms(ipMechanisms)

	// Only aggregate individual IPs - preserve existing CIDR blocks
	aggregatedIPv4 := aggregateIPv4WithConfig(ipv4IndividualIPs, config)
	aggregatedIPv6 := aggregateIPv6WithConfig(ipv6IndividualIPs, config)

	// Combine all results: aggregated individual IPs + preserved CIDR blocks + other mechanisms
	result := make([]string, 0, len(aggregatedIPv4)+len(aggregatedIPv6)+len(ipv4ExistingCIDRs)+len(ipv6ExistingCIDRs)+len(otherMechanisms))
	result = append(result, aggregatedIPv4...)
	result = append(result, aggregatedIPv6...)
	result = append(result, ipv4ExistingCIDRs...)
	result = append(result, ipv6ExistingCIDRs...)
	result = append(result, otherMechanisms...)

	return result
}

// separateIPv4IndividualFromCIDR separates individual IPv4 addresses from existing CIDR blocks.
// Returns (individual IPs for aggregation, existing CIDR blocks to preserve)
func separateIPv4IndividualFromCIDR(mechanisms []string) ([]net.IP, []string) {
	var individualIPs []net.IP
	var existingCIDRs []string

	for _, mechanism := range mechanisms {
		if !strings.HasPrefix(mechanism, "ip4:") {
			continue
		}

		// Remove "ip4:" prefix
		ipStr := strings.TrimPrefix(mechanism, "ip4:")

		// Check if it's a CIDR block
		if strings.Contains(ipStr, "/") {
			// It's already a CIDR block - preserve it as-is
			_, _, err := net.ParseCIDR(ipStr)
			if err == nil {
				existingCIDRs = append(existingCIDRs, mechanism)
			}
		} else {
			// It's an individual IP - add to aggregation list
			ip := net.ParseIP(ipStr)
			if ip != nil && ip.To4() != nil {
				individualIPs = append(individualIPs, ip.To4())
			}
		}
	}

	return individualIPs, existingCIDRs
}

// extractIPv4Addresses parses ip4: mechanisms and extracts IPv4 addresses.
// DEPRECATED: Use separateIPv4IndividualFromCIDR instead to avoid expanding existing CIDR blocks.
func extractIPv4Addresses(mechanisms []string) []net.IP {
	var ips []net.IP

	for _, mechanism := range mechanisms {
		if !strings.HasPrefix(mechanism, "ip4:") {
			continue
		}

		// Remove "ip4:" prefix
		ipStr := strings.TrimPrefix(mechanism, "ip4:")

		// Handle CIDR blocks
		if strings.Contains(ipStr, "/") {
			_, cidr, err := net.ParseCIDR(ipStr)
			if err != nil {
				continue // Skip invalid CIDR
			}

			// Expand CIDR to individual IPs
			expandedIPs := expandCIDRToIPv4s(cidr)
			ips = append(ips, expandedIPs...)
		} else {
			// Handle single IP
			ip := net.ParseIP(ipStr)
			if ip != nil && ip.To4() != nil {
				ips = append(ips, ip.To4())
			}
		}
	}

	return ips
}

// separateIPv6IndividualFromCIDR separates individual IPv6 addresses from existing CIDR blocks.
// Returns (individual IPs for aggregation, existing CIDR blocks to preserve)
func separateIPv6IndividualFromCIDR(mechanisms []string) ([]net.IP, []string) {
	var individualIPs []net.IP
	var existingCIDRs []string

	for _, mechanism := range mechanisms {
		if !strings.HasPrefix(mechanism, "ip6:") {
			continue
		}

		// Remove "ip6:" prefix
		ipStr := strings.TrimPrefix(mechanism, "ip6:")

		// Check if it's a CIDR block
		if strings.Contains(ipStr, "/") {
			// It's already a CIDR block - preserve it as-is
			_, _, err := net.ParseCIDR(ipStr)
			if err == nil {
				existingCIDRs = append(existingCIDRs, mechanism)
			}
		} else {
			// It's an individual IP - add to aggregation list
			ip := net.ParseIP(ipStr)
			if ip != nil && ip.To4() == nil { // IPv6
				individualIPs = append(individualIPs, ip.To16())
			}
		}
	}

	return individualIPs, existingCIDRs
}

// extractIPv6Addresses parses ip6: mechanisms and extracts IPv6 addresses.
// DEPRECATED: Use separateIPv6IndividualFromCIDR instead to avoid expanding existing CIDR blocks.
func extractIPv6Addresses(mechanisms []string) []net.IP {
	var ips []net.IP

	for _, mechanism := range mechanisms {
		if !strings.HasPrefix(mechanism, "ip6:") {
			continue
		}

		// Remove "ip6:" prefix
		ipStr := strings.TrimPrefix(mechanism, "ip6:")

		// Handle CIDR blocks
		if strings.Contains(ipStr, "/") {
			_, cidr, err := net.ParseCIDR(ipStr)
			if err != nil {
				continue // Skip invalid CIDR
			}

			// Expand CIDR to individual IPs (limited to prevent memory issues)
			expandedIPs := expandCIDRToIPv6s(cidr)
			ips = append(ips, expandedIPs...)
		} else {
			// Handle single IP
			ip := net.ParseIP(ipStr)
			if ip != nil && ip.To4() == nil {
				ips = append(ips, ip.To16())
			}
		}
	}

	return ips
}

// extractNonIPMechanisms returns all mechanisms that are not ip4: or ip6:.
// These include include:, a:, mx:, exists:, redirect:, and qualifiers.
func extractNonIPMechanisms(mechanisms []string) []string {
	var nonIP []string

	for _, mechanism := range mechanisms {
		if !strings.HasPrefix(mechanism, "ip4:") && !strings.HasPrefix(mechanism, "ip6:") {
			nonIP = append(nonIP, mechanism)
		}
	}

	return nonIP
}

// IPv4 Aggregation Functions

// aggregateIPv4 implements RFC 4632-compliant IPv4 CIDR aggregation.
// Converts IPs to uint32 for efficient processing and merges contiguous ranges.
func aggregateIPv4(ips []net.IP) []string {
	if len(ips) == 0 {
		return []string{}
	}

	// Remove duplicates and convert to uint32
	uniqueIPs := make(map[uint32]bool)
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			uniqueIPs[ipToUint32(ipv4)] = true
		}
	}

	// Convert to sorted slice
	uintIPs := make([]uint32, 0, len(uniqueIPs))
	for ip := range uniqueIPs {
		uintIPs = append(uintIPs, ip)
	}

	sort.Slice(uintIPs, func(i, j int) bool { return uintIPs[i] < uintIPs[j] })

	return mergeContiguousRanges(uintIPs)
}

// ipToUint32 converts an IPv4 address to uint32 for arithmetic operations.
func ipToUint32(ip net.IP) uint32 {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0
	}
	return uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])
}

// uint32ToIP converts a uint32 back to an IPv4 address.
func uint32ToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

// largestPowerOfTwoLessOrEqual finds the largest power of 2 that is <= value.
// Used for determining optimal CIDR block sizes.
func largestPowerOfTwoLessOrEqual(value uint32) uint32 {
	if value == 0 {
		return 1
	}

	// Find the highest bit position
	power := uint32(1)
	for power <= value {
		power <<= 1
	}
	return power >> 1
}

// mergeContiguousRanges finds contiguous IP ranges and converts them to exact CIDR blocks.
// CRITICAL: Only aggregates when the resulting CIDR block contains exactly the same
// IP addresses as the input, with no gaps or unintended inclusions.
func mergeContiguousRanges(sortedIPs []uint32) []string {
	if len(sortedIPs) == 0 {
		return []string{}
	}

	var cidrs []string

	for i := 0; i < len(sortedIPs); {
		start := sortedIPs[i]
		end := start
		j := i + 1

		// Find the end of the contiguous range (consecutive IPs only)
		for j < len(sortedIPs) && sortedIPs[j] == end+1 {
			end = sortedIPs[j]
			j++
		}

		// Convert range to exact CIDR blocks (no gaps allowed)
		cidrs = append(cidrs, rangeToExactCIDRs(start, end)...)

		// Move to next non-contiguous IP (j is already pointing to the next unprocessed IP)
		i = j
	}

	return cidrs
}

// rangeToExactCIDRs converts a contiguous IP range to CIDR blocks that cover
// exactly that range with no additional IP addresses.
func rangeToExactCIDRs(start, end uint32) []string {
	var cidrs []string

	for start <= end {
		// Find the largest CIDR block that starts at 'start' and doesn't exceed 'end'
		maxSize := largestPowerOfTwoLessOrEqual(end - start + 1)

		// Ensure the block is properly aligned (start address must be divisible by block size)
		for start%maxSize != 0 {
			maxSize /= 2
		}

		prefixLen := 32 - int(math.Log2(float64(maxSize)))
		if prefixLen == 32 {
			// Single IP - use bare address format for compatibility
			cidrs = append(cidrs, fmt.Sprintf("ip4:%s", uint32ToIP(start)))
		} else {
			// CIDR block - use CIDR notation
			cidrs = append(cidrs, fmt.Sprintf("ip4:%s/%d", uint32ToIP(start), prefixLen))
		}

		start += maxSize
	}

	return cidrs
}

// expandCIDRToIPv4s expands a CIDR block to individual IPv4 addresses.
// Limited to prevent memory exhaustion with large blocks.
func expandCIDRToIPv4s(cidr *net.IPNet) []net.IP {
	const maxIPs = 65536 // Limit to /16 networks

	var ips []net.IP

	// Calculate network size
	ones, bits := cidr.Mask.Size()
	if bits != 32 {
		return ips // Not IPv4
	}

	networkSize := uint32(1) << (32 - ones)
	if networkSize > maxIPs {
		return ips // Too large, skip expansion
	}

	// Convert network address to uint32
	networkIP := ipToUint32(cidr.IP)

	// Generate all IPs in the range
	for i := uint32(0); i < networkSize; i++ {
		ips = append(ips, uint32ToIP(networkIP+i))
	}

	return ips
}

// IPv6 Aggregation Functions

// aggregateIPv6 implements RFC 1887-compliant IPv6 CIDR aggregation.
// Uses big.Int for 128-bit arithmetic operations.
func aggregateIPv6(ips []net.IP) []string {
	if len(ips) == 0 {
		return []string{}
	}

	// Remove duplicates and convert to big.Int
	uniqueIPs := make(map[string]*big.Int)
	for _, ip := range ips {
		if ipv6 := ip.To16(); ipv6 != nil && ip.To4() == nil {
			bigInt := ipv6ToBigInt(ipv6)
			uniqueIPs[bigInt.String()] = bigInt
		}
	}

	// Convert to sorted slice
	bigInts := make([]*big.Int, 0, len(uniqueIPs))
	for _, bigInt := range uniqueIPs {
		bigInts = append(bigInts, bigInt)
	}

	sort.Slice(bigInts, func(i, j int) bool {
		return bigInts[i].Cmp(bigInts[j]) < 0
	})

	return mergeContiguousIPv6Ranges(bigInts)
}

// aggregateIPv4WithConfig implements IPv4 aggregation with advanced configuration options.
// Supports minimum prefix length restrictions and preservation of individual IPs.
func aggregateIPv4WithConfig(ips []net.IP, config *AggregationConfig) []string {
	if len(ips) == 0 {
		return []string{}
	}

	// Separate preserved IPs from aggregatable IPs
	preservedIPs := make(map[string]bool)
	for _, preserveIP := range config.PreserveIndividual {
		preservedIPs[preserveIP] = true
	}

	var aggregatableIPs []net.IP
	var individualIPs []string

	for _, ip := range ips {
		ipStr := ip.String()
		if preservedIPs[ipStr] {
			// Add as individual IP
			individualIPs = append(individualIPs, "ip4:"+ipStr)
		} else {
			aggregatableIPs = append(aggregatableIPs, ip)
		}
	}

	// Aggregate the non-preserved IPs using standard algorithm
	aggregatedCIDRs := aggregateIPv4(aggregatableIPs)

	// Filter aggregated CIDRs based on minimum prefix length
	var filteredCIDRs []string
	for _, cidr := range aggregatedCIDRs {
		if strings.Contains(cidr, "/") {
			// Extract prefix length
			parts := strings.Split(cidr, "/")
			if len(parts) == 2 {
				prefixStr := parts[1]
				var prefix int
				if _, err := fmt.Sscanf(prefixStr, "%d", &prefix); err == nil {
					if prefix >= config.IPv4MaxPrefix {
						filteredCIDRs = append(filteredCIDRs, cidr)
					} else {
						// Convert back to individual IPs if prefix is too small
						expandedIPs := expandCIDRToIPs(cidr)
						for _, expandedIP := range expandedIPs {
							individualIPs = append(individualIPs, "ip4:"+expandedIP)
						}
					}
				}
			}
		} else {
			// Single IP (format is "ip4:x.x.x.x"), add to individual IPs
			individualIPs = append(individualIPs, cidr)
		}
	}

	// Combine filtered CIDRs and individual IPs
	result := append(filteredCIDRs, individualIPs...)
	return result
}

// aggregateIPv6WithConfig implements IPv6 aggregation with advanced configuration options.
// Supports minimum prefix length restrictions and preservation of individual IPs.
func aggregateIPv6WithConfig(ips []net.IP, config *AggregationConfig) []string {
	if len(ips) == 0 {
		return []string{}
	}

	// Separate preserved IPs from aggregatable IPs
	preservedIPs := make(map[string]bool)
	for _, preserveIP := range config.PreserveIndividual {
		preservedIPs[preserveIP] = true
	}

	var aggregatableIPs []net.IP
	var individualIPs []string

	for _, ip := range ips {
		ipStr := ip.String()
		if preservedIPs[ipStr] {
			// Add as individual IP
			individualIPs = append(individualIPs, "ip6:"+ipStr)
		} else {
			aggregatableIPs = append(aggregatableIPs, ip)
		}
	}

	// Aggregate the non-preserved IPs using standard algorithm
	aggregatedCIDRs := aggregateIPv6(aggregatableIPs)

	// Filter aggregated CIDRs based on minimum prefix length
	var filteredCIDRs []string
	for _, cidr := range aggregatedCIDRs {
		if strings.Contains(cidr, "/") {
			// Extract prefix length
			parts := strings.Split(cidr, "/")
			if len(parts) == 2 {
				prefixStr := parts[1]
				var prefix int
				if _, err := fmt.Sscanf(prefixStr, "%d", &prefix); err == nil {
					if prefix >= config.IPv6MaxPrefix {
						filteredCIDRs = append(filteredCIDRs, cidr)
					} else {
						// Convert back to individual IPs if prefix is too small
						expandedIPs := expandCIDRToIPs(cidr)
						for _, expandedIP := range expandedIPs {
							individualIPs = append(individualIPs, "ip6:"+expandedIP)
						}
					}
				}
			}
		} else {
			// Single IP, add to individual IPs
			individualIPs = append(individualIPs, cidr)
		}
	}

	// Combine filtered CIDRs and individual IPs
	result := append(filteredCIDRs, individualIPs...)
	return result
}

// ipv6ToBigInt converts an IPv6 address to big.Int for arithmetic operations.
func ipv6ToBigInt(ip net.IP) *big.Int {
	ipv6 := ip.To16()
	if ipv6 == nil {
		return big.NewInt(0)
	}

	bigInt := big.NewInt(0)
	bigInt.SetBytes(ipv6)
	return bigInt
}

// bigIntToIPv6 converts a big.Int back to an IPv6 address.
func bigIntToIPv6(bigInt *big.Int) net.IP {
	bytes := bigInt.Bytes()

	// Pad to 16 bytes if necessary
	if len(bytes) < 16 {
		padded := make([]byte, 16)
		copy(padded[16-len(bytes):], bytes)
		bytes = padded
	}

	return net.IP(bytes)
}

// mergeContiguousIPv6Ranges finds contiguous IPv6 ranges and converts them to exact CIDR blocks.
func mergeContiguousIPv6Ranges(sortedIPs []*big.Int) []string {
	if len(sortedIPs) == 0 {
		return []string{}
	}

	var cidrs []string
	one := big.NewInt(1)

	for i := 0; i < len(sortedIPs); {
		start := new(big.Int).Set(sortedIPs[i])
		end := new(big.Int).Set(start)
		j := i + 1

		// Find the end of the contiguous range
		for j < len(sortedIPs) {
			expected := new(big.Int).Add(end, one)
			if sortedIPs[j].Cmp(expected) == 0 {
				end.Set(sortedIPs[j])
				j++
			} else {
				break
			}
		}

		// Convert range to exact CIDR blocks
		cidrs = append(cidrs, ipv6RangeToExactCIDRs(start, end)...)

		// Move to next non-contiguous IP (j is already pointing to the next unprocessed IP)
		i = j
	}

	return cidrs
}

// ipv6RangeToExactCIDRs converts a contiguous IPv6 range to CIDR blocks.
func ipv6RangeToExactCIDRs(start, end *big.Int) []string {
	var cidrs []string
	one := big.NewInt(1)
	current := new(big.Int).Set(start)

	for current.Cmp(end) <= 0 {
		// Find the largest CIDR block that starts at 'current' and doesn't exceed 'end'
		maxSize := new(big.Int).Sub(end, current)
		maxSize.Add(maxSize, one)

		// Find the largest power of 2 <= maxSize
		blockSize := big.NewInt(1)
		for new(big.Int).Lsh(blockSize, 1).Cmp(maxSize) <= 0 {
			blockSize.Lsh(blockSize, 1)
		}

		// Ensure proper alignment
		for {
			temp := new(big.Int).Mod(current, blockSize)
			if temp.Cmp(big.NewInt(0)) == 0 {
				break
			}
			blockSize.Rsh(blockSize, 1)
		}

		// Calculate prefix length
		prefixLen := 128
		tempSize := new(big.Int).Set(blockSize)
		for tempSize.Cmp(big.NewInt(1)) > 0 {
			tempSize.Rsh(tempSize, 1)
			prefixLen--
		}

		if prefixLen == 128 {
			// Single IP - use bare address format for compatibility
			cidrs = append(cidrs, fmt.Sprintf("ip6:%s", bigIntToIPv6(current)))
		} else {
			// CIDR block - use CIDR notation
			cidrs = append(cidrs, fmt.Sprintf("ip6:%s/%d", bigIntToIPv6(current), prefixLen))
		}
		current.Add(current, blockSize)
	}

	return cidrs
}

// expandCIDRToIPv6s expands a CIDR block to individual IPv6 addresses.
// Limited to prevent memory exhaustion with large blocks.
func expandCIDRToIPv6s(cidr *net.IPNet) []net.IP {
	const maxIPs = 1024 // Limit to /118 networks or smaller

	var ips []net.IP

	// Calculate network size
	ones, bits := cidr.Mask.Size()
	if bits != 128 {
		return ips // Not IPv6
	}

	if ones < 118 { // Network too large
		return ips
	}

	networkSize := uint64(1) << (128 - ones)
	if networkSize > maxIPs {
		return ips // Too large, skip expansion
	}

	// Convert network address to big.Int
	networkBigInt := ipv6ToBigInt(cidr.IP)

	// Generate all IPs in the range
	for i := uint64(0); i < networkSize; i++ {
		currentBigInt := new(big.Int).Add(networkBigInt, big.NewInt(int64(i)))
		ips = append(ips, bigIntToIPv6(currentBigInt))
	}

	return ips
}

// SPFSemanticallyDifferent compares two SPF records for functional equivalence.
// Returns true if the IP address sets covered by the records are different.
// This function enables proper change detection when CIDR aggregation is used,
// as two SPF records might have different string representations but cover
// the same set of IP addresses.
func SPFSemanticallyDifferent(oldSPF, newSPF string) bool {
	oldIPs := expandSPFToIPSet(oldSPF)
	newIPs := expandSPFToIPSet(newSPF)

	return !ipSetsEqual(oldIPs, newIPs)
}

// expandSPFToIPSet converts CIDR blocks in an SPF record back to individual IPs for comparison.
// This function extracts all ip4: and ip6: mechanisms and expands CIDR blocks to their
// constituent IP addresses, creating a set for comparison purposes.
func expandSPFToIPSet(spfRecord string) map[string]bool {
	ipSet := make(map[string]bool)
	mechanisms := strings.Fields(spfRecord)

	for _, mechanism := range mechanisms {
		if strings.HasPrefix(mechanism, "ip4:") || strings.HasPrefix(mechanism, "ip6:") {
			ips := expandCIDRToIPs(mechanism)
			for _, ip := range ips {
				ipSet[ip] = true
			}
		}
	}

	return ipSet
}

// expandCIDRToIPs expands a single IP mechanism (ip4: or ip6:) to individual IP addresses.
// Handles both single IPs and CIDR blocks. Returns string representations of IP addresses.
func expandCIDRToIPs(mechanism string) []string {
	var ips []string

	// Remove the ip4: or ip6: prefix
	var ipStr string
	if strings.HasPrefix(mechanism, "ip4:") {
		ipStr = strings.TrimPrefix(mechanism, "ip4:")
	} else if strings.HasPrefix(mechanism, "ip6:") {
		ipStr = strings.TrimPrefix(mechanism, "ip6:")
	} else {
		return ips
	}

	// Check if it's a CIDR block
	if strings.Contains(ipStr, "/") {
		_, cidr, err := net.ParseCIDR(ipStr)
		if err != nil {
			return ips
		}

		// Expand CIDR to individual IPs
		if cidr.IP.To4() != nil {
			// IPv4 CIDR
			expandedIPs := expandCIDRToIPv4s(cidr)
			for _, ip := range expandedIPs {
				ips = append(ips, ip.String())
			}
		} else {
			// IPv6 CIDR
			expandedIPs := expandCIDRToIPv6s(cidr)
			for _, ip := range expandedIPs {
				ips = append(ips, ip.String())
			}
		}
	} else {
		// Single IP address
		ip := net.ParseIP(ipStr)
		if ip != nil {
			ips = append(ips, ip.String())
		}
	}

	return ips
}

// ipSetsEqual compares two IP sets without expanding large CIDR blocks for performance.
// For small sets, uses direct comparison. For larger sets, uses CIDR-based comparison.
func ipSetsEqual(set1, set2 map[string]bool) bool {
	if len(set1) != len(set2) {
		return false
	}

	// For small sets, use direct comparison
	if len(set1) <= 1000 {
		for ip := range set1 {
			if !set2[ip] {
				return false
			}
		}
		return true
	}

	// For large sets, use CIDR-based comparison
	return compareLargeCIDRSets(set1, set2)
}

// compareLargeCIDRSets efficiently compares large IP sets by grouping into CIDR blocks.
// This avoids the memory overhead of expanding very large CIDR blocks for comparison.
func compareLargeCIDRSets(set1, set2 map[string]bool) bool {
	// Convert sets to slices for aggregation, adding appropriate prefixes
	ips1 := make([]string, 0, len(set1))
	ips2 := make([]string, 0, len(set2))

	for ip := range set1 {
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			if parsedIP.To4() != nil {
				ips1 = append(ips1, "ip4:"+ip)
			} else {
				ips1 = append(ips1, "ip6:"+ip)
			}
		}
	}
	for ip := range set2 {
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			if parsedIP.To4() != nil {
				ips2 = append(ips2, "ip4:"+ip)
			} else {
				ips2 = append(ips2, "ip6:"+ip)
			}
		}
	}

	// Aggregate both sets into their canonical CIDR representations
	aggregated1 := AggregateCIDRs(ips1)
	aggregated2 := AggregateCIDRs(ips2)

	// Sort both sets for comparison
	sort.Strings(aggregated1)
	sort.Strings(aggregated2)

	// Compare canonical representations
	if len(aggregated1) != len(aggregated2) {
		return false
	}

	for i, cidr := range aggregated1 {
		if cidr != aggregated2[i] {
			return false
		}
	}

	return true
}
