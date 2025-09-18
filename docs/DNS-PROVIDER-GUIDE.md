# DNS Provider Extension Guide

This guide explains how to extend the SPF Flattener to support additional DNS providers beyond Porkbun.

## Overview

The application uses a modular architecture that separates DNS provider implementation from core SPF processing logic. This allows for easy addition of new providers while maintaining consistent functionality. The system features provider-aware parallel processing where different DNS providers are processed concurrently, while domains using the same provider are processed sequentially to respect rate limits.

## Architecture

### Interfaces

The application defines two key interfaces for DNS operations:

1. **`DNSProvider`** - For DNS lookups (SPF resolution)
2. **`DNSAPIClient`** - For DNS record management (CRUD operations)

### Current Implementation

- **Porkbun**: Full implementation with API key + secret key authentication
- **System DNS**: Basic DNS lookups using Go's standard library
- **Custom DNS**: DNS lookups via specified DNS servers

## Adding a New Provider

### Step 1: Create Provider Package

```bash
mkdir internal/cloudflare
```

### Step 2: Define Client Structure

Create `internal/cloudflare/client.go`:

```go
package cloudflare

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    apiToken string
    client   *http.Client
    baseURL  string
    debug    bool
}

func NewClient(apiToken string, debug bool) *Client {
    return &Client{
        apiToken: apiToken,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        baseURL: "https://api.cloudflare.com/client/v4",
        debug:   debug,
    }
}
```

### Step 3: Implement Required Methods

#### Ping Method
```go
func (c *Client) Ping() (*PingResponse, error) {
    req, err := http.NewRequest("GET", c.baseURL+"/user/tokens/verify", nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+c.apiToken)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse response and return appropriate PingResponse
    // Implementation depends on Cloudflare API response format
}
```

#### RetrieveRecords Method
```go
func (c *Client) RetrieveRecords(domain string) ([]Record, error) {
    // 1. Get zone ID for domain
    // 2. Retrieve DNS records for zone
    // 3. Filter for TXT records
    // 4. Convert to standard Record format
}
```

#### CreateRecord Method
```go
func (c *Client) CreateRecord(domain, hostName, content string, ttl int) error {
    // 1. Get zone ID for domain
    // 2. Create DNS record via Cloudflare API
    // 3. Handle API response and errors
}
```

#### UpdateRecord Method
```go
func (c *Client) UpdateRecord(domain, recordID, hostName, content string, ttl int) error {
    // 1. Update existing DNS record
    // 2. Handle API response and errors
}
```

#### DeleteRecord Method
```go
func (c *Client) DeleteRecord(domain, recordID string) error {
    // 1. Delete DNS record by ID
    // 2. Handle API response and errors
}
```

#### Attribution Method
```go
func (c *Client) Attribution() string {
    // Return provider-specific attribution message
    // This will be displayed in the output to comply with provider terms of service
    return "Data provided by Cloudflare, Inc."
}
```

### Step 4: Add Provider Support

Update `cmd/spf-flattener/flatten.go` to include the new provider:

```go
import (
    // ... existing imports
    "github.com/dean/spf-flattener/internal/cloudflare"
)

// In the provider switch statement:
switch strings.ToLower(domain.Provider) {
case "porkbun":
    apiClient = porkbun.NewClient(domain.ApiKey, domain.SecretKey, cliConfig.Debug)
case "cloudflare":
    apiClient = cloudflare.NewClient(domain.ApiKey, cliConfig.Debug)
default:
    fmt.Printf("  No supported API client for provider: %s\n", domain.Provider)
    continue
}
```

### Step 5: Update Configuration Examples

Add provider-specific configuration documentation:

```yaml
# Cloudflare configuration
provider: cloudflare
domains:
  - name: example.com
    provider: cloudflare
    api_key: "your-cloudflare-api-token"
    # Note: Cloudflare uses single API tokens, not separate secret keys
    ttl: 3600
```

### Step 6: Add Tests

Create `internal/cloudflare/client_test.go`:

```go
package cloudflare

import (
    "testing"
    "net/http"
    "net/http/httptest"
)

func TestPing(t *testing.T) {
    // Mock HTTP server for testing
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock Cloudflare API response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"success": true, "result": {"status": "active"}}`))
    }))
    defer server.Close()

    client := &Client{
        apiToken: "test-token",
        client:   &http.Client{},
        baseURL:  server.URL,
        debug:    false,
    }

    resp, err := client.Ping()
    if err != nil {
        t.Fatalf("Ping failed: %v", err)
    }

    if resp.Status != "SUCCESS" {
        t.Errorf("Expected SUCCESS status, got %s", resp.Status)
    }
}

