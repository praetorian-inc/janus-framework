# Janus Framework

The Janus Framework is a powerful Go library for chaining security tools together to create complex, reusable workflows that can run at scale. It provides a uniform interface for connecting disparate security tools, enabling automation of multi-step security processes.

## Mission and Purpose

Janus creates a consistent abstraction layer that enables security engineers to:

- **Connect Disparate Tools**: Chain tools that weren't designed to work together
- **Build Reusable Workflows**: Compose complex security workflows from simple, testable components  
- **Scale Operations**: Run combinations of security tools at scale against large environments
- **Integrate with Chariot**: Provide a uniform mechanism for Product Engineers to pull new security tools into Chariot
- **Standardize Outputs**: Normalize output formats from diverse security tools
- **Focus on Results**: Concentrate on security outcomes rather than tool-specific implementation details

## Core Concepts

### Links
**Links** are the fundamental building blocks - individual processing units that transform data. Each link implements the `Link` interface and defines a `Process` method that handles one type of input and can send zero or more outputs to the next link in the chain.

### Chains  
**Chains** are collections of links that process data sequentially. Data flows from one link to the next through Go channels, enabling concurrent processing while maintaining order.

### Outputters
**Outputters** handle final results, supporting multiple formats like JSON, Markdown, and custom writers. They collect all chain outputs and format them for consumption.

### Configurations
The framework uses a robust configuration system supporting CLI arguments, environment variables, and programmatic configuration with type safety and validation.

## Quick Start

### Basic Chain with Multiple Links

```go
package main

import (
    "fmt"
    "github.com/praetorian-inc/janus-framework/pkg/chain"
    "github.com/praetorian-inc/janus-framework/pkg/links"
)

func main() {
    // Create a chain that converts strings to integers and counts them
    c := chain.NewChain(
        links.FromWrapper(func(s string) int {
            // Parse string to integer
            if num, err := strconv.Atoi(s); err == nil {
                return num
            }
            return 0
        }),
        links.FromWrapper(func(i int) int {
            // Double the number
            return i * 2
        }),
        links.NewCount(), // Count how many items processed
    )

    // Send data through the chain
    c.Send("10", "20", "30")
    c.Close()

    // Retrieve results
    for output, ok := chain.RecvAs[int](c); ok; output, ok = chain.RecvAs[int](c) {
        fmt.Printf("Result: %d\n", output)
    }
}
```

### Chain with Configuration and Output

```go
package main

import (
    "github.com/praetorian-inc/janus-framework/pkg/chain"
    "github.com/praetorian-inc/janus-framework/pkg/chain/cfg"
    "github.com/praetorian-inc/janus-framework/pkg/output"
    "github.com/praetorian-inc/janus-framework/pkg/links"
)

func main() {
    // Create a chain with JSON output
    c := chain.NewChain(
        links.FromWrapper(func(target string) map[string]interface{} {
            return map[string]interface{}{
                "target": target,
                "status": "processed",
                "timestamp": time.Now(),
            }
        }),
    ).WithConfigs(
        cfg.WithArg("jsonoutfile", "results.json"),
        cfg.WithArg("indent", 2),
    ).WithOutputters(
        output.NewJSONOutputter(),
    )

    // Process targets
    c.Send("example.com", "test.org", "demo.net")
    c.Close()
    c.Wait() // Wait for output to complete
}
```

## Advanced Patterns

### Multi-Chain for Parallel Processing

Multi-chains run multiple processing pipelines in parallel, useful for applying different analysis methods to the same data:

```go
// Create parallel chains for different analysis approaches
multi := chain.NewMulti(
    // Aggressive scan chain
    chain.NewChain(
        docker.NewDockerPull(),
        noseyparker.NewNoseyParker(cfg.WithArg("aggressive", true)),
    ),
    // Conservative scan chain  
    chain.NewChain(
        docker.NewDockerPull(),
        noseyparker.NewNoseyParker(cfg.WithArg("conservative", true)),
    ),
).WithOutputters(
    output.NewJSONOutputter(),
    output.NewMarkdownOutputter(),
)

// Send container image to both chains
multi.Send(types.DockerImage{
    Image: "nginx:latest",
    AuthConfig: registry.AuthConfig{},
})
multi.Close()
multi.Wait()
```

