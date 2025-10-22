// Package main documents examples for Cranberry SDK
package main

import (
	"context"

	"github.com/the-agent-c-ai/cran/sdk"
)

func main() {
	sdk.ConfigureDefaultLogger()
	_ = sdk.LoadEnv("../.env")

	ctx := context.Background()
	log := sdk.NewLogger()

	plan := sdk.NewPlan("example-pipeline")

	// Define images
	vectorImage := sdk.NewImage("timberio/vector").
		Version(sdk.GetEnv("VECTOR_VERSION")).
		Digest(sdk.GetEnv("VECTOR_DIGEST")).
		Build()

	// Caddy without digest - will show warning
	caddyImage := sdk.NewImage("caddy").
		Version(sdk.GetEnv("CADDY_VERSION")).
		Build()

	// Check for version updates
	plan.VersionCheck("vector-version").Source(vectorImage).Build()
	plan.VersionCheck("caddy-version").Source(caddyImage).Build()

	// Retrieve registry credentials from 1Password
	ghcrUsername, err := sdk.GetSecret(ctx, "op://Security (build)/deploy.registry.rw/username")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to retrieve Registry username from 1Password")
	}

	ghcrToken, err := sdk.GetSecret(ctx, "op://Security (build)/deploy.registry.rw/password")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to retrieve Registry token from 1Password")
	}

	dockerhubUsername, err := sdk.GetSecret(ctx, "op://Security (build)/deploy.docker.io.ro/username")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to retrieve Docker Hub username from 1Password")
	}

	dockerhubPassword, err := sdk.GetSecret(ctx, "op://Security (build)/deploy.docker.io.ro/password")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to retrieve Docker Hub password from 1Password")
	}

	// Define registries
	ghcr := plan.Registry("ghcr.io").
		Username(ghcrUsername).
		Password(ghcrToken).
		Build()

	dockerhub := plan.Registry("docker.io").
		Username(dockerhubUsername).
		Password(dockerhubPassword).
		Build()

	// Define buildkit nodes (SSH config resolves connection parameters)
	amd64Builder := plan.BuildNode("amd64-builder").
		Endpoint(sdk.GetEnv("AMD64_BUILDER_HOST")).
		Platform(sdk.PlatformAMD64).
		Build()

	arm64Builder := plan.BuildNode("arm64-builder").
		Endpoint(sdk.GetEnv("ARM64_BUILDER_HOST")).
		Platform(sdk.PlatformARM64).
		Build()

	// Build multi-platform image
	plan.Build("example-build").
		Context("./docker/example").
		Dockerfile("Dockerfile").
		Node(amd64Builder).
		Node(arm64Builder).
		Registry(ghcr).
		Tag(sdk.GetEnv("EXAMPLE_IMAGE_TAG")).
		Build()

	// Define example image for syncing
	exampleImage := sdk.NewImage(sdk.GetEnv("EXAMPLE_IMAGE_TAG")).
		Digest(sdk.GetEnv("EXAMPLE_IMAGE_DIGEST")).
		Build()

	// Audit Dockerfile and image
	plan.Audit("example-audit").
		Dockerfile("./docker/example/Dockerfile").
		Source(exampleImage).
		RuleSet(sdk.RuleSetStrict).
		Build()

	exampleDest := sdk.NewImage("myorg/example").
		Version("latest").
		Build()

	exampleDestImage := plan.Sync("example-sync").
		Source(exampleImage, nil, ghcr).
		Destination(exampleDest, dockerhub).
		Platforms(sdk.PlatformAMD64, sdk.PlatformARM64).
		Build()

	// Scan destination image after sync (digest auto-populated during plan execution)
	_ = plan.Scan("example-scan").
		Source(exampleDestImage, dockerhub).
		Severity(sdk.SeverityCritical).
		Severity(sdk.SeverityHigh, sdk.ActionWarn).
		Format(sdk.FormatTable).
		Build()

	// Execute plan
	if err := plan.Execute(ctx); err != nil {
		log := sdk.NewLogger()
		log.Fatal().Err(err).Msg("Plan execution failed")
	}
}
