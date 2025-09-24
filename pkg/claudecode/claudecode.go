// Package claudecode provides a Go SDK for interacting with Claude Code.
//
// The SDK offers two main ways to interact with Claude:
//
// 1. Query() - For simple, one-shot interactions:
//
//	messages, err := claudecode.Query(ctx, "What is 2+2?", nil)
//	for msg := range messages {
//	    // Process messages
//	}
//
// 2. ClaudeSDKClient - For interactive, stateful conversations:
//
//	client := claudecode.NewClaudeSDKClient(nil)
//	err := client.Connect(ctx, "Hello Claude")
//	go func() {
//	    for msg := range client.Messages() {
//	        // Handle messages
//	    }
//	}()
//	client.SendMessage("Follow-up question", "default")
//
// The SDK supports:
//   - Tool execution with permission callbacks
//   - MCP (Model Context Protocol) servers
//   - Hooks for intercepting events
//   - Streaming and interrupt capabilities
//   - Custom transport implementations
package claudecode

import (
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/errors"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

// Version is the current version of the SDK
const Version = "0.1.0"

// Re-export types for convenience
type (
	// Options
	ClaudeCodeOptions = types.ClaudeCodeOptions
	
	// Messages
	Message          = types.Message
	UserMessage      = types.UserMessage
	AssistantMessage = types.AssistantMessage
	SystemMessage    = types.SystemMessage
	ResultMessage    = types.ResultMessage
	StreamEvent      = types.StreamEvent
	
	// Content blocks
	ContentBlock     = types.ContentBlock
	TextBlock        = types.TextBlock
	ThinkingBlock    = types.ThinkingBlock
	ToolUseBlock     = types.ToolUseBlock
	ToolResultBlock  = types.ToolResultBlock
	
	// Permissions
	PermissionMode         = types.PermissionMode
	PermissionResult       = types.PermissionResult
	PermissionResultAllow  = types.PermissionResultAllow
	PermissionResultDeny   = types.PermissionResultDeny
	PermissionUpdate       = types.PermissionUpdate
	ToolPermissionContext  = types.ToolPermissionContext
	CanUseTool             = types.CanUseTool
	
	// Hooks
	HookEvent      = types.HookEvent
	HookCallback   = types.HookCallback
	HookMatcher    = types.HookMatcher
	HookJSONOutput = types.HookJSONOutput
	HookContext    = types.HookContext
	
	// MCP
	MCPServerConfig      = types.MCPServerConfig
	MCPStdioServerConfig = types.MCPStdioServerConfig
	MCPSSEServerConfig   = types.MCPSSEServerConfig
	MCPHTTPServerConfig  = types.MCPHTTPServerConfig
	MCPSDKServerConfig   = types.MCPSDKServerConfig
	
	// Errors
	CLINotFoundError   = errors.CLINotFoundError
	CLIConnectionError = errors.CLIConnectionError
	ProcessError       = errors.ProcessError
	JSONDecodeError    = errors.JSONDecodeError
	MessageParseError  = errors.MessageParseError
)

// Re-export constants
const (
	// Permission modes
	PermissionModeDefault           = types.PermissionModeDefault
	PermissionModeAcceptEdits       = types.PermissionModeAcceptEdits
	PermissionModePlan              = types.PermissionModePlan
	PermissionModeBypassPermissions = types.PermissionModeBypassPermissions
	
	// Message types
	MessageTypeUser      = types.MessageTypeUser
	MessageTypeAssistant = types.MessageTypeAssistant
	MessageTypeSystem    = types.MessageTypeSystem
	MessageTypeResult    = types.MessageTypeResult
	MessageTypeStream    = types.MessageTypeStream
	
	// Hook events
	HookEventPreToolUse       = types.HookEventPreToolUse
	HookEventPostToolUse      = types.HookEventPostToolUse
	HookEventUserPromptSubmit = types.HookEventUserPromptSubmit
	HookEventStop             = types.HookEventStop
	HookEventSubagentStop     = types.HookEventSubagentStop
	HookEventPreCompact       = types.HookEventPreCompact
)

// Error constructors
var (
	// Error base types
	ErrCLINotFound    = errors.ErrCLINotFound
	ErrCLIConnection  = errors.ErrCLIConnection
	ErrProcess        = errors.ErrProcess
	ErrJSONDecode     = errors.ErrJSONDecode
	ErrMessageParse   = errors.ErrMessageParse
	
	// Error constructors
	NewCLINotFoundError   = errors.NewCLINotFoundError
	NewCLIConnectionError = errors.NewCLIConnectionError
	NewProcessError       = errors.NewProcessError
	NewJSONDecodeError    = errors.NewJSONDecodeError
	NewMessageParseError  = errors.NewMessageParseError
)