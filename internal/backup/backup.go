package backup

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// DNSAPIClient abstracts DNS record management for extensibility.
// This interface is defined here to avoid circular imports.
type DNSAPIClient interface {
	Ping() (*PingResponse, error)
	RetrieveRecords(domain string) (*RetrieveRecordsResponse, error)
	UpdateRecord(domain, recordID, content string) (*UpdateRecordResponse, error)
	UpdateRecordWithDetails(domain, recordID, name, recordType, content, ttl, prio, notes string) (*UpdateRecordResponse, error)
	CreateRecord(domain, name, recordType, content string, ttl int) (*CreateRecordResponse, error)
	CreateRecordWithOptions(domain, name, recordType, content string, ttl int, prio string, notes string) (*CreateRecordResponse, error)
	DeleteRecord(domain, recordID string) (*DeleteRecordResponse, error)
	DeleteRecordByNameType(domain, recordType, subdomain string) (*DeleteRecordResponse, error)

	// Backup/Restore methods
	RetrieveAllRecords(domain string) ([]BackupDNSRecord, error)
	BulkCreateRecords(domain string, records []BackupDNSRecord) error
	BulkUpdateRecords(domain string, records []BackupDNSRecord) error
	BulkDeleteRecords(domain string, recordIDs []string) error

	// Attribution returns the attribution message for the DNS provider
	Attribution() string
}

// BackupDNSRecord represents a DNS record for backup/restore operations.
type BackupDNSRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// PingResponse represents the response from a ping operation.
type PingResponse struct {
	Status string `json:"status"`
	YourIP string `json:"yourIp"`
}

// RetrieveRecordsResponse represents the response from retrieving records.
type RetrieveRecordsResponse struct {
	Status  string   `json:"status"`
	Records []Record `json:"records"`
}

// Record represents a DNS record from the API.
type Record struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     string `json:"ttl"`
	Prio    string `json:"prio"`
	Notes   string `json:"notes"`
}

