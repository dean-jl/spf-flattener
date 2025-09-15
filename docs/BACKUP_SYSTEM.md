# DNS Backup System Documentation

## Overview

The DNS Backup System provides comprehensive backup and restore functionality for DNS records. This system allows users to export DNS records from their DNS provider to a structured format, and import them back when needed. The system supports multiple formats, comprehensive validation, and safety features like dry-run mode.

## Architecture

### Core Components

**Backup Manager (`internal/backup/backup.go`)**
- Central component managing backup and restore operations
- Handles validation, error handling, and retry logic with `golang.org/x/time/rate`
- Implements safety features including dry-run mode
- Supports concurrent operations with proper context handling
- Uses standardized rate limiting (2 req/sec, burst 1) across all providers

**Format Handlers (`internal/backup/formats.go`)**
- Pluggable format system supporting JSON and TXT formats
- Extensible architecture for adding new formats
- Consistent interface for serialization and deserialization

**Validation System (`internal/backup/validation.go`)**
- Comprehensive DNS record validation
- Supports all major DNS record types
- Provides both errors and warnings for best practices
- Validates individual records and complete record sets

**CLI Commands (`cmd/spf-flattener/backup.go`)**
- `export` command for backing up DNS records with provider-aware parallel processing
- `import` command for restoring DNS records with intelligent provider grouping
- Comprehensive flag support for all operations
- Optimized performance for multi-provider configurations

### Data Flow

```
Export Flow:
CLI Command → Backup Manager → DNS Provider → Record Validation → Format Handler → File

Import Flow:
CLI Command → Format Handler → Record Validation → Backup Manager → DNS Provider → Result
```

## Features

### Export Functionality

- **Complete Record Backup**: Exports all DNS records for a domain
- **Multiple Formats**: Supports JSON (structured) and TXT (human-readable) formats
- **Provider-Aware Parallel Processing**: Different DNS providers processed in parallel for optimal performance
- **Validation**: Comprehensive validation before export
- **Intelligent Rate Limiting**: Uses `golang.org/x/time/rate` with provider-specific limiters
- **Retry Logic**: Automatic retry for temporary failures with exponential backoff
- **Dry-run Mode**: Test connectivity without exporting (default mode)
- **Domain Filtering**: Export specific domains from multi-domain configurations
- **Metadata Preservation**: Includes timestamps and provider information
- **Sequential Same-Provider Processing**: Domains using the same provider are processed sequentially to avoid rate conflicts

### Import Functionality

- **Conflict Resolution**: Multiple strategies for handling existing records
- **Provider-Aware Processing**: Automatic provider grouping from backup files for parallel import
- **Safety Features**: Dry-run mode (default mode)and backup-before-import options
- **Validation**: Pre-import validation to prevent invalid records
- **Intelligent Rate Limiting**: Provider-specific rate limiting to prevent conflicts
- **Batch Processing**: Efficient bulk operations with concurrent provider processing
- **Error Handling**: Graceful handling of failures with detailed reporting
- **Progress Tracking**: Real-time progress updates during import with provider grouping information

### Validation Features

- **Record Type Validation**: Validates all standard DNS record types
- **Content Validation**: Type-specific content validation (IP addresses, domain names, etc.)
- **Syntax Validation**: DNS syntax and format checking
- **Best Practice Warnings**: Recommendations for DNS record configuration
- **Duplicate Detection**: Identifies duplicate records in record sets

## Usage

### Available Command Line Flags

**Export Command (`export`):**
- `--format`, `-f`: Output format - "json" (machine-readable) or "txt" (human-readable) [default: json]
- `--output-dir`, `-o`: Output directory for backup files [default: current directory]
- `--domains`, `-d`: Specific domains to export (comma-separated) [default: all configured domains]
- `--record-types`, `-t`: Specific DNS record types to export (comma-separated) [default: all types]
- `--dry-run`: Test connectivity and preview export without creating files [default: true]
- `--production`: Enable production mode to create actual backup files [default: false]
- `--config`: Path to configuration file [default: config.yaml] (inherited from root)
- `--debug`: Enable debug output (inherited from root)

