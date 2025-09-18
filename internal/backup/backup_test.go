package backup

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDNSClient implements the DNSAPIClient interface for testing
type MockDNSClient struct {
	mock.Mock
}

func (m *MockDNSClient) Ping() (*PingResponse, error) {
	args := m.Called()
	return args.Get(0).(*PingResponse), args.Error(1)
}

func (m *MockDNSClient) RetrieveRecords(domain string) (*RetrieveRecordsResponse, error) {
	args := m.Called(domain)
	return args.Get(0).(*RetrieveRecordsResponse), args.Error(1)
}

func (m *MockDNSClient) UpdateRecord(domain, recordID, content string) (*UpdateRecordResponse, error) {
	args := m.Called(domain, recordID, content)
	return args.Get(0).(*UpdateRecordResponse), args.Error(1)
}

func (m *MockDNSClient) UpdateRecordWithDetails(domain, recordID, name, recordType, content, ttl, prio, notes string) (*UpdateRecordResponse, error) {
	args := m.Called(domain, recordID, name, recordType, content, ttl, prio, notes)
	return args.Get(0).(*UpdateRecordResponse), args.Error(1)
}

func (m *MockDNSClient) CreateRecord(domain, name, recordType, content string, ttl int) (*CreateRecordResponse, error) {
	args := m.Called(domain, name, recordType, content, ttl)
	return args.Get(0).(*CreateRecordResponse), args.Error(1)
}

func (m *MockDNSClient) CreateRecordWithOptions(domain, name, recordType, content string, ttl int, prio string, notes string) (*CreateRecordResponse, error) {
	args := m.Called(domain, name, recordType, content, ttl, prio, notes)
	return args.Get(0).(*CreateRecordResponse), args.Error(1)
}

func (m *MockDNSClient) DeleteRecord(domain, recordID string) (*DeleteRecordResponse, error) {
	args := m.Called(domain, recordID)
	return args.Get(0).(*DeleteRecordResponse), args.Error(1)
}

func (m *MockDNSClient) DeleteRecordByNameType(domain, recordType, subdomain string) (*DeleteRecordResponse, error) {
	args := m.Called(domain, recordType, subdomain)
	return args.Get(0).(*DeleteRecordResponse), args.Error(1)
}

func (m *MockDNSClient) RetrieveAllRecords(domain string) ([]BackupDNSRecord, error) {
	args := m.Called(domain)
	return args.Get(0).([]BackupDNSRecord), args.Error(1)
}

func (m *MockDNSClient) BulkCreateRecords(domain string, records []BackupDNSRecord) error {
	args := m.Called(domain, records)
	return args.Error(0)
}

func (m *MockDNSClient) BulkUpdateRecords(domain string, records []BackupDNSRecord) error {
	args := m.Called(domain, records)
	return args.Error(0)
}

func (m *MockDNSClient) BulkDeleteRecords(domain string, recordIDs []string) error {
	args := m.Called(domain, recordIDs)
	return args.Error(0)
}

func (m *MockDNSClient) Attribution() string {
	args := m.Called()
	return args.String(0)
}

func TestNewBackupManager(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	config := BackupManagerConfig{
		Client:     mockClient,
		Logger:     logger,
		DryRun:     false,
		RetryCount: 3,
		RetryDelay: 1 * time.Second,
	}

	manager := NewBackupManager(config)

	assert.NotNil(t, manager)
	assert.Equal(t, mockClient, manager.client)
	assert.Equal(t, logger, manager.logger)
	assert.False(t, manager.dryRun)
	assert.Equal(t, 3, manager.retryCount)
	assert.Equal(t, 1*time.Second, manager.retryDelay)
}

func TestNewBackupManager_DefaultValues(t *testing.T) {
	mockClient := new(MockDNSClient)

	config := BackupManagerConfig{
		Client: mockClient,
		// Leave other fields empty to test defaults
	}

	manager := NewBackupManager(config)

	assert.NotNil(t, manager)
	assert.Equal(t, 5, manager.retryCount)             // Default retry count
	assert.Equal(t, 1*time.Second, manager.retryDelay) // Default retry delay
	assert.NotNil(t, manager.logger)                   // Default logger
}

func TestBackupManager_ExportRecords_Success(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	// Setup mock expectations
	sampleRecords := []BackupDNSRecord{
		{
			ID:      "rec1",
			Name:    "@",
			Type:    "A",
			Content: "192.168.1.1",
			TTL:     3600,
		},
		{
			ID:      "rec2",
			Name:    "www.example.com",
			Type:    "CNAME",
			Content: "example.com",
			TTL:     3600,
		},
	}

	mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
	mockClient.On("Attribution").Return("Test Provider")
	mockClient.On("RetrieveAllRecords", "example.com").Return(sampleRecords, nil)

	config := BackupManagerConfig{
		Client:     mockClient,
		Logger:     logger,
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	manager := NewBackupManager(config)
	ctx := context.Background()

	result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "example.com", result.Domain)
	assert.Equal(t, 2, len(result.Records))
	assert.Equal(t, "A", result.Records[0].Type)
	assert.Equal(t, "CNAME", result.Records[1].Type)

	mockClient.AssertExpectations(t)
}

