# Backup and Restore Workflow Examples

This document provides practical examples for using the backup and restore functionality.

## Basic Export

### Export All Domains
```bash
# Export all domains from configuration
./spf-flattener export --config config.yaml --format json --output-dir ./backups

# Export with verbose output
./spf-flattener export --config config.yaml --format json --output-dir ./backups --verbose

# Test connectivity before export (dry-run)
./spf-flattener export --config config.yaml --dry-run
```

### Export Specific Domains
```bash
# Export specific domains
./spf-flattener export --config config.yaml --domains example.com,mydomain.org --format json --output-dir ./backups

# Export single domain to TXT format
./spf-flattener export --config config.yaml --domains example.com --format txt --output-dir ./backups
```

### Export with Date Stamping
```bash
# Create timestamped backup directory
BACKUP_DIR="./backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Export to timestamped directory
./spf-flattener export --config config.yaml --format json --output-dir "$BACKUP_DIR"

# Example output file: ./backups/20240115_143022/example.com-dns-backup.json
```

## Basic Import

### Import with Safety Checks
```bash
# Test import without making changes (recommended first)
./spf-flattener import --files ./backups/example.com-dns-backup.json --dry-run

# Import with backup creation
./spf-flattener import --files ./backups/example.com-dns-backup.json --backup-before

# Import with verbose output
./spf-flattener import --files ./backups/example.com-dns-backup.json --verbose
```

### Conflict Resolution Strategies
```bash
# Skip existing records (safest)
./spf-flattener import --files ./backups/example.com-dns-backup.json --strategy skip

# Replace existing records (use with caution)
./spf-flattener import --files ./backups/example.com-dns-backup.json --strategy replace

# Merge records (may create duplicates)
./spf-flattener import --files ./backups/example.com-dns-backup.json --strategy merge
```

### Import Multiple Files
```bash
# Import multiple backup files
./spf-flattener import --files ./backups/example.com.json,./backups/mydomain.org.json --strategy skip

# Import all JSON files in directory
./spf-flattener import --files ./backups/*.json --strategy skip
```

## Backup Strategies

### Regular Automated Backups
```bash
#!/bin/bash
# daily-backup.sh - Automated daily backup script

CONFIG_FILE="/path/to/config.yaml"
BACKUP_DIR="/backups/daily"
DATE=$(date +%Y%m%d)
TIMESTAMP=$(date +%H%M%S)

# Create directory structure
mkdir -p "$BACKUP_DIR/$DATE"

# Export all domains
./spf-flattener export \
    --config "$CONFIG_FILE" \
    --format json \
    --output-dir "$BACKUP_DIR/$DATE/$TIMESTAMP"

# Keep only last 30 days
find "$BACKUP_DIR" -type d -mtime +30 -exec rm -rf {} \;

echo "Backup completed: $BACKUP_DIR/$DATE/$TIMESTAMP"
```

### Pre-Change Backup
```bash
#!/bin/bash
# pre-change-backup.sh - Backup before making DNS changes

DOMAIN="$1"
CONFIG_FILE="/path/to/config.yaml"
BACKUP_DIR="./backups/pre-change"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

if [ -z "$DOMAIN" ]; then
    echo "Usage: $0 <domain>"
    exit 1
fi

mkdir -p "$BACKUP_DIR"

# Backup specific domain before changes
./spf-flattener export \
    --config "$CONFIG_FILE" \
    --domains "$DOMAIN" \
    --format json \
    --output-dir "$BACKUP_DIR/$TIMESTAMP"

echo "Backup created for $DOMAIN: $BACKUP_DIR/$TIMESTAMP"
```

### Disaster Recovery Backup
```bash
#!/bin/bash
# disaster-recovery-backup.sh - Comprehensive backup for disaster recovery

CONFIG_FILE="/path/to/config.yaml"
BACKUP_DIR="./backups/disaster-recovery"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Export in multiple formats for redundancy
./spf-flattener export \
    --config "$CONFIG_FILE" \
    --format json \
    --output-dir "$BACKUP_DIR/$TIMESTAMP/json"

./spf-flattener export \
    --config "$CONFIG_FILE" \
    --format txt \
    --output-dir "$BACKUP_DIR/$TIMESTAMP/txt"

# Create archive
tar -czf "$BACKUP_DIR/backup-$TIMESTAMP.tar.gz" -C "$BACKUP_DIR" "$TIMESTAMP"

# Remove uncompressed files
rm -rf "$BACKUP_DIR/$TIMESTAMP"

echo "Disaster recovery backup created: $BACKUP_DIR/backup-$TIMESTAMP.tar.gz"
```

## Restore Scenarios

### Complete Restore
```bash
# Restore entire domain from backup
./spf-flattener import \
    --files ./backups/example.com-dns-backup.json \
    --strategy replace \
    --backup-before \
    --verbose
```

### Selective Restore
```bash
# If you need to restore specific records, you can:
# 1. Export current records first
./spf-flattener export --config config.yaml --domains example.com --output-dir ./temp

# 2. Edit the backup file to include only records you want to restore
# 3. Import with skip strategy to add missing records
./spf-flattener import \
    --files ./modified-backup.json \
    --strategy skip \
    --backup-before
```

### Emergency Restore
```bash
# Emergency restore from archive
tar -xzf ./backups/disaster-recovery/backup-20240115_143022.tar.gz

./spf-flattener import \
    --files ./20240115_143022/json/example.com-dns-backup.json \
    --strategy replace \
    --verbose
```

## Monitoring and Validation

### Backup Verification
```bash
#!/bin/bash
# verify-backups.sh - Verify backup integrity

BACKUP_DIR="./backups"
TEMP_DIR="./temp/verify"

mkdir -p "$TEMP_DIR"

# Find latest backup files
find "$BACKUP_DIR" -name "*-dns-backup.json" -mtime -7 | while read backup_file; do
    echo "Verifying: $backup_file"

    # Test file parsing
    if ./spf-flattener import --files "$backup_file" --dry-run; then
        echo "✓ Backup is valid"
    else
        echo "✗ Backup is corrupted"
    fi
done

rm -rf "$TEMP_DIR"
```

### Regular Health Checks
```bash
#!/bin/bash
# health-check.sh - Regular health checks for DNS infrastructure

CONFIG_FILE="/path/to/config.yaml"

# Test API connectivity
if ./spf-flattener ping --config "$CONFIG_FILE"; then
    echo "✓ API connectivity OK"
else
    echo "✗ API connectivity failed"
    exit 1
fi

# Test export functionality
TEMP_DIR="./temp/health"
mkdir -p "$TEMP_DIR"

if ./spf-flattener export --config "$CONFIG_FILE" --output-dir "$TEMP_DIR" --dry-run; then
    echo "✓ Export functionality OK"
else
    echo "✗ Export functionality failed"
    exit 1
fi

rm -rf "$TEMP_DIR"
echo "All health checks passed"
```

## Best Practices

### Backup Schedule
- **Daily**: Automated backups for critical domains
- **Weekly**: Full verification of backup integrity
- **Monthly**: Archive old backups to long-term storage
- **Pre-change**: Always backup before making DNS changes

### Security
- Store backup files in secure locations
- Encrypt backups containing sensitive information
- Limit access to backup tools and files
- Regular audit of backup access logs

### Testing
- Regular test restores from backup files
- Validate backup file integrity
- Test disaster recovery procedures
- Monitor backup success/failure rates

### Monitoring
- Set up alerts for backup failures
- Monitor disk space for backup storage
- Track backup completion times
- Regular review of backup retention policies