// Package main provides the Cranberry CLI.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog/log"
	hadron "github.com/the-agent-c-ai/hadron/sdk"
	"github.com/urfave/cli/v2"
)

var errPlanFileNotFound = errors.New("plan file not found")

func main() {
	// Configure zerolog with LOG_LEVEL env var support
	hadron.ConfigureDefaultLogger()

	app := &cli.App{
		Name:    "cranberry",
		Usage:   "Container image management tool",
		Version: "0.1.0",
		Commands: []*cli.Command{
			{
				Name:  "execute",
				Usage: "Execute a plan file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "plan",
						Aliases:  []string{"p"},
						Usage:    "Path to plan file",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "dry-run",
						Usage:   "Simulate execution without making changes",
						Aliases: []string{"n"},
					},
				},
				Action: executeCommand,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}

func executeCommand(cliCtx *cli.Context) error {
	planPath := cliCtx.String("plan")
	dryRun := cliCtx.Bool("dry-run")

	// Check if plan file exists
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", errPlanFileNotFound, planPath)
	}

	// Check for optional shared.go in same directory
	planDir := filepath.Dir(planPath)
	sharedPath := filepath.Join(planDir, "shared.go")

	// Set environment variables for plan execution
	if dryRun {
		if err := os.Setenv("CRANBERRY_DRY_RUN", "true"); err != nil {
			return fmt.Errorf("failed to set DRY_RUN env: %w", err)
		}
	}

	// Build command to execute plan
	args := []string{"run"}

	// Add shared.go if it exists
	if _, err := os.Stat(sharedPath); err == nil {
		args = append(args, sharedPath)
	}

	// Get absolute path for the plan
	absPath, err := filepath.Abs(planPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	args = append(args, absPath)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Dir = filepath.Dir(absPath) // Execute from plan's directory (matches hadron behavior)

	log.Info().Str("plan", absPath).Bool("dry-run", dryRun).Msg("executing plan")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("plan execution failed: %w", err)
	}

	return nil
}
