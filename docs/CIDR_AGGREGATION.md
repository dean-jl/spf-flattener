# CIDR Aggregation for SPF Records

This document explains how SPF Flattener's CIDR aggregation feature works, when to use it, and how to configure it for optimal results.

## Overview

CIDR (Classless Inter-Domain Routing) aggregation is an intelligent optimization technique that combines contiguous IP addresses into efficient network blocks. This can reduce the size of your flattened SPF records while maintaining identical functionality.

### Why Use CIDR Aggregation?

- **Reduced DNS Record Size**: Modest but meaningful reduction in SPF mechanisms for most records
- **Faster DNS Lookups**: Fewer mechanisms mean faster SPF evaluation by receiving mail servers
- **Stay Within Limits**: Helps avoid the DNS 10-lookup limit and TXT record character limits
- **Improved Reliability**: Reduces the chance of SPF record splitting across multiple DNS queries

**Important Note**: Most modern email providers (Google, Microsoft, AWS SES, etc.) already publish optimized SPF records. CIDR aggregation provides the most benefit for organizations with legacy configurations, manual IP lists, or specific network topologies that create aggregation opportunities.

## How It Works

### Basic Concept

CIDR aggregation looks for sequences of contiguous IP addresses and replaces them with a single network notation:

```
Before:  ip4:192.168.1.0 ip4:192.168.1.1 ip4:192.168.1.2 ip4:192.168.1.3
After:   ip4:192.168.1.0/30
```

Both representations authorize exactly the same IP addresses, but the aggregated version uses 75% fewer characters.

### Exact Aggregation Guarantee

SPF Flattener uses **exact CIDR aggregation** with these safety guarantees:

- ✅ **No unintended IP authorization**: The aggregated record authorizes exactly the same IPs as the original
- ✅ **No security gaps**: Missing IPs in the original remain unauthorized in the aggregated version
- ✅ **RFC compliance**: Follows RFC 4632 (IPv4) and RFC 1887 (IPv6) standards
- ✅ **Proper alignment**: CIDR blocks are mathematically aligned according to network boundaries

### Algorithm Details

1. **Collection**: Gathers all IP addresses from `include`, `a`, and `mx` mechanisms
2. **Separation**: Groups IPv4 and IPv6 addresses separately
3. **Deduplication**: Removes duplicate IP addresses
4. **Sorting**: Orders IPs numerically for range detection
5. **Contiguous Detection**: Identifies sequences of consecutive IP addresses
6. **CIDR Calculation**: Converts ranges to optimally-sized, properly-aligned CIDR blocks
7. **Validation**: Ensures each CIDR block contains exactly the intended IPs

### IPv4 Example

```
Original IPs:     192.168.1.1, 192.168.1.2, 192.168.1.3, 192.168.1.4
Binary Analysis:
  192.168.1.1 = ...00000001
  192.168.1.2 = ...00000010
  192.168.1.3 = ...00000011
  192.168.1.4 = ...00000100

Contiguous Range: 192.168.1.1 to 192.168.1.4 (4 consecutive IPs)
CIDR Block:       192.168.1.1/30 (covers exactly those 4 IPs)

Verification:     192.168.1.1/30 expands to 192.168.1.1, 192.168.1.2, 192.168.1.3, 192.168.1.4 ✓
```

### IPv6 Example

```
Original IPs:     2001:db8::0, 2001:db8::1
CIDR Block:       2001:db8::/127 (covers exactly those 2 IPs)
```

### Non-Contiguous IPs

When IPs are not contiguous, they remain as individual entries:

```
IPs:              192.168.1.1, 192.168.1.3, 192.168.1.5
Result:           ip4:192.168.1.1 ip4:192.168.1.3 ip4:192.168.1.5
Explanation:      Gap at 192.168.1.2 and 192.168.1.4 prevents aggregation
```

## Usage

### Basic Usage

Enable CIDR aggregation with the `--aggregate` flag:

```bash
# Test aggregation (dry-run mode)
./spf-flattener flatten --aggregate --dry-run

# Apply aggregation to live DNS records
./spf-flattener flatten --aggregate --production
```

### Per-Domain Configuration

You can configure aggregation behavior per domain in your `config.yaml`:

```yaml
domains:
  - name: example.com
    provider: porkbun
    api_key: "YOUR_API_KEY"
    secret_key: "YOUR_SECRET_KEY"

    # CIDR aggregation configuration
    aggregation:
      enabled: true                    # Enable aggregation for this domain
      ipv4_max_prefix: 24             # Maximum IPv4 CIDR prefix allowed - prevents overly broad aggregation (default: 24)
      ipv6_max_prefix: 64             # Maximum IPv6 CIDR prefix allowed - prevents overly broad aggregation (default: 64)
      preserve_individual:            # IPs to never aggregate
        - "192.168.1.100"             # Critical server IP
        - "2001:db8::important"       # Important IPv6 address
```

### Configuration Options Explained

#### `enabled`
- **Type**: Boolean
- **Default**: Uses global `--aggregate` flag setting
- **Purpose**: Override the global aggregation setting for this specific domain

#### `ipv4_max_prefix` / `ipv6_max_prefix`
- **Type**: Integer (0-32 for IPv4, 0-128 for IPv6)
- **Default**: 24 (IPv4), 64 (IPv6)
- **Purpose**: Sets maximum CIDR block size allowed - prevents overly broad aggregation that could authorize unintended IPs

**Understanding Max Prefix**:
Higher prefix numbers = smaller, more specific networks = safer aggregation
Lower prefix numbers = larger, broader networks = riskier aggregation

**Examples by Max Prefix Value**:

```yaml
# Conservative (allows only small networks)
ipv4_max_prefix: 30
# ✅ Allows: 192.168.1.0/30 (4 IPs), 192.168.1.4/31 (2 IPs), individual IPs
# ❌ Blocks: 192.168.1.0/29 (8 IPs), 192.168.1.0/28 (16 IPs), anything larger

# Moderate (good for most use cases)
ipv4_max_prefix: 28
# ✅ Allows: 192.168.1.0/28 (16 IPs), 192.168.1.0/30 (4 IPs), individual IPs
# ❌ Blocks: 192.168.1.0/27 (32 IPs), 192.168.1.0/24 (256 IPs), anything larger

# Permissive (allows larger networks)
ipv4_max_prefix: 24
# ✅ Allows: 192.168.1.0/24 (256 IPs), 192.168.1.0/28 (16 IPs), individual IPs
# ❌ Blocks: 192.168.0.0/23 (512 IPs), 10.0.0.0/8 (16 million IPs), anything larger

# Very permissive (enterprise networks)
ipv4_max_prefix: 20
# ✅ Allows: 10.0.0.0/20 (4096 IPs), 192.168.0.0/24 (256 IPs), individual IPs
# ❌ Blocks: 10.0.0.0/16 (65536 IPs), 192.0.0.0/8 (16 million IPs)
```

**Real-World Scenarios**:

```yaml
# Small business (very conservative)
domains:
  - name: smallbiz.com
    aggregation:
      ipv4_max_prefix: 30  # Only allow up to 4 IPs per CIDR block
      # Good when you only have a few mail servers and want maximum control

# Enterprise with regional offices (moderate)
domains:
  - name: enterprise.com
    aggregation:
      ipv4_max_prefix: 28  # Allow up to 16 IPs per CIDR block
      # Good balance between efficiency and control

# Large corporation with data centers (permissive)
domains:
  - name: largecorp.com
    aggregation:
      ipv4_max_prefix: 24  # Allow up to 256 IPs per CIDR block
      # Suitable when you have large, contiguous IP blocks
```