**Import Command (`import`):**
- `--files`, `-f`: Backup files to import (JSON or TXT format, comma-separated) [required]
- `--strategy`, `-s`: Conflict resolution strategy - "skip", "replace", "merge", or "abort" [default: skip]
- `--record-types`, `-t`: Specific DNS record types to import (comma-separated) [default: all types]
- `--backup-before`: Create backup of current records before importing [default: false]
- `--dry-run`: Test import operation without making any changes [default: true]
- `--production`: Enable production mode to make actual changes [default: false]
- `--config`: Path to configuration file [default: config.yaml] (inherited from root)
- `--debug`: Enable debug output (inherited from root)

**Important Notes:**
- Both commands default to **dry-run mode** for safety
- Use `--production` flag to perform actual operations
- Debug output is available via the global `--debug` flag

### Export DNS Records

```bash
# Export all domains from configuration (production mode)
./spf-flattener export --production --config config.yaml --format json --output-dir ./backups

# Export specific domain (production mode)
./spf-flattener export --production --domains example.com --format txt --output-dir ./backups

# Test connectivity before export (default dry-run mode)
./spf-flattener export --config config.yaml

# Export specific record types
./spf-flattener export --production --record-types A,AAAA --format json --output-dir ./backups
```

### Import DNS Records

```bash
# Import backup files (production mode)
./spf-flattener import --production --files backup1.json,backup2.json --strategy skip

# Import with backup creation (production mode)
./spf-flattener import --production --files backup.json --strategy replace --backup-before

# Test import without making changes (default dry-run mode)
./spf-flattener import --files backup.json

# Import specific record types (production mode)
./spf-flattener import --production --files backup.json --record-types A,CNAME --strategy skip

# Import with different conflict strategies
./spf-flattener import --production --files backup.json --strategy replace --backup-before
./spf-flattener import --production --files backup.json --strategy merge
./spf-flattener import --production --files backup.json --strategy abort
```

### Configuration

The backup system integrates with the existing SPF Flattener configuration structure:

```yaml
# Global provider setting
provider: porkbun

# DNS configuration (optional)
dns:
  - name: Google
    ip: 8.8.8.8

# Domain configurations
domains:
  - name: example.com
    provider: porkbun
    api_key: "your_api_key"
    secret_key: "your_secret_key"
    ttl: 3600
    dry_run: false
    logging: true
```

## File Formats

### JSON Format

```json
{
  "domain": "example.com",
  "provider": "porkbun",
  "version": "1.0",
  "exported_at": "2024-01-15T10:30:00Z",
  "records": [
    {
      "id": "rec1",
      "name": "@",
      "type": "A",
      "content": "192.168.1.1",
      "ttl": 3600,
      "priority": 0,
      "notes": "Main A record"
    }
  ]
}
```

### TXT Format

```
=== DNS Backup for example.com ===
Provider: porkbun
Version: 1.0
Exported At: 2024-01-15T10:30:00Z

=== Records ===
Record 1:
  ID: rec1
  Name: @
  Type: A
  Content: 192.168.1.1
  TTL: 3600
  Priority: 0
  Notes: Main A record
```

## Conflict Resolution Strategies

### Skip
- Existing records are left unchanged
- Only new records are created
- Safest option for preservation

### Replace
- Existing records are replaced with backup versions
- Use with caution - overwrites current configuration
- Best for complete restore scenarios

### Merge
- Combines existing and backup records
- Attempts to preserve both configurations
- May create duplicate records in some cases

## Error Handling

### Retry Logic
- **Automatic Retry**: Temporary failures are retried up to 3 times
- **Exponential Backoff**: Delays between retries increase exponentially
- **Context Cancellation**: Operations can be cancelled via context

### Validation Errors
- **Critical Errors**: Prevent operations from proceeding
- **Warnings**: Allow operations to continue with notifications
- **Detailed Reporting**: Clear error messages with specific record information

### Network Errors
- **Connectivity Testing**: Tests API connectivity before operations
- **Timeout Handling**: Proper timeout handling for network operations
- **Graceful Degradation**: Continues with available records when possible

## Safety Features

