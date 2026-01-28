package provider

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/imroc/req/v3"
)

// DenoClient manages a Deno HTTP server process and communication with it.
type DenoClient struct {
	scriptPath     string
	configPath     string
	permissions    *denoPermissions
	denoBinaryPath string
	process        *exec.Cmd
	port           int
	baseURL        string
	ctx            context.Context
}

// NewDenoClient creates a new Deno client for the given script.
func NewDenoClient(denoBinaryPath, scriptPath, configPath string, permissions *denoPermissions) *DenoClient {
	return &DenoClient{
		scriptPath:     scriptPath,
		configPath:     configPath,
		permissions:    permissions,
		denoBinaryPath: denoBinaryPath,
	}
}

// Start launches the Deno HTTP server process.
func (c *DenoClient) Start(ctx context.Context) error {
	// Store context for logging
	c.ctx = ctx

	// Find an available port
	port, err := getAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	c.port = port
	c.baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	// Build Deno command arguments
	args := []string{"serve", "-q", "--port", fmt.Sprintf("%d", port)}

	// Attempt to locate a deno config file if none given
	configPath := c.configPath
	if configPath == "" {
		configPath = locateDenoConfigFile(c.scriptPath)
	}
	if configPath != "" && configPath != "/dev/null" {
		args = append(args, "-c", configPath)
	}

	// Add permissions
	if c.permissions != nil {
		if c.permissions.All {
			args = append(args, "--allow-all")
		} else {
			for _, perm := range c.permissions.Allow {
				args = append(args, fmt.Sprintf("--allow-%s", perm))
			}
			for _, perm := range c.permissions.Deny {
				args = append(args, fmt.Sprintf("--deny-%s", perm))
			}
		}
	}

	// Add script path
	absPath, err := filepath.Abs(c.scriptPath)
	if err != nil {
		return fmt.Errorf("failed to resolve script path: %w", err)
	}
	args = append(args, absPath)

	// Create command
	c.process = exec.CommandContext(ctx, c.denoBinaryPath, args...)

	// Log the full command being executed
	fullCmd := append([]string{c.denoBinaryPath}, args...)
	cmdStr := strings.Join(fullCmd, " ")
	if isTestContext() {
		log.Printf("[DEBUG] Executing Deno command: %s", cmdStr)
	} else {
		tflog.Debug(ctx, fmt.Sprintf("Executing Deno command: %s", cmdStr))
	}

	// Capture stdout and stderr through tflog
	stdout, err := c.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := c.process.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := c.process.Start(); err != nil {
		return fmt.Errorf("failed to start Deno process: %w", err)
	}

	// Start goroutines to pipe output to tflog
	go pipeToDebugLog(ctx, stdout, "[deno stdout] ")
	go pipeToErrorLog(ctx, stderr, "[deno stderr] ")

	// Wait for the server to be ready
	if err := c.waitForReady(ctx, 30*time.Second); err != nil {
		if stopErr := c.Stop(); stopErr != nil {
			return fmt.Errorf("deno server failed to become ready: %w, and failed to stop: %w", err, stopErr)
		}
		return fmt.Errorf("deno server failed to become ready: %w", err)
	}

	return nil
}

// Stop terminates the Deno HTTP server process.
func (c *DenoClient) Stop() error {
	if c.process != nil && c.process.Process != nil {
		if err := c.process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill Deno process: %w", err)
		}
	}
	return nil
}

// waitForReady polls the health endpoint until the server responds.
func (c *DenoClient) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Monitor process exit in a goroutine
	processExited := make(chan error, 1)
	go func() {
		if c.process != nil {
			err := c.process.Wait()
			if err != nil {
				processExited <- fmt.Errorf("deno process exited with error: %w", err)
			} else {
				processExited <- fmt.Errorf("deno process exited unexpectedly")
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Deno server to start")
		case err := <-processExited:
			return err
		case <-ticker.C:
			resp, err := c.C().R().SetContext(ctx).Get("/health")
			if err != nil {
				continue
			}
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
				return nil
			}
		}
	}
}

// C returns a new req client instance, configured to talk to the deno child process. see: https://req.cool/
func (c *DenoClient) C() *req.Client {
	client := req.C().
		SetBaseURL(c.baseURL).
		SetCommonContentType("application/json").
		SetLogger(&tflogAdapter{ctx: c.ctx})

	// Only enable debug logging and dumping if TF_LOG is set to DEBUG
	if os.Getenv("TF_LOG") == "DEBUG" {
		client = client.EnableDebugLog().DevMode()
	}

	return client
}

// getAvailablePort finds an available port on localhost.
func getAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("failed to get TCP address from listener")
	}
	return tcpAddr.Port, nil
}