### Custom Link Implementation

```go
type PortScannerLink struct {
    *chain.Base
}

func NewPortScanner(configs ...cfg.Config) chain.Link {
    ps := &PortScannerLink{}
    ps.Base = chain.NewBase(ps, configs...)
    return ps
}

func (ps *PortScannerLink) Process(asset types.ScannableAsset) error {
    // Perform port scanning logic
    ports := ps.scanPorts(asset.Target)
    
    // Add discovered ports to the asset
    for _, port := range ports {
        asset.AddPort(types.TCP, port)
    }
    
    // Send enhanced asset to next link
    return ps.Send(asset)
}

func (ps *PortScannerLink) Params() []cfg.Param {
    return []cfg.Param{
        cfg.NewParam[int]("timeout", "scan timeout in seconds").WithDefault(30),
        cfg.NewParam[[]string]("ports", "ports to scan").WithDefault([]string{"80", "443", "22"}),
    }
}
```

## Configuration System

The framework provides a robust configuration system with a clear distinction between **Parameters** and **Arguments**:

### Parameters vs Arguments

**Parameters** are the *definitions* of what configuration a link accepts - think of them as a schema or contract:
- Defined by implementing the `Params()` method on your link
- Specify the name, type, description, defaults, and validation rules
- Similar to function parameters in programming

**Arguments** are the *actual values* provided when using a link:
- Provided using `cfg.WithArg()` when creating chains or links
- Must match the parameter names defined by the link
- Similar to function arguments in programming

### Complete Configuration Example

```go
// 1. DEFINE PARAMETERS - What your link accepts
type PortScannerLink struct {
    *chain.Base
}

func (ps *PortScannerLink) Params() []cfg.Param {
    return []cfg.Param{
        // Required parameter with no default
        cfg.NewParam[string]("api-key", "API key for vulnerability database").WithRequired(),
        
        // Parameter with default value
        cfg.NewParam[int]("timeout", "scan timeout in seconds").WithDefault(30),
        
        // Parameter with CLI shortcode
        cfg.NewParam[[]string]("ports", "ports to scan").WithShortcode("p").WithDefault([]string{"80", "443"}),
        
        // Boolean parameter with shortcode
        cfg.NewParam[bool]("verbose", "enable verbose output").WithShortcode("v").WithDefault(false),
        
        // Parameter with custom converter for complex types
        cfg.NewParam[io.Writer]("logfile", "log output destination").WithConverter(func(s string) (io.Writer, error) {
            return os.Create(s)
        }),
    }
}

// 2. PROVIDE ARGUMENTS - Actual values when using the link
func main() {
    // Method 1: Programmatic configuration
    scanner := NewPortScanner(
        cfg.WithArg("api-key", "secret-api-key-123"),
        cfg.WithArg("timeout", 60),
        cfg.WithArg("ports", []string{"22", "80", "443", "8080"}),
        cfg.WithArg("verbose", true),
    )
    
    // Method 2: Chain-level configuration
    c := chain.NewChain(
        scanner,
        NewVulnChecker(),
    ).WithConfigs(
        cfg.WithArg("api-key", "secret-api-key-123"),
        cfg.WithArg("timeout", 120),
    )
    
    // Method 3: Environment variables (if parameter name matches)
    // export API_KEY=secret-api-key-123
    // export TIMEOUT=60
    
    // Method 4: CLI arguments (when using shortcodes)
    // ./myapp -p 22,80,443 -v --timeout 60 --api-key secret-key
}
```

### Parameter Types and Features

```go
func (l *MyLink) Params() []cfg.Param {
    return []cfg.Param{
        // Basic types
        cfg.NewParam[string]("host", "target hostname"),
        cfg.NewParam[int]("port", "target port"),
        cfg.NewParam[bool]("ssl", "use SSL/TLS"),
        cfg.NewParam[[]string]("headers", "HTTP headers"),
        
        // With defaults
        cfg.NewParam[int]("retries", "number of retries").WithDefault(3),
        
        // Required parameters
        cfg.NewParam[string]("token", "auth token").WithRequired(),
        
        // CLI shortcodes
        cfg.NewParam[bool]("verbose", "verbose output").WithShortcode("v"),
        
        // Custom validation
        cfg.NewParam[int]("workers", "number of workers").WithDefault(10).WithValidator(func(v int) error {
            if v < 1 || v > 100 {
                return fmt.Errorf("workers must be between 1 and 100")
            }
            return nil
        }),
        
        // Type converters for complex types
        cfg.NewParam[time.Duration]("interval", "scan interval").WithConverter(func(s string) (time.Duration, error) {
            return time.ParseDuration(s)
        }),
    }
}
```

