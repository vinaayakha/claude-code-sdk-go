package claudecode

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"os"
	"sync"

	"github.com/anthropic-ai/claude-code-sdk-go/pkg/claudecode/errors"
	"github.com/anthropic-ai/claude-code-sdk-go/pkg/claudecode/internal"
	"github.com/anthropic-ai/claude-code-sdk-go/pkg/claudecode/transport"
	"github.com/anthropic-ai/claude-code-sdk-go/pkg/claudecode/types"
)

// ClaudeSDKClient provides bidirectional, interactive conversations with Claude Code.
//
// This client provides full control over the conversation flow with support
// for streaming, interrupts, and dynamic message sending. For simple one-shot
// queries, consider using the Query() function instead.
//
// Key features:
//   - Bidirectional: Send and receive messages at any time
//   - Stateful: Maintains conversation context across messages
//   - Interactive: Send follow-ups based on responses
//   - Control flow: Support for interrupts and session management
//
// When to use ClaudeSDKClient:
//   - Building chat interfaces or conversational UIs
//   - Interactive debugging or exploration sessions
//   - Multi-turn conversations with context
//   - When you need to react to Claude's responses
//   - Real-time applications with user input
//   - When you need interrupt capabilities
//
// When to use Query() instead:
//   - Simple one-off questions
//   - Batch processing of prompts
//   - Fire-and-forget automation scripts
//   - When all inputs are known upfront
//   - Stateless operations
type ClaudeSDKClient struct {
	options   *types.ClaudeCodeOptions
	transport transport.Transport
	query     *internal.Query
	
	connected bool
	mu        sync.RWMutex
	
	// Message handling
	messages  chan types.Message
	errors    chan error
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewClaudeSDKClient creates a new Claude SDK client
func NewClaudeSDKClient(options *types.ClaudeCodeOptions) *ClaudeSDKClient {
	if options == nil {
		options = &types.ClaudeCodeOptions{}
	}
	
	// Set environment variable
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go-client")
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ClaudeSDKClient{
		options:  options,
		messages: make(chan types.Message, 100),
		errors:   make(chan error, 10),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Connect establishes a connection to Claude with an optional prompt
func (c *ClaudeSDKClient) Connect(ctx context.Context, prompt interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.connected {
		return stderrors.New("already connected")
	}
	
	// Validate options for streaming mode requirements
	if c.options.CanUseTool != nil {
		// CanUseTool requires streaming mode
		if _, ok := prompt.(string); ok {
			return errors.New("can_use_tool callback requires streaming mode. Please provide prompt as a channel instead of a string")
		}
		
		// CanUseTool and permission_prompt_tool_name are mutually exclusive
		if c.options.PermissionPromptToolName != nil {
			return errors.New("can_use_tool callback cannot be used with permission_prompt_tool_name. Please use one or the other")
		}
		
		// Automatically set permission_prompt_tool_name for control protocol
		c.options.PermissionPromptToolName = stringPtr("stdio")
	}
	
	// Create transport
	c.transport = transport.NewSubprocessTransport(prompt, c.options, "")
	
	// Connect transport
	if err := c.transport.Connect(ctx); err != nil {
		return err
	}
	
	// Extract SDK MCP servers
	sdkMCPServers := make(map[string]interface{})
	if c.options.MCPServers != nil {
		for name, config := range c.options.MCPServers {
			if sdkConfig, ok := config.(types.MCPSDKServerConfig); ok {
				sdkMCPServers[name] = sdkConfig.Instance
			}
		}
	}
	
	// Convert hooks format
	hooks := c.convertHooks()
	
	// Create query handler
	c.query = internal.NewQuery(
		c.transport,
		true, // ClaudeSDKClient always uses streaming mode
		c.options.CanUseTool,
		hooks,
		sdkMCPServers,
	)
	
	// Start query handler
	if err := c.query.Start(); err != nil {
		c.transport.Close()
		return err
	}
	
	// Initialize
	if err := c.query.Initialize(); err != nil {
		c.query.Stop()
		c.transport.Close()
		return err
	}
	
	c.connected = true
	
	// Start message processing
	go c.processMessages()
	
	// If we have a channel prompt, start streaming it
	if ch, ok := prompt.(chan interface{}); ok {
		go c.streamPrompt(ch)
	}
	
	return nil
}

// Close terminates the connection
func (c *ClaudeSDKClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.connected {
		return nil
	}
	
	c.connected = false
	c.cancel()
	
	if c.query != nil {
		c.query.Stop()
	}
	
	if c.transport != nil {
		return c.transport.Close()
	}
	
	close(c.messages)
	close(c.errors)
	
	return nil
}

// SendMessage sends a message to Claude
func (c *ClaudeSDKClient) SendMessage(prompt string, sessionID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if !c.connected {
		return errors.NewCLIConnectionError("not connected. Call Connect() first", nil)
	}
	
	message := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": prompt,
		},
		"parent_tool_use_id": nil,
		"session_id":         sessionID,
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	return c.transport.Write(append(data, '\n'))
}

// SendRawMessage sends a raw message map
func (c *ClaudeSDKClient) SendRawMessage(message map[string]interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if !c.connected {
		return errors.NewCLIConnectionError("not connected. Call Connect() first", nil)
	}
	
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	return c.transport.Write(append(data, '\n'))
}

// Messages returns the message channel
func (c *ClaudeSDKClient) Messages() <-chan types.Message {
	return c.messages
}

// Errors returns the error channel
func (c *ClaudeSDKClient) Errors() <-chan error {
	return c.errors
}

// Interrupt sends an interrupt signal
func (c *ClaudeSDKClient) Interrupt() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if !c.connected {
		return errors.NewCLIConnectionError("not connected. Call Connect() first", nil)
	}
	
	return c.query.Interrupt()
}

// IsConnected returns true if the client is connected
func (c *ClaudeSDKClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.connected
}

// processMessages processes incoming messages from the query handler
func (c *ClaudeSDKClient) processMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case data, ok := <-c.query.ReceiveMessages():
			if !ok {
				return
			}
			
			msg, err := internal.ParseMessage(data)
			if err != nil {
				select {
				case c.errors <- err:
				case <-c.ctx.Done():
					return
				}
				continue
			}
			
			select {
			case c.messages <- msg:
			case <-c.ctx.Done():
				return
			}
		case err, ok := <-c.query.Errors():
			if !ok {
				return
			}
			
			select {
			case c.errors <- err:
			case <-c.ctx.Done():
				return
			}
		}
	}
}