### Dry-run Mode
- Tests operations without making actual changes
- Validates configuration and connectivity
- Shows what would be changed without executing

### Backup Before Import
- Creates backup of current records before import
- Provides rollback capability
- Stored in specified backup directory

### Validation
- Pre-operation validation prevents invalid changes
- Type-specific validation for all record types
- Best practice recommendations

## Extensibility

### Adding New DNS Providers

1. **Implement DNSAPIClient Interface**:
   ```go
   type DNSAPIClient interface {
       Ping() (*PingResponse, error)
       RetrieveAllRecords(domain string) ([]BackupDNSRecord, error)
       BulkCreateRecords(domain string, records []BackupDNSRecord) error
       // ... other required methods
   }
   ```

2. **Create Adapter** (if needed):
   - Handle interface compatibility issues
   - Convert between different record types
   - Maintain backward compatibility

3. **Update Configuration**:
   - Add provider-specific configuration options
   - Update configuration parsing logic
   - Add validation for provider-specific fields

### Adding New Formats

1. **Implement FormatHandler Interface**:
   ```go
   type FormatHandler interface {
       Serialize(recordSet *DNSRecordSet) ([]byte, error)
       Deserialize(data []byte) (*DNSRecordSet, error)
       GetFileExtension() string
   }
   ```

2. **Register Format**:
   - Add to format handler registry
   - Update CLI command options
   - Add format-specific validation

## Performance Considerations

### Concurrent Operations
- **Provider-Aware Parallelization**: Different DNS providers are processed in parallel for maximum performance
- **Same-Provider Sequential Processing**: Domains using the same provider are processed sequentially to respect rate limits
- **Intelligent Provider Grouping**: Automatic grouping of domains by DNS provider from configuration/backup files
- **Separate Rate Limiters**: Each provider group gets its own rate limiter (2 req/sec, burst 1) to prevent conflicts
- **Performance Improvement**: 50-70% faster processing for multi-provider configurations
- Proper context handling for cancellation and timeout management
- Resource management to prevent overwhelming APIs

### Memory Usage
- Streaming processing for large record sets
- Efficient data structures for record storage
- Proper cleanup of resources

### Network Efficiency
- Batch operations for multiple records
- Connection pooling and reuse
- Proper timeout handling

## Testing

### Unit Tests
- Comprehensive test coverage for all components
- Mock objects for external dependencies
- Edge case testing and error scenarios

### Integration Tests
- Complete workflow testing
- CLI command integration
- Format compatibility testing

### Manual Testing
- Real-world scenario validation
- Performance testing with large domains
- Error recovery testing

## Troubleshooting

### Common Issues

**API Connectivity Problems**
- Verify API keys and permissions
- Check network connectivity
- Validate DNS provider status

**Validation Failures**
- Review DNS record formats
- Check for invalid IP addresses or domain names
- Verify TTL values are within valid ranges

**Import Conflicts**
- Use appropriate conflict resolution strategy
- Consider backup-before-import option
- Review existing records before import

### Debug Information

Enable debug mode for detailed logging:
```bash
./spf-flattener export --production --debug --config config.yaml
```

### Log Analysis

The system provides structured logging with JSON format:
- Operation start/end timestamps
- Success/failure status
- Detailed error information
- Performance metrics

## Best Practices

### Backup Strategy
- Regular backups before making changes
- Store backups in multiple locations
- Use version control for critical configurations
- Test restore procedures periodically

### Import Safety
- Always use dry-run mode first
- Create backups before import operations
- Start with skip strategy for safety
- Review results before大规模操作

### Performance Optimization
- Use appropriate batch sizes
- Monitor API rate limits
- Consider off-peak hours for large operations
- Implement proper error handling and retry logic

## Security Considerations

### API Key Management
- Store API keys securely (environment variables or encrypted storage)
- Rotate keys regularly
- Use least privilege access
- Audit key usage

### Data Protection
- Encrypt backup files when stored
- Secure file permissions
- Regular access audits
- Secure transmission of backup data

### Access Control
- Restrict access to backup tools
- Implement proper authentication
- Log all backup/restore operations
- Monitor for unauthorized access attempts