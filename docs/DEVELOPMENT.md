# Development Guide

This guide covers building, developing, and contributing to the SPF Flattener project.

## Prerequisites

- **Go 1.21+** (project uses Go 1.24.2)
- **Git** for version control
- **Make** (optional, for automation)

## Building from Source

### Quick Build

```bash
# Clone the repository
git clone https://github.com/dean-jl/spf-flattener
cd spf-flattener

# Build the project
go build -o spf-flattener ./cmd/spf-flattener

# Or use the build script
./build-it.sh
```

### Build Commands

```bash
# Standard build
go build -o spf-flattener ./cmd/spf-flattener

# Build with version information
go build -ldflags "-X main.version=$(git describe --tags)" -o spf-flattener ./cmd/spf-flattener

# Cross-compilation examples
GOOS=linux GOARCH=amd64 go build -o spf-flattener-linux-amd64 ./cmd/spf-flattener
GOOS=windows GOARCH=amd64 go build -o spf-flattener-windows-amd64.exe ./cmd/spf-flattener
GOOS=darwin GOARCH=arm64 go build -o spf-flattener-darwin-arm64 ./cmd/spf-flattener
```

### Build Script

The `build-it.sh` script provides additional build options:

```bash
# Standard build
./build-it.sh

# Clean build (removes existing binary first)
./build-it.sh clean

# Build with debug symbols
./build-it.sh debug
```

## Development Environment

### Project Structure

```
spf-flattener/
├── cmd/spf-flattener/     # CLI application entry point
│   ├── main.go           # Root command and CLI setup
│   ├── flatten.go        # Flatten command implementation
│   ├── ping.go          # Ping command for API testing
│   ├── backup.go        # Backup/Restore command implementation
│   └── utils.go         # Utility functions
├── internal/            # Core application logic (private packages)
│   ├── backup/          # DNS backup and restore system
│   │   ├── backup.go   # Backup manager core logic
│   │   ├── types.go    # Data structures and interfaces
│   │   ├── formats.go  # Format handlers (JSON, TXT)
│   │   └── validation.go # DNS record validation
│   ├── config/          # Configuration handling
│   │   ├── config.go   # Configuration parsing and validation
│   │   └── types.go    # Configuration data structures
│   ├── spf/             # SPF record processing
│   │   ├── core_flatten.go # Core SPF flattening logic
│   │   ├── normalize.go    # SPF record normalization
│   │   ├── split.go        # Large record splitting
│   │   └── types.go        # SPF-related types
│   ├── porkbun/         # Porkbun DNS provider implementation
│   │   ├── client.go    # API client implementation
│   │   └── types.go     # Provider-specific types
│   └── processor/       # Domain processing business logic
├── docs/                # Documentation
├── tests/               # Test files and test data
└── examples/            # Example configurations
```

### Key Architecture Principles

1. **Separation of Concerns**: CLI, business logic, and providers are separate
2. **Interface-Driven**: Uses interfaces for extensibility (`DNSProvider`, `DNSAPIClient`)
3. **Context-Aware**: All operations support context for cancellation/timeouts
4. **Provider-Agnostic**: Core logic doesn't depend on specific DNS providers
5. **Safe by Default**: Dry-run mode is the default for all operations

## Code Quality

### Formatting and Linting

```bash
# Format code
go fmt ./...

# Check for common issues
go vet ./...

# Run static analysis (if golangci-lint is installed)
golangci-lint run

# Check for security issues (if gosec is installed)
gosec ./...
```

### Dependencies

```bash
# Download dependencies
go mod download

# Verify dependencies
go mod verify

# Clean up unused dependencies
go mod tidy

# Update dependencies
go get -u ./...
```

## Testing

See [TESTING.md](TESTING.md) for comprehensive testing documentation.

### Quick Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/spf/
go test ./internal/config/
go test ./internal/porkbun/

# Run benchmarks
go test -bench=. ./...
```

### Test Categories

- **Unit Tests**: Core logic, algorithms, and utilities
- **Integration Tests**: API clients with mocked responses
- **Manual Verification**: CLI workflows and real-world scenarios

## Development Workflow

### 1. Feature Development

```bash
# Create feature branch
git checkout -b feature/new-provider-support

# Make changes
# ... development work ...

# Test changes
go test ./...
go run ./cmd/spf-flattener flatten --dry-run

# Format and lint
go fmt ./...
go vet ./...

# Commit changes
git add .
git commit -m "Add support for new DNS provider"
```

### 2. Adding New DNS Providers

See [dns-provider-guide.md](DNS-PROVIDER-GUIDE.md) for detailed instructions.

Quick checklist:
- [ ] Create provider package in `internal/[provider]/`
- [ ] Implement `DNSAPIClient` interface
- [ ] Add provider to switch statement in CLI
- [ ] Write unit tests with mocked API responses
- [ ] Update configuration documentation
- [ ] Test manually with real API credentials

### 3. Code Style Guidelines

#### Go Style
- Follow standard Go conventions and idioms
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Add package documentation
- Handle errors explicitly

#### Error Handling
```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to retrieve DNS records for %s: %w", domain, err)
}