### Configuration Priority Order

The framework resolves configuration values in this priority order (highest to lowest):

1. **Programmatic Arguments** - `cfg.WithArg()` calls
2. **CLI Arguments** - Command line flags and arguments  
3. **Environment Variables** - Environment variable values
4. **Default Values** - Defaults specified in parameter definitions

### Accessing Configuration in Links

```go
func (ps *PortScannerLink) Process(target string) error {
    // Access configured values using the parameter names
    apiKey, err := cfg.As[string](ps.Arg("api-key"))
    if err != nil {
        return fmt.Errorf("missing required api-key: %w", err)
    }
    
    timeout, err := cfg.As[int](ps.Arg("timeout"))
    if err != nil {
        return fmt.Errorf("invalid timeout: %w", err)
    }
    
    ports, err := cfg.As[[]string](ps.Arg("ports"))
    if err != nil {
        return fmt.Errorf("invalid ports: %w", err)
    }
    
    verbose, err := cfg.As[bool](ps.Arg("verbose"))
    if err != nil {
        return fmt.Errorf("invalid verbose flag: %w", err)
    }
    
    // Use the configured values
    if verbose {
        fmt.Printf("Scanning %s on ports %v with timeout %ds\n", target, ports, timeout)
    }
    
    // ... scanning logic using apiKey, timeout, ports, verbose
    return ps.Send(result)
}
```

### Common Configuration Patterns

```go
// Outputter configuration
.WithOutputters(
    output.NewJSONOutputter(),
).WithConfigs(
    cfg.WithArg("jsonoutfile", "scan-results.json"),
    cfg.WithArg("indent", 2),
)

// Multi-chain with shared configuration
multi := chain.NewMulti(
    chain.NewChain(NewNmapScanner()),
    chain.NewChain(NewMasscanScanner()),
).WithConfigs(
    cfg.WithArg("timeout", 300),
    cfg.WithArg("output-dir", "/tmp/scans"),
)

// Per-link specific configuration
chain.NewChain(
    NewDNSResolver(cfg.WithArg("nameservers", []string{"8.8.8.8", "1.1.1.1"})),
    NewPortScanner(cfg.WithArg("technique", "syn-scan")),
    NewServiceDetector(cfg.WithArg("probe-intensity", "aggressive")),
)
```

This separation allows for flexible, reusable links that can be configured differently in various contexts while maintaining type safety and clear documentation of what each link expects.

## Output Formats

### JSON Output
```go
chain.WithOutputters(
    output.NewJSONOutputter(),
).WithConfigs(
    cfg.WithArg("jsonoutfile", "results.json"),
    cfg.WithArg("indent", 2),
)
```

### Markdown Output  
```go
chain.WithOutputters(
    output.NewMarkdownOutputter(),
).WithConfigs(
    cfg.WithArg("markdownoutfile", "report.md"),
    cfg.WithArg("template", "security-report"),
)
```

### Custom Output Writer
```go
chain.WithOutputters(
    output.NewWriterOutputter(),
).WithConfigs(
    cfg.WithArg("writer", os.Stdout),
)
```

## Error Handling

```go
c := chain.NewChain(/* links */)

c.Send("data")
c.Close()

// Check for errors
if err := c.Error(); err != nil {
    log.Fatalf("Chain error: %v", err)
}

// Process results with error checking
for output, ok := chain.RecvAs[string](c); ok; output, ok = chain.RecvAs[string](c) {
    fmt.Printf("Output: %s\n", output)
}
```


The Janus Framework provides the foundation for building scalable, maintainable security automation workflows that can integrate seamlessly with existing toolchains and the broader Chariot ecosystem.

