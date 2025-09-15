package backup

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// DNS record validation logic.

// ValidationResult contains the results of a validation operation.
type ValidationResult struct {
	IsValid  bool      `json:"is_valid"`
	Errors   []string  `json:"errors,omitempty"`
	Warnings []string  `json:"warnings,omitempty"`
	Record   DNSRecord `json:"record,omitempty"`
}

// Validator provides DNS record validation functionality.
type Validator struct {
	// Can be extended with custom validation rules
}

// NewValidator creates a new DNS record validator.
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateRecord validates a single DNS record against DNS standards and best practices.
func (v *Validator) ValidateRecord(record DNSRecord) *ValidationResult {
	result := &ValidationResult{
		Record:  record,
		IsValid: true,
	}

	// Validate required fields
	if record.Name == "" {
		result.addError("Record name is required")
	}

	if record.Type == "" {
		result.addError("Record type is required")
	}

	if record.Content == "" {
		result.addError("Record content is required")
	}

	// Validate record type
	if !v.isValidRecordType(record.Type) {
		result.addError(fmt.Sprintf("Invalid record type: %s", record.Type))
	}

	// Validate TTL
	if !v.isValidTTL(record.TTL) {
		result.addError(fmt.Sprintf("Invalid TTL: %d. Must be between 1 and 86400 seconds", record.TTL))
	}

	// Validate based on record type
	switch strings.ToUpper(record.Type) {
	case "A":
		v.validateARecord(record, result)
	case "AAAA":
		v.validateAAAARecord(record, result)
	case "CNAME":
		v.validateCNAMERecord(record, result)
	case "MX":
		v.validateMXRecord(record, result)
	case "TXT":
		v.validateTXTRecord(record, result)
	case "NS":
		v.validateNSRecord(record, result)
	case "SOA":
		v.validateSOARecord(record, result)
	case "SRV":
		v.validateSRVRecord(record, result)
	case "PTR":
		v.validatePTRRecord(record, result)
	case "CAA":
		v.validateCAARecord(record, result)
	}

	// Validate domain name format (with exceptions for SRV records)
	if record.Name != "@" && !v.isValidDomainName(record.Name) && !v.isValidServiceName(record.Name, record.Type) {
		result.addError(fmt.Sprintf("Invalid domain name format: %s", record.Name))
	}

	// Update validity based on errors
	result.IsValid = len(result.Errors) == 0

	return result
}

// ValidateRecordSet validates a collection of DNS records.
func (v *Validator) ValidateRecordSet(recordSet *DNSRecordSet) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
	}

	// Validate record set metadata
	if recordSet.Domain == "" {
		result.addError("Record set domain is required")
	}

	if recordSet.Provider == "" {
		result.addError("Record set provider is required")
	}

	if len(recordSet.Records) == 0 {
		result.addError("Record set must contain at least one record")
	}

	// Validate individual records
	for i, record := range recordSet.Records {
		recordResult := v.ValidateRecord(record)
		if !recordResult.IsValid {
			result.IsValid = false
			for _, err := range recordResult.Errors {
				result.addError(fmt.Sprintf("Record %d (%s %s): %s", i+1, record.Name, record.Type, err))
			}
		}
		for _, warning := range recordResult.Warnings {
			result.addWarning(fmt.Sprintf("Record %d (%s %s): %s", i+1, record.Name, record.Type, warning))
		}
	}

	// Check for duplicate records
	v.checkForDuplicates(recordSet, result)

	return result
}

// validateARecord validates an A record.
func (v *Validator) validateARecord(record DNSRecord, result *ValidationResult) {
	if net.ParseIP(record.Content) == nil {
		result.addError(fmt.Sprintf("Invalid IPv4 address: %s", record.Content))
	}
}

// validateAAAARecord validates an AAAA record.
func (v *Validator) validateAAAARecord(record DNSRecord, result *ValidationResult) {
	ip := net.ParseIP(record.Content)
	if ip == nil || ip.To4() != nil {
		result.addError(fmt.Sprintf("Invalid IPv6 address: %s", record.Content))
	}
}

// validateCNAMERecord validates a CNAME record.
func (v *Validator) validateCNAMERecord(record DNSRecord, result *ValidationResult) {
	if record.Name == "@" {
		result.addError("CNAME records cannot be used for the domain root (@)")
	}

	if !v.isValidDomainName(record.Content) {
		result.addError(fmt.Sprintf("Invalid CNAME target: %s", record.Content))
	}
}

// validateMXRecord validates an MX record.
func (v *Validator) validateMXRecord(record DNSRecord, result *ValidationResult) {
	if record.Priority <= 0 || record.Priority > 65535 {
		result.addError(fmt.Sprintf("Invalid MX priority: %d. Must be between 1 and 65535", record.Priority))
	}

	if !v.isValidDomainName(record.Content) {
		result.addError(fmt.Sprintf("Invalid MX mail server: %s", record.Content))
	}
}

