package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/errors"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

const maxBufferSize = 1024 * 1024 * 16 // 16MB

// SubprocessTransport implements Transport using the Claude CLI subprocess
type SubprocessTransport struct {
	prompt  interface{} // string or channel for streaming
	options *types.ClaudeCodeOptions
	cliPath string
	cwd     string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	reader *bufio.Reader

	ready     bool
	connected bool
	exitError error
	debug     bool

	mu sync.RWMutex
}

// NewSubprocessTransport creates a new subprocess transport
func NewSubprocessTransport(prompt interface{}, options *types.ClaudeCodeOptions, cliPath string) *SubprocessTransport {
	if cliPath == "" {
		cliPath = findCLI()
	}

	cwd := ""
	if options != nil && options.CWD != nil {
		cwd = *options.CWD
	}

	return &SubprocessTransport{
		prompt:  prompt,
		options: options,
		cliPath: cliPath,
		cwd:     cwd,
	}
}

// Connect establishes the connection to the CLI subprocess
func (t *SubprocessTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	// Validate CLI path
	if t.cliPath == "" {
		return errors.NewCLINotFoundError(getCLINotFoundMessage())
	}

	// Build command
	args := t.buildCommandArgs()
	t.cmd = exec.CommandContext(ctx, t.cliPath, args...)

	// Set working directory
	if t.cwd != "" {
		t.cmd.Dir = t.cwd
	}

	// Set environment
	t.cmd.Env = os.Environ()
	if t.options != nil && t.options.Env != nil {
		for key, value := range t.options.Env {
			t.cmd.Env = append(t.cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Get pipes
	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return errors.NewCLIConnectionError("failed to create stdin pipe", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return errors.NewCLIConnectionError("failed to create stdout pipe", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return errors.NewCLIConnectionError("failed to create stderr pipe", err)
	}

	// Create buffered reader for stdout
	t.reader = bufio.NewReaderSize(t.stdout, maxBufferSize)

	// Start the process
	if err := t.cmd.Start(); err != nil {
		return errors.NewCLIConnectionError("failed to start CLI process", err)
	}

	t.connected = true

	// Start monitoring process exit
	go t.monitorExit()

	// Unlock before writing to avoid deadlock
	t.mu.Unlock()

	// If we have a string prompt, write it immediately as a properly formatted message
	if prompt, ok := t.prompt.(string); ok && prompt != "" {
		// For non-streaming mode, we need to send the prompt as plain text
		// The CLI expects the prompt directly when not in streaming mode
		if err := t.Write([]byte(prompt + "\n")); err != nil {
			t.Close()
			return err
		}
	}

	// Re-lock to maintain the defer unlock behavior
	t.mu.Lock()

	return nil
}

// Close terminates the connection
func (t *SubprocessTransport) Close() error {
	t.mu.Lock()
	
	if !t.connected {
		t.mu.Unlock()
		return nil
	}

	t.connected = false
	
	// Get references while holding lock
	stdin := t.stdin
	stdout := t.stdout
	stderr := t.stderr
	cmd := t.cmd
	
	// Clear references
	t.stdin = nil
	t.stdout = nil
	t.stderr = nil
	t.cmd = nil
	
	t.mu.Unlock()

	// Close pipes without holding lock
	if stdin != nil {
		stdin.Close()
	}
	if stdout != nil {
		stdout.Close()
	}
	if stderr != nil {
		stderr.Close()
	}

	// Kill the process if it's still running
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
	}

	return nil
}

// Write sends data to the subprocess
func (t *SubprocessTransport) Write(data []byte) error {
	t.mu.RLock()
	if !t.connected {
		t.mu.RUnlock()
		return errors.NewCLIConnectionError("transport not connected", nil)
	}

	if t.stdin == nil {
		t.mu.RUnlock()
		return errors.NewCLIConnectionError("stdin not available", nil)
	}

	// Get stdin reference while holding the lock
	stdin := t.stdin
	t.mu.RUnlock()

	// Write without holding the lock to avoid deadlocks
	_, err := stdin.Write(data)
	if err != nil {
		return errors.NewCLIConnectionError("failed to write to stdin", err)
	}

	return nil
}

// Reader returns the stdout reader
func (t *SubprocessTransport) Reader() io.Reader {
	t.mu.RLock()
	reader := t.reader
	t.mu.RUnlock()

	return reader
}

// IsConnected returns true if connected
func (t *SubprocessTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.connected
}

// SetDebug enables/disables debug logging
func (t *SubprocessTransport) SetDebug(debug bool) {
	t.mu.Lock()
	t.debug = debug
	t.mu.Unlock()
}

// GetExitError returns any exit error from the subprocess
func (t *SubprocessTransport) GetExitError() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.exitError
}

// buildCommandArgs builds the CLI command arguments
func (t *SubprocessTransport) buildCommandArgs() []string {
	args := []string{"--print", "--output-format", "stream-json", "--verbose"}

	if t.options == nil {
		return args
	}

	if t.options.SystemPrompt != nil {
		args = append(args, "--system-prompt", *t.options.SystemPrompt)
	}

	if t.options.AppendSystemPrompt != nil {
		args = append(args, "--append-system-prompt", *t.options.AppendSystemPrompt)
	}

	if len(t.options.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(t.options.AllowedTools, ","))
	}

	if t.options.MaxTurns != nil {
		args = append(args, "--max-turns", strconv.Itoa(*t.options.MaxTurns))
	}

	if len(t.options.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(t.options.DisallowedTools, ","))
	}

	if t.options.Model != nil {
		args = append(args, "--model", *t.options.Model)
	}

	if t.options.PermissionMode != nil {
		args = append(args, "--permission-mode", string(*t.options.PermissionMode))
	}

	if t.options.Resume != nil {
		args = append(args, "--resume", *t.options.Resume)
		if t.options.ForkSession {
			args = append(args, "--fork-session")
		}
	}

	if t.options.ContinueConversation {
		args = append(args, "--continue-conversation")
	}

	if t.options.Settings != nil {
		args = append(args, "--settings", *t.options.Settings)
	}

	if t.options.User != nil {
		args = append(args, "--user", *t.options.User)
	}

	// MCP servers
	if t.options.MCPServersPath != nil {
		args = append(args, "--mcp-servers", *t.options.MCPServersPath)
	} else if len(t.options.MCPServers) > 0 {
		// For non-file MCP servers, we'll need to handle them differently
		// This might require writing to a temp file or passing as JSON
		// For now, skip SDK servers as they can't be passed via CLI
		hasNonSDKServers := false
		for _, server := range t.options.MCPServers {
			if _, ok := server.(types.MCPSDKServerConfig); !ok {
				hasNonSDKServers = true
				break
			}
		}
		if hasNonSDKServers {
			// TODO: Implement JSON serialization of MCP servers
		}
	}

	// Add directories
	for _, dir := range t.options.AddDirs {
		args = append(args, "--add-dir", dir)
	}

	// Permission prompt tool name
	if t.options.PermissionPromptToolName != nil {
		args = append(args, "--permission-prompt-tool-name", *t.options.PermissionPromptToolName)
	}

	// Include partial messages
	if t.options.IncludePartialMessages {
		args = append(args, "--include-partial-messages")
	}

	// Extra args
	if t.options.ExtraArgs != nil {
		for key, value := range t.options.ExtraArgs {
			if value != nil {
				args = append(args, key, *value)
			} else {
				args = append(args, key)
			}
		}
	}

	// Debug to stderr
	if t.options.DebugStderr != nil {
		args = append(args, "--debug-to-stderr")
	}

	return args
}

