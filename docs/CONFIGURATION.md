# Configuration Guide

SPF Flattener uses YAML-based configuration to define domains, DNS providers, and processing options.

## Basic Configuration

Create a `config.yaml` file in the same directory as the executable:

```yaml
provider: porkbun

domains:
  - name: yourdomain.com
    provider: porkbun
    api_key: "YOUR_API_KEY"
    secret_key: "YOUR_SECRET_KEY"
    ttl: 3600
```

## Configuration Structure

### Global Settings

```yaml
# Default DNS provider for all domains (can be overridden per domain)
provider: porkbun

# Custom DNS servers for lookups (optional)
dns:
  - name: GoogleDNS
    ip: "8.8.8.8"
  - name: CloudflareDNS
    ip: "1.1.1.1"
```

### Domain Configuration

```yaml
domains:
  - name: example.com              # Domain name (required)
    provider: porkbun              # DNS provider (required)
    api_key: "your_api_key"        # Provider API key (required)
    secret_key: "your_secret_key"  # Provider secret (if required)
    ttl: 3600                      # DNS record TTL in seconds (optional, default: 600)

    # CIDR aggregation settings (optional)
    aggregation:
      enabled: true                     # Enable CIDR aggregation for this domain
      ipv4_max_prefix: 24              # Maximum IPv4 CIDR prefix (default: 24)
      ipv6_max_prefix: 64              # Maximum IPv6 CIDR prefix (default: 64)
      preserve_individual:             # IPs to never aggregate
        - "192.168.1.100"
        - "2001:db8::special"
```

## Provider-Specific Configuration

### Porkbun

```yaml
provider: porkbun

domains:
  - name: example.com
    provider: porkbun
    api_key: "pk1_1234567890abcdef1234567890abcdef"
    secret_key: "sk1_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
    ttl: 3600
```

**Required Fields:**
- `api_key`: Porkbun API key (starts with `pk1_`)
- `secret_key`: Porkbun secret key (starts with `sk1_`)

**Important**: Ensure your API keys have DNS management permissions and your IP is whitelisted.

### Future Providers (Extensible)

The configuration is designed to support additional providers:

```yaml
# Cloudflare example (when implemented)
domains:
  - name: example.com
    provider: cloudflare
    api_key: "your-cloudflare-api-token"
    ttl: 3600

# AWS Route53 example (when implemented)
domains:
  - name: example.com
    provider: route53
    api_key: "AKIA..."
    secret_key: "your-secret-access-key"
    region: "us-east-1"
    ttl: 3600
```

## Environment Variables

For enhanced security, use environment variables instead of storing credentials in configuration files:

### Setup

```bash
export SPF_FLATTENER_API_KEY="your_actual_api_key"
export SPF_FLATTENER_SECRET_KEY="your_actual_secret_key"
```

### Configuration with Environment Variables

```yaml
provider: porkbun

domains:
  - name: yourdomain.com
    provider: porkbun
    api_key: ""      # Will use SPF_FLATTENER_API_KEY
    secret_key: ""   # Will use SPF_FLATTENER_SECRET_KEY
    ttl: 3600
```

The application automatically uses environment variables when config file fields are empty.

## Multi-Domain Configuration

```yaml
provider: porkbun  # Default provider

domains:
  # Domain using default provider
  - name: example.com
    api_key: "pk1_example_key"
    secret_key: "sk1_example_secret"
    ttl: 3600

  # Domain with different provider
  - name: anotherdomain.org
    provider: porkbun  # Override default
    api_key: "pk1_another_key"
    secret_key: "sk1_another_secret"
    ttl: 1800

  # Domain with CIDR aggregation
  - name: enterprise.com
    api_key: "pk1_enterprise_key"
    secret_key: "sk1_enterprise_secret"
    ttl: 7200
    aggregation:
      enabled: true
      ipv4_max_prefix: 22  # Allow broader aggregation
      preserve_individual:
        - "203.0.113.100"  # Critical mail server
```

## CIDR Aggregation Configuration

CIDR aggregation combines contiguous IP addresses into efficient CIDR blocks:

### Global Enable (Command Line)
```bash
./spf-flattener flatten --aggregate
```

