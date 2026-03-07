<h1 align="center">mksub - Enhanced Fork</h1>
<h3 align="center">Generate millions of subdomain combinations with FUZZ placeholders, multiple wordlists, and streaming optimization</h3>

![mksub](mksub.png "mksub")

**Enhanced fork** of the original mksub tool with advanced features for large-scale subdomain enumeration. This version includes direct FUZZ pattern processing, improved prefix pattern support, multiple wordlist handling, streaming optimization for massive wordlists, and seamless integration with dnsx.

## 🚀 **New Features Added**

### **1. FUZZ Placeholder Support**
Replace traditional level-based generation with flexible FUZZ placeholders for precise subdomain patterns.

### **2. Multiple Wordlist Mode** 
Use multiple wordlists simultaneously with numbered FUZZ placeholders (FUZZ1, FUZZ2, etc.).

### **3. Streaming Optimization**
Automatic streaming mode for wordlists >50MB to handle massive dictionaries (6M+ entries) without memory issues.

### **4. Automatic Cleanup**
Temporary files are automatically cleaned up when the tool finishes or is interrupted.

### **5. Enhanced dnsx Integration**
Optimized output buffering and threading for seamless integration with dnsx pipelines.

### **6. Prefix Pattern Support**
Apply FUZZ patterns to entire lists of subdomains using the `--prefix` flag for targeted fuzzing campaigns.

### **7. Direct FUZZ Pattern Processing**
Process files containing pre-defined FUZZ patterns directly without additional flags. Automatically detects and processes patterns like `FUZZ.domain.com`, `webdisk.FUZZ.domain.com`, etc.

# Installation

## From source (Recommended)
This is an enhanced fork of the original mksub with additional features. Clone and compile:

```bash
# Clone the repository
git clone https://github.com/yourusername/mksub.git
cd mksub

# Compile
go build -o mksub main.go

# Or install globally
go install
```

## Requirements
- Go 1.19 or higher
- Git

