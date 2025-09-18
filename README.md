[![Go Version](https://img.shields.io/badge/go-1.24.2%2B-blue.svg)](https://golang.org/dl/) [![License](https://img.shields.io/github/license/dean-jl/spf-flattener.svg)](LICENSE)

# SPF Flattener

A comprehensive Go CLI tool for SPF record management, DNS backup/restore, and multi-domain operations. Intelligently flattens SPF records only when necessary (>10 DNS lookups per RFC 7208), with optional CIDR aggregation and extensible provider architecture.

## ✨ Key Features

- **🧠 Intelligent SPF Flattening**: Only flattens when RFC 7208 lookup limits are exceeded
- **📦 CIDR Aggregation**: Reduce record size by up to 97% with intelligent IP grouping
- **💾 DNS Backup & Restore**: Complete backup/restore with conflict resolution
- **🏗️ Multi-Provider Support**: Extensible architecture (currently supports Porkbun)
- **🔄 Provider-Aware Processing**: Parallel processing optimized for multiple DNS providers
- **🛡️ Safe by Default**: Dry-run mode prevents accidental changes
- **⚡ Performance Optimized**: Rate limiting, caching, and concurrent operations

## 🚀 Quick Start

### Installation
```bash
go install github.com/dean-jl/spf-flattener@latest
```

### Basic Configuration
Create `config.yaml`:
```yaml
provider: porkbun
domains:
  - name: yourdomain.com
    provider: porkbun
    api_key: "YOUR_API_KEY"
    secret_key: "YOUR_SECRET_KEY"
    ttl: 3600
```

### Basic Usage
```bash
# Test configuration and see what would change
./spf-flattener flatten --dry-run --verbose

# Test API connectivity
./spf-flattener ping

# Apply SPF flattening (when ready)
./spf-flattener flatten --production

# Backup DNS records
./spf-flattener export --production

# Enable CIDR aggregation for smaller records
./spf-flattener flatten --production --aggregate
```

## 📚 Documentation

- **[📖 Complete Usage Guide](docs/USAGE.md)** - All commands, flags, and examples
- **[⚙️ Configuration Guide](docs/CONFIGURATION.md)** - Detailed configuration options and examples
- **[🧪 Testing Strategy](docs/TESTING.md)** - Testing approach and verification procedures
- **[🛠️ Development Guide](docs/DEVELOPMENT.md)** - Building, contributing, and extending
- **[🔌 DNS Provider Guide](docs/DNS-PROVIDER-GUIDE.md)** - Adding new DNS provider support
- **[📊 CIDR Aggregation](docs/CIDR_AGGREGATION.md)** - IP optimization details

## 🎯 Core Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `flatten` | Process SPF records | `./spf-flattener flatten --production` |
| `ping` | Test API connectivity | `./spf-flattener ping` |
| `export` | Backup DNS records | `./spf-flattener export --production` |
| `import` | Restore DNS records | `./spf-flattener import --files backup.json --production` |

## 🔒 Security Features

- **Environment Variables**: Store API keys securely outside config files
- **Input Validation**: RFC-compliant validation for all inputs
- **Secure Logging**: Never logs sensitive data (API keys, secrets)
- **Safe Defaults**: Dry-run mode prevents accidental changes

**Secure credential setup:**
```bash
export SPF_FLATTENER_API_KEY="your_api_key"
export SPF_FLATTENER_SECRET_KEY="your_secret_key"
```

## 🏗️ Architecture Highlights

- **Modern Go Structure**: Uses `internal/` packages with clean interfaces
- **Provider-Agnostic Core**: Easy to add new DNS providers
- **Context-Aware**: All operations support cancellation and timeouts
- **Hybrid Testing**: Unit tests + manual verification approach
- **Performance Optimized**: Provider-aware parallel processing, rate limiting

## 🤝 Contributing

This project welcomes contributions! See the [Development Guide](docs/DEVELOPMENT.md) for:
- Building from source
- Adding new DNS providers
- Code style guidelines
- Testing procedures

**Quick contribution setup:**
```bash
git clone https://github.com/dean-jl/spf-flattener
cd spf-flattener
go build -o spf-flattener ./cmd/spf-flattener
go test ./...
```

## ⚠️ Important Notes

- Only flattens SPF records exceeding 10 DNS lookups (RFC 7208 compliant)
- Currently supports Porkbun (extensible for other providers)
- Requires API keys with DNS read/write permissions
- Consider trade-offs before flattening your SPF records

## 📝 License

Licensed under [GNU AGPLv3](https://www.gnu.org/licenses/agpl-3.0.en.html)

---

**Need help?** Check the [complete documentation](docs/) or run `./spf-flattener --help` for command-specific guidance.