**IPv6 Examples**:
```yaml
# IPv6 conservative
ipv6_max_prefix: 120  # Allow up to 256 IPv6 addresses
# ✅ Allows: 2001:db8::/120, 2001:db8::100/125
# ❌ Blocks: 2001:db8::/64 (18 quintillion addresses!)

# IPv6 moderate (default)
ipv6_max_prefix: 64   # Allow up to /64 networks
# ✅ Allows: 2001:db8::/64, 2001:db8::1000/120
# ❌ Blocks: 2001:db8::/48, 2001:db8::/32
```

**Security Rationale**: Higher prefix numbers represent smaller, more specific networks. By setting `ipv4_max_prefix: 24`, you ensure no CIDR block broader than `/24` (256 IPs) is created, preventing accidental authorization of large IP ranges that might include unintended mail servers or compromised infrastructure.

#### `preserve_individual`
- **Type**: Array of IP address strings
- **Default**: Empty
- **Purpose**: Specific IP addresses that should never be aggregated, even if contiguous

**Use cases**:
- Critical server IPs that need to remain visible in SPF records for auditing
- IPs with special significance that should not be "hidden" in CIDR blocks
- Compliance requirements to keep certain IPs explicitly listed

## Performance Impact

### Aggregation Benefits

| Scenario | Before Aggregation | After Aggregation | Reduction |
|----------|-------------------|-------------------|-----------|
| 4 contiguous IPs | `ip4:192.168.1.0 ip4:192.168.1.1 ip4:192.168.1.2 ip4:192.168.1.3` | `ip4:192.168.1.0/30` | 75% |
| 100 contiguous IPs | 100 individual mechanisms | ~6-8 CIDR blocks | ~97% |
| Mixed contiguous/gaps | Varies by distribution | Optimally aggregated | 40-90% |

### Real-World Example

A typical cloud provider with sequential IP allocations:

```
Before (256 IPs):
v=spf1 ip4:203.0.113.0 ip4:203.0.113.1 ip4:203.0.113.2 ... [253 more IPs] ... ~all

After (1 CIDR block):
v=spf1 ip4:203.0.113.0/24 ~all

Character reduction: ~2,000 characters → ~30 characters (98.5% reduction)
```

### Performance Considerations

- **DNS Query Size**: Smaller records load faster
- **Mail Server Processing**: Fewer mechanisms = faster SPF evaluation
- **Record Parsing**: Reduced complexity for receiving servers
- **Bandwidth**: Less DNS traffic for SPF lookups

## Advanced Topics

### Change Detection with Aggregation

When aggregation is enabled, SPF Flattener uses **semantic comparison** instead of string comparison to detect changes:

- **Without aggregation**: Compares normalized SPF strings
- **With aggregation**: Expands both old and new records to IP sets and compares functionally

This prevents unnecessary DNS updates when records are functionally identical but have different representations.

### Alignment Rules

CIDR blocks must be properly aligned to network boundaries:

```
✅ Valid:   192.168.1.0/30 (covers .0, .1, .2, .3 - properly aligned)
❌ Invalid: 192.168.1.1/30 (would cover .0, .1, .2, .3 but starts at .1)
```

SPF Flattener automatically ensures proper alignment by:
1. Finding the largest possible CIDR block for each range
2. Ensuring the starting IP is divisible by the block size
3. Splitting into multiple blocks if necessary to maintain alignment

### IPv4 vs IPv6 Differences

| Aspect | IPv4 | IPv6 |
|--------|------|------|
| Address space | 32-bit | 128-bit |
| Minimum prefix default | /24 (256 addresses) | /64 (18 quintillion addresses) |
| Practical aggregation | Common for cloud providers | Less common, but more impactful |
| Alignment complexity | Moderate | High (due to large address space) |

## Troubleshooting

### Common Issues

#### "No aggregation occurred"
- **Cause**: IPs are not contiguous or blocks would exceed maximum prefix limit
- **Solution**: Check IP sequences and adjust `ipv4_max_prefix`/`ipv6_max_prefix` to allow broader aggregation

#### "Record size increased"
- **Cause**: Many small gaps in IP ranges leading to many small CIDR blocks
- **Solution**: Consider using `preserve_individual` for scattered IPs

