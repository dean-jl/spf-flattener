package spf

import (
	"fmt"
	"sort"
	"strings"
)

// isSPFMechanism checks if a given string part represents a valid SPF mechanism.
//
// This helper function identifies SPF mechanisms by their prefixes, supporting:
//   - include: - Include another domain's SPF policy
//   - a/a: - Authorize IPs that resolve from A records
//   - mx/mx: - Authorize IPs that resolve from MX records
//   - ip4:/ip6: - Explicitly authorize IPv4/IPv6 addresses
//   - ptr - Reverse DNS mechanism (discouraged)
//   - exists: - Test for existence of A record
//   - redirect= - Redirect to another domain's SPF record
//   - exp= - Explanation string for failures
//
// Returns true if the part is a recognized SPF mechanism, false otherwise.
func isSPFMechanism(part string) bool {
	return strings.HasPrefix(part, "include:") ||
		strings.HasPrefix(part, "a") ||
		strings.HasPrefix(part, "mx") ||
		strings.HasPrefix(part, "ip4:") ||
		strings.HasPrefix(part, "ip6:") ||
		strings.HasPrefix(part, "ptr") ||
		strings.HasPrefix(part, "exists:") ||
		strings.HasPrefix(part, "redirect=") ||
		strings.HasPrefix(part, "exp=")
}

// NormalizeSPF takes an SPF record string and returns a normalized version with
// consistent mechanism ordering.
//
// The normalization process:
//  1. Validates the record starts with "v=spf1"
//  2. Separates the "all" mechanism (e.g., ~all, -all)
//  3. Sorts all other mechanisms alphabetically
//  4. Reconstructs the record with "v=spf1" first, sorted mechanisms, and "all" last
//
// This normalization is useful for comparing SPF records where the mechanisms
// might be in different orders but are functionally equivalent.
//
// Parameters:
//   - spfRecord: The SPF record string to normalize
//
// Returns:
//   - string: The normalized SPF record
//   - error: Error if the record is invalid (doesn't start with v=spf1 or is empty)
//
// Example:
//
//	normalized, err := NormalizeSPF("v=spf1 mx a include:_spf.google.com ~all")
//	// Result: "v=spf1 a include:_spf.google.com mx ~all"
func NormalizeSPF(spfRecord string) (string, error) {
	if !strings.HasPrefix(spfRecord, "v=spf1") {
		return "", fmt.Errorf("invalid SPF record: must start with v=spf1")
	}

	parts := strings.Fields(spfRecord)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty SPF record")
	}

	// The first part is always "v=spf1"
	normalizedParts := []string{parts[0]}

	// Separate the "all" mechanism
	var allMechanism string
	mechanisms := []string{}

	for _, part := range parts[1:] {
		if strings.HasSuffix(part, "all") { // Check for ~all, -all, +all, ?all
			allMechanism = part
		} else if isSPFMechanism(part) {
			mechanisms = append(mechanisms, part)
		} else {
			// Handle unknown or malformed parts by appending them as is, but they won't be sorted
			normalizedParts = append(normalizedParts, part)
		}
	}

	// Sort mechanisms for consistent order
	sort.Strings(mechanisms)

	// Append sorted mechanisms
	normalizedParts = append(normalizedParts, mechanisms...)

	// Append the "all" mechanism last
	if allMechanism != "" {
		normalizedParts = append(normalizedParts, allMechanism)
	}

	return strings.Join(normalizedParts, " "), nil
}

// ExtractMechanisms parses an SPF record and returns a sorted slice of mechanisms (excluding v=spf1 and all).
func ExtractMechanisms(spfRecord string) ([]string, error) {
	if !strings.HasPrefix(spfRecord, "v=spf1") {
		return nil, fmt.Errorf("invalid SPF record: must start with v=spf1")
	}
	parts := strings.Fields(spfRecord)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty SPF record")
	}
	mechanisms := []string{}
	for _, part := range parts[1:] {
		if strings.HasSuffix(part, "all") {
			continue // skip all mechanism
		} else if isSPFMechanism(part) {
			mechanisms = append(mechanisms, part)
		}
	}
	sort.Strings(mechanisms)
	return mechanisms, nil
}

// ExtractMechanismSet parses an SPF record string and returns a set of mechanisms (ignoring order and duplicates).
func ExtractMechanismSet(spfRecord string) map[string]struct{} {
	mechSet := make(map[string]struct{})
	parts := strings.Fields(spfRecord)
	for _, part := range parts[1:] { // skip v=spf1
		if strings.HasSuffix(part, "all") {
			mechSet[part] = struct{}{}
		} else if isSPFMechanism(part) {
			mechSet[part] = struct{}{}
		}
	}
	return mechSet
}
