package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestExportImportWorkflowIntegration tests the complete export/import workflow
// using in-memory operations without external dependencies
func TestExportImportWorkflowIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create sample DNS record set for export
	exportRecordSet := &DNSRecordSet{
		Domain:   "example.com",
		Provider: "porkbun",
		Version:  "1.0",
		Records: []DNSRecord{
			{
				ID:        "rec1",
				Name:      "@",
				Type:      "A",
				Content:   "192.168.1.1",
				TTL:       3600,
				Priority:  0,
				Notes:     "Main A record",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        "rec2",
				Name:      "www.example.com",
				Type:      "CNAME",
				Content:   "example.com",
				TTL:       3600,
				Priority:  0,
				Notes:     "WWW alias",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        "rec3",
				Name:      "@",
				Type:      "TXT",
				Content:   "v=spf1 include:_spf.google.com ~all",
				TTL:       3600,
				Priority:  0,
				Notes:     "SPF record",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	// Test JSON export/import workflow
	t.Run("JSON_Format", func(t *testing.T) {
		testExportImportFormat(t, tempDir, exportRecordSet, "json")
	})

	// Test TXT export/import workflow
	t.Run("TXT_Format", func(t *testing.T) {
		testExportImportFormat(t, tempDir, exportRecordSet, "txt")
	})
}

// testExportImportFormat tests export/import workflow for a specific format
func testExportImportFormat(t *testing.T, tempDir string, recordSet *DNSRecordSet, format string) {
	// Get format handler
	handler, err := GetFormatHandler(format)
	assert.NoError(t, err)

	// Export records
	exportData, err := handler.Serialize(recordSet)
	assert.NoError(t, err)
	assert.NotNil(t, exportData)

	// Write to file
	filename := filepath.Join(tempDir, "test-backup."+format)
	err = os.WriteFile(filename, exportData, 0644)
	assert.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(filename)
	assert.NoError(t, err)

	// Read file back
	importData, err := os.ReadFile(filename)
	assert.NoError(t, err)

	// Import records
	importedRecordSet, err := handler.Deserialize(importData)
	assert.NoError(t, err)
	assert.NotNil(t, importedRecordSet)

	// Verify imported data matches original
	assert.Equal(t, recordSet.Domain, importedRecordSet.Domain)
	assert.Equal(t, recordSet.Provider, importedRecordSet.Provider)
	assert.Equal(t, recordSet.Version, importedRecordSet.Version)
	assert.Equal(t, len(recordSet.Records), len(importedRecordSet.Records))

	// Verify each record
	for i, expectedRecord := range recordSet.Records {
		actualRecord := importedRecordSet.Records[i]
		assert.Equal(t, expectedRecord.ID, actualRecord.ID)
		assert.Equal(t, expectedRecord.Name, actualRecord.Name)
		assert.Equal(t, expectedRecord.Type, actualRecord.Type)
		assert.Equal(t, expectedRecord.Content, actualRecord.Content)
		assert.Equal(t, expectedRecord.TTL, actualRecord.TTL)
		assert.Equal(t, expectedRecord.Priority, actualRecord.Priority)
		assert.Equal(t, expectedRecord.Notes, actualRecord.Notes)
	}
}

// TestFormatHandlerIntegration tests format handler detection and conversion
func TestFormatHandlerIntegration(t *testing.T) {
	// Test getting format handlers
	jsonHandler, err := GetFormatHandler("json")
	assert.NoError(t, err)
	assert.NotNil(t, jsonHandler)

	txtHandler, err := GetFormatHandler("txt")
	assert.NoError(t, err)
	assert.NotNil(t, txtHandler)

	// Test getting format handlers from filename
	jsonHandlerFromFilename, err := GetFormatHandlerFromFilename("backup.json")
	assert.NoError(t, err)
	assert.NotNil(t, jsonHandlerFromFilename)

	txtHandlerFromFilename, err := GetFormatHandlerFromFilename("backup.txt")
	assert.NoError(t, err)
	assert.NotNil(t, txtHandlerFromFilename)

	// Test invalid format
	invalidHandler, err := GetFormatHandler("invalid")
	assert.Error(t, err)
	assert.Nil(t, invalidHandler)

	// Test invalid filename
	invalidHandlerFromFilename, err := GetFormatHandlerFromFilename("backup.invalid")
	assert.Error(t, err)
	assert.Nil(t, invalidHandlerFromFilename)
}

// TestValidatorIntegration tests validation of complete record sets
func TestValidatorIntegration(t *testing.T) {
	validator := NewValidator()

	// Test valid record set
	validRecordSet := &DNSRecordSet{
		Domain:   "example.com",
		Provider: "porkbun",
		Version:  "1.0",
		Records: []DNSRecord{
			{
				ID:      "rec1",
				Name:    "@",
				Type:    "A",
				Content: "192.168.1.1",
				TTL:     3600,
			},
		},
	}

	result := validator.ValidateRecordSet(validRecordSet)
	assert.True(t, result.IsValid)
	assert.Len(t, result.Warnings, 0)
	assert.Len(t, result.Errors, 0)

	// Test invalid record set
	invalidRecordSet := &DNSRecordSet{
		Domain:   "example.com",
		Provider: "porkbun",
		Version:  "1.0",
		Records: []DNSRecord{
			{
				ID:      "rec1",
				Name:    "@",
				Type:    "INVALID_TYPE",
				Content: "192.168.1.1",
				TTL:     3600,
			},
		},
	}

	result = validator.ValidateRecordSet(invalidRecordSet)
	assert.False(t, result.IsValid)
	assert.Len(t, result.Warnings, 0)
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0], "Invalid record type")

	// Test empty record set
	emptyRecordSet := &DNSRecordSet{
		Domain:   "example.com",
		Provider: "porkbun",
		Version:  "1.0",
		Records:  []DNSRecord{},
	}

	result = validator.ValidateRecordSet(emptyRecordSet)
	assert.False(t, result.IsValid)
	assert.Len(t, result.Warnings, 0)
	assert.Greater(t, len(result.Errors), 0)
	assert.Contains(t, result.Errors[0], "Record set must contain at least one record")
}

