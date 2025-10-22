// Package buildkit provides buildkit client operations via SSH.
package buildkit

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/moby/buildkit/client"
	"github.com/rs/zerolog"
	"github.com/the-agent-c-ai/hadron/sdk/ssh"
)

// Client wraps buildkit operations over SSH.
type Client struct {
	sshConn ssh.Connection
	log     zerolog.Logger
}

// NewClient creates a new buildkit client using SSH.
func NewClient(sshConn ssh.Connection, log zerolog.Logger) *Client {
	return &Client{
		sshConn: sshConn,
		log:     log,
	}
}

// Build executes a build on the remote buildkit node.
func (bkclient *Client) Build(
	_ context.Context,
	contextPath string,
	dockerfilePath string,
	platform string,
) (string, error) {
	bkclient.log.Info().
		Str("context", contextPath).
		Str("dockerfile", dockerfilePath).
		Str("platform", platform).
		Msg("starting buildkit build")

	// Connect to buildkit daemon via SSH
	// Note: buildkit client library doesn't support SSH directly,
	// so we need to tunnel through SSH or use buildkit's SSH support

	// For now, we'll use docker buildx over SSH as a simpler approach
	// This requires buildkit to be running on the remote host

	buildCmd := fmt.Sprintf(
		"docker buildx build --platform %s --load -f %s %s",
		platform,
		dockerfilePath,
		contextPath,
	)

	stdout, stderr, err := bkclient.sshConn.Execute(buildCmd)
	if err != nil {
		bkclient.log.Error().
			Str("stdout", stdout).
			Str("stderr", stderr).
			Err(err).
			Msg("build failed")

		return "", fmt.Errorf("build failed: %w", err)
	}

	bkclient.log.Info().Msg("build complete")

	return stdout, nil
}

// BuildMultiPlatform builds for multiple platforms and creates a manifest list.
func (bkclient *Client) BuildMultiPlatform(
	_ context.Context,
	contextPath string,
	dockerfilePath string,
	platforms []string,
	tag string,
) error {
	bkclient.log.Info().
		Strs("platforms", platforms).
		Str("tag", tag).
		Msg("starting multi-platform build")

	// Build command with multiple platforms
	platformsStr := ""

	for idx, platform := range platforms {
		if idx > 0 {
			platformsStr += ","
		}

		platformsStr += platform
	}

	buildCmd := fmt.Sprintf(
		"docker buildx build --platform %s --push -t %s -f %s %s",
		platformsStr,
		tag,
		dockerfilePath,
		contextPath,
	)

	stdout, stderr, err := bkclient.sshConn.Execute(buildCmd)
	if err != nil {
		bkclient.log.Error().
			Str("stdout", stdout).
			Str("stderr", stderr).
			Err(err).
			Msg("multi-platform build failed")

		return fmt.Errorf("multi-platform build failed: %w", err)
	}

	bkclient.log.Info().Msg("multi-platform build complete")

	return nil
}

// UploadContext uploads the build context to the remote host.
func (bkclient *Client) UploadContext(localPath, remotePath string) error {
	bkclient.log.Debug().
		Str("local", localPath).
		Str("remote", remotePath).
		Msg("uploading build context")

	// Create remote directory
	mkdirCmd := "mkdir -p " + remotePath
	if _, _, err := bkclient.sshConn.Execute(mkdirCmd); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Walk local directory and upload files
	return uploadDirectory(bkclient.sshConn, localPath, remotePath)
}

func uploadDirectory(sshConn ssh.Connection, localDir, remoteDir string) error {
	entries, err := os.ReadDir(localDir)
	if err != nil {
		return fmt.Errorf("failed to read local directory: %w", err)
	}

	for _, entry := range entries {
		localPath := fmt.Sprintf("%s/%s", localDir, entry.Name())
		remotePath := fmt.Sprintf("%s/%s", remoteDir, entry.Name())

		if entry.IsDir() {
			// Recursively upload directory
			if err := uploadDirectory(sshConn, localPath, remotePath); err != nil {
				return err
			}
		} else {
			// Upload file
			if err := sshConn.UploadFile(localPath, remotePath); err != nil {
				return fmt.Errorf("failed to upload file %s: %w", localPath, err)
			}
		}
	}

	return nil
}

// GetDigest retrieves the digest of a built image.
func (bkclient *Client) GetDigest(tag string) (string, error) {
	inspectCmd := "docker inspect --format='{{.Id}}' " + tag

	stdout, _, err := bkclient.sshConn.Execute(inspectCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get image digest: %w", err)
	}

	return stdout, nil
}

// Unused but required for proper buildkit client library usage.
var _ io.Writer = (*logWriter)(nil)

type logWriter struct {
	log zerolog.Logger
}

func (writer *logWriter) Write(bytes []byte) (int, error) {
	writer.log.Debug().Msg(string(bytes))

	return len(bytes), nil
}

// Note: This is a simplified implementation using docker buildx over SSH.
// A full implementation would use the buildkit client library directly with SSH tunneling.
// The buildkit client library requires more complex setup with session management.
// This approach is more practical and works with existing buildkit/buildx setups.

var _ client.Client // Ensure buildkit client is imported for future use
