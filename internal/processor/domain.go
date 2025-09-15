package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/dean-jl/spf-flattener/internal/config"
	"github.com/dean-jl/spf-flattener/internal/porkbun"
	"github.com/dean-jl/spf-flattener/internal/spf"
)

// DomainProcessor handles the business logic for processing SPF records for domains
type DomainProcessor struct {
	dnsProvider spf.DNSProvider
	debug       bool
	dryRun      bool
	spfUnflat   bool
}

// NewDomainProcessor creates a new domain processor
func NewDomainProcessor(dnsProvider spf.DNSProvider, debug, dryRun, spfUnflat bool) *DomainProcessor {
	return &DomainProcessor{
		dnsProvider: dnsProvider,
		debug:       debug,
		dryRun:      dryRun,
		spfUnflat:   spfUnflat,
	}
}

// ProcessResult represents the result of processing a domain
type ProcessResult struct {
	Domain     string
	Success    bool
	Error      error
	Report     string
	HasChanges bool
	Changes    []string
}

// ProcessDomain processes SPF records for a single domain
func (dp *DomainProcessor) ProcessDomain(ctx context.Context, domain config.Domain) ProcessResult {
	result := ProcessResult{
		Domain:  domain.Name,
		Success: false,
	}

	// Create Porkbun client for this domain
	client := porkbun.NewClient(domain.ApiKey, domain.SecretKey, dp.debug)

	// Determine SPF lookup name
	spfLookupName := domain.Name
	if dp.spfUnflat {
		spfLookupName = "spf-unflat." + domain.Name
	}

	// Flatten SPF record
	originalSPF, flattenedSPF, err := spf.FlattenSPF(ctx, spfLookupName, dp.dnsProvider)
	if err != nil {
		result.Error = fmt.Errorf("failed to flatten SPF for %s: %w", domain.Name, err)
		return result
	}

	// Retrieve existing records
	existingRecordsResp, err := client.RetrieveRecords(domain.Name)
	if err != nil {
		result.Error = fmt.Errorf("failed to retrieve existing records for %s: %w", domain.Name, err)
		return result
	}

	// Process existing SPF TXT records
	existingSPFTXTRecords := make(map[string]string)
	for _, record := range existingRecordsResp.Records {
		if record.Type != "TXT" {
			continue
		}
		recordName := strings.TrimSuffix(record.Name, ".")
		if recordName == domain.Name {
			recordName = "@"
		}
		if strings.Contains(record.Content, "v=spf1") {
			existingSPFTXTRecords[recordName] = record.Content
		}
		// Handle spfX.domain records (split records)
		if strings.HasPrefix(recordName, "spf") && strings.HasSuffix(recordName, "."+domain.Name) {
			existingSPFTXTRecords[recordName] = record.Content
		}
	}

	// Detect changes
	changes, hasChanges := dp.detectChanges(existingSPFTXTRecords, flattenedSPF, domain.Name)
	result.HasChanges = hasChanges
	result.Changes = changes

	// Generate report
	result.Report = dp.generateReport(domain.Name, originalSPF, flattenedSPF, existingSPFTXTRecords, changes, hasChanges)
	result.Success = true

	return result
}

// detectChanges compares existing records with flattened SPF and returns changes needed
func (dp *DomainProcessor) detectChanges(existingRecords map[string]string, flattenedSPF, domain string) ([]string, bool) {
	var changes []string
	hasChanges := false

	// Handle splitting if needed
	splitRecords := spf.SplitAndChainSPF(flattenedSPF, domain)

	// Check for changes in each expected record
	for recordName, expectedContent := range splitRecords {
		displayName := recordName
		if recordName == domain {
			displayName = "@"
		}

		if existing, exists := existingRecords[displayName]; exists {
			normalizedExisting, _ := spf.NormalizeSPF(existing)
			normalizedExpected, _ := spf.NormalizeSPF(expectedContent)

			if normalizedExisting != normalizedExpected {
				changes = append(changes, fmt.Sprintf("UPDATE %s: %s -> %s", displayName, existing, expectedContent))
				hasChanges = true
			}
		} else {
			changes = append(changes, fmt.Sprintf("CREATE %s: %s", displayName, expectedContent))
			hasChanges = true
		}
	}

	// Check for records to delete (existing SPF records not in expected set)
	for recordName := range existingRecords {
		found := false
		for expectedName := range splitRecords {
			expectedDisplay := expectedName
			if expectedName == domain {
				expectedDisplay = "@"
			}
			if recordName == expectedDisplay {
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, fmt.Sprintf("DELETE %s", recordName))
			hasChanges = true
		}
	}

	return changes, hasChanges
}

// generateReport creates a formatted report for the domain processing
func (dp *DomainProcessor) generateReport(domain, originalSPF, flattenedSPF string, existingRecords map[string]string, changes []string, hasChanges bool) string {
	var report strings.Builder

	report.WriteString("\n===== Processing domain: ")
	report.WriteString(domain)
	report.WriteString(" =====\n\n")

	report.WriteString("Original SPF: ")
	report.WriteString(originalSPF)
	report.WriteString("\n")

	report.WriteString("Flattened SPF: ")
	report.WriteString(flattenedSPF)
	report.WriteString("\n\n")

	// Show existing records
	if len(existingRecords) > 0 {
		report.WriteString("Existing SPF TXT records:\n")
		for name, content := range existingRecords {
			report.WriteString("  ")
			report.WriteString(name)
			report.WriteString(": ")
			report.WriteString(content)
			report.WriteString("\n")
		}
		report.WriteString("\n")
	} else {
		report.WriteString("No existing SPF TXT records found.\n\n")
	}

	// Show changes
	if hasChanges {
		report.WriteString("Changes needed:\n")
		for _, change := range changes {
			report.WriteString("  - ")
			report.WriteString(change)
			report.WriteString("\n")
		}
		report.WriteString("\n")

		if dp.dryRun {
			report.WriteString("DRY-RUN: No changes will be applied.\n")
		} else {
			report.WriteString("Changes will be applied.\n")
		}
	} else {
		report.WriteString("✓ No changes needed - records are already up to date.\n")
	}

	return report.String()
}
