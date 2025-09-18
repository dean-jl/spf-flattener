package backup

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestEdgeCases tests various edge cases in the backup system
func TestEdgeCases(t *testing.T) {
	t.Run("NilRecordSet_Export", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		// Setup mock to return ping error
		mockClient.On("Attribution").Return("Test Provider").Maybe()
		mockClient.On("Ping").Return(&PingResponse{}, errors.New("connection failed"))

		config := BackupManagerConfig{
			Client: mockClient,
			Logger: logger,
		}

		manager := NewBackupManager(config)
		ctx := context.Background()

		result, err := manager.ExportRecords(ctx, "", ExportOptions{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to connect to DNS provider")

		mockClient.AssertExpectations(t)
	})

	t.Run("EmptyRecords_Export", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		mockClient.On("Attribution").Return("Test Provider").Maybe()
		mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
		mockClient.On("RetrieveAllRecords", "example.com").Return([]BackupDNSRecord{}, nil)

		config := BackupManagerConfig{
			Client: mockClient,
			Logger: logger,
		}

		manager := NewBackupManager(config)
		ctx := context.Background()

		result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "example.com", result.Domain)
		assert.Equal(t, 0, len(result.Records))

		mockClient.AssertExpectations(t)
	})

	t.Run("VeryLargeRecordSet_Import", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		// Create a very large record set
		records := make([]DNSRecord, 100)
		for i := 0; i < 100; i++ {
			records[i] = DNSRecord{
				ID:      fmt.Sprintf("rec%d", i),
				Name:    fmt.Sprintf("test%d.example.com", i),
				Type:    "A",
				Content: fmt.Sprintf("192.168.1.%d", i%255),
				TTL:     3600,
			}
		}

		recordSet := &DNSRecordSet{
			Domain:   "example.com",
			Provider: "test",
			Version:  "1.0",
			Records:  records,
		}

		config := BackupManagerConfig{
			Client: mockClient,
			Logger: logger,
			DryRun: true, // Use dry run to avoid actual API calls
		}

		manager := NewBackupManager(config)
		ctx := context.Background()

		options := ImportOptions{
			DryRun:           false,
			ReplaceExisting:  false,
			ConflictStrategy: "skip",
			BackupBefore:     false,
			Verbose:          false,
		}

		result, err := manager.ImportRecords(ctx, recordSet, options)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 100, result.TotalRecords)
		assert.Equal(t, 100, result.Skipped) // All skipped due to dry run
		assert.Equal(t, 0, result.Failed)
	})

	t.Run("SpecialCharactersInContent", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		// Records with special characters in content (valid domain names)
		records := []BackupDNSRecord{
			{
				ID:       "rec1",
				Name:     "@",
				Type:     "TXT",
				Content:  "v=spf1 include:_spf.google.com ~all",
				TTL:      3600,
				Priority: 0,
				Notes:    "SPF with special chars",
			},
			{
				ID:       "rec2",
				Name:     "dmarc.example.com",
				Type:     "TXT",
				Content:  "v=DMARC1; p=reject; rua=mailto:admin@example.com",
				TTL:      3600,
				Priority: 0,
				Notes:    "DMARC record",
			},
		}

		mockClient.On("Attribution").Return("Test Provider").Maybe()
		mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
		mockClient.On("RetrieveAllRecords", "example.com").Return(records, nil)

		config := BackupManagerConfig{
			Client: mockClient,
			Logger: logger,
		}

		manager := NewBackupManager(config)
		ctx := context.Background()

		result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, len(result.Records))
		assert.Contains(t, result.Records[0].Content, "v=spf1")
		assert.Contains(t, result.Records[1].Content, "v=DMARC1")

		mockClient.AssertExpectations(t)
	})
}

// TestRetryLogic tests the retry mechanism for API failures
func TestRetryLogic(t *testing.T) {
	t.Run("RetryOnFailureThenSuccess", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		// Setup mock to return error on first call, success on second
		mockClient.On("Attribution").Return("Test Provider").Maybe()
		mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
		mockClient.On("RetrieveAllRecords", "example.com").Return([]BackupDNSRecord{}, errors.New("temporary failure")).Once()
		mockClient.On("RetrieveAllRecords", "example.com").Return([]BackupDNSRecord{
			{
				ID:       "rec1",
				Name:     "@",
				Type:     "A",
				Content:  "192.168.1.1",
				TTL:      3600,
				Priority: 0,
			},
		}, nil).Once()

		config := BackupManagerConfig{
			Client:     mockClient,
			Logger:     logger,
			RetryCount: 3,
			RetryDelay: 1 * time.Millisecond,
		}

		manager := NewBackupManager(config)
		ctx := context.Background()

		result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, len(result.Records))

		mockClient.AssertExpectations(t)
	})

	t.Run("ExhaustRetries", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		mockClient.On("Attribution").Return("Test Provider").Maybe()
		mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
		mockClient.On("RetrieveAllRecords", "example.com").Return([]BackupDNSRecord{}, errors.New("persistent error"))

		config := BackupManagerConfig{
			Client:     mockClient,
			Logger:     logger,
			RetryCount: 3,
			RetryDelay: 1 * time.Millisecond,
		}

		manager := NewBackupManager(config)
		ctx := context.Background()

		result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to retrieve records after 3 attempts")

		mockClient.AssertExpectations(t)
	})
}

