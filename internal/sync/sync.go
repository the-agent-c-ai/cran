// Package sync provides image synchronization operations.
package sync

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/rs/zerolog"

	"github.com/the-agent-c-ai/cran/internal/registry"
)

// Syncer handles image synchronization between registries.
type Syncer struct {
	srcClient *registry.Client
	dstClient *registry.Client
	log       zerolog.Logger
}

// NewSyncer creates a new image syncer.
func NewSyncer(srcClient, dstClient *registry.Client, log zerolog.Logger) *Syncer {
	return &Syncer{
		srcClient: srcClient,
		dstClient: dstClient,
		log:       log,
	}
}

// SyncImage synchronizes an image from source to destination.
// For multi-platform images, copies each platform separately and creates manifest list.
// This matches the approach used by black/scripts/sync-images.sh.
// Returns the destination image digest (computed locally, not from registry for security).
func (syncer *Syncer) SyncImage(srcImage, dstImage string) (string, error) {
	syncer.log.Debug().
		Str("source", srcImage).
		Str("destination", dstImage).
		Msg("starting image sync")

	// Check if source exists and get descriptor
	desc, err := syncer.srcClient.GetImage(srcImage)
	if err != nil {
		return "", fmt.Errorf("failed to get source image: %w", err)
	}

	// Determine if this is an index (multi-platform) or single image
	if desc.MediaType.IsIndex() {
		syncer.log.Debug().Msg("detected multi-platform image index")

		return syncer.syncMultiPlatform(srcImage, dstImage)
	}

	syncer.log.Debug().Msg("detected single-platform image")

	return syncer.syncSinglePlatform(srcImage, dstImage)
}

// syncMultiPlatform syncs a multi-platform image by copying each platform separately.
// This is the same approach as black/scripts/sync-images.sh:
// 1. Get platform digests from source
// 2. Copy each platform image by digest
// 3. Create and push manifest list at destination
// Returns the destination manifest list digest (computed locally for security).
func (syncer *Syncer) syncMultiPlatform(srcImage, dstImage string) (string, error) {
	// Get platform-specific digests
	platformDigests, err := syncer.srcClient.GetPlatformDigests(srcImage)
	if err != nil {
		return "", fmt.Errorf("failed to get platform digests: %w", err)
	}

	syncer.log.Debug().
		Int("platforms", len(platformDigests)).
		Msg("found platforms in source image")

	// Only sync linux/amd64 and linux/arm64 platforms
	supportedPlatforms := []string{"linux/amd64", "linux/arm64"}

	// Copy each supported platform separately and collect the images
	platformImages := make(map[string]v1.Image)

	for platform, digest := range platformDigests {
		// Skip unsupported platforms
		supported := false

		for _, sp := range supportedPlatforms {
			if platform == sp {
				supported = true

				break
			}
		}

		if !supported {
			syncer.log.Debug().
				Str("platform", platform).
				Msg("skipping unsupported platform")

			continue
		}

		syncer.log.Debug().
			Str("platform", platform).
			Str("digest", digest).
			Msg("copying platform image")

		// Build destination ref with platform suffix
		dstPlatform := fmt.Sprintf("%s-%s", dstImage, sanitizePlatform(platform))

		// Copy this platform
		if err := syncer.srcClient.CopyPlatformImage(stripTag(srcImage), digest, dstPlatform, syncer.dstClient); err != nil {
			return "", fmt.Errorf("failed to copy platform %s: %w", platform, err)
		}

		// Fetch the image we just pushed to include in manifest list
		img, err := syncer.fetchImage(dstPlatform)
		if err != nil {
			return "", fmt.Errorf("failed to fetch pushed image for platform %s: %w", platform, err)
		}

		platformImages[platform] = img
	}

	// Create and push manifest list
	syncer.log.Debug().
		Str("destination", dstImage).
		Msg("creating manifest list")

	digest, err := syncer.dstClient.PushManifestList(dstImage, platformImages)
	if err != nil {
		return "", fmt.Errorf("failed to create manifest list: %w", err)
	}

	syncer.log.Debug().
		Str("digest", digest).
		Msg("manifest list created successfully")

	return digest, nil
}

// syncSinglePlatform syncs a single-platform image.
// Returns the destination image digest (computed locally for security).
func (syncer *Syncer) syncSinglePlatform(srcImage, dstImage string) (string, error) {
	// Copy the image
	if err := syncer.srcClient.CopyImage(srcImage, dstImage, syncer.dstClient); err != nil {
		return "", fmt.Errorf("failed to copy image: %w", err)
	}

	// Fetch the image we just pushed to get its digest
	img, err := syncer.dstClient.GetImageHandle(dstImage)
	if err != nil {
		return "", fmt.Errorf("failed to get destination image handle: %w", err)
	}

	// Compute digest locally
	digest, err := img.Digest()
	if err != nil {
		return "", fmt.Errorf("failed to compute image digest: %w", err)
	}

	syncer.log.Debug().
		Str("digest", digest.String()).
		Msg("single-platform image synced successfully")

	return digest.String(), nil
}

// fetchImage fetches an image from the destination registry.
func (syncer *Syncer) fetchImage(imageRef string) (v1.Image, error) {
	img, err := syncer.dstClient.GetImageHandle(imageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get image handle: %w", err)
	}

	return img, nil
}

// Handles both formats: "registry/repo:tag" and "registry/repo@digest".
func stripTag(imageRef string) string {
	// Check for digest format first (@sha256:...)
	if atIdx := strings.Index(imageRef, "@"); atIdx != -1 {
		return imageRef[:atIdx]
	}

	// Check for tag format (:tag) - but not registry port (host:port)
	// Scan backwards from end, stop at first / (which means we're in the repo part, not host part)
	for i := len(imageRef) - 1; i >= 0; i-- {
		if imageRef[i] == ':' {
			return imageRef[:i]
		}

		if imageRef[i] == '/' {
			break
		}
	}

	return imageRef
}

// sanitizePlatform converts "linux/amd64" to "amd64" for image tag suffixes.
func sanitizePlatform(platform string) string {
	// Extract just the architecture part
	for i := len(platform) - 1; i >= 0; i-- {
		if platform[i] == '/' {
			return platform[i+1:]
		}
	}

	return platform
}

// CheckExists checks if an image exists in the destination registry.
func (syncer *Syncer) CheckExists(imageRef string) (bool, error) {
	exists, err := syncer.dstClient.CheckExists(imageRef)
	if err != nil {
		return false, fmt.Errorf("failed to check image existence: %w", err)
	}

	return exists, nil
}