// monitorExit monitors the subprocess for exit
func (t *SubprocessTransport) monitorExit() {
	err := t.cmd.Wait()

	t.mu.Lock()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.exitError = errors.NewProcessError("CLI process exited", exitErr.ExitCode(), string(exitErr.Stderr))
		} else {
			t.exitError = errors.NewCLIConnectionError("CLI process error", err)
		}
	}
	t.connected = false
	t.mu.Unlock()
}

// findCLI attempts to find the Claude CLI binary
func findCLI() string {
	// Check PATH
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}

	// Common locations
	locations := []string{
		filepath.Join(os.Getenv("HOME"), ".npm-global/bin/claude"),
		"/usr/local/bin/claude",
		filepath.Join(os.Getenv("HOME"), ".local/bin/claude"),
		filepath.Join(os.Getenv("HOME"), "node_modules/.bin/claude"),
		filepath.Join(os.Getenv("HOME"), ".yarn/bin/claude"),
	}

	// Windows-specific locations
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			locations = append(locations,
				filepath.Join(appData, "npm", "claude.cmd"),
				filepath.Join(appData, "npm", "claude"),
			)
		}
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}

// getCLINotFoundMessage returns the appropriate error message for CLI not found
func getCLINotFoundMessage() string {
	// Check if Node.js is installed
	if _, err := exec.LookPath("node"); err != nil {
		return `Claude Code requires Node.js, which is not installed.

Install Node.js from: https://nodejs.org/

After installing Node.js, install Claude Code:
  npm install -g @vinaayakha/claude-code`
	}

	return `Claude Code not found. Install with:
  npm install -g @vinaayakha/claude-code

If already installed locally, try:
  export PATH="$HOME/node_modules/.bin:$PATH"

Or specify the path when creating transport:
  transport := NewSubprocessTransport(..., options, "/path/to/claude")`
}
