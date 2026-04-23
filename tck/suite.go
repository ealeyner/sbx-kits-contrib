// Package tck provides a Technology Compatibility Kit for validating sandbox kit artifacts.
// It loads an artifact from a directory, derives test expectations from its spec.yaml,
// and verifies them against a real container using testcontainers-go.
package tck

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/sbx-kits-contrib/spec"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const (
	// DefaultShellImage is the Docker image used for kind=mixin container tests.
	DefaultShellImage = "docker/sandbox-templates:shell-docker"

	// HomeDir is the agent's home directory inside sandbox containers.
	HomeDir = "/home/agent"
)

// Suite holds test expectations derived from a kit artifact.
type Suite struct {
	// Artifact is the loaded and validated kit artifact.
	Artifact *spec.Artifact

	// Image is the container image used for integration tests.
	Image string

	// Derived expectations
	ExpectedEnvVars        []string
	ExpectedContainerFiles []string
	ExpectedAllowedDomains []string
	ExpectedServiceDomains map[string]string
	ExpectedServiceAuth    map[string]spec.ServiceAuth
}

// RunAll runs all TCK tests for the kit artifact.
func (s *Suite) RunAll(t *testing.T) {
	t.Run(s.Artifact.Manifest.Name+"_TCK", func(t *testing.T) {
		s.RunValidationTests(t)
		s.RunNetworkPolicyTests(t)
		s.RunCredentialPolicyTests(t)
		s.RunCommandsValidationTests(t)
		s.RunEnvironmentTests(t)
		s.RunContainerFileTests(t)
		s.RunSecurityTests(t)
	})
}

// RunValidationTests verifies the artifact's manifest is well-formed.
func (s *Suite) RunValidationTests(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		a := s.Artifact
		require.NotEmpty(t, a.Manifest.SchemaVersion, "schemaVersion is required")
		require.NotEmpty(t, a.Manifest.Kind, "kind is required")
		require.NotEmpty(t, a.Manifest.Name, "name is required")
		require.NotEmpty(t, a.Manifest.DisplayName, "displayName is required")
		require.NotEmpty(t, a.Manifest.Description, "description is required")

		if a.Manifest.Kind == spec.KindMixin {
			require.Empty(t, a.Manifest.Template, "mixins should not define their own template")
			require.Empty(t, a.Manifest.Binary, "mixins should not define a binary")
		}
		if a.Manifest.Kind == spec.KindAgent {
			require.NotEmpty(t, a.Manifest.Template, "agents must define a template")
		}
	})
}

// RunNetworkPolicyTests verifies the artifact's network policy is consistent.
func (s *Suite) RunNetworkPolicyTests(t *testing.T) {
	if s.Artifact.Network == nil && len(s.ExpectedAllowedDomains) == 0 && len(s.ExpectedServiceDomains) == 0 {
		return
	}

	t.Run("network_policy", func(t *testing.T) {
		net := s.Artifact.Network
		if net == nil {
			require.Empty(t, s.ExpectedAllowedDomains, "expected allowed domains but network policy is nil")
			require.Empty(t, s.ExpectedServiceDomains, "expected service domains but network policy is nil")
			return
		}

		if len(s.ExpectedAllowedDomains) > 0 {
			require.ElementsMatch(t, s.ExpectedAllowedDomains, net.AllowedDomains,
				"allowed domains should match")
		}

		if len(s.ExpectedServiceDomains) > 0 {
			require.Equal(t, s.ExpectedServiceDomains, net.ServiceDomains,
				"service domains should match")
		}

		if len(s.ExpectedServiceAuth) > 0 {
			require.NotNil(t, net.ServiceAuth)
			for service, expected := range s.ExpectedServiceAuth {
				actual, ok := net.ServiceAuth[service]
				require.True(t, ok, "service auth for %q not found", service)
				require.Equal(t, expected.HeaderName, actual.HeaderName,
					"headerName mismatch for service %q", service)
				require.Contains(t, actual.ValueFormat, "%s",
					"valueFormat for service %q must contain %%s", service)
				require.Equal(t, expected.ValueFormat, actual.ValueFormat,
					"valueFormat mismatch for service %q", service)
			}
		}
	})
}

