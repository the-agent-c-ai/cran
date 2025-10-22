package sdk

import (
	"fmt"

	"github.com/rs/zerolog"
)

// Image represents a container image reference with optional version and digest.
type Image struct {
	name    string
	version string
	digest  string
	log     zerolog.Logger
}

// ImageBuilder builds an Image.
type ImageBuilder struct {
	image *Image
}

// NewImage creates a new Image builder with the specified name.
// Name can be a simple repository name (e.g., "timberio/vector")
// or include a registry host (e.g., "ghcr.io/org/vector").
func NewImage(name string) *ImageBuilder {
	return &ImageBuilder{
		image: &Image{
			name: name,
			log:  globalLogger.With().Str("image", name).Logger(),
		},
	}
}

// Version sets the image version/tag.
// Can include variant suffix (e.g., "0.50.0-distroless-static").
func (builder *ImageBuilder) Version(version string) *ImageBuilder {
	builder.image.version = version

	return builder
}

// Digest sets the image digest for verification and secure operations.
func (builder *ImageBuilder) Digest(digest string) *ImageBuilder {
	builder.image.digest = digest

	return builder
}

// Build validates and returns the Image.
func (builder *ImageBuilder) Build() *Image {
	if builder.image.name == "" {
		builder.image.log.Fatal().Msg("image name is required")
	}

	return builder.image
}

// Name returns the image name.
func (img *Image) Name() string {
	return img.name
}

// Version returns the image version/tag.
func (img *Image) Version() string {
	return img.version
}

// Digest returns the image digest.
func (img *Image) Digest() string {
	return img.digest
}

// Returns: "name:version".
func (img *Image) tagRef() string {
	if img.version == "" {
		img.log.Fatal().Msg("cannot create tag reference without version")
	}

	return fmt.Sprintf("%s:%s", img.name, img.version)
}

// Returns: "name@digest".
func (img *Image) digestRef() string {
	if img.digest == "" {
		img.log.Fatal().Msg("cannot create digest reference without digest")
	}

	return fmt.Sprintf("%s@%s", img.name, img.digest)
}