// TestBackupManagerIntegration tests backup manager with mock client
func TestBackupManagerIntegration(t *testing.T) {
	// Create mock client
	mockClient := new(MockDNSClient)

	// Setup mock expectations for export phase
	exportRecords := []BackupDNSRecord{
		{
			ID:       "rec1",
			Name:     "@",
			Type:     "A",
			Content:  "192.168.1.1",
			TTL:      3600,
			Priority: 0,
			Notes:    "Main A record",
		},
		{
			ID:       "rec2",
			Name:     "www.example.com",
			Type:     "CNAME",
			Content:  "example.com",
			TTL:      3600,
			Priority: 0,
			Notes:    "WWW alias",
		},
	}

	mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
	// First call for export
	mockClient.On("RetrieveAllRecords", "example.com").Return(exportRecords, nil).Once()

	// Create backup manager
	config := BackupManagerConfig{
		Client:     mockClient,
		DryRun:     false,
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	backupManager := NewBackupManager(config)
	assert.NotNil(t, backupManager)

	// Test export
	ctx := context.Background()
	exportedRecordSet, err := backupManager.ExportRecords(ctx, "example.com", ExportOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, exportedRecordSet)
	assert.Equal(t, "example.com", exportedRecordSet.Domain)
	assert.Equal(t, 2, len(exportedRecordSet.Records))

	// Test import with exported records
	// Import calls RetrieveAllRecords once per record for conflict checking
	existingRecords := []BackupDNSRecord{
		{
			ID:      "existing1",
			Name:    "mail",
			Type:    "A",
			Content: "10.0.0.1",
			TTL:     3600,
		},
	}
	mockClient.On("RetrieveAllRecords", "example.com").Return(existingRecords, nil).Times(2)                                                // Called once per imported record
	mockClient.On("BulkCreateRecords", mock.AnythingOfType("string"), mock.AnythingOfType("[]backup.BackupDNSRecord")).Return(nil).Times(2) // Called once per imported record

	importOptions := ImportOptions{
		DryRun:           false,
		ReplaceExisting:  false,
		ConflictStrategy: "skip",
		BackupBefore:     false,
		Verbose:          false,
	}

	importResult, err := backupManager.ImportRecords(ctx, exportedRecordSet, importOptions)
	assert.NoError(t, err)
	assert.NotNil(t, importResult)
	assert.Equal(t, 2, importResult.TotalRecords)
	assert.Equal(t, 2, importResult.Created)
	assert.Equal(t, 0, importResult.Failed)

	// Verify mock expectations
	mockClient.AssertExpectations(t)
}

// TestErrorHandlingIntegration tests error handling across the workflow
func TestErrorHandlingIntegration(t *testing.T) {
	// Test validation error handling
	t.Run("ValidationError", func(t *testing.T) {
		invalidRecordSet := &DNSRecordSet{
			Domain:   "example.com",
			Provider: "porkbun",
			Version:  "1.0",
			Records: []DNSRecord{
				{
					ID:      "rec1",
					Name:    "@",
					Type:    "INVALID_TYPE",
					Content: "192.168.1.1",
					TTL:     3600,
				},
			},
		}

		validator := NewValidator()
		result := validator.ValidateRecordSet(invalidRecordSet)
		assert.Len(t, result.Warnings, 0)
		assert.Greater(t, len(result.Errors), 0)
	})

	// Test format handler error handling
	t.Run("FormatError", func(t *testing.T) {
		invalidJSON := []byte("invalid json content")

		jsonHandler, err := GetFormatHandler("json")
		assert.NoError(t, err)

		_, err = jsonHandler.Deserialize(invalidJSON)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})

	// Test file error handling
	t.Run("FileError", func(t *testing.T) {
		// Try to read non-existent file
		_, err := os.ReadFile("nonexistent-file.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})
}

// TestConcurrentOperations tests concurrent export/import operations
func TestConcurrentOperations(t *testing.T) {
	// Create multiple backup managers for concurrent testing
	var managers []*BackupManager
	for i := 0; i < 3; i++ {
		mockClient := new(MockDNSClient)
		mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
		mockClient.On("RetrieveAllRecords", "example.com").Return([]BackupDNSRecord{
			{
				ID:       "rec1",
				Name:     "@",
				Type:     "A",
				Content:  "192.168.1.1",
				TTL:      3600,
				Priority: 0,
			},
		}, nil)

		config := BackupManagerConfig{
			Client:     mockClient,
			DryRun:     false,
			RetryCount: 1,
			RetryDelay: 1 * time.Millisecond,
		}
		managers = append(managers, NewBackupManager(config))
	}

	// Test concurrent exports
	results := make(chan *DNSRecordSet, len(managers))
	errors := make(chan error, len(managers))

	for i, manager := range managers {
		go func(idx int, mgr *BackupManager) {
			ctx := context.Background()
			recordSet, err := mgr.ExportRecords(ctx, "example.com", ExportOptions{})
			if err != nil {
				errors <- err
			} else {
				results <- recordSet
			}
		}(i, manager)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < len(managers); i++ {
		select {
		case result := <-results:
			assert.NotNil(t, result)
			assert.Equal(t, "example.com", result.Domain)
			successCount++
		case err := <-errors:
			assert.Error(t, err)
			errorCount++
		}
	}

	assert.Equal(t, len(managers), successCount)
	assert.Equal(t, 0, errorCount)
}