// RunCredentialPolicyTests verifies the artifact's credential policy is well-formed.
func (s *Suite) RunCredentialPolicyTests(t *testing.T) {
	if s.Artifact.Credentials == nil {
		return
	}

	t.Run("credential_policy", func(t *testing.T) {
		for service, source := range s.Artifact.Credentials.Sources {
			t.Run(service, func(t *testing.T) {
				require.True(t, len(source.Env) > 0 || source.File != nil,
					"credential source for %q must have at least one of env or file", service)

				if source.File != nil {
					require.NotEmpty(t, source.File.Path,
						"credential file path for %q must not be empty", service)
				}

				if source.Priority != "" {
					require.Contains(t, []string{"env-first", "file-first"}, source.Priority,
						"invalid priority %q for service %q", source.Priority, service)
				}
			})
		}
	})
}

// RunCommandsValidationTests verifies install and startup commands are well-formed.
func (s *Suite) RunCommandsValidationTests(t *testing.T) {
	if s.Artifact.Commands == nil {
		return
	}

	t.Run("commands_validation", func(t *testing.T) {
		for i, cmd := range s.Artifact.Commands.Install {
			require.NotEmpty(t, cmd.Command,
				"install command [%d] must not be empty", i)
		}

		for i, cmd := range s.Artifact.Commands.Startup {
			require.NotEmpty(t, cmd.Command,
				"startup command [%d] must not be empty", i)
		}

		for i, f := range s.Artifact.Commands.InitFiles {
			require.NotEmpty(t, f.Path,
				"initFile [%d] path must not be empty", i)
			require.True(t, strings.HasPrefix(f.Path, "/"),
				"initFile [%d] path must be absolute (got %q)", i, f.Path)
		}
	})
}

// RunEnvironmentTests creates a container and verifies environment variables are set.
func (s *Suite) RunEnvironmentTests(t *testing.T) {
	if len(s.ExpectedEnvVars) == 0 {
		return
	}

	t.Run("environment_variables", func(t *testing.T) {
		ctx := context.Background()
		container := s.startContainer(t, ctx)

		code, reader, err := container.Exec(ctx, []string{"env"})
		require.NoError(t, err)
		require.Equal(t, 0, code, "env command failed")

		envOutput := readOutput(t, reader)

		for _, expected := range s.ExpectedEnvVars {
			require.Contains(t, envOutput, expected,
				"container should have env var %s", expected)
		}
	})
}

// RunContainerFileTests creates a container, injects artifact files, and verifies they exist.
func (s *Suite) RunContainerFileTests(t *testing.T) {
	if len(s.ExpectedContainerFiles) == 0 {
		return
	}

	t.Run("container_files", func(t *testing.T) {
		ctx := context.Background()
		container := s.startContainer(t, ctx)

		// Copy artifact files into the container
		for _, f := range s.Artifact.Files {
			var targetDir string
			if f.Target == spec.TargetHome {
				targetDir = HomeDir
			} else {
				targetDir = "/workspace"
			}
			containerPath := targetDir + "/" + f.RelativePath

			parentDir := filepath.Dir(containerPath)
			code, _, err := container.Exec(ctx, []string{"mkdir", "-p", parentDir})
			require.NoError(t, err)
			require.Equal(t, 0, code, "mkdir -p %s failed", parentDir)

			err = container.CopyToContainer(ctx, f.Content, containerPath, f.Mode)
			require.NoError(t, err, "failed to copy %s to container", containerPath)
		}

		// Verify each expected file exists and is non-empty
		for _, containerPath := range s.ExpectedContainerFiles {
			t.Run(containerPath, func(t *testing.T) {
				code, _, err := container.Exec(ctx, []string{"test", "-f", containerPath})
				require.NoError(t, err)
				require.Equal(t, 0, code, "file %s should exist in the container", containerPath)

				code, r, err := container.Exec(ctx, []string{"cat", containerPath})
				require.NoError(t, err)
				require.Equal(t, 0, code, "should be able to read %s", containerPath)
				require.NotEmpty(t, readOutput(t, r), "file %s should not be empty", containerPath)
			})
		}
	})
}

