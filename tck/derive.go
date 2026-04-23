package tck

import (
	"fmt"
	"sort"

	"github.com/docker/sbx-kits-contrib/spec"
)

// Option is a functional option for customizing a Suite.
type Option func(*Suite)

// WithImage overrides the container image used for integration tests.
func WithImage(image string) Option {
	return func(s *Suite) {
		s.Image = image
	}
}

// NewSuiteFromDir loads a kit artifact from the given directory and derives
// all test expectations from the spec.yaml and files/ directory.
//
// For kind=agent artifacts, the container image is taken from the manifest's template.
// For kind=mixin artifacts, the image is resolved from the extends field using
// well-known agent templates, or defaults to the shell template.
// Use WithImage to override the image for any artifact.
func NewSuiteFromDir(dir string, opts ...Option) (*Suite, error) {
	artifact, err := spec.LoadFromDirectory(dir)
	if err != nil {
		return nil, fmt.Errorf("load artifact from %q: %w", dir, err)
	}

	image, err := containerImage(artifact)
	if err != nil {
		return nil, err
	}

	suite := &Suite{
		Artifact:               artifact,
		Image:                  image,
		ExpectedEnvVars:        deriveEnvVars(artifact.Environment),
		ExpectedContainerFiles: deriveContainerFiles(artifact.Files, artifact.Commands),
	}

	// Derive network expectations
	if artifact.Network != nil {
		suite.ExpectedAllowedDomains = artifact.Network.AllowedDomains
		suite.ExpectedServiceDomains = artifact.Network.ServiceDomains
		suite.ExpectedServiceAuth = artifact.Network.ServiceAuth
	}

	// Apply functional options (e.g., WithImage)
	for _, opt := range opts {
		opt(suite)
	}

	return suite, nil
}

// deriveEnvVars builds expected "KEY=VALUE" strings from Environment.Variables.
// Results are sorted for deterministic output.
func deriveEnvVars(env *spec.EnvironmentPolicy) []string {
	if env == nil || len(env.Variables) == 0 {
		return nil
	}

	vars := make([]string, 0, len(env.Variables))
	for k, v := range env.Variables {
		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(vars)
	return vars
}

// deriveContainerFiles builds expected absolute container paths from artifact
// files and initFiles. Only "home" target files are derived since workspace
// paths depend on the workdir.
func deriveContainerFiles(files []spec.ArtifactFile, commands *spec.CommandsPolicy) []string {
	var paths []string

	for _, f := range files {
		if f.Target == spec.TargetHome {
			paths = append(paths, HomeDir+"/"+f.RelativePath)
		}
	}

	if commands != nil {
		for _, f := range commands.InitFiles {
			paths = append(paths, f.Path)
		}
	}

	return paths
}
