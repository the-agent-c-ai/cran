// Package audit provides Dockerfile and image quality auditing.
package audit

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"

	"github.com/the-agent-c-ai/cran/internal/tools"
)

// Auditor wraps hadolint and dockle CLI operations.
type Auditor struct {
	log       zerolog.Logger
	installer *tools.Installer
}

// NewAuditor creates a new auditor.
func NewAuditor(log zerolog.Logger) *Auditor {
	return &Auditor{
		log:       log,
		installer: tools.NewInstaller(log),
	}
}

// ImageAuditOptions configures image audit behavior.
type ImageAuditOptions struct {
	RegistryHost string // Registry host for authentication (optional)
	Username     string // Registry username (optional)
	Password     string // Registry password (optional)
	RuleSet      string // Rule set: "strict", "recommended", or "minimal"
}

// HadolintIssue represents a single hadolint issue.
type HadolintIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Line    int    `json:"line"`
	Level   string `json:"level"`
}

// HadolintResult represents hadolint scan results.
//
//nolint:tagliatelle
type HadolintResult struct {
	Issues []HadolintIssue `json:""`
}

// DockleDetail represents a single dockle issue detail.
type DockleDetail struct {
	Code   string   `json:"code"`
	Title  string   `json:"title"`
	Level  string   `json:"level"`
	Alerts []string `json:"alerts"`
}

// DockleResult represents dockle scan results.
type DockleResult struct {
	Details []DockleDetail `json:"details"`
}

// Result aggregates audit results.
type Result struct {
	DockerfileIssues int
	ImageIssues      int
	Passed           bool
	Output           string
}

// AuditDockerfile audits a Dockerfile with hadolint.
func (auditor *Auditor) AuditDockerfile(dockerfilePath string) (*Result, error) {
	auditor.log.Info().
		Str("dockerfile", dockerfilePath).
		Msg("auditing Dockerfile with hadolint")

	cmd := exec.Command("hadolint", "--format", "json", dockerfilePath)
	output, err := cmd.CombinedOutput()

	// hadolint returns non-zero when issues found
	var issues []map[string]any
	if len(output) > 0 {
		if parseErr := json.Unmarshal(output, &issues); parseErr != nil {
			auditor.log.Debug().Err(parseErr).Msg("failed to parse hadolint output")
		}
	}

	result := &Result{
		DockerfileIssues: len(issues),
		Passed:           len(issues) == 0 && err == nil,
		Output:           formatHadolintOutput(issues),
	}

	auditor.log.Info().
		Int("issues", result.DockerfileIssues).
		Bool("passed", result.Passed).
		Msg("Dockerfile audit complete")

	return result, nil
}

// AuditImage audits an image with dockle.
func (auditor *Auditor) AuditImage(imageRef string, opts ImageAuditOptions) (*Result, error) {
	// Ensure dockle is installed
	docklePath, err := auditor.installer.Ensure(tools.Dockle)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure dockle is installed: %w", err)
	}

	auditor.log.Info().
		Str("image", imageRef).
		Msg("auditing image with dockle")

	// Build dockle command
	args := []string{"--format", "json", "--exit-code", "1", imageRef}

	//nolint:gosec // Image ref is from user config
	cmd := exec.Command(docklePath, args...)

	// Set credentials via environment variables to avoid exposing in process list
	// DOCKLE_AUTH_URL scopes credentials to the specific registry
	if opts.Username != "" && opts.Password != "" && opts.RegistryHost != "" {
		authURL := "https://" + opts.RegistryHost
		cmd.Env = append(cmd.Env,
			"DOCKLE_AUTH_URL="+authURL,
			"DOCKLE_USERNAME="+opts.Username,
			"DOCKLE_PASSWORD="+opts.Password,
		)
	}

	output, err := cmd.CombinedOutput()

	// Parse dockle JSON output
	var dockleResult DockleResult
	if len(output) > 0 {
		if parseErr := json.Unmarshal(output, &dockleResult); parseErr != nil {
			auditor.log.Error().
				Err(parseErr).
				Str("output", string(output)).
				Msg("failed to parse dockle JSON output")

			return nil, fmt.Errorf("failed to parse dockle output: %w", parseErr)
		}
	} else if err != nil {
		// Command failed with no output
		return nil, fmt.Errorf("dockle command failed: %w", err)
	}

	// Count issues by severity level
	var fatalCount, warnCount, infoCount int

	for _, detail := range dockleResult.Details {
		switch detail.Level {
		case "FATAL":
			fatalCount++
		case "WARN":
			warnCount++
		case "INFO":
			// SKIP is not counted as an issue
			infoCount++
		}
	}

	totalIssues := fatalCount + warnCount + infoCount

	// Determine which issues should cause failure based on RuleSet
	var failingIssues int

	switch opts.RuleSet {
	case "strict":
		// Fail on FATAL and WARN
		failingIssues = fatalCount + warnCount
	case "recommended":
		// Fail only on FATAL
		failingIssues = fatalCount
	case "minimal":
		// Fail only on FATAL
		failingIssues = fatalCount
	default:
		// Default to strict
		failingIssues = fatalCount + warnCount
	}

	result := &Result{
		ImageIssues: totalIssues,
		Passed:      failingIssues == 0,
		Output:      formatDockleOutput(&dockleResult),
	}

	auditor.log.Info().
		Int("issues", result.ImageIssues).
		Bool("passed", result.Passed).
		Msg("image audit complete")

	return result, nil
}

func formatHadolintOutput(issues []map[string]any) string {
	if len(issues) == 0 {
		return "No Dockerfile issues found\n"
	}

	var builder strings.Builder

	_, _ = builder.WriteString("DOCKERFILE AUDIT RESULTS (hadolint)\n")
	_, _ = builder.WriteString(strings.Repeat("=", 80) + "\n\n")

	for _, issue := range issues {
		code := getString(issue, "code")
		message := getString(issue, "message")
		line := getInt(issue, "line")
		level := getString(issue, "level")

		_, _ = builder.WriteString(fmt.Sprintf(
			"[%s] Line %d: %s\n  %s\n\n",
			level,
			line,
			code,
			message,
		))
	}

	_, _ = builder.WriteString(fmt.Sprintf("Total issues: %d\n", len(issues)))

	return builder.String()
}

func formatDockleOutput(result *DockleResult) string {
	if len(result.Details) == 0 {
		return "No image issues found\n"
	}

	var builder strings.Builder

	_, _ = builder.WriteString("IMAGE AUDIT RESULTS (dockle)\n")
	_, _ = builder.WriteString(strings.Repeat("=", 80) + "\n\n")

	for _, detail := range result.Details {
		_, _ = builder.WriteString(fmt.Sprintf(
			"[%s] %s - %s\n",
			detail.Level,
			detail.Code,
			detail.Title,
		))

		for _, alert := range detail.Alerts {
			_, _ = builder.WriteString(fmt.Sprintf("  - %s\n", alert))
		}

		_, _ = builder.WriteString("\n")
	}

	_, _ = builder.WriteString(fmt.Sprintf("Total issues: %d\n", len(result.Details)))

	return builder.String()
}

func getString(data map[string]any, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}

	return ""
}

func getInt(data map[string]any, key string) int {
	if val, ok := data[key]; ok {
		if num, ok := val.(float64); ok {
			return int(num)
		}
	}

	return 0
}
