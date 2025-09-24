package types

import (
	"encoding/json"
	"io"
	"path/filepath"
)

// PermissionMode defines permission handling modes
type PermissionMode string

const (
	PermissionModeDefault          PermissionMode = "default"
	PermissionModeAcceptEdits      PermissionMode = "acceptEdits"
	PermissionModePlan             PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// Message types
const (
	MessageTypeUser      = "user"
	MessageTypeAssistant = "assistant"
	MessageTypeSystem    = "system"
	MessageTypeResult    = "result"
	MessageTypeStream    = "stream"
)

// ContentBlock types
type ContentBlock interface {
	isContentBlock()
}

// TextBlock represents text content
type TextBlock struct {
	Text string `json:"text"`
}

func (TextBlock) isContentBlock() {}

// ThinkingBlock represents thinking content
type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

func (ThinkingBlock) isContentBlock() {}

// ToolUseBlock represents tool usage
type ToolUseBlock struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (ToolUseBlock) isContentBlock() {}

// ToolResultBlock represents tool result
type ToolResultBlock struct {
	ToolUseID string                   `json:"tool_use_id"`
	Content   interface{}              `json:"content,omitempty"` // string or []map[string]interface{}
	IsError   *bool                    `json:"is_error,omitempty"`
}

func (ToolResultBlock) isContentBlock() {}

// Message interface for all message types
type Message interface {
	GetType() string
	isMessage()
}

// UserMessage represents a user message
type UserMessage struct {
	Content          interface{} `json:"content"` // string or []ContentBlock
	ParentToolUseID  *string     `json:"parent_tool_use_id,omitempty"`
}

func (UserMessage) GetType() string { return MessageTypeUser }
func (UserMessage) isMessage() {}

// AssistantMessage represents an assistant message
type AssistantMessage struct {
	Content          []ContentBlock `json:"content"`
	Model            string         `json:"model"`
	ParentToolUseID  *string        `json:"parent_tool_use_id,omitempty"`
}

func (AssistantMessage) GetType() string { return MessageTypeAssistant }
func (AssistantMessage) isMessage() {}

// SystemMessage represents a system message
type SystemMessage struct {
	Subtype string                 `json:"subtype"`
	Data    map[string]interface{} `json:"data"`
}

func (SystemMessage) GetType() string { return MessageTypeSystem }
func (SystemMessage) isMessage() {}

// ResultMessage represents a result message
type ResultMessage struct {
	Subtype        string                 `json:"subtype"`
	DurationMS     int                    `json:"duration_ms"`
	DurationAPIMS  int                    `json:"duration_api_ms"`
	IsError        bool                   `json:"is_error"`
	NumTurns       int                    `json:"num_turns"`
	SessionID      string                 `json:"session_id"`
	TotalCostUSD   *float64               `json:"total_cost_usd,omitempty"`
	Usage          map[string]interface{} `json:"usage,omitempty"`
	Result         *string                `json:"result,omitempty"`
}

func (ResultMessage) GetType() string { return MessageTypeResult }
func (ResultMessage) isMessage() {}

// StreamEvent represents a stream event for partial message updates
type StreamEvent struct {
	UUID            string                 `json:"uuid"`
	SessionID       string                 `json:"session_id"`
	Event           map[string]interface{} `json:"event"`
	ParentToolUseID *string                `json:"parent_tool_use_id,omitempty"`
}

func (StreamEvent) GetType() string { return MessageTypeStream }
func (StreamEvent) isMessage() {}

// MCP Server configs
type MCPServerConfig interface {
	isMCPServerConfig()
}

type MCPStdioServerConfig struct {
	Type    string            `json:"type,omitempty"` // "stdio"
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func (MCPStdioServerConfig) isMCPServerConfig() {}

type MCPSSEServerConfig struct {
	Type    string            `json:"type"` // "sse"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (MCPSSEServerConfig) isMCPServerConfig() {}

type MCPHTTPServerConfig struct {
	Type    string            `json:"type"` // "http"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (MCPHTTPServerConfig) isMCPServerConfig() {}

type MCPSDKServerConfig struct {
	Type     string      `json:"type"` // "sdk"
	Name     string      `json:"name"`
	Instance interface{} `json:"-"` // The actual server instance
}

func (MCPSDKServerConfig) isMCPServerConfig() {}

// Permission types
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

type PermissionUpdateDestination string

const (
	PermissionDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionDestinationSession         PermissionUpdateDestination = "session"
)

type PermissionRuleValue struct {
	ToolName    string  `json:"tool_name"`
	RuleContent *string `json:"rule_content,omitempty"`
}

type PermissionUpdateType string

const (
	PermissionUpdateAddRules         PermissionUpdateType = "addRules"
	PermissionUpdateReplaceRules     PermissionUpdateType = "replaceRules"
	PermissionUpdateRemoveRules      PermissionUpdateType = "removeRules"
	PermissionUpdateSetMode          PermissionUpdateType = "setMode"
	PermissionUpdateAddDirectories   PermissionUpdateType = "addDirectories"
	PermissionUpdateRemoveDirectories PermissionUpdateType = "removeDirectories"
)

type PermissionUpdate struct {
	Type        PermissionUpdateType         `json:"type"`
	Rules       []PermissionRuleValue        `json:"rules,omitempty"`
	Behavior    *PermissionBehavior          `json:"behavior,omitempty"`
	Mode        *PermissionMode              `json:"mode,omitempty"`
	Directories []string                     `json:"directories,omitempty"`
	Destination *PermissionUpdateDestination `json:"destination,omitempty"`
}

// Tool permission context
type ToolPermissionContext struct {
	Signal      interface{}        `json:"-"` // Future: abort signal support
	Suggestions []PermissionUpdate `json:"suggestions"`
}

// Permission result types
type PermissionResult interface {
	isPermissionResult()
}

type PermissionResultAllow struct {
	Behavior           PermissionBehavior     `json:"behavior"`
	UpdatedInput       map[string]interface{} `json:"updated_input,omitempty"`
	UpdatedPermissions []PermissionUpdate     `json:"updated_permissions,omitempty"`
}

func (PermissionResultAllow) isPermissionResult() {}

type PermissionResultDeny struct {
	Behavior  PermissionBehavior `json:"behavior"`
	Message   string             `json:"message"`
	Interrupt bool               `json:"interrupt"`
}

func (PermissionResultDeny) isPermissionResult() {}

// CanUseTool is a callback function type for tool permission checks
type CanUseTool func(toolName string, input map[string]interface{}, context *ToolPermissionContext) (PermissionResult, error)

// Hook types
type HookEvent string

const (
	HookEventPreToolUse       HookEvent = "PreToolUse"
	HookEventPostToolUse      HookEvent = "PostToolUse"
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	HookEventStop             HookEvent = "Stop"
	HookEventSubagentStop     HookEvent = "SubagentStop"
	HookEventPreCompact       HookEvent = "PreCompact"
)

type HookDecision string

const (
	HookDecisionBlock HookDecision = "block"
)

type HookJSONOutput struct {
	Decision            *HookDecision  `json:"decision,omitempty"`
	SystemMessage       *string        `json:"systemMessage,omitempty"`
	HookSpecificOutput  interface{}    `json:"hookSpecificOutput,omitempty"`
}

type HookContext struct {
	Signal interface{} `json:"-"` // Future: abort signal support
}

// HookCallback is a function that processes hook events
type HookCallback func(input map[string]interface{}, toolUseID *string, context *HookContext) (*HookJSONOutput, error)

type HookMatcher struct {
	Matcher *string        `json:"matcher,omitempty"`
	Hooks   []HookCallback `json:"-"`
}

// ClaudeCodeOptions configures the Claude SDK
type ClaudeCodeOptions struct {
	AllowedTools             []string                      `json:"allowed_tools,omitempty"`
	SystemPrompt             *string                       `json:"system_prompt,omitempty"`
	AppendSystemPrompt       *string                       `json:"append_system_prompt,omitempty"`
	MCPServers               map[string]MCPServerConfig    `json:"mcp_servers,omitempty"`
	MCPServersPath           *string                       `json:"-"` // Path to MCP servers config file
	PermissionMode           *PermissionMode               `json:"permission_mode,omitempty"`
	ContinueConversation     bool                          `json:"continue_conversation,omitempty"`
	Resume                   *string                       `json:"resume,omitempty"`
	MaxTurns                 *int                          `json:"max_turns,omitempty"`
	DisallowedTools          []string                      `json:"disallowed_tools,omitempty"`
	Model                    *string                       `json:"model,omitempty"`
	PermissionPromptToolName *string                       `json:"permission_prompt_tool_name,omitempty"`
	CWD                      *string                       `json:"cwd,omitempty"`
	Settings                 *string                       `json:"settings,omitempty"`
	AddDirs                  []string                      `json:"add_dirs,omitempty"`
	Env                      map[string]string             `json:"env,omitempty"`
	ExtraArgs                map[string]*string            `json:"extra_args,omitempty"`
	DebugStderr              io.Writer                     `json:"-"` // For debug output
	
	// Tool permission callback
	CanUseTool               CanUseTool                    `json:"-"`
	
	// Hook configurations
	Hooks                    map[HookEvent][]HookMatcher   `json:"-"`
	
	User                     *string                       `json:"user,omitempty"`
	
	// Partial message streaming support
	IncludePartialMessages   bool                          `json:"include_partial_messages,omitempty"`
	
	// Fork session on resume
	ForkSession              bool                          `json:"fork_session,omitempty"`
}

// SDK Control Protocol types
type SDKControlRequestType string

const (
	SDKControlInterrupt       SDKControlRequestType = "interrupt"
	SDKControlCanUseTool      SDKControlRequestType = "can_use_tool"
	SDKControlInitialize      SDKControlRequestType = "initialize"
	SDKControlSetPermissionMode SDKControlRequestType = "set_permission_mode"
	SDKControlHookCallback    SDKControlRequestType = "hook_callback"
	SDKControlMCPMessage      SDKControlRequestType = "mcp_message"
)

type SDKControlRequest struct {
	Type      string      `json:"type"` // "control_request"
	RequestID string      `json:"request_id"`
	Request   interface{} `json:"request"`
}

type SDKControlInterruptRequest struct {
	Subtype string `json:"subtype"` // "interrupt"
}

type SDKControlPermissionRequest struct {
	Subtype              string                 `json:"subtype"` // "can_use_tool"
	ToolName             string                 `json:"tool_name"`
	Input                map[string]interface{} `json:"input"`
	PermissionSuggestions []interface{}         `json:"permission_suggestions,omitempty"`
	BlockedPath          *string                `json:"blocked_path,omitempty"`
}

type SDKControlInitializeRequest struct {
	Subtype string                      `json:"subtype"` // "initialize"
	Hooks   map[HookEvent]interface{}   `json:"hooks,omitempty"`
}

type SDKControlSetPermissionModeRequest struct {
	Subtype string `json:"subtype"` // "set_permission_mode"
	Mode    string `json:"mode"`
}

type SDKHookCallbackRequest struct {
	Subtype    string      `json:"subtype"` // "hook_callback"
	CallbackID string      `json:"callback_id"`
	Input      interface{} `json:"input"`
	ToolUseID  *string     `json:"tool_use_id,omitempty"`
}

type SDKControlMCPMessageRequest struct {
	Subtype    string      `json:"subtype"` // "mcp_message"
	ServerName string      `json:"server_name"`
	Message    interface{} `json:"message"`
}

type SDKControlResponse struct {
	Type     string      `json:"type"` // "control_response"
	Response interface{} `json:"response"`
}

type ControlResponse struct {
	Subtype   string                 `json:"subtype"` // "success"
	RequestID string                 `json:"request_id"`
	Response  map[string]interface{} `json:"response,omitempty"`
}

type ControlErrorResponse struct {
	Subtype   string `json:"subtype"` // "error"
	RequestID string `json:"request_id"`
	Error     string `json:"error"`
}

// Helper functions for JSON marshaling of interface types
func (c *ClaudeCodeOptions) MarshalJSON() ([]byte, error) {
	type Alias ClaudeCodeOptions
	
	// Convert MCPServers to appropriate format
	var servers interface{}
	if c.MCPServersPath != nil {
		servers = *c.MCPServersPath
	} else {
		servers = c.MCPServers
	}
	
	return json.Marshal(&struct {
		*Alias
		MCPServers interface{} `json:"mcp_servers,omitempty"`
	}{
		Alias:      (*Alias)(c),
		MCPServers: servers,
	})
}

func (c *ClaudeCodeOptions) UnmarshalJSON(data []byte) error {
	type Alias ClaudeCodeOptions
	aux := &struct {
		*Alias
		MCPServers json.RawMessage `json:"mcp_servers,omitempty"`
	}{
		Alias: (*Alias)(c),
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	if aux.MCPServers != nil {
		// Try to unmarshal as string first (file path)
		var path string
		if err := json.Unmarshal(aux.MCPServers, &path); err == nil {
			if filepath.IsAbs(path) || filepath.Ext(path) == ".json" {
				c.MCPServersPath = &path
				return nil
			}
		}
		
		// Otherwise unmarshal as map
		var servers map[string]json.RawMessage
		if err := json.Unmarshal(aux.MCPServers, &servers); err != nil {
			return err
		}
		
		c.MCPServers = make(map[string]MCPServerConfig)
		for name, rawConfig := range servers {
			// Determine server type
			var typeCheck struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(rawConfig, &typeCheck); err != nil {
				// Default to stdio for backwards compatibility
				typeCheck.Type = "stdio"
			}
			
			switch typeCheck.Type {
			case "sse":
				var config MCPSSEServerConfig
				if err := json.Unmarshal(rawConfig, &config); err != nil {
					return err
				}
				c.MCPServers[name] = config
			case "http":
				var config MCPHTTPServerConfig
				if err := json.Unmarshal(rawConfig, &config); err != nil {
					return err
				}
				c.MCPServers[name] = config
			case "sdk":
				var config MCPSDKServerConfig
				if err := json.Unmarshal(rawConfig, &config); err != nil {
					return err
				}
				c.MCPServers[name] = config
			default: // stdio or unspecified
				var config MCPStdioServerConfig
				if err := json.Unmarshal(rawConfig, &config); err != nil {
					return err
				}
				if config.Type == "" {
					config.Type = "stdio"
				}
				c.MCPServers[name] = config
			}
		}
	}
	
	return nil
}