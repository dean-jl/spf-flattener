package backup

import (
	"testing"
	"time"
)

func TestValidator_ValidateRecord_ValidARecord(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "@",
		Type:    "A",
		Content: "192.168.1.1",
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	if !result.IsValid {
		t.Errorf("Expected valid A record, got errors: %v", result.Errors)
	}
	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
}

func TestValidator_ValidateRecord_InvalidARecord(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "@",
		Type:    "A",
		Content: "invalid.ip.address",
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	if result.IsValid {
		t.Error("Expected invalid A record to be marked as invalid")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid IP address")
	}
}

func TestValidator_ValidateRecord_ValidMXRecord(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:     "@",
		Type:     "MX",
		Content:  "mail.example.com",
		TTL:      3600,
		Priority: 10,
	}

	result := validator.ValidateRecord(record)

	if !result.IsValid {
		t.Errorf("Expected valid MX record, got errors: %v", result.Errors)
	}
}

func TestValidator_ValidateRecord_InvalidMXPriority(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:     "@",
		Type:     "MX",
		Content:  "mail.example.com",
		TTL:      3600,
		Priority: 0, // Invalid priority
	}

	result := validator.ValidateRecord(record)

	if result.IsValid {
		t.Error("Expected invalid MX record due to invalid priority")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid priority")
	}
}

func TestValidator_ValidateRecord_InvalidTTL(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "@",
		Type:    "A",
		Content: "192.168.1.1",
		TTL:     0, // Invalid TTL
	}

	result := validator.ValidateRecord(record)

	if result.IsValid {
		t.Error("Expected invalid record due to invalid TTL")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid TTL")
	}
}

func TestValidator_ValidateRecord_InvalidRecordType(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "@",
		Type:    "INVALID",
		Content: "192.168.1.1",
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	if result.IsValid {
		t.Error("Expected invalid record due to invalid record type")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid record type")
	}
}

func TestValidator_ValidateRecord_MissingRequiredFields(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		// Missing Name, Type, and Content
		TTL: 3600,
	}

	result := validator.ValidateRecord(record)

	if result.IsValid {
		t.Error("Expected invalid record due to missing required fields")
	}
	if len(result.Errors) < 3 {
		t.Errorf("Expected at least 3 errors for missing required fields, got: %d", len(result.Errors))
	}
}

func TestValidator_ValidateRecord_CNAMEAtRoot(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "@", // CNAME at domain root is invalid
		Type:    "CNAME",
		Content: "example.com",
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	if result.IsValid {
		t.Error("Expected invalid CNAME record at domain root")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for CNAME at domain root")
	}
}

func TestValidator_ValidateRecord_SPFWarning(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "@",
		Type:    "TXT",
		Content: "v=spf1 include:_spf.google.com", // Missing ~all or -all
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	// Should be valid but with warnings
	if !result.IsValid {
		t.Errorf("Expected valid SPF record, got errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for incomplete SPF record")
	}
}

func TestValidator_ValidateRecordSet_Valid(t *testing.T) {
	validator := NewValidator()
	recordSet := &DNSRecordSet{
		Domain:     "example.com",
		Provider:   "porkbun",
		Version:    "1.0",
		ExportedAt: time.Now(),
		Records: []DNSRecord{
			{
				Name:    "@",
				Type:    "A",
				Content: "192.168.1.1",
				TTL:     3600,
			},
			{
				Name:     "@",
				Type:     "MX",
				Content:  "mail.example.com",
				TTL:      3600,
				Priority: 10,
			},
		},
	}

	result := validator.ValidateRecordSet(recordSet)

	if !result.IsValid {
		t.Errorf("Expected valid record set, got errors: %v", result.Errors)
	}
	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
}

func TestValidator_ValidateRecordSet_EmptyRecords(t *testing.T) {
	validator := NewValidator()
	recordSet := &DNSRecordSet{
		Domain:     "example.com",
		Provider:   "porkbun",
		Version:    "1.0",
		ExportedAt: time.Now(),
		Records:    []DNSRecord{}, // Empty records
	}

	result := validator.ValidateRecordSet(recordSet)

	if result.IsValid {
		t.Error("Expected invalid record set due to empty records")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for empty records")
	}
}

