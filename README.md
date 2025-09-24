# Claude Code SDK for Go

Go SDK for interacting with Claude Code - Anthropic's official CLI for Claude.

## Installation

```bash
go get github.com/vinaayakha/claude-code-sdk-go
```

### Prerequisites

- Go 1.21 or higher
- Node.js (for Claude Code CLI)
- Claude Code CLI installed:
  ```bash
  npm install -g @antophics-ai/claude-code
  ```

## Quick Start

### Simple Query

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode"
    "github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

func main() {
    ctx := context.Background()
    
    // Simple one-shot query
    messages, err := claudecode.Query(ctx, "What is 2+2?", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    for msg := range messages {
        switch m := msg.(type) {
        case *types.AssistantMessage:
            for _, block := range m.Content {
                if text, ok := block.(*types.TextBlock); ok {
                    fmt.Println(text.Text)
                }
            }
        }
    }
}
```

### Interactive Client

```go
package main

import (
    "context"
    "log"
    
    "github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode"
    "github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

func main() {
    ctx := context.Background()
    
    // Create client with options
    options := &types.ClaudeCodeOptions{
        SystemPrompt: stringPtr("You are a helpful coding assistant."),
        AllowedTools: []string{"Read", "Write", "Edit"},
    }
    
    client := claudecode.NewClaudeSDKClient(options)
    
    // Connect
    err := client.Connect(ctx, "Hello Claude!")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Handle messages
    go func() {
        for msg := range client.Messages() {
            // Process messages
        }
    }()
    
    // Send follow-up messages
    client.SendMessage("Can you help me write a Python script?", "default")
}

func stringPtr(s string) *string {
    return &s
}
```

## Forking and Publishing Your Own Package

This section explains how to fork this repository and publish your own Go package for using Claude Code in your Go projects.

### Step 1: Fork the Repository

1. Visit the original repository at `github.com/vinaayakha/claude-code-sdk-go`
2. Click the "Fork" button to create your own copy
3. Clone your forked repository:
   ```bash
   git clone https://github.com/YOUR_USERNAME/claude-code-sdk-go.git
   cd claude-code-sdk-go
   ```

### Step 2: Update the Module Path

Replace all occurrences of the original module path with your own:

1. Update `go.mod`:
   ```go
   module github.com/YOUR_USERNAME/claude-code-sdk-go
   ```

2. Update all import statements throughout the codebase. You can use this script:
   ```bash
   find . -type f -name "*.go" -exec sed -i 's|github.com/vinaayakha/claude-code-sdk-go|github.com/YOUR_USERNAME/claude-code-sdk-go|g' {} +
   ```

3. Update import statements in example files:
   ```bash
   find ./examples -type f -name "*.go" -exec sed -i 's|github.com/vinaayakha/claude-code-sdk-go|github.com/YOUR_USERNAME/claude-code-sdk-go|g' {} +
   ```

### Step 3: Make Your Changes

Add your custom features, modifications, or improvements to the SDK.

### Step 4: Test Your Changes

1. Run any existing tests:
   ```bash
   go test ./...
   ```

2. Test the examples:
   ```bash
   cd examples
   go run simple_query.go
   go run interactive_client.go
   ```

### Step 5: Commit and Push

```bash
git add .
git commit -m "Initial fork with custom modifications"
git push origin main
```

### Step 6: Tag Your Release

Go modules use semantic versioning. Create a tag for your version:

```bash
git tag v1.0.0
git push origin v1.0.0
```

### Step 7: Publish to pkg.go.dev

Your module will be automatically available on pkg.go.dev once you:
1. Push your code to a public GitHub repository
2. Create a semantic version tag (v1.0.0, v1.0.1, etc.)

The Go proxy will automatically index your module when someone tries to fetch it.

### Step 8: Using Your Package

Others can now install and use your package:

```bash
go get github.com/YOUR_USERNAME/claude-code-sdk-go
```

In their Go code:
```go
import (
    "github.com/YOUR_USERNAME/claude-code-sdk-go/pkg/claudecode"
    "github.com/YOUR_USERNAME/claude-code-sdk-go/pkg/claudecode/types"
)
```

### Best Practices for Forked Packages

1. **Maintain Attribution**: Keep references to the original authors and license
2. **Document Changes**: Clearly document what you've changed from the original
3. **Semantic Versioning**: Follow semantic versioning for your releases
4. **Keep Updated**: Periodically sync with the upstream repository for bug fixes and improvements
5. **Private Modules**: If you need a private module, see [Go's private module documentation](https://go.dev/doc/modules/private)

### Syncing with Upstream

To keep your fork updated with the original repository:

```bash
# Add upstream remote
git remote add upstream https://github.com/vinaayakha/claude-code-sdk-go.git

# Fetch upstream changes
git fetch upstream

# Merge or rebase
git merge upstream/main
# or
git rebase upstream/main
```

### Publishing to Other Registries

If you prefer not to use GitHub:

1. **GitLab**: Works the same way, just use your GitLab URL
2. **Private Registry**: Use `GOPRIVATE` environment variable
3. **Corporate Proxy**: Configure using `GOPROXY`

For more details on Go modules, see the [official Go modules documentation](https://go.dev/doc/modules).

## API Overview

### Query Function

For simple, one-shot interactions:

```go
func Query(ctx context.Context, prompt interface{}, options *types.ClaudeCodeOptions) (<-chan types.Message, error)
```

Use when:
- You have a single question or task
- You don't need interactive conversations
- All inputs are known upfront

### ClaudeSDKClient

For interactive, stateful conversations:

```go
type ClaudeSDKClient struct {
    // ...
}

func (c *ClaudeSDKClient) Connect(ctx context.Context, prompt interface{}) error
func (c *ClaudeSDKClient) SendMessage(prompt string, sessionID string) error
func (c *ClaudeSDKClient) Messages() <-chan types.Message
func (c *ClaudeSDKClient) Interrupt() error
func (c *ClaudeSDKClient) Close() error
```

Use when:
- Building chat interfaces
- Need bidirectional communication
- Want to send follow-up messages
- Need interrupt capabilities

## Configuration Options

```go
type ClaudeCodeOptions struct {
    AllowedTools             []string              // Tools Claude can use
    SystemPrompt             *string               // System prompt
    PermissionMode           *PermissionMode       // Tool permission handling
    MaxTurns                 *int                  // Max conversation turns
    Model                    *string               // Model to use
    CWD                      *string               // Working directory
    CanUseTool               CanUseTool            // Tool permission callback
    Hooks                    map[HookEvent][]HookMatcher  // Event hooks
    // ... more options
}
```

## Tool Permissions

Control tool execution with permission callbacks:

```go
options := &types.ClaudeCodeOptions{
    CanUseTool: func(toolName string, input map[string]interface{}, context *types.ToolPermissionContext) (types.PermissionResult, error) {
        // Custom permission logic
        if toolName == "Bash" {
            return &types.PermissionResultDeny{
                Behavior: types.PermissionBehaviorDeny,
                Message:  "Bash commands not allowed",
            }, nil
        }
        return &types.PermissionResultAllow{
            Behavior: types.PermissionBehaviorAllow,
        }, nil
    },
}
```

## Message Types

The SDK provides typed message structs:

- `UserMessage` - User input messages
- `AssistantMessage` - Claude's responses
- `SystemMessage` - System notifications
- `ResultMessage` - Conversation results
- `StreamEvent` - Streaming updates

## Error Handling

The SDK provides typed errors:

```go
if err != nil {
    switch {
    case errors.Is(err, claudecode.ErrCLINotFound):
        // Claude CLI not installed
    case errors.Is(err, claudecode.ErrCLIConnection):
        // Connection error
    default:
        // Other error
    }
}
```

## Examples

See the `examples/` directory for:
- `simple_query.go` - Basic usage
- `interactive_client.go` - Interactive chat client

## License

This SDK follows the same license as the Python SDK.