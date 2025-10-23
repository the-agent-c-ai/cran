// Package sdk provides the public API for Cranberry container image management.
package sdk

import (
	"context"

	"github.com/rs/zerolog"

	hadron "github.com/the-agent-c-ai/hadron/sdk"
)

// LoadEnv loads environment variables from a .env file.
// Wraps Hadron's LoadEnv for convenience.
//
//nolint:wrapcheck
func LoadEnv(path string) error {
	return hadron.LoadEnv(path)
}

// GetEnv retrieves a required environment variable.
// Wraps Hadron's GetEnv for convenience.
func GetEnv(key string) string {
	return hadron.GetEnv(key)
}

// GetEnvDefault retrieves an environment variable or returns a default value.
// Wraps Hadron's GetEnvDefault for convenience.
func GetEnvDefault(key, defaultValue string) string {
	return hadron.GetEnvDefault(key, defaultValue)
}

// MustGetEnv retrieves a required environment variable.
// Wraps Hadron's MustGetEnv for convenience.
func MustGetEnv(key string) string {
	return hadron.MustGetEnv(key)
}

// GetSecret retrieves a secret from 1Password using a secret reference.
// Reference format: "op://vault/item/field"
//
// Example usage in plans:
//
//	registry := plan.Registry("ghcr.io").
//	    Username(sdk.GetEnv("GITHUB_USERNAME")).
//	    Password(func() string {
//	        password, err := sdk.GetSecret(context.Background(), "op://Production/github/token")
//	        if err != nil {
//	            log.Fatal().Err(err).Msg("failed to retrieve GitHub token from 1Password")
//	        }
//	        return password
//	    }()).
//	    Build()
//
// Or retrieve before building the plan:
//
//	ctx := context.Background()
//	githubToken, err := sdk.GetSecret(ctx, "op://Production/github/token")
//	if err != nil {
//	    log.Fatal().Err(err).Msg("failed to retrieve GitHub token")
//	}
//
//	registry := plan.Registry("ghcr.io").
//	    Username(sdk.GetEnv("GITHUB_USERNAME")).
//	    Password(githubToken).
//	    Build()
//
// Requires OP_SERVICE_ACCOUNT_TOKEN environment variable to be set.
// Wraps Hadron's GetSecret for convenience.
//
//nolint:wrapcheck
func GetSecret(ctx context.Context, reference string) (string, error) {
	return hadron.GetSecret(ctx, reference)
}

// ConfigureDefaultLogger configures the global zerolog logger with sensible defaults.
// It uses a console writer with RFC3339 timestamps for human-readable output.
// If a log level is provided, it sets that level. Otherwise, it reads from the LOG_LEVEL
// environment variable (defaults to "info" if not set or invalid).
// Wraps Hadron's ConfigureDefaultLogger for convenience.
func ConfigureDefaultLogger(level ...zerolog.Level) {
	hadron.ConfigureDefaultLogger(level...)
}

// NewLogger creates a new zerolog logger with console output.
// This can be used if you want a separate logger instance instead of the global one.
// Optionally accepts a log level; if not provided, uses the global level.
// Wraps Hadron's NewLogger for convenience.
func NewLogger(level ...zerolog.Level) zerolog.Logger {
	return hadron.NewLogger(level...)
}

//nolint:gochecknoglobals // Shared logger instance.
var globalLogger = NewLogger()