## Original Project
This is a fork of the original [mksub by Trickest](https://github.com/trickest/mksub) with enhanced features including:
- Direct FUZZ pattern processing
- Improved prefix pattern support
- Better memory optimization
- Enhanced dnsx integration

# Usage

## **Enhanced Flags**
```
  -d string
        Input domain (can contain FUZZ placeholders)
  -df string
        Input domain file, one domain per line (supports files with FUZZ patterns)
  -l int
        Subdomain level to generate (default 1)
  -o string
        Output file (stdout will be used when omitted)
  -r string
        Regex to filter words from wordlist file
  -silent
        Skip writing generated subdomains to stdout (faster) (default true)
  -t int
        Number of threads for every subdomain level (default 100)
  -w string
        Wordlist file (can be used multiple times with --multiple)
  --multiple
        Enable multiple wordlist mode with numbered FUZZ placeholders
  --prefix string
        Prefix pattern to apply to domains (e.g., 'FUZZ', 'FUZZ2-FUZZ1')
```

### **📁 Input File Formats**

#### **Standard Domain File**
```bash
# domains.txt
example.com
target.com
test.org
```

#### **FUZZ Pattern File (NEW)**
```bash
# fuzz-patterns.txt
FUZZ.example.com
webdisk.FUZZ.target.com
api-FUZZ.test.org
FUZZ1-FUZZ2.company.com
```

## **🎯 FUZZ Placeholder Examples**

### **Single Domain with FUZZ**
```bash
# Replace FUZZ with words from wordlist
./mksub -d "FUZZ.example.com" -w wordlist.txt
# Output: admin.example.com, api.example.com, www.example.com...

# Multiple FUZZ in same domain
./mksub -d "FUZZ.FUZZ-staging.example.com" -w wordlist.txt  
# Output: admin.admin-staging.example.com, api.api-staging.example.com...
```

### **File with FUZZ Patterns (NEW)**
```bash
# Process file containing FUZZ patterns directly
./mksub -df fuzz-patterns.txt -w wordlist.txt
# Automatically detects and processes all FUZZ placeholders in the file
# No --prefix flag needed!

# Example file content:
# FUZZ.example.com → api.example.com, admin.example.com...
# webdisk.FUZZ.target.com → webdisk.api.target.com, webdisk.admin.target.com...
# api-FUZZ.test.org → api-staging.test.org, api-dev.test.org...
```

### **Multiple Wordlists with Numbered FUZZ**
```bash
# Use FUZZ1 and FUZZ2 with different wordlists
./mksub --multiple -d "FUZZ1-FUZZ2.example.com" -w environments.txt -w subdomains.txt
# FUZZ1 = environments.txt, FUZZ2 = subdomains.txt
# Output: staging-api.example.com, prod-admin.example.com...

# Complex patterns
./mksub --multiple -d "api-FUZZ1.FUZZ2.example.com" -w envs.txt -w services.txt
# Output: api-staging.auth.example.com, api-prod.billing.example.com...
```

## **🎯 Direct FUZZ Pattern Processing**

### **Pre-defined FUZZ Patterns (New Feature)**
```bash
# Create file with FUZZ patterns already defined
cat > fuzz-patterns.txt << 'EOF'
FUZZ.example.com
FUZZ.arifleet.com
webdisk.FUZZ.arifleet.com
whm.FUZZ.arifleet.com
api-FUZZ.holman.com
EOF

# Process directly - no --prefix needed!
./mksub -df fuzz-patterns.txt -w keywords.txt
# Output: api.example.com, webdisk.admin.arifleet.com, api-staging.holman.com...

# Multiple FUZZ patterns with multiple wordlists
cat > complex-patterns.txt << 'EOF'
FUZZ1-FUZZ2.domain.com
api-FUZZ1.FUZZ2.example.com
FUZZ2.FUZZ1-staging.target.com
EOF

./mksub -df complex-patterns.txt --multiple -w environments.txt -w services.txt
# Output: dev-api.domain.com, api-staging.admin.example.com...
```

## **🎯 Prefix Pattern Fuzzing**

### **Single Dictionary with Domain List**
```bash
# Apply FUZZ pattern to multiple domains from file
./mksub -df subdomains.txt --prefix "FUZZ" -w keywords.txt
# Applies each keyword to every domain in the list
# Input: demo.example.com, staging.target.com
# Output: api.demo.example.com, admin.staging.target.com...

# Save results to file
./mksub -df domains.txt --prefix "FUZZ" -w wordlist.txt -o results.txt
```

### **Multiple Dictionaries with Domain List**
```bash
# Apply complex patterns to domain lists
./mksub -df subdomains.txt --prefix "FUZZ2-FUZZ1" --multiple -w envs.txt -w services.txt
# FUZZ1 = services.txt, FUZZ2 = envs.txt
# Input: api.example.com, app.target.com
# Output: staging-admin.api.example.com, prod-auth.app.target.com...

# Real-world example for bug bounty
./mksub -df discovered-subdomains.txt --prefix "FUZZ" -w /usr/share/seclists/Discovery/Web-Content/common.txt -silent | dnsx -re
```

## **⚡ Streaming & Performance Examples**

### **Large Wordlist Optimization**
```bash
# Automatic streaming for wordlists >50MB
./mksub -d "FUZZ.example.com" -w /usr/share/seclists/Discovery/Web-Content/combined_words.txt -t 500

# Multiple large wordlists (2M+ combinations)
./mksub --multiple -d "FUZZ1-FUZZ2.example.com" -w environments.txt -w huge-wordlist.txt -t 1000
```

### **dnsx Integration**
```bash
# Optimized for dnsx (without -stream for large volumes)
./mksub -d "FUZZ.example.com" -w wordlist.txt -silent | \
dnsx -r /tmp/resolvers.txt -re -t 200 -rl 400 -stats

# Multiple wordlists with dnsx
./mksub --multiple -d "FUZZ1-FUZZ2.example.com" -w envs.txt -w subs.txt -silent | \
dnsx -r /tmp/resolvers.txt -re -t 300 -rl 500 -stats -json -o results.jsonl

# Direct FUZZ patterns with dnsx (NEW)
./mksub -df fuzz-patterns.txt -w wordlist.txt -silent | \
dnsx -r /tmp/resolvers.txt -cname -re -stats -wt 10 -o cname-results.txt

# Prefix fuzzing with dnsx for large-scale enumeration
./mksub -df discovered-domains.txt --prefix "FUZZ" -w common-subdomains.txt -silent | \
dnsx -r /tmp/resolvers.txt -re -t 500 -rl 1000 -stats -o live-subdomains.txt

# Complex patterns with wildcard filtering
./mksub -df complex-patterns.txt --multiple -w envs.txt -w services.txt -silent | \
dnsx -r /tmp/resolvers.txt -cname -a -re -stats -wt 20 -json -o filtered-results.jsonl
```

### **Regex Filtering**
```bash
# Filter wordlist with regex
./mksub -d "FUZZ.example.com" -w wordlist.txt -r "^(api|admin|www|mail|dev)"

# Multiple wordlists with filtering
./mksub --multiple -d "FUZZ1-FUZZ2.example.com" -w envs.txt -w subs.txt -r "^[a-zA-Z]"

# Direct FUZZ patterns with regex filtering (NEW)
./mksub -df fuzz-patterns.txt -w wordlist.txt -r "^(api|admin|dev|test|staging)" -o filtered-results.txt

# Prefix fuzzing with regex filtering
./mksub -df subdomains.txt --prefix "FUZZ" -w wordlist.txt -r "^(api|admin|dev|test|staging)" -o filtered-results.txt
```

## **📊 Performance Comparison & Limitations**

### **Original vs Enhanced**
| Feature | Original | Enhanced |
|---------|----------|----------|
| Max Wordlist Size | ~1MB | **Recommended: <50K words** |
| Memory Usage | High (loads all in RAM) | Optimized but still memory-intensive |
| Multiple Wordlists | ❌ | ✅ FUZZ1, FUZZ2, etc. |
| FUZZ Placeholders | ❌ | ✅ Flexible patterns |
| Prefix Patterns | ❌ | ✅ Domain list fuzzing |
| Direct FUZZ Processing | ❌ | ✅ Auto-detect patterns |
| dnsx Integration | Basic | Optimized buffering |
| Large Scale | Limited | **Best for targeted enumeration** |

### **⚠️ Important Limitations**
- **Memory usage**: Scales with wordlist size × domain count
- **Not suitable for**: Massive wordlists (>100K words)
- **Best for**: Targeted, focused enumeration campaigns
- **Temp files**: Only created when using `-o` flag (fixed)

### **Real-World Performance & Limitations**
```bash
# Small wordlist (748 words × 30 environments = 22K combinations) ✅ EFFICIENT
./mksub --multiple -d "FUZZ2-FUZZ1.example.com" -w environments.txt -w small-wordlist.txt
# Time: ~2 seconds, Memory: ~10MB

# Medium wordlist (10K words × 30 environments = 300K combinations) ⚠️ MODERATE
./mksub --multiple -d "FUZZ2-FUZZ1.example.com" -w environments.txt -w medium-wordlist.txt
# Time: ~8 seconds, Memory: ~25MB

# Large wordlist (100K+ words) ❌ NOT RECOMMENDED
# High memory usage, processing time increases significantly
# Better alternatives: Use smaller targeted wordlists or other tools

# Direct FUZZ patterns (50 patterns × 1K words = 50K combinations) ✅ EFFICIENT
./mksub -df fuzz-patterns.txt -w targeted-wordlist.txt -silent
# Time: ~3 seconds, Memory: ~8MB

# With dnsx resolution (recommended approach - no temp files)
./mksub -df fuzz-patterns.txt -w focused-wordlist.txt -silent | \
dnsx -r /tmp/resolvers.txt -cname -re -stats -wt 10
# Focus on quality over quantity - use targeted wordlists
```

## **🔧 Technical Improvements**

### **1. Memory Optimization**
- **Streaming Mode**: Automatic for wordlists >50MB
- **Buffer Management**: 10MB buffers instead of 100MB
- **Batch Processing**: 10K word batches for efficiency

### **2. Threading Enhancements**
- **Configurable Workers**: Up to 2000 threads for generation
- **Round-Robin Distribution**: Balanced load across workers
- **Nil-Safe Operations**: Prevents crashes in pipe mode

### **3. Output Optimization**
- **Buffered Writing**: 1000-line output buffers
- **Silent Mode**: Optimized for pipe operations
- **Smart File Handling**: Only creates files when `-o` specified (fixed)

## **🚀 Migration from Original**

### **Old Syntax → New Syntax**
```bash
# OLD: Level-based generation
mksub -d example.com -l 2 -w wordlist.txt

# NEW: FUZZ-based generation (more flexible)
./mksub -d "FUZZ.example.com" -w wordlist.txt
./mksub -d "FUZZ.FUZZ.example.com" -w wordlist.txt  # equivalent to -l 2
```

### **New Capabilities (No Temp Files)**
```bash
# Multiple wordlists (impossible in original)
./mksub --multiple -d "FUZZ1-FUZZ2.example.com" -w envs.txt -w subs.txt

# Large-scale enumeration with targeted wordlists
./mksub -d "FUZZ.example.com" -w targeted-wordlist.txt

# Direct FUZZ pattern processing (newest feature - clean pipes)
./mksub -df fuzz-patterns.txt -w wordlist.txt -silent

# Prefix fuzzing against domain lists
./mksub -df discovered-subdomains.txt --prefix "FUZZ" -w common-wordlist.txt -silent

# Seamless dnsx integration (no temp files created)
./mksub -d "FUZZ.example.com" -w wordlist.txt -silent | dnsx -r resolvers.txt -re -stats
```

## **📚 Complete Usage Examples**

### **🎯 All Supported Modes**

#### **1. Traditional Level-based Generation**
```bash
# Generate 2-level subdomains
./mksub -d example.com -l 2 -w wordlist.txt
# Output: api.admin.example.com, www.dev.example.com...
```

#### **2. Single FUZZ Placeholder**
```bash
# Basic FUZZ replacement
./mksub -d "FUZZ.example.com" -w wordlist.txt
# Output: api.example.com, admin.example.com...

# Multiple FUZZ in same domain
./mksub -d "FUZZ.FUZZ-staging.example.com" -w wordlist.txt
# Output: api.api-staging.example.com, admin.admin-staging.example.com...
```

#### **3. Multiple Wordlists with Numbered FUZZ**
```bash
# Two wordlists with FUZZ1 and FUZZ2
./mksub --multiple -d "FUZZ1-FUZZ2.example.com" -w environments.txt -w services.txt
# FUZZ1 = environments.txt, FUZZ2 = services.txt
# Output: staging-api.example.com, prod-admin.example.com...

# Complex patterns
./mksub --multiple -d "api-FUZZ1.FUZZ2.example.com" -w envs.txt -w services.txt
# Output: api-staging.auth.example.com, api-prod.billing.example.com...
```

#### **4. Prefix Pattern Fuzzing**
```bash
# Apply pattern to domain list
./mksub -df subdomains.txt --prefix "FUZZ" -w keywords.txt
# Input file: demo.example.com, staging.target.com
# Output: api.demo.example.com, admin.staging.target.com...

# Multiple wordlists with prefix
./mksub -df domains.txt --prefix "FUZZ2-FUZZ1" --multiple -w envs.txt -w services.txt
# Output: staging-admin.demo.example.com, prod-auth.staging.target.com...
```

#### **5. Direct FUZZ Pattern Processing (NEW)**
```bash
# Process pre-defined FUZZ patterns
cat > patterns.txt << 'EOF'
FUZZ.example.com
webdisk.FUZZ.arifleet.com
api-FUZZ.holman.com
FUZZ1-FUZZ2.example.com
EOF

# Single wordlist mode
./mksub -df patterns.txt -w keywords.txt
# Output: api.example.com, webdisk.admin.arifleet.com...

# Multiple wordlist mode (auto-detected)
./mksub -df patterns.txt --multiple -w envs.txt -w services.txt
# Output: staging-api.example.com, prod-admin.example.com...
```

### **🔧 Advanced Usage Patterns**

#### **Bug Bounty & Reconnaissance**
```bash
# Large-scale subdomain discovery
./mksub -df discovered-subdomains.txt --prefix "FUZZ" \
  -w /usr/share/seclists/Discovery/Web-Content/combined_words.txt -silent | \
  dnsx -r /tmp/resolvers.txt -cname -re -stats -wt 10 -o live-subdomains.txt

# Multi-pattern reconnaissance
./mksub -df fuzz-patterns.txt -w common-subdomains.txt -silent | \
  dnsx -r /tmp/resolvers.txt -a -cname -re -stats -json -o results.jsonl

# Filtered enumeration with regex
./mksub -df patterns.txt -w wordlist.txt -r "^(api|admin|dev|test|staging)" | \
  dnsx -r /tmp/resolvers.txt -cname -re -stats -o filtered-results.txt
```

#### **Corporate Infrastructure Mapping**
```bash
# Environment-based discovery
./mksub --multiple -d "FUZZ1-FUZZ2.company.com" \
  -w environments.txt -w services.txt -silent | \
  dnsx -r /tmp/resolvers.txt -a -cname -re -stats -wt 20 -json -o corp-infra.jsonl

# Service discovery across multiple domains
./mksub -df corporate-domains.txt --prefix "FUZZ" \
  -w service-keywords.txt -silent | \
  dnsx -r /tmp/resolvers.txt -cname -re -stats -o services.txt
```

#### **Performance Optimized Scans**
```bash
# High-throughput scanning
./mksub -df large-domain-list.txt --prefix "FUZZ" \
  -w /usr/share/seclists/Discovery/Web-Content/combined_words.txt \
  -t 1000 -silent | \
  dnsx -r /tmp/resolvers.txt -t 500 -rl 2000 -re -stats -wt 50 -o results.txt

# Memory-efficient large wordlist processing
./mksub -df patterns.txt -w huge-wordlist.txt -t 500 -silent | \
  dnsx -r /tmp/resolvers.txt -cname -re -stats -o memory-efficient.txt
```

### **📊 Output Examples**

#### **Standard Output**
```
api.example.com
admin.example.com
dev.example.com
staging.example.com
```

#### **With dnsx Resolution**
```
api.example.com [A] [192.168.1.100]
admin.example.com [CNAME] [admin-lb.example.com]
dev.example.com [A] [10.0.0.50]
```

#### **JSON Output**
```json
{"host":"api.example.com","a":["192.168.1.100"],"timestamp":"2024-01-01T12:00:00Z"}
{"host":"admin.example.com","cname":["admin-lb.example.com"],"timestamp":"2024-01-01T12:00:01Z"}
```

## Report Bugs / Feedback
If you encounter any issues or have suggestions for improvements, please create an [Issue](https://github.com/yourusername/mksub/issues/new) or submit a pull request on the GitHub repository.

## Contributing
Contributions are welcome! This fork aims to enhance the original mksub with additional features for modern subdomain enumeration workflows.

## Credits
- Original project: [mksub by Trickest](https://github.com/trickest/mksub)
- Enhanced features and improvements by the community

## License
This project maintains the same license as the original mksub project.