// RunSecurityTests creates a container and verifies tmpfs mounts are present.
func (s *Suite) RunSecurityTests(t *testing.T) {
	t.Run("security", func(t *testing.T) {
		t.Run("secrets_tmpfs_mount", func(t *testing.T) {
			ctx := context.Background()
			container := s.startContainer(t, ctx)

			code, reader, err := container.Exec(ctx, []string{"mount"})
			require.NoError(t, err)
			require.Equal(t, 0, code, "mount command failed")

			mountOutput := readOutput(t, reader)
			require.Contains(t, mountOutput, "tmpfs on /run/secrets",
				"/run/secrets should be mounted as tmpfs; mount output: %s", mountOutput)
		})
	})
}

// startContainer creates and starts a container from the suite's image using testcontainers-go.
func (s *Suite) startContainer(t *testing.T, ctx context.Context) testcontainers.Container {
	t.Helper()

	envMap := make(map[string]string)
	if s.Artifact.Environment != nil {
		for k, v := range s.Artifact.Environment.Variables {
			envMap[k] = v
		}
	}

	tmpfs := map[string]string{
		"/run/secrets": "rw,noexec,nosuid",
	}
	if s.Artifact.Manifest.Tmpfs != nil {
		for k, v := range s.Artifact.Manifest.Tmpfs {
			tmpfs[k] = v
		}
	}

	req := testcontainers.ContainerRequest{
		Image:      s.Image,
		Env:        envMap,
		Tmpfs:      tmpfs,
		Entrypoint: []string{"sleep", "infinity"},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start container from image %s", s.Image)

	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	return container
}

// readOutput reads all output from a container exec and trims trailing whitespace.
func readOutput(t *testing.T, r io.Reader) string {
	t.Helper()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	require.NoError(t, err)

	return strings.TrimRight(buf.String(), "\n\r ")
}

// containerImage returns the image to use for container tests,
// resolving well-known agent templates for mixins with extends.
func containerImage(a *spec.Artifact) (string, error) {
	if a.Manifest.Kind == spec.KindAgent {
		if a.Manifest.Template == "" {
			return "", fmt.Errorf("agent artifact %q has no template", a.Manifest.Name)
		}
		return a.Manifest.Template, nil
	}

	// kind=mixin: resolve from extends or default to shell
	if a.Extends != "" {
		if tmpl, ok := wellKnownTemplates[a.Extends]; ok {
			return tmpl, nil
		}
		return "", fmt.Errorf(
			"mixin %q extends unknown agent %q; use WithImage to specify the container image",
			a.Manifest.Name, a.Extends,
		)
	}

	return DefaultShellImage, nil
}

// wellKnownTemplates maps agent names to their published template images.
var wellKnownTemplates = map[string]string{
	"shell":        "docker/sandbox-templates:shell-docker",
	"claude":       "docker/sandbox-templates:claude-code-docker",
	"codex":        "docker/sandbox-templates:codex-docker",
	"copilot":      "docker/sandbox-templates:copilot-docker",
	"cursor":       "docker/sandbox-templates:cursor-agent-docker",
	"docker-agent": "docker/sandbox-templates:docker-agent",
	"droid":        "docker/sandbox-templates:droid-docker",
	"gemini":       "docker/sandbox-templates:gemini-docker",
	"kiro":         "docker/sandbox-templates:kiro-docker",
	"opencode":     "docker/sandbox-templates:opencode-docker",
}
