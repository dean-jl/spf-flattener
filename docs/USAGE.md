# Usage Guide

Complete command reference for the SPF Flattener CLI tool.

## Global Flags

These flags are available for all commands:

- `--config` (string, default: `config.yaml`): Path to the configuration file
- `--debug` (boolean, default: `false`): Enable detailed debug logging for troubleshooting
- `--spf-unflat` (boolean, default: `false`): Use spf-unflat.<domain> TXT record as source instead of main SPF record (preserves original unflattened SPF for future updates)

## Commands Overview

- `flatten` - Process and flatten SPF records
- `ping` - Test API connectivity
- `export` - Backup DNS records to files
- `import` - Restore DNS records from backup files
- `completion` - Generate shell autocompletion scripts
- `help` - Get help for any command

---

## `flatten` Command

Process SPF records for domains defined in your configuration file.

```bash
spf-flattener flatten [flags]
```

### Intelligent Processing

The tool automatically counts DNS lookups in SPF records and only flattens when necessary:
- **≤10 lookups**: Considered RFC 7208 compliant, won't be flattened
- **>10 lookups**: Automatically flattened to prevent SPF failures
- Use `--force-flatten` to override this behavior

### Flags

- `--dry-run` (boolean, default: `true`): Simulate changes without applying them (safe default)
- `--production` (boolean, default: `false`): Enable production mode to apply live DNS updates
- `--verbose` (boolean, default: `false`): Show detailed processing information
- `--force` (boolean, default: `false`): Force update DNS records even if no changes detected
- `--force-flatten` (boolean, default: `false`): Force flattening even for RFC-compliant records
- `--aggregate` (boolean, default: `false`): Enable CIDR aggregation to optimize record size
- `--output` (string): Write final report to file instead of console

### Examples

**Safe dry-run (default behavior):**
```bash
./spf-flattener flatten
./spf-flattener flatten --verbose
```

**Apply changes to live DNS:**
```bash
./spf-flattener flatten --production
./spf-flattener flatten --production --verbose
```

**Force operations:**
```bash
# Force update even if no changes detected
./spf-flattener flatten --production --force

# Force flatten RFC-compliant records
./spf-flattener flatten --production --force-flatten
```

**CIDR aggregation:**
```bash
# Test aggregation
./spf-flattener flatten --aggregate --dry-run

# Apply with aggregation
./spf-flattener flatten --production --aggregate
```

**Using spf-unflat source:**
```bash
./spf-flattener flatten --production --spf-unflat
```

**Save report to file:**
```bash
./spf-flattener flatten --production --output report.txt
```

---

## `ping` Command

Test API connectivity for all configured domains.

```bash
spf-flattener ping [flags]
```

This command verifies:
- API credentials are valid
- Network connectivity to DNS provider
- Required permissions are available
- IP address is whitelisted (if required by provider)

### Examples

```bash
# Test all domains in config
./spf-flattener ping

# Test with specific config file
./spf-flattener ping --config production-config.yaml

# Test with debug output
./spf-flattener ping --debug
```

---

## `export` Command

Backup DNS records for configured domains to files.

```bash
spf-flattener export [flags]
```

### Flags

- `-o, --output-dir` (string): Output directory for backup files (default: current directory)
- `-f, --format` (string): Output format - `json` (machine-readable) or `txt` (human-readable) (default: "json")
- `-d, --domains` (strings): Specific domains to export (comma-separated, default: all)
- `-t, --record-types` (strings): Specific DNS record types to export (comma-separated, default: all)
- `--dry-run` (boolean, default: `true`): Test connectivity without creating files
- `--production` (boolean, default: `false`): Create actual backup files

### Supported Record Types

A, AAAA, CNAME, MX, TXT, NS, SOA, SRV, PTR, CAA, DNSKEY, DS, RRSIG, NSEC, NSEC3, NSEC3PARAM

### Examples

**Test export (dry-run):**
```bash
./spf-flattener export --dry-run
./spf-flattener export --format txt --dry-run
```

**Export all domains:**
```bash
./spf-flattener export --production --format json
./spf-flattener export --production --format txt --output-dir ./backups
```

**Export specific domains:**
```bash
./spf-flattener export --production --domains example.com --format json
./spf-flattener export --production --domains example.com,mydomain.org --output-dir ./backups
```

**Export specific record types:**
```bash
# Export only TXT and CNAME records
./spf-flattener export --production --record-types TXT,CNAME --format json

# Export only A records for specific domain
./spf-flattener export --production --domains example.com --record-types A --format txt
```

### Output Files

Files are automatically named as `{domain}-dns-backup-{timestamp}.{format}`:
- `example.com-dns-backup-20240115-143022.json`
- `example.com-dns-backup-20240115-143022.txt`

---

## `import` Command

Restore DNS records from backup files with validation and conflict resolution.

```bash
spf-flattener import [flags]
```

### Flags

