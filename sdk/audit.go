package sdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/the-agent-c-ai/cran/internal/audit"
)

var errAuditFoundIssues = errors.New("audit found issues")

// RuleSet represents audit rule severity.
type RuleSet string

const (
	// RuleSetStrict represents strict audit rules.
	RuleSetStrict RuleSet = "strict"
	// RuleSetRecommended represents recommended audit rules.
	RuleSetRecommended RuleSet = "recommended"
	// RuleSetMinimal represents minimal audit rules.
	RuleSetMinimal RuleSet = "minimal"
)

// Audit represents a Dockerfile and image quality audit.
type Audit struct {
	name         string
	dockerfile   string
	image        *Image
	registry     *Registry
	ruleSet      RuleSet
	ignoreChecks []string
	log          zerolog.Logger
}

// AuditBuilder builds an Audit.
type AuditBuilder struct {
	plan  *Plan
	audit *Audit
}

// Dockerfile sets the Dockerfile path to audit.
func (builder *AuditBuilder) Dockerfile(dockerfile string) *AuditBuilder {
	builder.audit.dockerfile = dockerfile

	return builder
}

// Source sets the image to audit with optional registry credentials.
func (builder *AuditBuilder) Source(image *Image, registry ...*Registry) *AuditBuilder {
	builder.audit.image = image
	if len(registry) > 0 {
		builder.audit.registry = registry[0]
	}

	return builder
}

// RuleSet sets the rule set severity.
func (builder *AuditBuilder) RuleSet(ruleSet RuleSet) *AuditBuilder {
	builder.audit.ruleSet = ruleSet

	return builder
}

// IgnoreChecks sets specific Dockle checks to ignore (e.g., "DKL-DI-0005").
func (builder *AuditBuilder) IgnoreChecks(checks ...string) *AuditBuilder {
	builder.audit.ignoreChecks = append(builder.audit.ignoreChecks, checks...)

	return builder
}

// Build validates and adds the audit to the plan.
func (builder *AuditBuilder) Build() *Audit {
	if builder.audit.dockerfile == "" && builder.audit.image == nil {
		builder.audit.log.Fatal().Msg("audit requires either dockerfile or image")
	}

	if builder.audit.ruleSet == "" {
		builder.audit.ruleSet = RuleSetStrict
	}

	builder.plan.audits = append(builder.plan.audits, builder.audit)

	return builder.audit
}

func (auditJob *Audit) execute(_ context.Context) error {
	var imageRef string
	if auditJob.image != nil {
		imageRef = auditJob.image.tagRef()
	}

	auditJob.log.Info().
		Str("dockerfile", auditJob.dockerfile).
		Str("image", imageRef).
		Str("ruleset", string(auditJob.ruleSet)).
		Msg("auditing")

	auditor := audit.NewAuditor(auditJob.log)
	allPassed := true

	// Audit Dockerfile if provided
	if auditJob.dockerfile != "" {
		result, err := auditor.AuditDockerfile(auditJob.dockerfile)
		if err != nil {
			return fmt.Errorf("failed to audit Dockerfile: %w", err)
		}

		auditJob.log.Info().Msg(result.Output)

		if !result.Passed {
			allPassed = false
		}
	}

	// Audit image if provided
	if auditJob.image != nil {
		opts := audit.ImageAuditOptions{
			RuleSet:      string(auditJob.ruleSet),
			IgnoreChecks: auditJob.ignoreChecks,
		}

		if auditJob.registry != nil {
			opts.RegistryHost = auditJob.registry.host
			opts.Username = auditJob.registry.username
			opts.Password = auditJob.registry.password
		}

		result, err := auditor.AuditImage(imageRef, opts)
		if err != nil {
			return fmt.Errorf("failed to audit image: %w", err)
		}

		auditJob.log.Info().Msg(result.Output)

		if !result.Passed {
			allPassed = false
		}
	}

	if !allPassed {
		auditJob.log.Warn().Msg("audit found issues")

		return errAuditFoundIssues
	}

	auditJob.log.Info().Msg("audit passed")

	return nil
}