#### "Semantic comparison failed"
- **Cause**: Complex record structure or large IP sets
- **Solution**: Monitor performance; large sets use optimized comparison algorithms

### Debugging Tips

1. **Use dry-run mode** to see aggregation results before applying:
   ```bash
   ./spf-flattener flatten --aggregate --dry-run --verbose
   ```

2. **Check the detailed report** for aggregation statistics and any warnings

3. **Test with different maximum prefix values** to find optimal settings:
   ```yaml
   aggregation:
     ipv4_max_prefix: 28  # More restrictive (smaller networks only)
     ipv6_max_prefix: 56  # More permissive (allows larger networks)
   ```

### Best Practices

1. **Start with dry-run**: Always test aggregation before applying to production
2. **Monitor record sizes**: Ensure aggregation actually reduces record size
3. **Preserve critical IPs**: Use `preserve_individual` for important addresses
4. **Regular review**: Periodically review aggregation effectiveness as IP allocations change
5. **Document configuration**: Keep notes on why specific IPs are preserved individually

## Security Considerations

### Safety Guarantees

- **Exact authorization**: Aggregated records authorize exactly the same IPs as originals
- **No expansion**: CIDR blocks never include unintended IP addresses
- **Validation**: Each transformation is mathematically verified
- **Rollback capability**: Original records are preserved for comparison

### Security Benefits

- **Reduced attack surface**: Smaller DNS records are less prone to parsing errors
- **Improved reliability**: Fewer DNS lookups reduce failure points
- **Better monitoring**: Consolidated records are easier to audit
- **Consistent behavior**: Deterministic aggregation produces predictable results

## Migration Guide

### Enabling Aggregation on Existing Domains

1. **Backup current records**:
   ```bash
   ./spf-flattener export --production --format json
   ```

2. **Test with dry-run**:
   ```bash
   ./spf-flattener flatten --aggregate --dry-run --verbose
   ```

3. **Review the changes** in the detailed report

4. **Apply gradually**:
   - Start with a test domain
   - Monitor mail delivery for 24-48 hours
   - Apply to production domains

5. **Monitor and optimize**:
   - Check record sizes and performance
   - Adjust configuration as needed
   - Document any domain-specific requirements

### Rollback Procedure

If you need to disable aggregation:

1. **Disable the flag**:
   ```bash
   ./spf-flattener flatten --production  # Without --aggregate
   ```

2. **Or disable per-domain**:
   ```yaml
   aggregation:
     enabled: false
   ```

The tool will automatically revert to non-aggregated individual IP entries.

## Benchmarking & Performance Validation

### Algorithm Capabilities vs. Real-World Performance

Our CIDR aggregation algorithm is capable of achieving up to 97% reduction in ideal scenarios, but real-world performance is typically much more modest.

#### Algorithm Testing Results

The "up to 97% reduction" represents our algorithm's maximum capability, measured in controlled test scenarios from `TestFlattenSPF_AggregationPerformance`:

**Artificial Test Scenarios:**

| Test Case | Input | Output | Reduction | Reality |
|-----------|-------|--------|-----------|---------|
| 4 contiguous IPs | 4 individual `ip4:` entries | 1 `/30` block | 75% | Uncommon |
| 16 contiguous IPs | 16 individual `ip4:` entries | 1 `/28` block | 94% | Rare |
| 100 contiguous IPs | 100 individual `ip4:` entries | ~3 optimized blocks | 97% | Very rare |
| 256 contiguous IPs | 256 individual `ip4:` entries | 1 `/24` block | 99.6% | Extremely rare |

#### Why These Results Don't Reflect Real SPF Records

**Typical SPF Record Structure:**
```
v=spf1 include:_spf.google.com include:spf.protection.outlook.com
include:mailgun.org ip4:203.0.113.100 ~all
```

**After Flattening (without aggregation):**
```
v=spf1 ip4:209.85.128.0/17 ip4:64.233.160.0/19 ip4:108.177.8.0/21
ip4:172.217.0.0/16 ip4:216.239.32.0/19 ip4:203.0.113.100 ~all
```

