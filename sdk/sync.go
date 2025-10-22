package sdk

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/the-agent-c-ai/cran/internal/registry"
	syncsvc "github.com/the-agent-c-ai/cran/internal/sync"
)

// Sync represents an image sync operation from source to destination registry.
type Sync struct {
	name           string
	sourceRegistry *Registry
	sourceImage    *Image
	sourceScan     *Scan // Optional scan verification
	destRegistry   *Registry
	destImage      *Image
	platforms      []Platform
	destDigest     string // Destination image digest (computed locally, not from registry)
	log            zerolog.Logger
}

// SyncBuilder builds a Sync.
type SyncBuilder struct {
	plan *Plan
	sync *Sync
}

// Source sets the source image, optional scan verification, and optional registry.
// The image MUST have a digest specified - syncing by tag alone is not allowed for security.
// Scan parameter is OPTIONAL. If not provided, a warning will be issued (syncing without scan verification).
// Registry credentials are optional but recommended for private images and rate limit avoidance.
// If registry is not provided, it will be inferred from the image name (e.g., timberio/vector → docker.io).
func (builder *SyncBuilder) Source(image *Image, scan *Scan, sourceRegistry ...*Registry) *SyncBuilder {
	builder.sync.sourceImage = image
	builder.sync.sourceScan = scan

	if len(sourceRegistry) > 0 {
		builder.sync.sourceRegistry = sourceRegistry[0]
	}

	return builder
}

// Destination sets the destination image and registry.
// The image should have name and version. Digest will be computed after sync.
// Registry is required for destination (authentication needed for push operations).
func (builder *SyncBuilder) Destination(image *Image, destRegistry *Registry) *SyncBuilder {
	builder.sync.destImage = image
	builder.sync.destRegistry = destRegistry

	return builder
}

// Platforms sets the platforms to sync.
func (builder *SyncBuilder) Platforms(platforms ...Platform) *SyncBuilder {
	builder.sync.platforms = platforms

	return builder
}

// Build validates and adds the sync to the plan.
// Returns the destination image, which will have its digest populated during plan execution.
func (builder *SyncBuilder) Build() *Image {
	if builder.sync.sourceImage == nil {
		builder.sync.log.Fatal().Msg("sync source image is required")
	}

	if builder.sync.sourceImage.digest == "" {
		builder.sync.log.Fatal().
			Str("image", builder.sync.sourceImage.name).
			Msg("sync source image MUST have digest specified (syncing by tag alone is not allowed)")
	}

	// FAT WARNING if no scan verification provided
	if builder.sync.sourceScan == nil {
		builder.sync.log.Warn().Msgf(
			"⚠️  WARNING: Syncing %s WITHOUT scan verification! "+
				"This means you are syncing an image that has NOT been scanned for vulnerabilities. "+
				"Pass a scan as the second parameter to Source() to enable scan verification.",
			builder.sync.sourceImage.name,
		)
	}

	if builder.sync.destRegistry == nil {
		builder.sync.log.Fatal().Msg("sync destination registry is required")
	}

	if builder.sync.destImage == nil {
		builder.sync.log.Fatal().Msg("sync destination image is required")
	}

	if len(builder.sync.platforms) == 0 {
		// Default to both platforms
		builder.sync.platforms = []Platform{PlatformAMD64, PlatformARM64}
	}

	builder.plan.syncs = append(builder.plan.syncs, builder.sync)

	return builder.sync.destImage
}

func (sync *Sync) execute(_ context.Context) error {
	// Use digestRef for source (immutable, secure)
	sourceRef := sync.sourceImage.digestRef()

	// Construct destination ref with tag (we're pushing new content)
	var destRef string
	if sync.destImage.version != "" {
		destRef = fmt.Sprintf("%s:%s", sync.destImage.name, sync.destImage.version)
	} else {
		destRef = sync.destImage.name
	}

	sync.log.Info().
		Str("source", sourceRef).
		Str("destination", destRef).
		Msg("syncing image")

	// Create source registry client
	// If no registry provided, use empty credentials (for public images)
	// Registry host will be inferred from image name by go-containerregistry
	var srcClient *registry.Client
	if sync.sourceRegistry != nil {
		srcClient = registry.NewClient(
			sync.sourceRegistry.host,
			sync.sourceRegistry.username,
			sync.sourceRegistry.password,
			sync.log.With().Str("registry", "source").Logger(),
		)
	} else {
		// No auth - for public images
		srcClient = registry.NewClient(
			"", // Host inferred from image name
			"", // No username
			"", // No password
			sync.log.With().Str("registry", "source").Logger(),
		)
	}

	dstClient := registry.NewClient(
		sync.destRegistry.host,
		sync.destRegistry.username,
		sync.destRegistry.password,
		sync.log.With().Str("registry", "destination").Logger(),
	)

	// Create syncer
	syncer := syncsvc.NewSyncer(srcClient, dstClient, sync.log)

	// Sync the image by digest and capture destination digest
	destDigest, err := syncer.SyncImage(sourceRef, destRef)
	if err != nil {
		return fmt.Errorf("failed to sync image: %w", err)
	}

	// Store the destination digest (computed locally for security)
	sync.destDigest = destDigest

	// Auto-populate destination image digest for subsequent operations (e.g., scanning)
	sync.destImage.digest = destDigest

	sync.log.Info().
		Str("dest_digest", destDigest).
		Msg("image sync complete")

	return nil
}

// DestDigest returns the destination image digest after sync execution.
// The digest is computed locally from the pushed image/manifest, not retrieved
// from the registry, providing defense in depth against compromised registries.
// Returns empty string if sync has not been executed yet.
func (sync *Sync) DestDigest() string {
	return sync.destDigest
}
