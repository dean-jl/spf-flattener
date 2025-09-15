package spf

import (
	"fmt"
	"strings"
)

const (
	// maxSPFChars is the maximum number of characters allowed in a single SPF string.
	maxSPFChars = 255
)

// SplitAndChainSPF splits a flattened SPF record into multiple chained TXT records for a domain.
// Returns a map of record names to values, plus the main domain record.
func SplitAndChainSPF(spfRecord, domain string) map[string]string {
	if len(spfRecord) <= maxSPFChars {
		return map[string]string{
			domain: spfRecord,
		}
	}

	// Remove the trailing "~all" or "-all" if present, as we'll add it back during chaining
	spfRecord = strings.TrimSuffix(spfRecord, " ~all")
	spfRecord = strings.TrimSuffix(spfRecord, " -all")

	parts := strings.Fields(spfRecord)
	var records []string
	var currentRecord strings.Builder

	currentRecord.WriteString(parts[0]) // "v=spf1"

	for i, part := range parts[1:] {
		// Calculate the chaining string that will be added later
		chaining := " ~all"
		if i < len(parts)-1 {
			chaining = " include:spfX." + domain + " ~all" // X will be replaced later
		}
		// Estimate the max length for this record including chaining
		if currentRecord.Len()+len(part)+1+len(chaining) > maxSPFChars {
			records = append(records, currentRecord.String())
			currentRecord.Reset()
			currentRecord.WriteString(parts[0])
		}
		currentRecord.WriteString(" ")
		currentRecord.WriteString(part)
	}

	records = append(records, currentRecord.String())

	result := make(map[string]string)
	for i := 0; i < len(records); i++ {
		name := "spf" + fmt.Sprintf("%d.%s", i, domain)
		chaining := " ~all"
		if i < len(records)-1 {
			chaining = " include:spf" + fmt.Sprintf("%d.%s", i+1, domain) + " ~all"
		}
		// Ensure the final record does not exceed 255 chars
		record := records[i]
		maxLen := maxSPFChars - len(chaining)
		if len(record) > maxLen {
			record = record[:maxLen]
		}
		result[name] = record + chaining
	}
	// Main domain record includes spf0
	result[domain] = "v=spf1 include:spf0." + domain + " ~all"
	return result
}

// Exported wrapper for tests and compatibility
func SplitSPF(spfRecord string) []string {
	if len(spfRecord) <= maxSPFChars {
		return []string{spfRecord}
	}
	parts := strings.Fields(spfRecord)
	var records []string
	var currentRecord strings.Builder
	currentRecord.WriteString(parts[0]) // "v=spf1"
	for _, part := range parts[1:] {
		if currentRecord.Len()+len(part)+1+len(" ~all") > maxSPFChars {
			records = append(records, currentRecord.String()+" ~all")
			currentRecord.Reset()
			currentRecord.WriteString(parts[0])
		}
		currentRecord.WriteString(" ")
		currentRecord.WriteString(part)
	}
	if currentRecord.Len() > 0 {
		records = append(records, currentRecord.String()+" ~all")
	}
	return records
}