### Per-Domain Configuration
```yaml
domains:
  - name: example.com
    # ... other config ...
    aggregation:
      enabled: true                    # Override global --aggregate flag
      ipv4_max_prefix: 24             # Don't create networks larger than /24
      ipv6_max_prefix: 64             # Don't create networks larger than /64
      preserve_individual:            # Never aggregate these IPs
        - "192.168.1.100"             # Critical server
        - "203.0.113.50"              # Load balancer
        - "2001:db8::important"       # IPv6 service
```

### Aggregation Benefits
- Can reduce SPF record size by up to 97%
- Fewer DNS lookups required
- Better performance and reliability
- See [CIDR_AGGREGATION.md](CIDR_AGGREGATION.md) for detailed examples

## DNS Server Configuration

Configure custom DNS servers for SPF resolution:

```yaml
dns:
  - name: GoogleDNS
    ip: "8.8.8.8"
  - name: CloudflareDNS
    ip: "1.1.1.1"
  - name: CorporateDNS
    ip: "10.0.0.53"

domains:
  - name: example.com
    # ... provider config ...
```

**Use Cases:**
- Corporate environments with internal DNS
- Testing with specific DNS servers
- Avoiding DNS filtering or blocking

## Validation and Defaults

The application validates configuration and provides helpful defaults:

### Required Fields
- `domains[].name`: Domain name
- `domains[].provider`: DNS provider name
- `domains[].api_key`: Provider API key

### Default Values
- `ttl`: 600 seconds
- `aggregation.ipv4_max_prefix`: 24
- `aggregation.ipv6_max_prefix`: 64
- `aggregation.enabled`: false

### Validation Rules
- Domain names must be valid DNS names
- TTL must be between 60 and 86400 seconds
- CIDR prefixes must be within valid ranges
- API keys must not be empty (unless using environment variables)

## Configuration Examples

### Minimal Configuration
```yaml
provider: porkbun
domains:
  - name: example.com
    api_key: "pk1_key"
    secret_key: "sk1_secret"
```

### Production Configuration
```yaml
provider: porkbun

dns:
  - name: PrimaryDNS
    ip: "8.8.8.8"
  - name: SecondaryDNS
    ip: "1.1.1.1"

domains:
  - name: example.com
    provider: porkbun
    api_key: ""  # Use environment variable
    secret_key: ""  # Use environment variable
    ttl: 3600
    aggregation:
      enabled: true
      ipv4_max_prefix: 24
      preserve_individual:
        - "203.0.113.100"

  - name: staging.example.com
    provider: porkbun
    api_key: ""
    secret_key: ""
    ttl: 300  # Short TTL for staging
```

### Multi-Provider Configuration
```yaml
# When multiple providers are supported
domains:
  - name: example.com
    provider: porkbun
    api_key: "pk1_key"
    secret_key: "sk1_secret"
    ttl: 3600

  - name: another.org
    provider: cloudflare
    api_key: "cloudflare_token"
    ttl: 7200
```

## Troubleshooting Configuration

### Common Issues

1. **Invalid YAML Syntax**
   ```bash
   # Test configuration syntax
   ./spf-flattener flatten --dry-run --config config.yaml
   ```

2. **Missing API Permissions**
   ```bash
   # Test API connectivity
   ./spf-flattener ping --config config.yaml
   ```

3. **Environment Variables Not Set**
   ```bash
   # Check environment variables
   echo $SPF_FLATTENER_API_KEY
   echo $SPF_FLATTENER_SECRET_KEY
   ```

### Validation Commands

```bash
# Validate configuration file
./spf-flattener flatten --dry-run --verbose

# Test API connectivity for all domains
./spf-flattener ping

# Debug configuration loading
./spf-flattener flatten --debug --dry-run
```

## Security Best Practices

1. **Use Environment Variables**: Store API credentials in environment variables, not config files
2. **File Permissions**: Restrict config file permissions (`chmod 600 config.yaml`)
3. **API Key Rotation**: Regularly rotate API keys
4. **Principle of Least Privilege**: Use API keys with minimal required permissions
5. **IP Whitelisting**: Configure provider IP restrictions when available