// validateTXTRecord validates a TXT record.
func (v *Validator) validateTXTRecord(record DNSRecord, result *ValidationResult) {
	// TXT records can contain any text, but we'll check for common issues
	if len(record.Content) > 255 {
		result.addWarning(fmt.Sprintf("TXT record content is very long (%d characters). Consider splitting into multiple records", len(record.Content)))
	}

	// Check for SPF record syntax if it looks like an SPF record
	if strings.HasPrefix(record.Content, "v=spf1") {
		v.validateSPFRecord(record.Content, result)
	}
}

// validateNSRecord validates an NS record.
func (v *Validator) validateNSRecord(record DNSRecord, result *ValidationResult) {
	if !v.isValidDomainName(record.Content) {
		result.addError(fmt.Sprintf("Invalid NS nameserver: %s", record.Content))
	}
}

// validateSOARecord validates an SOA record.
func (v *Validator) validateSOARecord(record DNSRecord, result *ValidationResult) {
	// SOA records have a complex format: MNAME RNAME SERIAL REFRESH RETRY EXPIRE MINIMUM
	parts := strings.Fields(record.Content)
	if len(parts) != 7 {
		result.addError("SOA record must have exactly 7 fields: MNAME RNAME SERIAL REFRESH RETRY EXPIRE MINIMUM")
		return
	}

	if !v.isValidDomainName(parts[0]) {
		result.addError(fmt.Sprintf("Invalid SOA MNAME (primary nameserver): %s", parts[0]))
	}

	if !v.isValidEmailAddress(parts[1]) {
		result.addError(fmt.Sprintf("Invalid SOA RNAME (admin email): %s", parts[1]))
	}

	// Validate numeric fields
	for i, field := range parts[2:] {
		if !v.isValidSerialNumber(field) {
			fieldNames := []string{"SERIAL", "REFRESH", "RETRY", "EXPIRE", "MINIMUM"}
			result.addError(fmt.Sprintf("Invalid SOA %s: %s", fieldNames[i], field))
		}
	}
}

// validateSRVRecord validates an SRV record.
func (v *Validator) validateSRVRecord(record DNSRecord, result *ValidationResult) {
	// SRV format: priority weight port target
	parts := strings.Fields(record.Content)
	if len(parts) != 4 {
		result.addError("SRV record must have exactly 4 fields: priority weight port target")
		return
	}

	for i, field := range parts[:3] {
		if !v.isValidPortNumber(field) {
			fieldNames := []string{"priority", "weight", "port"}
			result.addError(fmt.Sprintf("Invalid SRV %s: %s", fieldNames[i], field))
		}
	}

	if !v.isValidDomainName(parts[3]) {
		result.addError(fmt.Sprintf("Invalid SRV target: %s", parts[3]))
	}
}

// validatePTRRecord validates a PTR record.
func (v *Validator) validatePTRRecord(record DNSRecord, result *ValidationResult) {
	if !v.isValidDomainName(record.Content) {
		result.addError(fmt.Sprintf("Invalid PTR target: %s", record.Content))
	}
}

// validateCAARecord validates a CAA record.
func (v *Validator) validateCAARecord(record DNSRecord, result *ValidationResult) {
	// CAA format: flags tag value
	parts := strings.Fields(record.Content)
	if len(parts) != 3 {
		result.addError("CAA record must have exactly 3 fields: flags tag value")
		return
	}

	if parts[1] != "issue" && parts[1] != "issuewild" && parts[1] != "iodef" {
		result.addError(fmt.Sprintf("Invalid CAA tag: %s. Must be 'issue', 'issuewild', or 'iodef'", parts[1]))
	}
}

// validateSPFRecord validates SPF record syntax.
func (v *Validator) validateSPFRecord(content string, result *ValidationResult) {
	// Basic SPF validation
	if !strings.HasPrefix(content, "v=spf1") {
		result.addWarning("SPF record should start with 'v=spf1'")
	}

	// Check for common SPF issues
	if strings.Contains(content, "include:_spf.google.com") && !strings.Contains(content, "~all") && !strings.Contains(content, "-all") {
		result.addWarning("SPF record with Google include should end with ~all or -all")
	}

	// Check record length (SPF records should be under 450 characters to stay within TXT limits)
	if len(content) > 450 {
		result.addWarning("SPF record is very long. Consider using include mechanisms to reduce length")
	}
}

// Helper functions

// isValidRecordType checks if the record type is valid.
func (v *Validator) isValidRecordType(recordType string) bool {
	validTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SOA", "SRV", "PTR", "CAA", "ALIAS", "DNSKEY", "DS", "RRSIG", "NSEC", "NSEC3", "NSEC3PARAM"}
	recordTypeUpper := strings.ToUpper(recordType)
	for _, validType := range validTypes {
		if recordTypeUpper == validType {
			return true
		}
	}
	return false
}

// isValidTTL checks if the TTL is valid.
func (v *Validator) isValidTTL(ttl int) bool {
	return ttl >= 1 && ttl <= 86400
}