func TestValidator_ValidateRecordSet_DuplicateRecords(t *testing.T) {
	validator := NewValidator()
	recordSet := &DNSRecordSet{
		Domain:     "example.com",
		Provider:   "porkbun",
		Version:    "1.0",
		ExportedAt: time.Now(),
		Records: []DNSRecord{
			{
				Name:    "@",
				Type:    "A",
				Content: "192.168.1.1",
				TTL:     3600,
			},
			{
				Name:    "@", // Duplicate name and type
				Type:    "A",
				Content: "192.168.1.1", // Duplicate content
				TTL:     3600,
			},
		},
	}

	result := validator.ValidateRecordSet(recordSet)

	// Should be valid but with warnings about duplicates
	if !result.IsValid {
		t.Errorf("Expected valid record set, got errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for duplicate records")
	}
}

func TestValidator_ValidateRecordSet_MissingMetadata(t *testing.T) {
	validator := NewValidator()
	recordSet := &DNSRecordSet{
		// Missing Domain and Provider
		Version:    "1.0",
		ExportedAt: time.Now(),
		Records: []DNSRecord{
			{
				Name:    "@",
				Type:    "A",
				Content: "192.168.1.1",
				TTL:     3600,
			},
		},
	}

	result := validator.ValidateRecordSet(recordSet)

	if result.IsValid {
		t.Error("Expected invalid record set due to missing metadata")
	}
	if len(result.Errors) < 2 { // Should have errors for missing domain and provider
		t.Errorf("Expected at least 2 errors for missing metadata, got: %d", len(result.Errors))
	}
}

func TestValidator_ValidateRecord_SOAValid(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "@",
		Type:    "SOA",
		Content: "ns1.example.com admin.example.com 2024011501 3600 1800 604800 86400",
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	if !result.IsValid {
		t.Errorf("Expected valid SOA record, got errors: %v", result.Errors)
	}
}

func TestValidator_ValidateRecord_SOAInvalidFormat(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "@",
		Type:    "SOA",
		Content: "invalid soa format",
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	if result.IsValid {
		t.Error("Expected invalid SOA record due to format")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid SOA format")
	}
}

func TestValidator_ValidateRecord_SRVValid(t *testing.T) {
	validator := NewValidator()
	record := DNSRecord{
		Name:    "_sip._tcp",
		Type:    "SRV",
		Content: "10 60 5060 sipserver.example.com",
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	if !result.IsValid {
		t.Errorf("Expected valid SRV record, got errors: %v", result.Errors)
	}
}

func TestValidator_ValidateRecord_TXTLongContent(t *testing.T) {
	validator := NewValidator()
	longContent := "This is a very long TXT record content that exceeds the recommended 255 character limit for optimal DNS performance and compatibility with various DNS servers and resolvers across the internet infrastructure. This additional text makes it longer than 255 characters to trigger the warning."
	record := DNSRecord{
		Name:    "@",
		Type:    "TXT",
		Content: longContent,
		TTL:     3600,
	}

	result := validator.ValidateRecord(record)

	// Should be valid but with warnings
	if !result.IsValid {
		t.Errorf("Expected valid TXT record, got errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for long TXT content")
	}
}

func TestHelperFunctions(t *testing.T) {
	validator := NewValidator()

	// Test isValidRecordType
	if !validator.isValidRecordType("A") {
		t.Error("Expected 'A' to be a valid record type")
	}
	if validator.isValidRecordType("INVALID") {
		t.Error("Expected 'INVALID' to be an invalid record type")
	}

	// Test isValidTTL
	if !validator.isValidTTL(3600) {
		t.Error("Expected 3600 to be a valid TTL")
	}
	if validator.isValidTTL(0) {
		t.Error("Expected 0 to be an invalid TTL")
	}
	if validator.isValidTTL(100000) {
		t.Error("Expected 100000 to be an invalid TTL")
	}

	// Test isValidDomainName
	if !validator.isValidDomainName("example.com") {
		t.Error("Expected 'example.com' to be a valid domain name")
	}
	if !validator.isValidDomainName("@") {
		t.Error("Expected '@' to be a valid domain name")
	}
	if validator.isValidDomainName("invalid..domain") {
		t.Error("Expected 'invalid..domain' to be an invalid domain name")
	}
}
