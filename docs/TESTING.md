# Testing Strategy

This project uses a **hybrid testing approach** that combines automated unit tests for core logic with manual verification for user-facing functionality.

## Automated Tests (Unit Tests)

### Running Tests

```bash
# Run all automated tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Run specific test suites
go test ./internal/spf/     # SPF processing algorithms
go test ./internal/config/  # Configuration parsing logic
go test ./internal/porkbun/ # API client functionality
go test ./internal/backup/  # Backup system functionality
```

### Test Coverage Areas

**Automated unit tests cover:**
- SPF processing algorithms and DNS lookup counting
- Configuration parsing and validation
- CIDR aggregation logic
- DNS record splitting algorithms
- API client functionality (with mocked responses)
- Backup system validation and conflict resolution
- Error handling and edge cases

## Manual Verification (Integration Testing)

For CLI functionality and real-world scenarios, perform these manual tests:

### 1. Configuration Validation
```bash
./spf-flattener flatten --config invalid.yaml
# Expected: Clear error message about configuration issues
```

### 2. API Connectivity Testing
```bash
./spf-flattener ping
# Expected: Success confirmation or clear failure reason
```

### 3. Full Workflow Dry-Run
```bash
./spf-flattener flatten --dry-run --verbose
# Expected: Shows what changes would be made without applying them
```

### 4. Export Functionality
```bash
./spf-flattener export --dry-run
# Expected: Shows what would be exported without creating file

./spf-flattener export --production --format json --output-dir ./test-backups
# Expected: Creates backup files in specified directory
```

### 5. Import Functionality
```bash
./spf-flattener import --dry-run --files example.com-dns-backup-20240115-143022.json
# Expected: Shows what changes would be made without applying them

./spf-flattener import --production --files backup.json --strategy skip
# Expected: Imports records using skip strategy for conflicts
```

### 6. Help and Usage Verification
```bash
./spf-flattener --help
./spf-flattener flatten --help
./spf-flattener export --help
./spf-flattener import --help
# Expected: Clear, accurate usage instructions
```

### 7. Output Functionality
```bash
./spf-flattener flatten --dry-run --output report.txt
# Expected: Results written to file instead of console
```

### 8. CIDR Aggregation Testing
```bash
./spf-flattener flatten --dry-run --aggregate --verbose
# Expected: Shows IP aggregation results and efficiency gains
```

### 9. Force Flatten Testing
```bash
./spf-flattener flatten --dry-run --force-flatten --verbose
# Expected: Flattens SPF records even if they're RFC 7208 compliant
```

### 10. Multi-Provider Testing
```bash
# Test with configuration containing multiple DNS providers
./spf-flattener flatten --dry-run --verbose
# Expected: Shows provider-aware parallel processing
```

## Why This Approach?

- **Unit tests** verify core algorithms and logic that users cannot easily test
- **Manual verification** ensures real-world functionality works correctly
- **No brittle integration tests** that depend on external services
- **User confidence** through actual usage rather than mocked scenarios

## Testing New DNS Providers

When adding a new DNS provider, follow this testing checklist:

### Unit Tests
- [ ] Mock HTTP responses for all API calls
- [ ] Test error conditions and edge cases
- [ ] Verify proper authentication header handling
- [ ] Test rate limiting behavior
- [ ] Test record CRUD operations

### Manual Integration Tests
```bash
# Test API connectivity
./spf-flattener ping --config config-newprovider.yaml

# Test dry-run functionality
./spf-flattener flatten --dry-run --config config-newprovider.yaml

# Test export/import functionality
./spf-flattener export --dry-run --config config-newprovider.yaml
./spf-flattener import --dry-run --files backup.json
```

## Performance Testing

### Benchmarks
```bash
# Run performance benchmarks
go test -bench=. ./internal/spf/
go test -bench=. ./internal/backup/

# Profile memory usage
go test -bench=. -memprofile=mem.prof ./internal/spf/
go tool pprof mem.prof
```

### Load Testing
For production environments, test with:
- Multiple domains (10+)
- Large SPF records (>10 includes)
- Multiple DNS providers simultaneously
- Rate limiting validation

## Continuous Integration

The project uses GitHub Actions for automated testing:
- Runs unit tests on Go 1.21+
- Tests on multiple platforms (Linux, macOS, Windows)
- Validates code formatting and linting
- Checks for security vulnerabilities

## Test Data

### Sample Configuration Files
- `tests/valid-config.yaml` - Valid configuration for testing
- `tests/invalid-config.yaml` - Invalid configuration for error testing
- `tests/multi-provider-config.yaml` - Multi-provider configuration

### Sample DNS Records
- `tests/sample-spf-records.txt` - Various SPF record formats for testing
- `tests/backup-samples/` - Sample backup files for import testing

## Troubleshooting Tests

### Common Issues
1. **API Rate Limiting**: Use `--verbose` to see rate limiting in action
2. **DNS Timeouts**: Check network connectivity and DNS server accessibility
3. **Configuration Errors**: Validate YAML syntax and required fields
4. **Provider Authentication**: Verify API keys and permissions

### Debug Mode
```bash
./spf-flattener flatten --debug --dry-run
# Provides detailed logging for troubleshooting
```