// UpdateRecordResponse represents the response from updating a record.
type UpdateRecordResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// CreateRecordResponse represents the response from creating a record.
type CreateRecordResponse struct {
	Status  string `json:"status"`
	ID      int    `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

// DeleteRecordResponse represents the response from deleting a record.
type DeleteRecordResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// BackupManager handles DNS record backup and restore operations.
type BackupManager struct {
	client            DNSAPIClient
	validator         *Validator
	logger            *log.Logger
	dryRun            bool
	retryCount        int
	retryDelay        time.Duration
	rateLimiter       *rate.Limiter
	maxRetryDelay     time.Duration
	backoffMultiplier float64
	jitterEnabled     bool
}

// BackupManagerConfig contains configuration for the BackupManager.
type BackupManagerConfig struct {
	Client            DNSAPIClient
	Logger            *log.Logger
	DryRun            bool
	RetryCount        int
	RetryDelay        time.Duration
	MaxRetryDelay     time.Duration
	BackoffMultiplier float64
	JitterEnabled     bool
	RateLimiter       *rate.Limiter // Standardized rate limiter using golang.org/x/time/rate
}

// isRateLimitError checks if an error indicates rate limiting.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "quota exceeded")
}

// NewBackupManager creates a new BackupManager with the given configuration.
func NewBackupManager(config BackupManagerConfig) *BackupManager {
	if config.RetryCount <= 0 {
		config.RetryCount = 5 // Increased default retry count
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.MaxRetryDelay <= 0 {
		config.MaxRetryDelay = 30 * time.Second
	}
	if config.BackoffMultiplier <= 0 {
		config.BackoffMultiplier = 2.0
	}
	// Set default rate limiter if not provided
	if config.RateLimiter == nil {
		// Match flatten command: 2 req/sec, burst 1
		config.RateLimiter = rate.NewLimiter(rate.Limit(2.0), 1)
	}
	if config.Logger == nil {
		config.Logger = log.Default()
	}

	return &BackupManager{
		client:            config.Client,
		validator:         NewValidator(),
		logger:            config.Logger,
		dryRun:            config.DryRun,
		retryCount:        config.RetryCount,
		retryDelay:        config.RetryDelay,
		rateLimiter:       config.RateLimiter,
		maxRetryDelay:     config.MaxRetryDelay,
		backoffMultiplier: config.BackoffMultiplier,
		jitterEnabled:     config.JitterEnabled,
	}
}

// ValidRecordTypes contains the list of supported DNS record types.
var ValidRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true, "TXT": true, "NS": true,
	"SOA": true, "SRV": true, "PTR": true, "CAA": true, "DNSKEY": true, "DS": true,
	"RRSIG": true, "NSEC": true, "NSEC3": true, "NSEC3PARAM": true,
}

// normalizeRecordType converts record type to uppercase and validates it.
func normalizeRecordType(recordType string) (string, error) {
	normalized := strings.ToUpper(recordType)
	if !ValidRecordTypes[normalized] {
		return "", fmt.Errorf("invalid DNS record type: %s", recordType)
	}
	return normalized, nil
}

// validateRecordTypes validates a slice of record types and returns normalized versions.
func validateRecordTypes(recordTypes []string) ([]string, error) {
	if len(recordTypes) == 0 {
		return nil, nil // Empty means all types
	}

	var normalized []string
	seen := make(map[string]bool)

	for _, recordType := range recordTypes {
		normalizedType, err := normalizeRecordType(recordType)
		if err != nil {
			return nil, fmt.Errorf("invalid record type '%s': %w", recordType, err)
		}

		if !seen[normalizedType] {
			seen[normalizedType] = true
			normalized = append(normalized, normalizedType)
		}
	}

	return normalized, nil
}

// calculateBackoffDelay calculates exponential backoff with optional jitter.
func (bm *BackupManager) calculateBackoffDelay(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}

	// Calculate exponential backoff
	delay := float64(bm.retryDelay) * math.Pow(bm.backoffMultiplier, float64(attempt-1))

	// Cap at maximum delay
	if delay > float64(bm.maxRetryDelay) {
		delay = float64(bm.maxRetryDelay)
	}

	duration := time.Duration(delay)

	// Add jitter if enabled
	if bm.jitterEnabled {
		jitter := time.Duration(rand.Float64() * float64(duration) * 0.1) // 10% jitter
		duration += jitter
	}

	return duration
}

// withRetry executes a function with intelligent retry logic and rate limiting.
func (bm *BackupManager) withRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < bm.retryCount; attempt++ {
		// Check context first
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("operation canceled: %w", err)
		}

		// Apply rate limiting
		if err := bm.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiting failed for %s: %w", operation, err)
		}

		// Execute the operation
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if this is a rate limiting or temporary error
		if isRateLimitError(err) || isTemporaryError(err) {
			if attempt < bm.retryCount-1 {
				delay := bm.calculateBackoffDelay(attempt)
				bm.logger.Printf("[RETRY] %s failed (attempt %d/%d), retrying in %v: %v",
					operation, attempt+1, bm.retryCount, delay, err)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
		} else {
			// Non-temporary error, don't retry
			break
		}
	}

	return fmt.Errorf("%s failed after %d attempts: %w", operation, bm.retryCount, lastErr)
}

// isTemporaryError checks if an error is temporary and should be retried.
func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "temporary failure")
}

// filterRecordsByType filters a slice of DNS records by type.
func filterRecordsByType(records []DNSRecord, filterTypes []string) []DNSRecord {
	if len(filterTypes) == 0 {
		return records // No filtering
	}

	var filtered []DNSRecord
	for _, record := range records {
		for _, filterType := range filterTypes {
			if strings.ToUpper(record.Type) == filterType {
				filtered = append(filtered, record)
				break
			}
		}
	}

	return filtered
}

// filterRecordSetByType creates a new DNSRecordSet with filtered records.
func filterRecordSetByType(recordSet *DNSRecordSet, filterTypes []string) (*DNSRecordSet, error) {
	if len(filterTypes) == 0 {
		return recordSet, nil // No filtering
	}

	filteredRecords := filterRecordsByType(recordSet.Records, filterTypes)

	// Create new record set with filtered records
	filteredSet := &DNSRecordSet{
		Domain:      recordSet.Domain,
		Provider:    recordSet.Provider,
		Version:     recordSet.Version,
		ExportedAt:  recordSet.ExportedAt,
		Records:     filteredRecords,
		Attribution: recordSet.Attribution,
	}

	return filteredSet, nil
}

// convertFromBackupRecord converts a BackupDNSRecord to DNSRecord.
func convertFromBackupRecord(record BackupDNSRecord) DNSRecord {
	return DNSRecord{
		ID:       record.ID,
		Name:     record.Name,
		Type:     record.Type,
		Content:  record.Content,
		TTL:      record.TTL,
		Priority: record.Priority,
		Notes:    record.Notes,
	}
}

// convertToBackupRecord converts a DNSRecord to BackupDNSRecord.
func convertToBackupRecord(record DNSRecord) BackupDNSRecord {
	return BackupDNSRecord{
		ID:       record.ID,
		Name:     record.Name,
		Type:     record.Type,
		Content:  record.Content,
		TTL:      record.TTL,
		Priority: record.Priority,
		Notes:    record.Notes,
	}
}

// ExportRecords exports DNS records for a domain to a DNSRecordSet with optional filtering.
func (bm *BackupManager) ExportRecords(ctx context.Context, domain string, options ExportOptions) (*DNSRecordSet, error) {
	// Validate and normalize record types if specified
	validatedTypes, err := validateRecordTypes(options.RecordTypes)
	if err != nil {
		return nil, fmt.Errorf("invalid record types: %w", err)
	}

	bm.logger.Printf("Starting export for domain: %s", domain)
	if len(validatedTypes) > 0 {
		bm.logger.Printf("Filtering by record types: %v", validatedTypes)
	}

	// Test API connectivity first
	if err := bm.testConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to DNS provider: %w", err)
	}

	// Retrieve all records with intelligent retry logic
	var backupRecords []BackupDNSRecord
	retrieveErr := bm.withRetry(ctx, "retrieve records", func() error {
		var err error
		backupRecords, err = bm.client.RetrieveAllRecords(domain)
		return err
	})
	if retrieveErr != nil {
		return nil, fmt.Errorf("failed to retrieve records after %d attempts: %w", bm.retryCount, retrieveErr)
	}

	// Convert backup records to DNS records
	records := make([]DNSRecord, len(backupRecords))
	for i, record := range backupRecords {
		records[i] = convertFromBackupRecord(record)
	}

	bm.logger.Printf("Successfully retrieved %d records for domain %s", len(records), domain)

	// Apply record type filtering if specified
	filteredRecords := filterRecordsByType(records, validatedTypes)
	if len(validatedTypes) > 0 {
		bm.logger.Printf("Filtered to %d records matching types: %v", len(filteredRecords), validatedTypes)
	}

	// Create DNSRecordSet
	recordSet := &DNSRecordSet{
		Domain:      domain,
		Provider:    "porkbun", // Could be made configurable
		Version:     "1.0",
		ExportedAt:  time.Now(),
		Records:     filteredRecords,
		Attribution: bm.client.Attribution(),
	}

	// Validate the record set
	validationResult := bm.validator.ValidateRecordSet(recordSet)
	if !validationResult.IsValid {
		bm.logger.Printf("Validation warnings for %s: %v", domain, validationResult.Warnings)
		bm.logger.Printf("Validation errors for %s: %v", domain, validationResult.Errors)

		// For export, we treat empty record sets as warnings, not errors
		if len(validationResult.Errors) > 0 {
			// Check if the only error is about empty records
			onlyEmptyRecordsError := true
			for _, err := range validationResult.Errors {
				if err != "Record set must contain at least one record" {
					onlyEmptyRecordsError = false
					break
				}
			}

			if !onlyEmptyRecordsError {
				return nil, fmt.Errorf("validation failed for domain %s: %v", domain, validationResult.Errors)
			}
		}
	}

	bm.logger.Printf("Export completed successfully for domain: %s", domain)
	return recordSet, nil
}

// ImportRecords imports DNS records from a DNSRecordSet to the DNS provider.
func (bm *BackupManager) ImportRecords(ctx context.Context, recordSet *DNSRecordSet, options ImportOptions) (*ImportResult, error) {
	bm.logger.Printf("Starting import for domain: %s (dry-run: %v)", recordSet.Domain, options.DryRun || bm.dryRun)

	startTime := time.Now()

	// Validate input
	if recordSet == nil {
		return nil, fmt.Errorf("record set cannot be nil")
	}
	if len(recordSet.Records) == 0 {
		return nil, fmt.Errorf("no records to import")
	}

	// Validate and normalize record types if specified
	validatedTypes, err := validateRecordTypes(options.RecordTypes)
	if err != nil {
		return nil, fmt.Errorf("invalid record types: %w", err)
	}

	// Apply record type filtering if specified
	filteredRecordSet, err := filterRecordSetByType(recordSet, validatedTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to filter records: %w", err)
	}

	if len(validatedTypes) > 0 {
		bm.logger.Printf("Filtering by record types: %v", validatedTypes)
		bm.logger.Printf("Filtered from %d to %d records", len(recordSet.Records), len(filteredRecordSet.Records))
	}

	// Use filtered record set for the rest of the operation
	recordSet = filteredRecordSet

	// Check if we still have records after filtering
	if len(recordSet.Records) == 0 {
		return nil, fmt.Errorf("no records to import after filtering by types: %v", validatedTypes)
	}

	// Validate records before import
	validationResult := bm.validator.ValidateRecordSet(recordSet)
	if !validationResult.IsValid {
		return nil, fmt.Errorf("record validation failed: %v", validationResult.Errors)
	}

	result := &ImportResult{
		Domain:       recordSet.Domain,
		TotalRecords: len(recordSet.Records),
		Errors:       []error{},
	}

	// Add validation warnings as conflicts
	for _, warning := range validationResult.Warnings {
		result.Conflicts = append(result.Conflicts, Conflict{
			ExistingRecord: DNSRecord{}, // No specific record for general warnings
			ImportedRecord: DNSRecord{},
			ConflictType:   "validation_warning",
			Resolution:     warning,
		})
	}

	// Process each record
	for i, record := range recordSet.Records {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			return result, ctx.Err()
		default:
		}

		bm.logger.Printf("Processing record %d/%d: %s %s", i+1, len(recordSet.Records), record.Name, record.Type)

		if options.DryRun || bm.dryRun {
			result.Skipped++
			bm.logger.Printf("[DRY-RUN] Would import record: %s %s -> %s", record.Name, record.Type, record.Content)
			continue
		}

		// Check for existing records if using skip strategy
		if options.ConflictStrategy == "skip" {
			existingRecord, exists, err := bm.recordExists(ctx, recordSet.Domain, record)
			if err != nil {
				bm.logger.Printf("Warning: Failed to check for existing record %s %s: %v", record.Name, record.Type, err)
				// Continue with import attempt if we can't check
			} else if exists {
				result.Skipped++
				result.Conflicts = append(result.Conflicts, Conflict{
					ExistingRecord: *existingRecord,
					ImportedRecord: record,
					ConflictType:   "existing_record",
					Resolution:     "skipped",
				})
				bm.logger.Printf("Skipped existing record: %s %s", record.Name, record.Type)
				continue
			}
		}

		// Import the record
		err := bm.importSingleRecord(ctx, record, recordSet.Domain, options)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("failed to import %s %s: %w", record.Name, record.Type, err))
			result.Conflicts = append(result.Conflicts, Conflict{
				ExistingRecord: DNSRecord{}, // No existing record for import failures
				ImportedRecord: record,
				ConflictType:   "import_failure",
			})
			bm.logger.Printf("Failed to import record %s %s: %v", record.Name, record.Type, err)
		} else {
			result.Created++
			bm.logger.Printf("Successfully imported record: %s %s -> %s", record.Name, record.Type, record.Content)
		}
	}

	result.Duration = time.Since(startTime)
	bm.logger.Printf("Import completed for domain %s: %d created, %d failed, %d skipped",
		result.Domain, result.Created, result.Failed, result.Skipped)

	return result, nil
}

// recordExists checks if a DNS record already exists for the given domain and record details.
// It returns the existing record, whether it exists, and any error encountered.
func (bm *BackupManager) recordExists(ctx context.Context, domain string, record DNSRecord) (*DNSRecord, bool, error) {
	var existingRecords []BackupDNSRecord

	// Retrieve all existing records for the domain with rate limiting
	err := bm.withRetry(ctx, "retrieve records for existence check", func() error {
		var err error
		existingRecords, err = bm.client.RetrieveAllRecords(domain)
		return err
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to retrieve existing records: %w", err)
	}

	// Extract the hostname we're looking for
	targetHostname := extractHostnameFromFQDN(record.Name, domain)

	// Look for a matching record
	for _, existingRecord := range existingRecords {
		existingHostname := extractHostnameFromFQDN(existingRecord.Name, domain)

		// Check if this is the same record (same name, type, and content)
		if existingHostname == targetHostname &&
			existingRecord.Type == record.Type &&
			existingRecord.Content == record.Content {
			// Convert BackupDNSRecord to DNSRecord for return
			matchingRecord := convertFromBackupRecord(existingRecord)
			return &matchingRecord, true, nil
		}
	}

	return nil, false, nil
}

// extractHostnameFromFQDN extracts the hostname portion from a fully qualified domain name.
// For example: "www.example.com" → "www", "example.com" → "@"
func extractHostnameFromFQDN(fqdn string, domain string) string {
	if fqdn == domain {
		return "@" // Root domain
	}

	if len(fqdn) > len(domain)+1 && strings.HasSuffix(fqdn, "."+domain) {
		return strings.TrimSuffix(fqdn, "."+domain)
	}

	// If it doesn't match the domain pattern, return as-is
	return fqdn
}

// importSingleRecord imports a single record with intelligent retry and error handling.
func (bm *BackupManager) importSingleRecord(ctx context.Context, record DNSRecord, domain string, options ImportOptions) error {
	// Extract the hostname from the FQDN
	hostname := extractHostnameFromFQDN(record.Name, domain)

	backupRecord := convertToBackupRecord(record)
	// Update the record name to use the extracted hostname
	backupRecord.Name = hostname

	return bm.withRetry(ctx, fmt.Sprintf("create record %s %s", record.Name, record.Type), func() error {
		return bm.client.BulkCreateRecords(domain, []BackupDNSRecord{backupRecord})
	})
}

// testConnectivity tests the connection to the DNS provider with retry logic.
func (bm *BackupManager) testConnectivity(ctx context.Context) error {
	bm.logger.Printf("Testing connectivity to DNS provider...")

	var pingResp *PingResponse
	err := bm.withRetry(ctx, "ping", func() error {
		var err error
		pingResp, err = bm.client.Ping()
		return err
	})
	if err != nil {
		return fmt.Errorf("ping failed after %d attempts: %w", bm.retryCount, err)
	}

	if pingResp.Status != "SUCCESS" {
		return fmt.Errorf("ping returned non-success status: %s", pingResp.Status)
	}

	bm.logger.Printf("Connectivity test successful. Your IP: %s", pingResp.YourIP)
	return nil
}
