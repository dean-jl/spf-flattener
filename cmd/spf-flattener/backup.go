package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dean-jl/spf-flattener/internal/backup"
	"github.com/dean-jl/spf-flattener/internal/config"
	"github.com/dean-jl/spf-flattener/internal/porkbun"
	"github.com/dean-jl/spf-flattener/internal/processor"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

// exportResult represents the result of an export operation
type exportResult struct {
	Domain   string
	Provider string
	Error    error
}

// importTask represents an import task for a specific file
type importTask struct {
	Filename string
	Domain   string
	Provider string
}

// importResult represents the result of an import operation
type importResult struct {
	Filename string
	Provider string
	Error    error
}

// DNSClientAdapter adapts porkbun.Client to implement backup.DNSAPIClient
type DNSClientAdapter struct {
	client *porkbun.Client
}

func NewDNSClientAdapter(client *porkbun.Client) *DNSClientAdapter {
	return &DNSClientAdapter{client: client}
}

func (a *DNSClientAdapter) Ping() (*backup.PingResponse, error) {
	resp, err := a.client.Ping()
	if err != nil {
		return nil, err
	}
	return &backup.PingResponse{
		Status: resp.Status,
		YourIP: resp.YourIP,
	}, nil
}

func (a *DNSClientAdapter) RetrieveRecords(domain string) (*backup.RetrieveRecordsResponse, error) {
	resp, err := a.client.RetrieveRecords(domain)
	if err != nil {
		return nil, err
	}

	records := make([]backup.Record, len(resp.Records))
	for i, record := range resp.Records {
		records[i] = backup.Record{
			ID:      record.ID,
			Name:    record.Name,
			Type:    record.Type,
			Content: record.Content,
			TTL:     record.TTL,
			Prio:    record.Prio,
			Notes:   record.Notes,
		}
	}

	return &backup.RetrieveRecordsResponse{
		Status:  resp.Status,
		Records: records,
	}, nil
}

func (a *DNSClientAdapter) UpdateRecord(domain, recordID, content string) (*backup.UpdateRecordResponse, error) {
	resp, err := a.client.UpdateRecord(domain, recordID, content)
	if err != nil {
		return nil, err
	}
	return &backup.UpdateRecordResponse{
		Status:  resp.Status,
		Message: resp.Message,
	}, nil
}

func (a *DNSClientAdapter) UpdateRecordWithDetails(domain, recordID, name, recordType, content, ttl, prio, notes string) (*backup.UpdateRecordResponse, error) {
	resp, err := a.client.UpdateRecordWithDetails(domain, recordID, name, recordType, content, ttl, prio, notes)
	if err != nil {
		return nil, err
	}
	return &backup.UpdateRecordResponse{
		Status:  resp.Status,
		Message: resp.Message,
	}, nil
}

func (a *DNSClientAdapter) CreateRecord(domain, name, recordType, content string, ttl int) (*backup.CreateRecordResponse, error) {
	resp, err := a.client.CreateRecord(domain, name, recordType, content, ttl)
	if err != nil {
		return nil, err
	}
	return &backup.CreateRecordResponse{
		Status:  resp.Status,
		ID:      resp.ID,
		Message: resp.Message,
	}, nil
}

func (a *DNSClientAdapter) CreateRecordWithOptions(domain, name, recordType, content string, ttl int, prio string, notes string) (*backup.CreateRecordResponse, error) {
	resp, err := a.client.CreateRecordWithOptions(domain, name, recordType, content, ttl, prio, notes)
	if err != nil {
		return nil, err
	}
	return &backup.CreateRecordResponse{
		Status:  resp.Status,
		ID:      resp.ID,
		Message: resp.Message,
	}, nil
}

func (a *DNSClientAdapter) DeleteRecord(domain, recordID string) (*backup.DeleteRecordResponse, error) {
	resp, err := a.client.DeleteRecord(domain, recordID)
	if err != nil {
		return nil, err
	}
	return &backup.DeleteRecordResponse{
		Status:  resp.Status,
		Message: resp.Message,
	}, nil
}

