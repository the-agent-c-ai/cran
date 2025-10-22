package sdk

import (
	"context"

	"github.com/rs/zerolog"
)

// Plan represents a declarative container image management plan.
type Plan struct {
	name string
	log  zerolog.Logger

	// Resources
	registries    []*Registry
	buildNodes    []*BuildNode
	syncs         []*Sync
	builds        []*Build
	scans         []*Scan
	audits        []*Audit
	versionChecks []*VersionCheck
}

// NewPlan creates a new Plan with the given name.
func NewPlan(name string) *Plan {
	return &Plan{
		name: name,
		log:  globalLogger.With().Str("plan", name).Logger(),
	}
}

// Registry creates a new Registry builder.
func (plan *Plan) Registry(host string) *RegistryBuilder {
	return &RegistryBuilder{
		plan: plan,
		registry: &Registry{
			host: host,
			log:  plan.log.With().Str("registry", host).Logger(),
		},
	}
}

// BuildNode creates a new BuildNode builder.
func (plan *Plan) BuildNode(name string) *BuildNodeBuilder {
	return &BuildNodeBuilder{
		plan: plan,
		node: &BuildNode{
			name: name,
			log:  plan.log.With().Str("buildnode", name).Logger(),
		},
	}
}

// Sync creates a new Sync builder.
func (plan *Plan) Sync(name string) *SyncBuilder {
	return &SyncBuilder{
		plan: plan,
		sync: &Sync{
			name: name,
			log:  plan.log.With().Str("sync", name).Logger(),
		},
	}
}

// Build creates a new Build builder.
func (plan *Plan) Build(name string) *BuildBuilder {
	return &BuildBuilder{
		plan: plan,
		build: &Build{
			name: name,
			log:  plan.log.With().Str("build", name).Logger(),
		},
	}
}

// Scan creates a new Scan builder.
func (plan *Plan) Scan(name string) *ScanBuilder {
	return &ScanBuilder{
		plan: plan,
		scan: &Scan{
			name: name,
			log:  plan.log.With().Str("scan", name).Logger(),
		},
	}
}

// Audit creates a new Audit builder.
func (plan *Plan) Audit(name string) *AuditBuilder {
	return &AuditBuilder{
		plan: plan,
		audit: &Audit{
			name: name,
			log:  plan.log.With().Str("audit", name).Logger(),
		},
	}
}

// VersionCheck creates a new VersionCheck builder.
func (plan *Plan) VersionCheck(name string) *VersionCheckBuilder {
	return &VersionCheckBuilder{
		plan: plan,
		check: &VersionCheck{
			name: name,
			log:  plan.log.With().Str("version_check", name).Logger(),
		},
	}
}

// Execute runs the plan with the given context.
func (plan *Plan) Execute(ctx context.Context) error {
	plan.log.Info().Msg("executing plan")

	// Execute in order: version checks → syncs → builds → scans → audits
	if err := plan.executeVersionChecks(ctx); err != nil {
		return err
	}

	if err := plan.executeSyncs(ctx); err != nil {
		return err
	}

	if err := plan.executeBuilds(ctx); err != nil {
		return err
	}

	if err := plan.executeScans(ctx); err != nil {
		return err
	}

	if err := plan.executeAudits(ctx); err != nil {
		return err
	}

	plan.log.Info().Msg("plan execution complete")

	return nil
}

// DryRun simulates plan execution without making changes.
func (plan *Plan) DryRun() error {
	plan.log.Info().Msg("dry run (no changes will be made)")

	return nil
}

func (plan *Plan) executeSyncs(ctx context.Context) error {
	for _, sync := range plan.syncs {
		if err := sync.execute(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (plan *Plan) executeBuilds(ctx context.Context) error {
	for _, build := range plan.builds {
		if err := build.execute(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (plan *Plan) executeScans(ctx context.Context) error {
	for _, scan := range plan.scans {
		if err := scan.execute(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (plan *Plan) executeAudits(ctx context.Context) error {
	for _, audit := range plan.audits {
		if err := audit.execute(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (plan *Plan) executeVersionChecks(ctx context.Context) error {
	for _, check := range plan.versionChecks {
		if err := check.execute(ctx); err != nil {
			return err
		}
	}

	return nil
}