func TestBackupManager_ExportRecords_APIError(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	// Setup mock to return error on first call, success on second
	mockClient.On("Attribution").Return("Test Provider").Maybe()
	mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
	mockClient.On("RetrieveAllRecords", "example.com").Return([]BackupDNSRecord{}, assert.AnError)

	config := BackupManagerConfig{
		Client:     mockClient,
		Logger:     logger,
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	manager := NewBackupManager(config)
	ctx := context.Background()

	result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve records after 1 attempts")

	mockClient.AssertExpectations(t)
}

func TestBackupManager_ExportRecords_ConnectivityError(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	// Setup mock to return ping error
	mockClient.On("Attribution").Return("Test Provider").Maybe()
	mockClient.On("Ping").Return(&PingResponse{}, assert.AnError)

	config := BackupManagerConfig{
		Client:     mockClient,
		Logger:     logger,
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	manager := NewBackupManager(config)
	ctx := context.Background()

	result, err := manager.ExportRecords(ctx, "example.com", ExportOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to connect to DNS provider")

	mockClient.AssertExpectations(t)
}

func TestBackupManager_ImportRecords_Success(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	// Setup mock expectations
	mockClient.On("Attribution").Return("Test Provider").Maybe()
	mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil).Maybe()
	mockClient.On("RetrieveAllRecords", "example.com").Return([]BackupDNSRecord{}, nil)
	mockClient.On("BulkCreateRecords", "example.com", mock.AnythingOfType("[]backup.BackupDNSRecord")).Return(nil)

	config := BackupManagerConfig{
		Client:     mockClient,
		Logger:     logger,
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	manager := NewBackupManager(config)
	ctx := context.Background()

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

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "example.com", result.Domain)
	assert.Equal(t, 1, result.TotalRecords)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Failed)

	mockClient.AssertExpectations(t)
}

func TestBackupManager_ImportRecords_DryRun(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	config := BackupManagerConfig{
		Client:     mockClient,
		Logger:     logger,
		DryRun:     true, // Enable dry run
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	manager := NewBackupManager(config)
	ctx := context.Background()

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

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Created) // No records should be created in dry run

	// No API calls should be made in dry run mode
	mockClient.AssertNotCalled(t, "BulkCreateRecords")
}

func TestBackupManager_ImportRecords_ValidationError(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	config := BackupManagerConfig{
		Client:     mockClient,
		Logger:     logger,
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	manager := NewBackupManager(config)
	ctx := context.Background()

	// Create invalid record set (missing required fields)
	recordSet := &DNSRecordSet{
		Domain:   "example.com",
		Provider: "test",
		Version:  "1.0",
		Records: []DNSRecord{
			{
				// Missing required fields
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
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "record validation failed")

	// No API calls should be made for invalid records
	mockClient.AssertNotCalled(t, "BulkCreateRecords")
}

func TestBackupManager_ImportRecords_EmptyRecordSet(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	config := BackupManagerConfig{
		Client:     mockClient,
		Logger:     logger,
		RetryCount: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	manager := NewBackupManager(config)
	ctx := context.Background()

	// Create empty record set
	recordSet := &DNSRecordSet{
		Domain:   "example.com",
		Provider: "test",
		Version:  "1.0",
		Records:  []DNSRecord{}, // Empty records
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
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no records to import")

	// No API calls should be made for empty record sets
	mockClient.AssertNotCalled(t, "BulkCreateRecords")
}

func TestBackupManager_ImportRecords_CancelledContext(t *testing.T) {
	mockClient := new(MockDNSClient)
	logger := log.Default()

	// Setup mock to delay execution
	mockClient.On("Ping").Return(&PingResponse{Status: "SUCCESS", YourIP: "1.2.3.4"}, nil)
	mockClient.On("Attribution").Return("Test Provider")
	mockClient.On("BulkCreateRecords", "example.com", mock.AnythingOfType("[]backup.BackupDNSRecord")).Return(nil).After(100 * time.Millisecond)

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
	assert.Equal(t, 0, result.Created) // No records should be created
}

func TestConvertFromBackupRecord(t *testing.T) {
	backupRecord := BackupDNSRecord{
		ID:       "rec1",
		Name:     "@",
		Type:     "A",
		Content:  "192.168.1.1",
		TTL:      3600,
		Priority: 10,
		Notes:    "Test record",
	}

	result := convertFromBackupRecord(backupRecord)

	assert.Equal(t, backupRecord.ID, result.ID)
	assert.Equal(t, backupRecord.Name, result.Name)
	assert.Equal(t, backupRecord.Type, result.Type)
	assert.Equal(t, backupRecord.Content, result.Content)
	assert.Equal(t, backupRecord.TTL, result.TTL)
	assert.Equal(t, backupRecord.Priority, result.Priority)
	assert.Equal(t, backupRecord.Notes, result.Notes)
}

func TestConvertToBackupRecord(t *testing.T) {
	dnsRecord := DNSRecord{
		ID:        "rec1",
		Name:      "@",
		Type:      "A",
		Content:   "192.168.1.1",
		TTL:       3600,
		Priority:  10,
		Notes:     "Test record",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result := convertToBackupRecord(dnsRecord)

	assert.Equal(t, dnsRecord.ID, result.ID)
	assert.Equal(t, dnsRecord.Name, result.Name)
	assert.Equal(t, dnsRecord.Type, result.Type)
	assert.Equal(t, dnsRecord.Content, result.Content)
	assert.Equal(t, dnsRecord.TTL, result.TTL)
	assert.Equal(t, dnsRecord.Priority, result.Priority)
	assert.Equal(t, dnsRecord.Notes, result.Notes)
}