func (a *DNSClientAdapter) DeleteRecordByNameType(domain, recordType, subdomain string) (*backup.DeleteRecordResponse, error) {
	resp, err := a.client.DeleteRecordByNameType(domain, recordType, subdomain)
	if err != nil {
		return nil, err
	}
	return &backup.DeleteRecordResponse{
		Status:  resp.Status,
		Message: resp.Message,
	}, nil
}

func (a *DNSClientAdapter) RetrieveAllRecords(domain string) ([]backup.BackupDNSRecord, error) {
	records, err := a.client.RetrieveAllRecords(domain)
	if err != nil {
		return nil, err
	}

	backupRecords := make([]backup.BackupDNSRecord, len(records))
	for i, record := range records {
		backupRecords[i] = backup.BackupDNSRecord{
			ID:       record.ID,
			Name:     record.Name,
			Type:     record.Type,
			Content:  record.Content,
			TTL:      record.TTL,
			Priority: record.Priority,
			Notes:    record.Notes,
		}
	}

	return backupRecords, nil
}

func (a *DNSClientAdapter) BulkCreateRecords(domain string, records []backup.BackupDNSRecord) error {
	porkbunRecords := make([]porkbun.BackupDNSRecord, len(records))
	for i, record := range records {
		porkbunRecords[i] = porkbun.BackupDNSRecord{
			ID:       record.ID,
			Name:     record.Name,
			Type:     record.Type,
			Content:  record.Content,
			TTL:      record.TTL,
			Priority: record.Priority,
			Notes:    record.Notes,
		}
	}
	return a.client.BulkCreateRecords(domain, porkbunRecords)
}

func (a *DNSClientAdapter) BulkUpdateRecords(domain string, records []backup.BackupDNSRecord) error {
	porkbunRecords := make([]porkbun.BackupDNSRecord, len(records))
	for i, record := range records {
		porkbunRecords[i] = porkbun.BackupDNSRecord{
			ID:       record.ID,
			Name:     record.Name,
			Type:     record.Type,
			Content:  record.Content,
			TTL:      record.TTL,
			Priority: record.Priority,
			Notes:    record.Notes,
		}
	}
	return a.client.BulkUpdateRecords(domain, porkbunRecords)
}

func (a *DNSClientAdapter) BulkDeleteRecords(domain string, recordIDs []string) error {
	return a.client.BulkDeleteRecords(domain, recordIDs)
}

func (a *DNSClientAdapter) Attribution() string {
	return a.client.Attribution()
}