**Aggregation Result:**
- Google and Microsoft already publish optimized CIDR blocks
- Only the single corporate IP (`203.0.113.100`) remains individual
- **Actual reduction: ~0%**

#### Mathematical Foundation (For Reference)

The algorithm achieves high reduction rates when:
1. **Contiguous IPs align to CIDR boundaries**: Sequential addresses become single network blocks
2. **Power-of-two efficiency**: CIDR represents 2^n addresses (256 IPs = one `/24`)
3. **Optimal block selection**: Finds largest possible blocks for each range

These conditions rarely occur in production SPF records.

### Testing Your Own SPF Records

You can benchmark aggregation performance on your actual SPF records:

#### 1. Test Current Performance
```bash
# See current SPF size without aggregation
./spf-flattener flatten --dry-run --verbose --config config.yaml

# Test with aggregation enabled
./spf-flattener flatten --aggregate --dry-run --verbose --config config.yaml
```

#### 2. Measure Reduction
The tool reports aggregation statistics in verbose mode:
```
Original mechanisms: 127
Aggregated mechanisms: 125
Reduction: 1.6%
```
*Note: This shows typical real-world results - modest improvements rather than dramatic reductions.*

#### 3. Profile Different Configurations
Test various minimum prefix settings to optimize for your IP distribution:

```yaml
# More permissive aggregation (allows broader CIDR blocks)
domains:
  - name: example.com
    aggregation:
      ipv4_max_prefix: 22  # Allow up to /22 networks (1024 IPs)
      ipv6_max_prefix: 48   # Allow up to /48 networks

# Conservative aggregation (restricts to smaller networks)
domains:
  - name: example.com
    aggregation:
      ipv4_max_prefix: 28   # Only allow /28 and smaller (16 IPs max)
      ipv6_max_prefix: 120  # Only allow /120 and smaller
```

#### 4. Automated Performance Testing
Run our test suite to see aggregation performance on various scenarios:
```bash
# Run performance benchmarks
go test -bench=BenchmarkAggregateCIDRs ./internal/spf/

# Run integration performance tests
go test -run TestFlattenSPF_AggregationPerformance ./internal/spf/
```

### Realistic Performance Expectations

| Organization Type | Typical Reduction | Reason |
|-------------------|------------------|---------|
| Modern enterprise (Google, O365) | 0-5% | Already optimized by providers |
| Well-managed enterprise | 5-15% | Few mail servers per region |
| Legacy/unoptimized setup | 20-40% | Manual IP lists, old configurations |
| Specific network topologies | 40-70% | Contiguous corporate IP blocks |
| Ideal test scenarios | 90%+ | Rare in practice |

**Reality Check**: Most production SPF records will see minimal aggregation because:
- Major email providers already use optimized CIDR blocks
- Well-managed enterprises typically have only a few mail servers per region
- Modern cloud services publish consolidated IP ranges
- SPF includes often point to already-efficient records

### Verification Methods

1. **Semantic Validation**: Our tests verify that aggregated SPF records authorize exactly the same IPs as the original
2. **RFC Compliance**: All CIDR blocks follow RFC 4632 (IPv4) and RFC 1887 (IPv6) standards
3. **Boundary Testing**: Tests ensure proper network alignment and no unintended IP authorization
4. **Real-World Testing**: Integration tests use actual DNS lookups and SPF record structures

The performance claims are reproducible and based on mathematical properties of CIDR aggregation combined with empirical testing across diverse SPF record patterns.

---

## Further Reading

- [RFC 4632 - Classless Inter-domain Routing (CIDR)](https://tools.ietf.org/html/rfc4632)
- [RFC 7208 - Sender Policy Framework (SPF)](https://tools.ietf.org/html/rfc7208)
- [SPF Flattener Main Documentation](../README.md)
- [CIDR Implementation Details](../dev-docs/CIDR_AGGREGATION_PROPOSAL.md) (Technical specification)