// TestConcurrentAccess tests concurrent access to the backup manager
func TestConcurrentAccess(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	// Setup mock to handle concurrent calls
	mockClient.On("Attribution").Return("Test Provider").Maybe()
	mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
	mockClient.On("RetrieveAllRecords", mock.AnythingOfType("string")).Return([]BackupDNSRecord{
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
		Logger:     logger,
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	manager := NewBackupManager(config)

	// Test concurrent exports
	numGoroutines := 10
	results := make(chan *DNSRecordSet, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(domain string) {
			ctx := context.Background()
			result, err := manager.ExportRecords(ctx, domain, ExportOptions{})
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(testDomain(i))
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			assert.NotNil(t, result)
			assert.Equal(t, 1, len(result.Records))
			successCount++
		case err := <-errors:
			assert.Error(t, err)
			errorCount++
		}
	}

	assert.Equal(t, numGoroutines, successCount)
	assert.Equal(t, 0, errorCount)

	mockClient.AssertExpectations(t)
}

// TestInvalidDataHandling tests handling of invalid or malformed data
func TestInvalidDataHandling(t *testing.T) {
	t.Run("InvalidTTLValues", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		// Records with invalid TTL values - should be rejected
		records := []BackupDNSRecord{
			{
				ID:       "rec1",
				Name:     "@",
				Type:     "A",
				Content:  "192.168.1.1",
				TTL:      -1, // Invalid negative TTL
				Priority: 0,
			},
			{
				ID:       "rec2",
				Name:     "www.example.com",
				Type:     "A",
				Content:  "192.168.1.2",
				TTL:      0, // Minimum valid TTL
				Priority: 0,
			},
		}

		mockClient.On("Attribution").Return("Test Provider").Maybe()
		mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
		mockClient.On("RetrieveAllRecords", "example.com").Return(records, nil)

		config := BackupManagerConfig{
			Client: mockClient,
			Logger: logger,
		}

		manager := NewBackupManager(config)
		ctx := context.Background()

		result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})
		// This should fail due to validation errors
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "validation failed")

		mockClient.AssertExpectations(t)
	})

	t.Run("EmptyContentFields", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		// Records with empty content - should be rejected
		records := []BackupDNSRecord{
			{
				ID:       "rec1",
				Name:     "@",
				Type:     "A",
				Content:  "", // Empty content
				TTL:      3600,
				Priority: 0,
			},
		}

		mockClient.On("Attribution").Return("Test Provider").Maybe()
		mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
		mockClient.On("RetrieveAllRecords", "example.com").Return(records, nil)

		config := BackupManagerConfig{
			Client: mockClient,
			Logger: logger,
		}

		manager := NewBackupManager(config)
		ctx := context.Background()

		result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})
		// This should fail due to validation errors
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "validation failed")

		mockClient.AssertExpectations(t)
	})
}

// TestConfigurationEdgeCases tests edge cases in backup manager configuration
func TestConfigurationEdgeCases(t *testing.T) {
	t.Run("ZeroRetryCount", func(t *testing.T) {
		mockClient := new(MockDNSClient)

		config := BackupManagerConfig{
			Client:     mockClient,
			RetryCount: 0, // Should default to 3
			RetryDelay: 0, // Should default to 1 second
		}

		manager := NewBackupManager(config)
		assert.NotNil(t, manager)
		assert.Equal(t, 5, manager.retryCount)
		assert.Equal(t, 1*time.Second, manager.retryDelay)
	})

	t.Run("NilLogger", func(t *testing.T) {
		mockClient := new(MockDNSClient)

		config := BackupManagerConfig{
			Client: mockClient,
			Logger: nil, // Should default to log.Default()
		}

		manager := NewBackupManager(config)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.logger)
	})

	t.Run("NegativeRetryDelay", func(t *testing.T) {
		mockClient := new(MockDNSClient)

		config := BackupManagerConfig{
			Client:     mockClient,
			RetryCount: 3,
			RetryDelay: -1 * time.Second, // Should default to 1 second
		}

		manager := NewBackupManager(config)
		assert.NotNil(t, manager)
		assert.Equal(t, 1*time.Second, manager.retryDelay)
	})
}

// TestCancellationHandling tests context cancellation handling
func TestCancellationHandling(t *testing.T) {
	t.Run("ExportCancellation", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		// Setup mock to return error on cancelled context
		mockClient.On("Attribution").Return("Test Provider").Maybe()
		mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil).Maybe()
		mockClient.On("RetrieveAllRecords", "example.com").Return([]BackupDNSRecord{}, context.Canceled).Maybe()

		// Create cancelled context before any operations
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		config := BackupManagerConfig{
			Client:     mockClient,
			Logger:     logger,
			RetryCount: 1,
			RetryDelay: 1 * time.Millisecond,
		}

		manager := NewBackupManager(config)

		result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		assert.Nil(t, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("ImportCancellation", func(t *testing.T) {
		mockClient := new(MockDNSClient)
		logger := log.Default()

		config := BackupManagerConfig{
			Client:     mockClient,
			Logger:     logger,
			RetryCount: 1,
			RetryDelay: 1 * time.Millisecond,
		}

		manager := NewBackupManager(config)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		recordSet := &DNSRecordSet{
			Domain:   "example.com",
			Provider: "test",
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

		options := ImportOptions{
			DryRun:           false,
			ReplaceExisting:  false,
			ConflictStrategy: "skip",
			BackupBefore:     false,
			Verbose:          false,
		}

		result, err := manager.ImportRecords(ctx, recordSet, options)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.Created)
	})
}

// Helper function to generate test domain names
func testDomain(i int) string {
	return "test" + string(rune(i%26+'a')) + ".example.com"
}