var (
	exportFormat      string
	exportOutputDir   string
	exportDomains     []string
	exportRecordTypes []string
	importFiles       []string
	importStrategy    string
	importBackup      bool
	importRecordTypes []string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export DNS records from configured domains to backup files",
	Long: `Export DNS records from your DNS provider to backup files.

This command exports all DNS records for the specified domains and saves them
in either JSON format (machine-readable) or TXT format (human-readable).

The export process includes:
- DNS record validation and sanitization
- Complete record set export (A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, etc.)
- Metadata preservation (timestamps, provider information)
- Automatic retry for temporary failures
- Comprehensive error reporting

Examples:
  # Export all domains from config to JSON files
  spf-flattener export --config config.yaml --format json --output-dir ./backups

  # Export specific domains in human-readable format
  spf-flattener export --config config.yaml --format txt --output-dir ./backups --domains example.com,mydomain.org

  # Export only specific record types (TXT and CNAME)
  spf-flattener export --config config.yaml --record-types TXT,CNAME --format json

  # Export only A records for specific domain
  spf-flattener export --config config.yaml --domains example.com --record-types A --format txt

  # Test connectivity without exporting (dry-run)
  spf-flattener export --config config.yaml --dry-run

  # Export with custom filename (will be: example.com-dns-backup-20240115-143022.json)
  spf-flattener export --config config.yaml --domains example.com --format json --output-dir ./backups

  # Export multiple domains with specific record types
  spf-flattener export --config config.yaml --format json --output-dir ./backups --domains example.com,api.example.com --record-types A,AAAA,CNAME

Output files are named as: {domain}-dns-backup-{timestamp}.{format}

Format Options:
  json  - Machine-readable format suitable for automated processing
  txt   - Human-readable format for manual review and documentation

Use --dry-run to test connectivity and preview export operation without creating files.`,
	Run: func(cmd *cobra.Command, args []string) {
		populateConfigFromFlags(cmd)

		// Production mode overrides dry-run for actual operations
		if cliConfig.Production {
			cliConfig.DryRun = false
		}

		if cliConfig.DryRun {
			fmt.Println("DRY-RUN: Testing connectivity and previewing export operation.")
		}

		// Load configuration
		cfg, err := config.LoadConfig(cliConfig.ConfigPath)
		if err != nil {
			cmd.PrintErrf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Determine which domains to export
		domainsToExport := getDomainsToExport(cfg.Domains, exportDomains)
		if len(domainsToExport) == 0 {
			cmd.PrintErrln("No domains specified or configured for export")
			os.Exit(1)
		}

		// Create output directory if it doesn't exist
		if exportOutputDir != "" {
			if err := os.MkdirAll(exportOutputDir, 0755); err != nil {
				cmd.PrintErrf("Error creating output directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Setup logger
		logger := log.Default()
		verbosePrintlnf("[VERBOSE] Starting export for %d domains\n", len(domainsToExport))
		debugPrintlnf("[DEBUG] Export configuration: format=%s, output-dir=%s, record-types=%v\n",
			exportFormat, exportOutputDir, exportRecordTypes)

		// Group domains by provider for parallel processing
		ctx := context.Background()
		providerGroups := processor.GroupDomainsByProvider(domainsToExport)

		verbosePrintlnf("[VERBOSE] Grouped %d domains into %d provider groups\n", len(domainsToExport), len(providerGroups))
		debugPrintlnf("[DEBUG] Provider groups: %+v\n", func() map[string]int {
			counts := make(map[string]int)
			for name, group := range providerGroups {
				counts[name] = len(group.Domains)
			}
			return counts
		}())

		// Process each provider group in parallel
		var wg sync.WaitGroup
		resultsChan := make(chan exportResult, len(domainsToExport))

		for providerName, group := range providerGroups {
			wg.Add(1)
			go func(provider string, domains *processor.ProviderGroup) {
				defer wg.Done()

				// Process domains for this provider sequentially to respect rate limits
				for i, domain := range domains.Domains {
					verbosePrintlnf("[VERBOSE] [%d/%d] Exporting domain: %s (provider: %s)\n", i+1, len(domains.Domains), domain.Name, provider)
					debugPrintlnf("[DEBUG] Domain %s config: TTL=%d, API key length=%d\n", domain.Name, domain.TTL, len(domain.ApiKey))

					err := exportDomainWithRateLimit(ctx, &domain, domains.RateLimiter, logger)
					resultsChan <- exportResult{
						Domain:   domain.Name,
						Provider: provider,
						Error:    err,
					}
				}
			}(providerName, group)
		}

		// Wait for all provider groups to complete
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Process results
		successCount := 0
		failureCount := 0

		for result := range resultsChan {
			if result.Error != nil {
				failureCount++
				verbosePrintlnf("[VERBOSE] Failed to export domain %s: %v\n", result.Domain, result.Error)
				debugPrintlnf("[DEBUG] Export error details for %s (provider: %s): %+v\n", result.Domain, result.Provider, result.Error)
				if !cliConfig.Verbose {
					cmd.PrintErrf("Error exporting %s: %v\n", result.Domain, result.Error)
				}
			} else {
				successCount++
				verbosePrintlnf("[VERBOSE] Successfully exported domain: %s\n", result.Domain)
				debugPrintlnf("[DEBUG] Export completed for %s (provider: %s)\n", result.Domain, result.Provider)
			}
		}

		// Print summary
		fmt.Printf("\nExport Summary:\n")
		fmt.Printf("  Success: %d domains\n", successCount)
		fmt.Printf("  Failed:  %d domains\n", failureCount)
		fmt.Printf("  Format:  %s\n", strings.ToUpper(exportFormat))
		if exportOutputDir != "" {
			fmt.Printf("  Output:  %s\n", exportOutputDir)
		}

		if failureCount > 0 {
			os.Exit(1)
		}
	},
}

func getDomainsToExport(configuredDomains []config.Domain, requestedDomains []string) []config.Domain {
	if len(requestedDomains) == 0 {
		return configuredDomains
	}

	var result []config.Domain
	requestedSet := make(map[string]bool)
	for _, name := range requestedDomains {
		requestedSet[name] = true
	}

	for _, domain := range configuredDomains {
		if requestedSet[domain.Name] {
			result = append(result, domain)
		}
	}

	return result
}

func exportDomain(ctx context.Context, domain *config.Domain, logger *log.Logger) error {
	if cliConfig.DryRun {
		return testDomainConnectivity(ctx, domain, logger)
	}

	// Create Porkbun client
	client := porkbun.NewClient(domain.ApiKey, domain.SecretKey, cliConfig.Debug)

	// Create adapter to interface with backup package
	adapter := NewDNSClientAdapter(client)

	// Create backup manager
	backupConfig := backup.BackupManagerConfig{
		Client:            adapter,
		Logger:            logger,
		DryRun:            cliConfig.DryRun,
		RetryCount:        5,
		RetryDelay:        1 * time.Second,
		MaxRetryDelay:     30 * time.Second,
		BackoffMultiplier: 2.0,
		JitterEnabled:     true,
		RateLimiter:       rate.NewLimiter(rate.Limit(2.0), 1), // Match flatten command
	}

	backupManager := backup.NewBackupManager(backupConfig)

	// Setup export options with record type filtering
	exportOptions := backup.ExportOptions{
		RecordTypes: exportRecordTypes,
		DryRun:      cliConfig.DryRun,
		Verbose:     cliConfig.Verbose,
	}

	// Export records
	recordSet, err := backupManager.ExportRecords(ctx, domain.Name, exportOptions)
	if err != nil {
		return fmt.Errorf("failed to export records: %w", err)
	}

	// Get format handler
	formatHandler, err := backup.GetFormatHandler(exportFormat)
	if err != nil {
		return fmt.Errorf("failed to get format handler: %w", err)
	}

	// Serialize records
	data, err := formatHandler.Serialize(recordSet)
	if err != nil {
		return fmt.Errorf("failed to serialize records: %w", err)
	}

	// Determine output filename
	filename := getOutputFilename(domain.Name, exportFormat)
	if exportOutputDir != "" {
		filename = filepath.Join(exportOutputDir, filename)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	logger.Printf("Exported %d records to %s", len(recordSet.Records), filename)
	return nil
}

// exportDomainWithRateLimit wraps exportDomain with provider-specific rate limiting
func exportDomainWithRateLimit(ctx context.Context, domain *config.Domain, rateLimiter *rate.Limiter, logger *log.Logger) error {
	// Apply rate limiting before domain export
	if err := rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiting failed for domain %s: %w", domain.Name, err)
	}

	return exportDomain(ctx, domain, logger)
}

// extractDomainProviderFromBackup reads a backup file to extract domain and provider information
func extractDomainProviderFromBackup(filename string, cfg *config.Config) (string, string, error) {
	// Read the backup file
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", "", fmt.Errorf("failed to read backup file: %w", err)
	}

	// Determine format from file extension
	formatHandler, err := backup.GetFormatHandlerFromFilename(filename)
	if err != nil {
		return "", "", fmt.Errorf("failed to determine format for file %s: %w", filename, err)
	}

	// Deserialize records to get domain information
	recordSet, err := formatHandler.Deserialize(data)
	if err != nil {
		return "", "", fmt.Errorf("failed to deserialize backup file: %w", err)
	}

	// Find the domain configuration to get provider
	for _, domain := range cfg.Domains {
		if domain.Name == recordSet.Domain {
			return recordSet.Domain, strings.ToLower(domain.Provider), nil
		}
	}

	return "", "", fmt.Errorf("domain %s not found in configuration", recordSet.Domain)
}

func testDomainConnectivity(ctx context.Context, domain *config.Domain, logger *log.Logger) error {
	client := porkbun.NewClient(domain.ApiKey, domain.SecretKey, cliConfig.Debug)

	// Test connectivity with ping
	pingResp, err := client.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping DNS provider: %w", err)
	}

	if pingResp.Status != "SUCCESS" {
		return fmt.Errorf("DNS provider ping failed: %s", pingResp.Status)
	}

	// Try to retrieve a small set of records to test API access
	recordsResp, err := client.RetrieveRecords(domain.Name)
	if err != nil {
		return fmt.Errorf("failed to test record retrieval: %w", err)
	}

	if recordsResp.Status != "SUCCESS" {
		return fmt.Errorf("record retrieval test failed: %s", recordsResp.Status)
	}

	logger.Printf("[DRY-RUN] Connectivity test successful for domain %s. Would export %d records.",
		domain.Name, len(recordsResp.Records))
	return nil
}

func getOutputFilename(domain, format string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-dns-backup-%s.%s", domain, timestamp, format)
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import DNS records from backup files to your DNS provider",
	Long: `Import DNS records from backup files previously created with the export command.

This command reads DNS records from backup files (JSON or TXT format) and imports them
to your DNS provider. Various conflict resolution strategies are available for handling
existing records.

The import process includes:
- Pre-import validation of all DNS records
- Conflict detection and resolution
- Automatic backup creation (optional)
- Detailed progress reporting
- Comprehensive error handling
- Dry-run support for testing

Conflict Resolution Strategies:
  skip    - Skip existing records, only import new ones (safest)
  replace - Replace existing records with backup versions (use with caution)
  merge   - Attempt to merge backup and existing records
  abort   - Stop import if any conflicts are detected

Examples:
  # Test import without making changes (recommended first)
  spf-flattener import --files backup.json --dry-run

  # Import only A records from backup file
  spf-flattener import --files backup.json --record-types A --dry-run

  # Import with backup creation and skip strategy
  spf-flattener import --files example.com-backup.json --strategy skip --backup-before

  # Import only specific record types (A and AAAA)
  spf-flattener import --files backup.json --record-types A,AAAA --strategy skip

  # Import multiple files with replace strategy
  spf-flattener import --files backup1.json,backup2.json --strategy replace --config config.yaml

  # Import with custom conflict handling
  spf-flattener import --files backup.json --strategy merge --backup-before --config config.yaml

Best Practices:
  1. Always use --dry-run first to test import operations
  2. Use --backup-before to create restore points
  3. Start with 'skip' strategy for safety
  4. Review import summary before making changes`,
	Run: func(cmd *cobra.Command, args []string) {
		populateConfigFromFlags(cmd)

		if len(importFiles) == 0 {
			cmd.PrintErrln("No backup files specified")
			os.Exit(1)
		}

		// Production mode overrides dry-run for actual operations
		if cliConfig.Production {
			cliConfig.DryRun = false
		}

		if cliConfig.DryRun {
			fmt.Println("DRY-RUN: Testing import operation without making changes.")
		}

		// Load configuration
		cfg, err := config.LoadConfig(cliConfig.ConfigPath)
		if err != nil {
			cmd.PrintErrf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Setup logger
		logger := log.Default()
		verbosePrintlnf("[VERBOSE] Starting import for %d files\n", len(importFiles))
		debugPrintlnf("[DEBUG] Import configuration: strategy=%s, record-types=%v, backup-before=%v\n",
			importStrategy, importRecordTypes, importBackup)

		// Group import tasks by provider for parallel processing
		ctx := context.Background()
		tasks := make([]importTask, 0, len(importFiles))

		// Extract domain and provider information from each backup file
		for _, file := range importFiles {
			domain, provider, err := extractDomainProviderFromBackup(file, cfg)
			if err != nil {
				logger.Printf("Failed to determine provider for file %s: %v", file, err)
				continue
			}

			tasks = append(tasks, importTask{
				Filename: file,
				Domain:   domain,
				Provider: provider,
			})
		}

		verbosePrintlnf("[VERBOSE] Processing %d import tasks\n", len(tasks))
		debugPrintlnf("[DEBUG] Task list: %+v\n", func() []string {
			var taskNames []string
			for _, task := range tasks {
				taskNames = append(taskNames, fmt.Sprintf("%s->%s", task.Filename, task.Domain))
			}
			return taskNames
		}())

		// Group tasks by provider
		providerTasks := make(map[string][]importTask)
		for _, task := range tasks {
			providerTasks[task.Provider] = append(providerTasks[task.Provider], task)
		}

		verbosePrintlnf("[VERBOSE] Grouped tasks into %d provider groups\n", len(providerTasks))
		debugPrintlnf("[DEBUG] Provider task distribution: %+v\n", func() map[string]int {
			counts := make(map[string]int)
			for provider, tasks := range providerTasks {
				counts[provider] = len(tasks)
			}
			return counts
		}())

		// Process each provider group in parallel
		var wg sync.WaitGroup
		resultsChan := make(chan importResult, len(importFiles))

		for provider, tasks := range providerTasks {
			wg.Add(1)
			go func(providerName string, providerTasks []importTask) {
				defer wg.Done()

				// Create rate limiter for this provider
				rateLimiter := rate.NewLimiter(rate.Limit(2.0), 1)

				// Process imports for this provider sequentially
				for _, task := range providerTasks {
					if cliConfig.Verbose {
						logger.Printf("Importing from file: %s (provider: %s)", task.Filename, providerName)
					}

					// Apply rate limiting before import
					if err := rateLimiter.Wait(ctx); err != nil {
						resultsChan <- importResult{
							Filename: task.Filename,
							Provider: providerName,
							Error:    fmt.Errorf("rate limiting failed: %w", err),
						}
						continue
					}

					err := importFromFile(ctx, task.Filename, cfg, logger)
					resultsChan <- importResult{
						Filename: task.Filename,
						Provider: providerName,
						Error:    err,
					}
				}
			}(provider, tasks)
		}

		// Wait for all provider groups to complete
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Process results
		successCount := 0
		failureCount := 0

		for result := range resultsChan {
			if result.Error != nil {
				failureCount++
				logger.Printf("Failed to import from file %s: %v", result.Filename, result.Error)
				if !cliConfig.Verbose {
					cmd.PrintErrf("Error importing from %s: %v\n", result.Filename, result.Error)
				}
			} else {
				successCount++
				if cliConfig.Verbose {
					logger.Printf("Successfully imported from file: %s", result.Filename)
				}
			}
		}

		// Print summary
		fmt.Printf("\nImport Summary:\n")
		fmt.Printf("  Success: %d files\n", successCount)
		fmt.Printf("  Failed:  %d files\n", failureCount)
		fmt.Printf("  Strategy: %s\n", importStrategy)
		if importBackup {
			fmt.Printf("  Backup:   Enabled before import\n")
		}

		if failureCount > 0 {
			os.Exit(1)
		}
	},
}

func importFromFile(ctx context.Context, filename string, cfg *config.Config, logger *log.Logger) error {
	// Read the backup file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Determine format from file extension
	formatHandler, err := backup.GetFormatHandlerFromFilename(filename)
	if err != nil {
		return fmt.Errorf("failed to determine format for file %s: %w", filename, err)
	}

	// Deserialize records
	recordSet, err := formatHandler.Deserialize(data)
	if err != nil {
		return fmt.Errorf("failed to deserialize backup file: %w", err)
	}

	// Find the domain configuration
	var domainConfig *config.Domain
	for _, domain := range cfg.Domains {
		if domain.Name == recordSet.Domain {
			domainConfig = &domain
			break
		}
	}

	if domainConfig == nil {
		return fmt.Errorf("domain %s not found in configuration", recordSet.Domain)
	}

	if cliConfig.Verbose {
		logger.Printf("Importing %d records for domain %s", len(recordSet.Records), recordSet.Domain)
	}

	// Create Porkbun client and adapter
	client := porkbun.NewClient(domainConfig.ApiKey, domainConfig.SecretKey, cliConfig.Debug)
	adapter := NewDNSClientAdapter(client)

	// Create backup manager
	backupConfig := backup.BackupManagerConfig{
		Client:            adapter,
		Logger:            logger,
		DryRun:            cliConfig.DryRun,
		RetryCount:        5,
		RetryDelay:        1 * time.Second,
		MaxRetryDelay:     30 * time.Second,
		BackoffMultiplier: 2.0,
		JitterEnabled:     true,
		RateLimiter:       rate.NewLimiter(rate.Limit(2.0), 1), // Match flatten command
	}

	backupManager := backup.NewBackupManager(backupConfig)

	// Setup import options
	importOptions := backup.ImportOptions{
		RecordTypes:      importRecordTypes,
		DryRun:           cliConfig.DryRun,
		ReplaceExisting:  importStrategy == "replace",
		ConflictStrategy: importStrategy,
		BackupBefore:     importBackup,
		Verbose:          cliConfig.Verbose,
	}

	// Optional: Create backup before import if requested
	if importBackup && !cliConfig.DryRun {
		if cliConfig.Verbose {
			logger.Printf("Creating backup before import...")
		}

		backupFile := fmt.Sprintf("%s-pre-import-backup-%s.json",
			recordSet.Domain, time.Now().Format("20060102-150405"))

		backupRecordSet, err := backupManager.ExportRecords(ctx, recordSet.Domain, backup.ExportOptions{})
		if err != nil {
			logger.Printf("Warning: Failed to create pre-import backup: %v", err)
		} else {
			jsonHandler, _ := backup.GetFormatHandler("json")
			backupData, _ := jsonHandler.Serialize(backupRecordSet)
			os.WriteFile(backupFile, backupData, 0644)
			logger.Printf("Pre-import backup saved to: %s", backupFile)
		}
	}

	// Import records
	result, err := backupManager.ImportRecords(ctx, recordSet, importOptions)
	if err != nil {
		return fmt.Errorf("import operation failed: %w", err)
	}

	// Print results
	fmt.Printf("\nImport Results for %s:\n", recordSet.Domain)
	fmt.Printf("  Total Records: %d\n", result.TotalRecords)
	fmt.Printf("  Created: %d\n", result.Created)
	fmt.Printf("  Updated: %d\n", result.Updated)
	fmt.Printf("  Skipped: %d\n", result.Skipped)
	fmt.Printf("  Failed: %d\n", result.Failed)
	fmt.Printf("  Duration: %v\n", result.Duration)

	if len(result.Conflicts) > 0 {
		fmt.Printf("  Conflicts: %d\n", len(result.Conflicts))
		if cliConfig.Verbose {
			for _, conflict := range result.Conflicts {
				fmt.Printf("    - %s: %s\n", conflict.ConflictType, conflict.Resolution)
			}
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("  Errors: %d\n", len(result.Errors))
		if cliConfig.Verbose {
			for _, err := range result.Errors {
				fmt.Printf("    - %v\n", err)
			}
		}
	}

	return nil
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Output format: json (machine-readable) or txt (human-readable)")
	exportCmd.Flags().StringVarP(&exportOutputDir, "output-dir", "o", "", "Output directory for backup files (default: current directory)")
	exportCmd.Flags().StringSliceVarP(&exportDomains, "domains", "d", []string{}, "Specific domains to export (comma-separated, default: all configured domains)")
	exportCmd.Flags().StringSliceVarP(&exportRecordTypes, "record-types", "t", []string{}, "Specific DNS record types to export (comma-separated, default: all types)")
	exportCmd.Flags().Bool("dry-run", true, "Test connectivity and preview export without creating files (default: true for safety)")
	exportCmd.Flags().Bool("production", false, "Enable production mode to create actual backup files (default: false)")

	importCmd.Flags().StringSliceVarP(&importFiles, "files", "f", []string{}, "Backup files to import (JSON or TXT format, comma-separated, required)")
	importCmd.Flags().StringVarP(&importStrategy, "strategy", "s", "skip", "Conflict resolution strategy: skip, replace, merge, or abort")
	importCmd.Flags().StringSliceVarP(&importRecordTypes, "record-types", "t", []string{}, "Specific DNS record types to import (comma-separated, default: all types)")
	importCmd.Flags().BoolVar(&importBackup, "backup-before", false, "Create backup of current records before importing")
	importCmd.Flags().Bool("dry-run", true, "Test import operation without making any changes to DNS records (default: true for safety)")
	importCmd.Flags().Bool("production", false, "Enable production mode to make actual changes to DNS records (default: false)")
}