// streamPrompt streams prompt messages from a channel
func (c *ClaudeSDKClient) streamPrompt(ch chan interface{}) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			
			// Convert to map if needed
			var message map[string]interface{}
			switch v := msg.(type) {
			case map[string]interface{}:
				message = v
			case string:
				message = map[string]interface{}{
					"type": "user",
					"message": map[string]interface{}{
						"role":    "user",
						"content": v,
					},
					"parent_tool_use_id": nil,
					"session_id":         "default",
				}
			default:
				continue
			}
			
			if err := c.SendRawMessage(message); err != nil {
				select {
				case c.errors <- err:
				case <-c.ctx.Done():
					return
				}
			}
		}
	}
}

// convertHooks converts ClaudeCodeOptions hooks to internal format
func (c *ClaudeSDKClient) convertHooks() map[types.HookEvent][]types.HookMatcher {
	if c.options.Hooks == nil {
		return nil
	}
	return c.options.Hooks
}

// GetServerInfo returns server initialization info
func (c *ClaudeSDKClient) GetServerInfo() (map[string]interface{}, error) {
	// This would be implemented based on the first system message received
	// For now, return a placeholder
	return map[string]interface{}{
		"commands": []string{},
		"output_styles": []string{
			"text",
			"json",
			"stream-json",
		},
	}, nil
}

// Helper function to get string pointer
func stringPtr(s string) *string {
	return &s
}