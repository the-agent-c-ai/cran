package sdk

import (
	"github.com/rs/zerolog"
)

// Registry represents a container registry with authentication.
type Registry struct {
	host     string
	username string
	password string
	log      zerolog.Logger
}

// RegistryBuilder builds a Registry.
type RegistryBuilder struct {
	plan     *Plan
	registry *Registry
}

// Username sets the registry username.
func (builder *RegistryBuilder) Username(username string) *RegistryBuilder {
	builder.registry.username = username

	return builder
}

// Password sets the registry password.
func (builder *RegistryBuilder) Password(password string) *RegistryBuilder {
	builder.registry.password = password

	return builder
}

// Build validates and adds the registry to the plan.
func (builder *RegistryBuilder) Build() *Registry {
	if builder.registry.host == "" {
		builder.registry.log.Fatal().Msg("registry host is required")
	}

	builder.plan.registries = append(builder.plan.registries, builder.registry)

	return builder.registry
}

// Host returns the registry host.
func (registry *Registry) Host() string {
	return registry.host
}

// Username returns the registry username.
func (registry *Registry) Username() string {
	return registry.username
}

// Password returns the registry password.
func (registry *Registry) Password() string {
	return registry.password
}
