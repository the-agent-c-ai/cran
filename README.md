# Cranberry

Cranberry is a declarative container image management tool written in Go, designed for building, syncing, scanning, and auditing container images across multiple platforms and registries.

## Features

- **Multi-Platform Image Sync**: Copy images between registries with digest verification (linux/amd64, linux/arm64)
- **Distributed Builds**: Build multi-platform images using SSH-accessible buildkit nodes
- **Vulnerability Scanning**: Scan images with Trivy for CVEs and security vulnerabilities
- **Quality Auditing**: Audit Dockerfiles (hadolint) and images (dockle) for best practices
- **Version Checking**: Monitor upstream image registries for new releases
- **Type-Safe Plans**: Define operations as Go programs with compile-time validation
- **Infrastructure Agnostic**: No hard-coded dependencies on specific registries or infrastructure
- **Idempotent Operations**: Digest-based change detection prevents unnecessary work

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

import (
    "context"
    "github.com/the-agent-c-ai/cran/sdk"
)

func main() {
    _ = sdk.LoadEnv(".env")
    plan := sdk.NewPlan("my-pipeline")

    // Define registries
    ghcr := plan.Registry("ghcr.io").
        Username(sdk.GetEnv("GHCR_USERNAME")).
        Password(sdk.GetEnv("GHCR_PASSWORD")).
        Build()

    // Build multi-platform image
    plan.Build("my-build").
        Context("./docker").
        Registry(ghcr).
        Tag("ghcr.io/org/app:v1.0").
        Platform("linux/amd64").
        Platform("linux/arm64").
        Build()

    // Sync image between registries
    plan.Sync("docker-to-ghcr").
        SourceImage("docker.io/org/app:latest").
        DestinationRegistry(ghcr).
        DestinationImage("ghcr.io/org/app:latest").
        Build()

    // Scan image for vulnerabilities
    plan.Scan("scan-app").
        Image("ghcr.io/org/app:v1.0").
        Build()

    // Audit Dockerfile
    plan.Audit("audit-dockerfile").
        Dockerfile("./docker/Dockerfile").
        Build()

    // Check for new versions
    plan.VersionCheck("check-alpine").
        Image("alpine").
        Tag("3.18").
        Build()

    ctx := context.Background()
    if err := plan.Execute(ctx); err != nil {
        log.Fatal().Err(err).Msg("Plan execution failed")
    }
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

## SDK Operations

Cranberry provides these operations in the SDK:

### Build
Build multi-platform container images using buildkit:
```go
plan.Build("name").
    Context("./path").
    Dockerfile("Dockerfile").  // optional
    Registry(registry).
    Tag("image:tag").
    Platform("linux/amd64").
    Platform("linux/arm64").
    Build()
```

### Sync
Copy images between registries with digest verification:
```go
plan.Sync("name").
    SourceImage("source.io/org/image:tag").
    DestinationRegistry(registry).
    DestinationImage("dest.io/org/image:tag").
    Build()
```

### Scan
Scan images for vulnerabilities using Trivy:
```go
plan.Scan("name").
    Image("image:tag").
    Build()
```

### Audit
Audit Dockerfiles and images for best practices:
```go
plan.Audit("name").
    Dockerfile("./Dockerfile").  // hadolint
    Image("image:tag").          // dockle
    Build()
```

### VersionCheck
Check for new image versions in registries:
```go
plan.VersionCheck("name").
    Image("library/alpine").
    Tag("3.18").
    Build()
```

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