// Add tests for other methods: TestRetrieveRecords, TestCreateRecord, etc.
```

## Provider-Specific Considerations

### Authentication Methods

Different providers use different authentication:
- **Porkbun**: API key + Secret key
- **Cloudflare**: Bearer token
- **AWS Route53**: AWS credentials (Access Key + Secret)
- **DigitalOcean**: Personal Access Token

### API Rate Limits

**Standardized Rate Limiting**: All providers now use standardized rate limiting with `golang.org/x/time/rate`:
- **Default Rate**: 2 requests/second with burst of 1
- **Provider-Aware Processing**: Each provider gets its own rate limiter to prevent conflicts
- **Parallel Processing**: Different providers are processed in parallel for optimal performance
- **Same-Provider Sequential**: Domains using the same provider are processed sequentially

**Provider-Specific Limits** (for reference, but standardized implementation handles these automatically):
- Porkbun: ~120 requests/minute
- Cloudflare: 1200 requests/5 minutes
- AWS Route53: 5 requests/second per hosted zone

### Error Handling

Each provider has different error response formats. Standardize error handling by:
1. Parsing provider-specific error responses
2. Converting to standard Go errors with context
3. Using consistent error messages across providers

### Record Management

Different providers handle DNS records differently:
- Some require zone ID lookup before operations
- Some support bulk operations, others require individual calls
- Record ID formats vary between providers

## Testing Strategy

### Unit Tests
- Mock HTTP responses for all API calls
- Test error conditions and edge cases
- Verify proper authentication header handling

### Integration Tests
- Use real API credentials in test environment
- Test against actual provider APIs
- Include in manual verification steps, not automated CI

### Manual Verification
Users should test new providers with:
```bash
# Test API connectivity
./spf-flattener ping --config config-cloudflare.yaml

# Test dry-run functionality
./spf-flattener flatten --dry-run --config config-cloudflare.yaml
```

## Validation Checklist

Before submitting a new DNS provider:

- [ ] Implements all required interface methods (including `Attribution()`)
- [ ] Includes comprehensive unit tests
- [ ] Handles authentication properly
- [ ] Implements appropriate attribution message for provider terms of service
- [ ] Uses standardized rate limiting (automatically handled by provider grouping system)
- [ ] Follows established error handling patterns
- [ ] Includes configuration documentation
- [ ] Provides manual testing instructions
- [ ] Updates main application switch statement
- [ ] Tests provider-aware parallel processing behavior
- [ ] Considers provider-specific limitations
- [ ] Verifies integration with backup/restore system (`export`/`import` commands)

## Provider-Aware Processing

### Automatic Provider Grouping

The system automatically groups domains and operations by DNS provider:

- **Export Operations**: Domains are grouped by provider, different providers process in parallel
- **Import Operations**: Backup files are analyzed to determine provider, then grouped accordingly
- **Rate Limiting**: Each provider group gets its own rate limiter (2 req/sec, burst 1)
- **Performance**: Up to 50-70% improvement for multi-provider configurations

### Integration Requirements

New providers automatically benefit from provider-aware processing. No special implementation required, but providers should:

1. **Be Identified Consistently**: Provider name in configuration should be case-insensitive
2. **Support Backup Operations**: Implement full `DNSAPIClient` interface for export/import functionality
3. **Handle Concurrent Safe Operations**: While same-provider domains are processed sequentially, the provider implementation should be safe for concurrent use across different provider instances

## Common Patterns

### Zone ID Resolution
Many providers require zone ID lookup:

```go
func (c *Client) getZoneID(domain string) (string, error) {
    // Implementation varies by provider
    // Usually involves API call to list zones and find matching domain
}
```

### Pagination
Handle paginated responses:

```go
func (c *Client) getAllRecords(zoneID string) ([]Record, error) {
    var allRecords []Record
    page := 1

    for {
        records, hasMore, err := c.getRecordsPage(zoneID, page)
        if err != nil {
            return nil, err
        }

        allRecords = append(allRecords, records...)

        if !hasMore {
            break
        }
        page++
    }

    return allRecords, nil
}
```

### Request Retry Logic
Implement exponential backoff for transient failures:

```go
func (c *Client) makeRequestWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
    // Implementation with exponential backoff
}
```

This modular approach ensures consistent behavior across all DNS providers while allowing for provider-specific optimizations and requirements.