// Good: Use structured logging
log.Printf("Processing domain %s with provider %s", domain, provider)
```

#### Interface Design
```go
// Follow existing interface patterns
type DNSProvider interface {
    LookupTXT(ctx context.Context, domain string) ([]string, error)
    LookupIP(ctx context.Context, domain string) ([]net.IP, error)
    Close() error
}
```

#### Testing Patterns
```go
// Use table-driven tests
func TestSPFFlattening(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []string
        wantErr  bool
    }{
        {
            name:     "basic include",
            input:    "v=spf1 include:_spf.google.com ~all",
            expected: []string{"ip4:209.85.128.0/17", "ip4:64.233.160.0/19"},
            wantErr:  false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FlattenSPF(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FlattenSPF() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // ... assertions
        })
    }
}
```

## Performance Optimization

### Key Performance Features

1. **Provider-Aware Parallel Processing**: Different DNS providers process concurrently
2. **Rate Limiting**: Standardized `golang.org/x/time/rate` implementation
3. **DNS Client Reuse**: Connection pooling for better performance
4. **Context Timeouts**: Prevent hanging operations
5. **Caching**: DNS response caching for repeated lookups

### Benchmarking

```bash
# Run benchmarks
go test -bench=. ./internal/spf/
go test -bench=. ./internal/backup/

# Profile CPU usage
go test -bench=. -cpuprofile=cpu.prof ./internal/spf/
go tool pprof cpu.prof

# Profile memory usage
go test -bench=. -memprofile=mem.prof ./internal/spf/
go tool pprof mem.prof
```

### Performance Guidelines

- Use context timeouts for all external operations
- Implement provider-specific rate limiting
- Cache DNS responses when appropriate
- Use worker pools for concurrent operations
- Profile before optimizing

## Security Considerations

### API Key Management

```go
// Good: Check for empty API keys
if domain.ApiKey == "" && os.Getenv("SPF_FLATTENER_API_KEY") == "" {
    return fmt.Errorf("API key not provided for domain %s", domain.Name)
}

// Good: Don't log sensitive data
log.Printf("Processing domain %s", domain.Name) // Don't log API keys
```

### Input Validation

```go
// Validate domain names
if !isValidDomain(domain) {
    return fmt.Errorf("invalid domain name: %s", domain)
}

// Validate DNS responses
if len(records) == 0 {
    return fmt.Errorf("no DNS records returned for %s", domain)
}
```

### Safe Defaults

- Default to dry-run mode
- Validate all configuration before operations
- Use HTTPS for all API communications
- Implement proper error handling and recovery

## Release Process

### Version Management

```bash
# Tag a new version
git tag -a v1.2.3 -m "Release version 1.2.3"
git push origin v1.2.3

# Build release binaries
./scripts/build-release.sh v1.2.3
```

### Pre-Release Checklist

- [ ] All tests pass (`go test ./...`)
- [ ] Code is formatted (`go fmt ./...`)
- [ ] No linting issues (`go vet ./...`)
- [ ] Documentation is updated
- [ ] Manual testing completed
- [ ] Security review completed
- [ ] Performance benchmarks run

## Debugging

### Debug Mode

```bash
# Enable debug logging
./spf-flattener flatten --debug --dry-run

# Debug specific operations
./spf-flattener ping --debug
./spf-flattener export --debug --dry-run
```

### Common Issues

1. **Configuration Errors**: Use `--debug` to see detailed config parsing
2. **API Connectivity**: Use `ping` command to test credentials
3. **DNS Resolution**: Check custom DNS server configuration
4. **Rate Limiting**: Watch for rate limit messages in verbose mode

### Debugging Tools

```bash
# Use Go's built-in tools
go tool trace trace.out
go tool pprof cpu.prof
go tool pprof mem.prof

# Network debugging
tcpdump -i any port 53  # Monitor DNS traffic
```

## Contributing

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Update documentation as needed
7. Submit a pull request

### Code Review Guidelines

- Follow Go best practices and idioms
- Include comprehensive tests
- Update documentation for user-facing changes
- Consider backward compatibility
- Handle errors appropriately

### Community Guidelines

- Be respectful and constructive
- Follow the project's coding standards
- Write clear commit messages
- Respond to feedback promptly
- Help maintain project quality

## Tools and IDE Setup

### VS Code

Recommended extensions:
- Go extension pack
- Go test explorer
- Go outliner

### GoLand/IntelliJ

Built-in Go support with:
- Integrated testing
- Debugging capabilities
- Code analysis
- Refactoring tools

### Vim/Neovim

Use vim-go plugin for:
- Syntax highlighting
- Code completion
- Integrated testing
- Go-specific commands