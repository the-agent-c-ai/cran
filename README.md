# Cranberry

Cranberry is a declarative container image management tool written in Go, designed for building, syncing, scanning, and auditing container images across multiple platforms and registries.

## Features

- **Multi-Platform Image Sync**: Copy images between registries (linux/amd64, linux/arm64)
- **Distributed Builds**: Build images using SSH-accessible buildkit nodes
- **Vulnerability Scanning**: Scan images with Trivy for security vulnerabilities
- **Quality Auditing**: Audit Dockerfiles and images with hadolint and dockle
- **Type-Safe Plans**: Define infrastructure as Go programs with compile-time validation
- **Infrastructure Agnostic**: No hard-coded dependencies on specific registries or infrastructure

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Cranberry CLI                           │
│          (cranberry execute -p plan.go)                     │
└────────────────────┬────────────────────────────────────────┘
                     │
         ┌───────────┴──────────┐
         │                      │
    ┌────▼─────┐          ┌─────▼──────┐
    │   SDK    │          │  Internal  │
    │          │          │  Packages  │
    └────┬─────┘          └─────┬──────┘
         │                      │
    ┌────▼──────────────────────▼─────┐
    │  • Registry Client (OCI)        │
    │  • Buildkit Client (SSH)        │
    │  • Trivy Scanner (local)        │
    │  • Audit Tools (hadolint/dockle)│
    └────┬────────────────────────────┘
         │
    ┌────▼──────────────────────┐
    │  Target Systems           │
    │  • Registries (GHCR, etc) │
    │  • Buildkit Nodes (SSH)   │
    └───────────────────────────┘
```

## Installation

```bash
make build
make install  # Installs to $GOPATH/bin
```

## Usage

### Create a Plan File

Plans are Go programs that define your container image operations:

```go
package main

import "github.com/the-agent-c-ai/cran/sdk"

func main() {
    _ = sdk.LoadEnv(".env")
    plan := sdk.NewPlan("my-pipeline")

    // Define registries
    ghcr := plan.Registry("ghcr.io").
        Username(sdk.GetEnv("GHCR_USERNAME")).
        Password(sdk.GetEnv("GHCR_PASSWORD")).
        Build()

    // Build image
    plan.Build("my-build").
        Context("./docker").
        Registry(ghcr).
        Tag("ghcr.io/org/app:v1.0").
        Build()

    plan.Execute()
}
```

### Execute a Plan

```bash
cranberry execute -p plan.go
cranberry execute -p plan.go --dry-run  # Simulate without changes
```

## Design Principles

1. **Infrastructure Agnostic**: No hard-coded registries or infrastructure dependencies
2. **SSH-Based Security**: Buildkit nodes accessed via SSH (no custom TLS)
3. **Type-Safe Configuration**: Plans are Go programs with compile-time validation
4. **Idempotent Operations**: Digest-based change detection prevents unnecessary work
5. **Builder Pattern**: Fluent, readable API inspired by Hadron

## Examples

See `examples/example_plan.go` for a comprehensive example showcasing all features.

## Technology Stack

- **Language**: Go 1.24.0
- **CLI**: urfave/cli/v2
- **Logging**: zerolog
- **Registry**: google/go-containerregistry
- **Build**: moby/buildkit
- **Scanning**: Trivy
- **Linting**: hadolint, dockle