- `-f, --files` (strings, **required**): Backup files to import (comma-separated)
- `-s, --strategy` (string): Conflict resolution strategy (default: "skip")
- `-t, --record-types` (strings): Specific record types to import (comma-separated, default: all)
- `--backup-before` (boolean): Create backup before importing (default: false)
- `--dry-run` (boolean, default: `true`): Test import without making changes
- `--production` (boolean, default: `false`): Make actual changes to DNS records

### Conflict Resolution Strategies

- `skip`: Skip conflicting records (safest)
- `replace`: Replace existing records with backup data
- `merge`: Attempt to merge records intelligently
- `abort`: Stop import if conflicts are detected

### Examples

**Test import (dry-run):**
```bash
./spf-flattener import --files example.com-dns-backup-20240115-143022.json --dry-run
./spf-flattener import --files backup.json --strategy replace --dry-run
```

**Import with backup:**
```bash
./spf-flattener import --production --files backup.json --backup-before
```

**Handle conflicts:**
```bash
# Skip conflicting records
./spf-flattener import --production --files backup.json --strategy skip

# Replace conflicting records
./spf-flattener import --production --files backup.json --strategy replace

# Merge records
./spf-flattener import --production --files backup.json --strategy merge
```

**Import multiple files:**
```bash
./spf-flattener import --production --files example1.json,example2.txt --strategy merge
```

**Import specific record types:**
```bash
# Import only A and AAAA records
./spf-flattener import --production --files backup.json --record-types A,AAAA

# Import only TXT records
./spf-flattener import --production --files backup.json --record-types TXT --dry-run
```

---

## `completion` Command

Generate shell autocompletion scripts for improved usability.

```bash
spf-flattener completion [bash|zsh|fish|powershell]
```

### Quick Setup (Temporary)

Load completions in your current shell session:

```bash
# Bash
source <(spf-flattener completion bash)

# Zsh
source <(spf-flattener completion zsh)

# Fish
spf-flattener completion fish | source
```

### Persistent Setup

**Bash:**
```bash
# Linux
spf-flattener completion bash > /etc/bash_completion.d/spf-flattener

# macOS (with Homebrew)
spf-flattener completion bash > $(brew --prefix)/etc/bash_completion.d/spf-flattener
```

**Zsh:**
```bash
# Linux
spf-flattener completion zsh > "${fpath[1]}/_spf-flattener"

# macOS (with Homebrew)
spf-flattener completion zsh > $(brew --prefix)/share/zsh/site-functions/_spf-flattener
```

**Fish:**
```bash
spf-flattener completion fish > ~/.config/fish/completions/spf-flattener.fish
```

**PowerShell:**
```bash
spf-flattener completion powershell > spf-flattener.ps1
```

### Using Tab Completion

After setup, you can use tab completion:
- `spf-flattener <TAB>` - Shows available commands
- `spf-flattener flatten --<TAB>` - Shows available flags
- `spf-flattener export --format <TAB>` - Shows format options

---

## Common Usage Patterns

### Daily SPF Management

```bash
# Check what would change
./spf-flattener flatten --verbose --dry-run

# Apply changes if needed
./spf-flattener flatten --production --verbose

# Save report for records
./spf-flattener flatten --production --output daily-report-$(date +%Y%m%d).txt
```

### Before Making Changes

```bash
# Test connectivity
./spf-flattener ping

# Create backup
./spf-flattener export --production --output-dir ./backups

# Test changes
./spf-flattener flatten --dry-run --verbose

# Apply changes
./spf-flattener flatten --production
```

### Disaster Recovery

```bash
# List available backups
ls -la backups/

# Test restore (dry-run)
./spf-flattener import --files backups/example.com-dns-backup-20240115.json --dry-run

# Restore from backup
./spf-flattener import --production --files backups/example.com-dns-backup-20240115.json --strategy replace
```

### Automation with Cron

```bash
# Add to crontab for daily SPF maintenance
0 2 * * * /path/to/spf-flattener flatten --production --config /path/to/config.yaml >> /var/log/spf-flattener.log 2>&1

# Weekly backup
0 1 * * 0 /path/to/spf-flattener export --production --config /path/to/config.yaml --output-dir /backups/dns >> /var/log/spf-flattener.log 2>&1
```

### Multi-Environment Management

```bash
# Development environment
./spf-flattener flatten --config dev-config.yaml --dry-run

# Staging environment
./spf-flattener flatten --config staging-config.yaml --production

# Production environment (with backup)
./spf-flattener export --config prod-config.yaml --production --output-dir ./prod-backups
./spf-flattener flatten --config prod-config.yaml --production --output prod-report.txt
```

## Error Handling

### Common Exit Codes

- `0`: Success
- `1`: General error (configuration, network, etc.)
- `2`: Authentication/authorization error
- `3`: DNS provider API error
- `4`: Validation error

### Troubleshooting Commands

```bash
# Debug configuration issues
./spf-flattener flatten --debug --dry-run

# Test API connectivity
./spf-flattener ping --debug

# Validate specific operations
./spf-flattener export --dry-run --verbose
./spf-flattener import --files backup.json --dry-run --verbose
```

### Log Analysis

All operations log to stderr in structured format:
- Use `--verbose` for operational details
- Use `--debug` for troubleshooting information
- Use `--output` to separate results from logs