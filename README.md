[![Go Version](https://img.shields.io/badge/go-1.24.2%2B-blue.svg)](https://golang.org/dl/) [![License](https://img.shields.io/github/license/dean-jl/spf-flattener.svg)](LICENSE)

# SPF Flattener

A CLI tool to automatically flatten SPF DNS records for domains using DNS providers such as Porkbun (with extensibility for others). This tool helps manage complex SPF records by resolving `include`, `a`, and `mx` mechanisms into IP addresses, ensuring your domain's SPF record adheres to DNS lookup limits and character limits.

## Features

-   **SPF Flattening**: Resolves complex SPF records into a single, flattened record containing only IP addresses. Supports `include`, `a`, and `mx` mechanisms.
-   **DNS Backup & Restore**: Export and import complete DNS record sets with validation, conflict resolution, and support for multiple formats (JSON, TXT).
-   **Multi-Domain & Multi-Provider Support**: Manages SPF records for multiple domains, each with its own DNS provider and API credentials.
-   **Extensible Provider Architecture**: Easily add support for other DNS providers (e.g., Cloudflare, Route53) via config and pluggable interfaces.
-   **Automatic Splitting**: Automatically splits large flattened SPF records into multiple TXT records (e.g., `spf0.yourdomain.com`, `spf1.yourdomain.com`) to comply with the 255-character limit per TXT record.
-   **Flexible Source**: Can retrieve the unflattened SPF record from the root domain or a specified subdomain (e.g., `spf-unflat.yourdomain.com`).
-   **Safe by Default**: Runs in dry-run mode unless explicitly told to update DNS records.
-   **Performance Optimized**: Features provider-aware parallel processing, standardized rate limiting with `golang.org/x/time/rate`, and DNS client reuse for optimal performance. Up to 70% faster for multi-provider setups.
-   **Security Enhanced**: Includes comprehensive input validation, DNS response validation, and secure logging practices.
-   **Context-Aware**: Full context support for cancellation and timeout control across all operations.

## Installation

```bash
go install github.com/dean-jl/spf-flattener@latest
```

## Architecture & Quality

This project follows Go best practices and modern software engineering patterns:

### Code Quality
- **Modern Go Structure**: Uses `internal/` directory structure with proper package organization
- **Comprehensive Testing**: Includes unit tests, integration tests, error path tests, and performance benchmarks
- **Context-Aware Operations**: All DNS operations support context for cancellation and timeouts
- **Error Handling**: Proper error wrapping with context and detailed error messages

### Performance
- **Provider-Aware Parallel Processing**: Different DNS providers are processed in parallel while same-provider domains are processed sequentially to avoid rate conflicts
- **Intelligent Rate Limiting**: Standardized use of `golang.org/x/time/rate` (2 req/sec, burst 1) across all commands with provider-specific limiters
- **Multi-Provider Optimization**: Up to 50-70% performance improvement for configurations using multiple DNS providers
- **DNS Client Reuse**: Optimized DNS client pooling for reduced connection overhead
- **Response Caching**: DNS response caching for improved performance

### Security
- **Input Validation**: RFC-compliant domain name validation and configuration validation
- **DNS Response Validation**: Comprehensive validation of DNS responses to prevent malformed data processing
- **Secure Logging**: No API keys, secrets, or sensitive data are logged anywhere in the application
- **Safe Defaults**: Dry-run mode by default to prevent accidental changes

### Testing

This project uses a **hybrid testing approach** that combines automated unit tests for core logic with manual verification for user-facing functionality.

**Automated Tests (Unit Tests)**
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
```

**Manual Verification (Integration Testing)**

For CLI functionality and real-world scenarios, perform these manual tests:

```bash
# 1. Configuration validation
./spf-flattener flatten --config invalid.yaml
# Expected: Clear error message about configuration issues

# 2. API connectivity testing
./spf-flattener ping
# Expected: Success confirmation or clear failure reason

# 3. Full workflow dry-run
./spf-flattener flatten --dry-run --verbose
# Expected: Shows what changes would be made without applying them

# 4. Export functionality
./spf-flattener export --dry-run
# Expected: Shows what would be exported without creating file

# 5. Import functionality (dry-run)
./spf-flattener import --dry-run --files example.com-dns-backup-20240115-143022.json
# Expected: Shows what changes would be made without applying them

# 6. Help and usage verification
./spf-flattener --help
./spf-flattener flatten --help
./spf-flattener export --help
./spf-flattener import --help
# Expected: Clear, accurate usage instructions

# 7. Output functionality
./spf-flattener flatten --dry-run --output report.txt
# Expected: Results written to file instead of console
```

**Why This Approach?**
- **Unit tests** verify core algorithms and logic that users cannot easily test
- **Manual verification** ensures real-world functionality works correctly
- **No brittle integration tests** that depend on external services
- **User confidence** through actual usage rather than mocked scenarios

## Configuration

Create a `config.yaml` file in the same directory as the executable. This file should define the domains you want to manage, along with their respective provider and API credentials.

**Example `config.yaml`:**
```yaml
provider: porkbun

dns:
  - name: GoogleDNS
    ip: "8.8.8.8"

domains:
  - name: yourdomain.com
    provider: porkbun
    api_key: "YOUR_API_KEY_FOR_YOURDOMAIN"
    secret_key: "YOUR_SECRET_KEY_FOR_YOURDOMAIN"
    ttl: 3600 # Optional: Time-To-Live for DNS records. Default is 600 seconds.
```

**Important**: Ensure your API keys have the necessary permissions to retrieve and update DNS records. If you encounter "403 Forbidden" errors, ensure your IP address is whitelisted in your provider account settings.

### Environment Variables

For enhanced security, you can use environment variables instead of storing API credentials directly in your configuration file. Set the API key and secret key fields to empty strings (`""`) in your config file, then export the following environment variables:

```bash
export SPF_FLATTENER_API_KEY="your_actual_api_key"
export SPF_FLATTENER_SECRET_KEY="your_actual_secret_key"
```

**Example configuration with environment variables:**
```yaml
provider: porkbun
domains:
  - name: yourdomain.com
    provider: porkbun
    api_key: ""      # Will use SPF_FLATTENER_API_KEY
    secret_key: ""   # Will use SPF_FLATTENER_SECRET_KEY
    ttl: 3600
```

The application will automatically use the environment variables when the config file fields are empty, providing a secure way to manage credentials.

## Usage

The `spf-flattener` tool provides commands for SPF flattening, DNS backup/restore, and API testing.

### `flatten` Command

The `flatten` command processes the SPF records for the domains defined in your `config.yaml`.

```bash
spf-flattener flatten [flags]
```

**Global Flags** (available for all commands):

-   `--config` (string, default: `config.yaml`): Path to the configuration file.
-   `--debug` (boolean, default: `false`): Enables detailed debug logging for troubleshooting.
-   `--spf-unflat` (boolean, default: `false`): If set, the tool will retrieve the initial unflattened SPF record from `spf-unflat.<domain>` instead of the root domain.

**Flatten Command Flags:**

-   `--dry-run` (boolean, default: `true`): Simulate changes without applying them. This is the default safe mode.
-   `--production` (boolean, default: `false`): Enable production mode to apply live DNS updates. Use with caution.
-   `--verbose` (boolean, default: `false`): Enables informational logging, showing the start and end of processing for each domain.
-   `--force` (boolean, default: `false`): Force update DNS records regardless of whether changes are detected. Useful for refreshing records even when SPF mechanisms haven't changed.
-   `--output` (string): If specified, the final report is written to this file instead of to the console.

**Examples:**

-   **Perform a dry run (default behavior):**
    ```bash
    ./spf-flattener flatten
    ```

-   **Perform a dry run with verbose output:**
    ```bash
    ./spf-flattener flatten --verbose
    ```

-   **Update live DNS records:**
    ```bash
    ./spf-flattener flatten --production
    ```

-   **Force update DNS records even if no changes detected:**
    ```bash
    ./spf-flattener flatten --production --force
    ```

-   **Update live DNS records using `spf-unflat` source:**
    ```bash
    ./spf-flattener flatten --production --spf-unflat
    ```

-   **Run in production and save the report to a file:**
    ```bash
    ./spf-flattener flatten --production --output report.txt
    ```

### `ping` Command

The `ping` command allows you to test your Porkbun API credentials for each configured domain.

```bash
spf-flattener ping
```

### `completion` Command

The `completion` command generates shell autocompletion scripts that enable tab completion for the `spf-flattener` command in your shell. This greatly improves usability by allowing you to tab-complete commands, flags, and options.

**Quick setup (temporary - current shell session only):**
```bash
# Load completions in your current shell session
source <(spf-flattener completion bash)    # For Bash
source <(spf-flattener completion zsh)     # For Zsh
```

**Persistent setup (survives shell restarts):**

```bash
# Bash (Linux)
spf-flattener completion bash > /etc/bash_completion.d/spf-flattener

# Bash (macOS)
spf-flattener completion bash > $(brew --prefix)/etc/bash_completion.d/spf-flattener

# Zsh (Linux)
spf-flattener completion zsh > "${fpath[1]}/_spf-flattener"

# Zsh (macOS)
spf-flattener completion zsh > $(brew --prefix)/share/zsh/site-functions/_spf-flattener

# Fish
spf-flattener completion fish > ~/.config/fish/completions/spf-flattener.fish

# PowerShell
spf-flattener completion powershell > spf-flattener.ps1
```

**After setup, you can use tab completion:**
- `spf-flattener <TAB>` - Shows available commands (completion, flatten, help, ping)
- `spf-flattener flatten --<TAB>` - Shows available flags (--config, --debug, --dry-run, etc.)

**For detailed shell-specific instructions, run:**
- `spf-flattener completion bash --help`
- `spf-flattener completion zsh --help`
- `spf-flattener completion fish --help`
- `spf-flattener completion powershell --help`

### `export` Command

The `export` command backs up DNS records for configured domains to a file. It uses provider-aware parallel processing to optimize performance when working with multiple DNS providers.

```bash
spf-flattener export [flags]
```

**Export Command Flags:**

-   `-o, --output-dir` (string): Output directory for backup files (default: current directory).
-   `-f, --format` (string): Output format: `json` (machine-readable) or `txt` (human-readable) (default: "json").
-   `-d, --domains` (strings): Specific domains to export (comma-separated, default: all configured domains).
-   `-t, --record-types` (strings): Specific DNS record types to export (comma-separated, default: all types). Valid types: A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, PTR, CAA, DNSKEY, DS, RRSIG, NSEC, NSEC3, NSEC3PARAM.
-   `--dry-run` (boolean): Test connectivity and preview export without creating files (default: true).
-   `--production` (boolean): Enable production mode to create actual backup files (default: false).

**Examples:**

-   **Export all domains to JSON in current directory:**
    ```bash
    ./spf-flattener export --format json
    ```

-   **Export specific domain to text format in custom directory:**
    ```bash
    ./spf-flattener export --domains example.com --format txt --output-dir ./backups
    ```

-   **Export multiple domains:**
    ```bash
    ./spf-flattener export --domains example.com,mydomain.org --output-dir ./backups
    ```

-   **Export only specific record types (TXT and CNAME):**
    ```bash
    ./spf-flattener export --record-types TXT,CNAME --format json
    ```

-   **Export only A records for a specific domain:**
    ```bash
    ./spf-flattener export --domains example.com --record-types A --format txt
    ```

-   **Validate configuration before export:**
    ```bash
    ./spf-flattener export --dry-run
    ```

-   **Export all domains to backup files (production mode):**
    ```bash
    ./spf-flattener export --production
    ```

**Note:** Output files are automatically named as `{domain}-dns-backup-{timestamp}.{format}` (e.g., `example.com-dns-backup-20240115-143022.json`).

### `import` Command

The `import` command restores DNS records from a backup file with validation and conflict resolution. It automatically groups import tasks by DNS provider and processes different providers in parallel while maintaining sequential processing for the same provider to avoid rate conflicts.

```bash
spf-flattener import [flags]
```

**Import Command Flags:**

-   `-f, --files` (strings, required): Backup files to import (JSON or TXT format, comma-separated).
-   `-s, --strategy` (string): Conflict resolution strategy: `skip`, `replace`, `merge`, or `abort` (default: "skip").
-   `-t, --record-types` (strings): Specific DNS record types to import (comma-separated, default: all types). Valid types: A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, PTR, CAA, DNSKEY, DS, RRSIG, NSEC, NSEC3, NSEC3PARAM.
-   `--backup-before` (boolean): Create backup of current records before importing (default: false).
-   `--dry-run` (boolean): Test import operation without making any changes to DNS records (default: true).
-   `--production` (boolean): Enable production mode to make actual changes to DNS records (default: false).

**Examples:**

-   **Import records with dry-run (safe mode):**
    ```bash
    ./spf-flattener import --files example.com-dns-backup-20240115-143022.json --dry-run
    ```

-   **Import records with automatic backup:**
    ```bash
    ./spf-flattener import --files backup.json --backup-before
    ```

-   **Import and replace conflicting records:**
    ```bash
    ./spf-flattener import --files backup.json --strategy replace
    ```

-   **Import multiple backup files:**
    ```bash
    ./spf-flattener import --files example1.json,example2.txt --strategy merge
    ```

-   **Import only specific record types (A and AAAA):**
    ```bash
    ./spf-flattener import --files backup.json --record-types A,AAAA --strategy skip
    ```

-   **Import only TXT records from backup file:**
    ```bash
    ./spf-flattener import --files backup.json --record-types TXT --dry-run
    ```

-   **Import records in production mode (actual changes):**
    ```bash
    ./spf-flattener import --files backup.json --production --backup-before
    ```

## Automating with Cron Jobs

You can automate SPF flattening by running the tool periodically with a cron job. This example runs every day at midnight.

```cron
0 0 * * * /path/to/spf-flattener flatten --production --config /path/to/config.yaml >> /var/log/spf-flattener.log 2>&1
```

## Error Handling & Logging

- All actions and errors are logged in a structured, human-readable format to the console's standard error.
- Use the `--verbose` and `--debug` flags to increase the level of detail in the logs.
- The `ptr` mechanism is not supported and will be ignored if encountered.

## Known Limitations
- Only Porkbun is supported as a DNS provider by default (though the architecture is extensible).
- The `ptr` SPF mechanism is not supported and is ignored due to security considerations.
- DNS propagation delays may affect record updates immediately after changes.
- The `spf-unflat` source record is excluded from deletion during production updates for safety.
- Flatten/Backup/restore functionality requires API keys with read/write permissions for DNS records.

## Development

### Building from Source
```bash
# Clone the repository
git clone https://github.com/dean-jl/spf-flattener
cd spf-flattener

# Build the project
go build -o spf-flattener ./cmd/spf-flattener

# Or use the build script
./build-it.sh
```

### Project Structure
```
cmd/spf-flattener/     # CLI application entry point
internal/              # Core application logic (not importable by external packages)
├── backup/            # DNS backup/restore functionality
├── config/            # Configuration handling and validation
├── spf/               # SPF record processing and DNS operations
├── porkbun/           # Porkbun DNS provider implementation
└── processor/         # Domain processing business logic
tests/                 # Integration tests
docs/                  # Documentation and improvement tracking
```

### Adding New DNS Providers

The application is designed to be extensible for additional DNS providers beyond Porkbun. Here's how to add support for a new provider (e.g., Cloudflare, Route53):

#### 1. Create Provider Package
```bash
# Create new provider package
mkdir internal/cloudflare
```

#### 2. Implement DNS Provider Interface
Create `internal/cloudflare/client.go`:

```go
package cloudflare

import (
    "net"
    "net/http"
    "time"
)

// Client implements DNS operations for Cloudflare
type Client struct {
    apiToken string
    client   *http.Client
    debug    bool
}

func NewClient(apiToken string, debug bool) *Client {
    return &Client{
        apiToken: apiToken,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        debug: debug,
    }
}

// Implement the DNSAPIClient interface methods:
// - Ping() (*PingResponse, error)
// - RetrieveRecords(domain string) ([]Record, error)
// - UpdateRecord(domain, recordID, hostName, content string, ttl int) error
// - CreateRecord(domain, hostName, content string, ttl int) error
// - DeleteRecord(domain, recordID string) error
```

#### 3. Add Provider Configuration Support
Update configuration parsing in `cmd/spf-flattener/flatten.go`:

```go
// Add to the provider switch statement
switch strings.ToLower(domain.Provider) {
case "porkbun":
    apiClient = porkbun.NewClient(domain.ApiKey, domain.SecretKey, cliConfig.Debug)
case "cloudflare":
    apiClient = cloudflare.NewClient(domain.ApiKey, cliConfig.Debug)
// Add other providers here
default:
    return fmt.Errorf("unsupported DNS provider: %s", domain.Provider)
}
```

#### 4. Update Configuration Documentation
Add provider-specific configuration examples:

```yaml
# For Cloudflare
provider: cloudflare
domains:
  - name: example.com
    provider: cloudflare
    api_key: "your-cloudflare-api-token"
    # Note: Cloudflare uses API tokens, not separate secret keys
```

#### 5. Add Tests
Create `internal/cloudflare/client_test.go` with unit tests for the new provider.

#### 6. Required Interface Methods

All DNS providers must implement these methods:

- **Ping()** - Test API connectivity
- **RetrieveRecords()** - Get existing DNS records for a domain
- **CreateRecord()** - Create new DNS record
- **UpdateRecord()** - Update existing DNS record
- **DeleteRecord()** - Delete DNS record
- **BulkCreateRecords()** - Create multiple DNS records in a single operation (for backup/restore)

#### 7. Error Handling Standards

Follow the established error handling patterns:
- Wrap errors with context using `fmt.Errorf("operation failed: %w", err)`
- Return structured errors that can be logged appropriately
- Handle rate limiting and API timeouts gracefully

#### 8. Documentation
- Package documentation follows GoDoc conventions
- Architecture decisions are documented in code comments

## 🤝 Community Contributions

This project is open to the community—whether you're looking to fork, extend, or improve it.

### 🛠 Fork-Friendly by Design
- The architecture is modular and extensible.
- Adding new DNS providers is straightforward (see `docs/` and `internal/` structure).
- Dry-run mode ensures safe experimentation.

Feel free to fork this repository and adapt it to your needs. If you build something useful or add support for a new provider, I’d love to hear about it.

### 📬 Pull Requests Welcome (with Caveats)
You’re welcome to submit pull requests, but please note:
- I may not be able to actively review or merge contributions.
- This project is maintained as a reference implementation and personal utility.
- Contributions may be acknowledged or incorporated at my discretion.

If you’d like to collaborate more deeply or help maintain the project, open a discussion or issue to introduce yourself.

### 🧭 Contribution Guidelines
If you do submit a PR:
- Please run `go fmt ./...`, `go vet ./...`, and `go test ./...` before submitting.
- Include a brief description of what your change does and why it’s useful.
- For new DNS providers, include integration test instructions.

## 🤖 Authorship & AI Assistance

This application was developed with the assistance of AI tools, guided and validated by human authorship. All architectural decisions, testing, and final implementation were reviewed and refined by the maintainer.

While AI tools supported code generation and scaffolding, the design, logic, and operational validation were shaped by human insight. This includes scenario modeling, DNS architecture decisions, and CLI behavior refinement.

The maintainer retains copyright over the human-authored portions of this work.

## 🧾 License

This project is licensed under the [GNU AGPLv3](https://www.gnu.org/licenses/agpl-3.0.en.html).