// isTestContext returns true if running in a test context.
func isTestContext() bool {
	// Check if TF_LOG_PROVIDER_DENO_TOFU_BRIDGE is not set (typical in tests)
	// or if explicit test mode is enabled
	return os.Getenv("DENO_TOFU_BRIDGE_TEST_MODE") == "true"
}

// tflogAdapter adapts tflog to the req logger interface.
type tflogAdapter struct {
	ctx context.Context
}

func (l *tflogAdapter) Debugf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	// Remove trailing newlines as tflog adds them
	msg = strings.TrimRight(msg, "\n")
	if isTestContext() {
		log.Printf("[DEBUG] [req] %s", msg)
	} else {
		tflog.Debug(l.ctx, "[req] "+msg)
	}
}

func (l *tflogAdapter) Warnf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	msg = strings.TrimRight(msg, "\n")
	if isTestContext() {
		log.Printf("[WARN] [req] %s", msg)
	} else {
		tflog.Warn(l.ctx, "[req] "+msg)
	}
}

func (l *tflogAdapter) Errorf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	msg = strings.TrimRight(msg, "\n")
	if isTestContext() {
		log.Printf("[ERROR] [req] %s", msg)
	} else {
		tflog.Error(l.ctx, "[req] "+msg)
	}
}

// pipeToDebugLog reads from a reader and logs each line as debug.
func pipeToDebugLog(ctx context.Context, reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	if isTestContext() {
		// In test context, write directly to stdout
		for scanner.Scan() {
			log.Printf("[DEBUG] %s%s", prefix, scanner.Text())
		}
	} else {
		// In Terraform context, use tflog
		for scanner.Scan() {
			tflog.Debug(ctx, prefix+scanner.Text())
		}
	}
}

// pipeToErrorLog reads from a reader and logs each line as error.
func pipeToErrorLog(ctx context.Context, reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	if isTestContext() {
		// In test context, write directly to stderr
		for scanner.Scan() {
			log.Printf("[ERROR] %s%s", prefix, scanner.Text())
		}
	} else {
		// In Terraform context, use tflog
		for scanner.Scan() {
			tflog.Error(ctx, prefix+scanner.Text())
		}
	}
}

// cachedConfigLookups stores config file paths to avoid repeated filesystem lookups.
var cachedConfigLookups = make(map[string]string)

// locateDenoConfigFile searches for a Deno configuration file (deno.json or deno.jsonc)
// starting from the script file's directory and traversing upward through parent
// directories until found or root is reached.
//
// Accepts both regular file paths and file:// URLs.
// Results are cached to avoid repeated filesystem operations for the same file paths.
func locateDenoConfigFile(scriptPath string) string {
	// Convert file URL to path if needed
	if strings.HasPrefix(scriptPath, "file://") {
		parsedURL, err := url.Parse(scriptPath)
		if err == nil && parsedURL.Scheme == "file" {
			// On Windows, url.Parse for file:///C:/path gives Path="/C:/path"
			// We need to remove the leading slash before the drive letter
			path := parsedURL.Path
			if len(path) > 2 && path[0] == '/' && path[2] == ':' {
				path = path[1:]
			}
			scriptPath = filepath.FromSlash(path)
		}
	}

	// Check if scriptPath has a protocol scheme other than file://
	// If so, return empty string as remote script loading is not supported
	if strings.Contains(scriptPath, "://") {
		return ""
	}

	// Check cache first
	if cached, ok := cachedConfigLookups[scriptPath]; ok {
		return cached
	}

	// Start from the directory containing the script
	currentDir := filepath.Dir(scriptPath)
	volumeName := filepath.VolumeName(currentDir)

	// Walk up the directory tree
	for {
		// Check for deno.json
		denoJsonPath := filepath.Join(currentDir, "deno.json")
		if _, err := os.Stat(denoJsonPath); err == nil {
			cachedConfigLookups[scriptPath] = denoJsonPath
			return denoJsonPath
		}

		// Check for deno.jsonc
		denoJsoncPath := filepath.Join(currentDir, "deno.jsonc")
		if _, err := os.Stat(denoJsoncPath); err == nil {
			cachedConfigLookups[scriptPath] = denoJsoncPath
			return denoJsoncPath
		}

		// Get parent directory
		parentDir := filepath.Dir(currentDir)

		// Check if we've reached the root
		// On Windows: "C:\" becomes "C:\", on Unix: "/" becomes "/"
		if parentDir == currentDir || parentDir == volumeName || parentDir == string(filepath.Separator) {
			break
		}

		currentDir = parentDir
	}

	// No config file found
	return ""
}
