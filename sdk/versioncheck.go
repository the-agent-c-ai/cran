package sdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/the-agent-c-ai/cran/internal/version"
)

var errDigestMismatch = errors.New("DIGEST MISMATCH (possible tag mutation or supply chain attack)")

// VersionCheck represents a version check operation.
type VersionCheck struct {
	name     string
	image    *Image
	registry *Registry
	log      zerolog.Logger
}

// VersionCheckBuilder builds a VersionCheck.
type VersionCheckBuilder struct {
	plan  *Plan
	check *VersionCheck
}

// Source sets the source image and optional registry credentials.
// The image must have a version specified. Digest is optional:
// - If digest is provided: verifies the version tag points to expected digest (fails on mismatch)
// - If digest is not provided: shows warning with actual digest
// If registry is not provided, version check will use anonymous access (public repos only).
// For private repositories, provide registry with authentication credentials.
//
// Signature matches Sync.Source() and Scan.Source() for consistency.
func (builder *VersionCheckBuilder) Source(image *Image, registry ...*Registry) *VersionCheckBuilder {
	builder.check.image = image
	if len(registry) > 0 {
		builder.check.registry = registry[0]
	}

	return builder
}

// Build validates and adds the version check to the plan.
func (builder *VersionCheckBuilder) Build() *VersionCheck {
	if builder.check.image == nil {
		builder.check.log.Fatal().Msg("version check image is required")
	}

	if builder.check.image.version == "" {
		builder.check.log.Fatal().Msg("version check image must have version specified")
	}

	builder.plan.versionChecks = append(builder.plan.versionChecks, builder.check)

	return builder.check
}

func (check *VersionCheck) execute(_ context.Context) error {
	img := check.image

	check.log.Info().
		Str("image", img.name).
		Str("version", img.version).
		Msg("checking for version updates")

	// Create version checker with optional registry credentials
	var username, password string
	if check.registry != nil {
		username = check.registry.username
		password = check.registry.password
	}

	checker := version.NewChecker(username, password, check.log)

	// Use tagRef to query what the tag points to
	tagReference := img.tagRef()

	// Verify current version digest if provided
	if img.digest != "" {
		check.log.Debug().
			Str("expected_digest", img.digest).
			Msg("verifying current version digest")

		actualDigest, err := checker.GetTagDigest(tagReference)
		if err != nil {
			return fmt.Errorf("failed to get current version digest: %w", err)
		}

		if actualDigest != img.digest {
			check.log.Error().
				Str("expected", img.digest).
				Str("actual", actualDigest).
				Str("version", img.version).
				Msg("current version digest mismatch")

			return fmt.Errorf(
				"%w: current version %s points to %s, expected %s",
				errDigestMismatch,
				tagReference,
				actualDigest,
				img.digest,
			)
		}

		check.log.Info().
			Str("digest", actualDigest).
			Msg("current version digest verification passed")
	} else {
		// Warn if no digest provided - show actual digest
		actualDigest, err := checker.GetTagDigest(tagReference)
		if err != nil {
			check.log.Warn().
				Err(err).
				Str("version", img.version).
				Msg("failed to retrieve current version digest for verification")
		} else {
			check.log.Warn().
				Str("tag", tagReference).
				Str("digest", actualDigest).
				Msgf("⚠ WARNING: No digest verification for %s. Add .Digest(\"%s\") to your Image to enable verification", tagReference, actualDigest)
		}
	}

	// Check for updates - variant auto-extracted from version
	info, err := checker.CheckVersion(img.name, img.version, "")
	if err != nil {
		return fmt.Errorf("failed to check version: %w", err)
	}

	if info.UpdateAvailable {
		check.log.Warn().
			Str("image", img.name).
			Str("current", info.CurrentVersion).
			Str("latest", info.LatestVersion).
			Str("digest", info.LatestDigest).
			Msg("⚠ UPDATE AVAILABLE")
	} else {
		check.log.Info().
			Str("tag", tagReference).
			Msg("✓ Up to date")
	}

	return nil
}