// isValidDomainName checks if the domain name format is valid according to DNS standards.
// This implements proper DNS naming conventions from RFC 1035 and RFC 1123.
func (v *Validator) isValidDomainName(domain string) bool {
	if domain == "@" {
		return true // @ represents the domain root
	}

	// Remove trailing dot if present (FQDN format)
	domain = strings.TrimSuffix(domain, ".")

	// Check total domain name length (must not exceed 255 octets)
	if len(domain) > 255 {
		return false
	}

	// Empty domain is not valid
	if len(domain) == 0 {
		return false
	}

	// Split into labels (parts separated by dots)
	labels := strings.Split(domain, ".")

	// Check each label
	for i, label := range labels {
		if !v.isValidDNSLabel(label, i == 0) {
			return false
		}
	}

	return true
}

// isValidDNSLabel validates individual DNS labels according to RFC specifications.
// firstLabel indicates if this is the first (leftmost) label, which may have special rules.
func (v *Validator) isValidDNSLabel(label string, firstLabel bool) bool {
	// Label length must be between 1 and 63 octets
	if len(label) < 1 || len(label) > 63 {
		return false
	}

	// Check for special DNS record names that may start with underscore
	// These are allowed for DNS-specific records like _dmarc, _acme-challenge, etc.
	// ANY label (not just first) can contain underscores for DNS records
	if len(label) > 0 && label[0] == '_' {
		return v.isValidSpecialDNSLabel(label)
	}

	// Standard label validation (RFC 1035 "preferred name syntax")
	// Labels must start and end with alphanumeric characters
	if !isAlphanumeric(rune(label[0])) || !isAlphanumeric(rune(label[len(label)-1])) {
		return false
	}

	// Check all characters in the label
	for _, char := range label {
		if !isAlphanumeric(char) && char != '-' {
			return false
		}
	}

	return true
}

// isValidSpecialDNSLabel validates special DNS labels that start with underscore.
// These are used for DNS-specific records like SRV, TXT records for domain verification, etc.
func (v *Validator) isValidSpecialDNSLabel(label string) bool {
	// Must start with underscore
	if len(label) == 0 || label[0] != '_' {
		return false
	}

	// Must have at least one character after the underscore
	if len(label) < 2 {
		return false
	}

	// Check remaining characters (after the underscore)
	for i := 1; i < len(label); i++ {
		char := rune(label[i])
		if !isAlphanumeric(char) && char != '-' && char != '_' {
			return false
		}
	}

	return true
}

// isAlphanumeric checks if a character is alphanumeric (A-Z, a-z, 0-9).
func isAlphanumeric(char rune) bool {
	return (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
}

// isValidServiceName checks if a service name (like SRV records) is valid.
func (v *Validator) isValidServiceName(name, recordType string) bool {
	if strings.ToUpper(recordType) == "SRV" {
		// SRV service names can contain underscores and follow pattern: _service._protocol
		srvRegex := `^_[a-zA-Z0-9\-]+\._[a-zA-Z0-9\-]+(\..*)?$`
		matched, _ := regexp.MatchString(srvRegex, name)
		return matched
	}
	return false
}

// isValidEmailAddress checks if the email address format is valid (for SOA RNAME).
func (v *Validator) isValidEmailAddress(email string) bool {
	// SOA RNAME uses dots instead of @, so user@domain.com becomes user.domain.com
	if !strings.Contains(email, ".") {
		return false
	}
	emailRegex := `^[a-zA-Z0-9._%+-]+(\.[a-zA-Z0-9._%+-]+)*$`
	matched, _ := regexp.MatchString(emailRegex, email)
	return matched
}

// isValidSerialNumber checks if the serial number is valid.
func (v *Validator) isValidSerialNumber(serial string) bool {
	_, err := strconv.ParseInt(serial, 10, 32)
	return err == nil
}

// isValidPortNumber checks if the port number is valid.
func (v *Validator) isValidPortNumber(port string) bool {
	num, err := strconv.Atoi(port)
	return err == nil && num >= 0 && num <= 65535
}

// checkForDuplicates checks for duplicate records in the record set.
func (v *Validator) checkForDuplicates(recordSet *DNSRecordSet, result *ValidationResult) {
	seen := make(map[string]bool)
	for i, record := range recordSet.Records {
		key := fmt.Sprintf("%s:%s:%s", record.Name, record.Type, record.Content)
		if seen[key] {
			result.addWarning(fmt.Sprintf("Duplicate record found at position %d: %s %s → %s", i+1, record.Name, record.Type, record.Content))
		}
		seen[key] = true
	}
}

// addError adds an error message to the validation result.
func (r *ValidationResult) addError(message string) {
	r.Errors = append(r.Errors, message)
	r.IsValid = false
}

// addWarning adds a warning message to the validation result.
func (r *ValidationResult) addWarning(message string) {
	r.Warnings = append(r.Warnings, message)
}
