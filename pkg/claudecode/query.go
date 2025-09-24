package claudecode

import (
	"context"
	"os"

	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/errors"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/internal"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/transport"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

// Query performs a one-shot or unidirectional streaming interaction with Claude Code.
//
// This function is ideal for simple, stateless queries where you don't need
// bidirectional communication or conversation management. For interactive,
// stateful conversations, use ClaudeSDKClient instead.
//
// Key differences from ClaudeSDKClient:
//   - Unidirectional: Send all messages upfront, receive all responses
//   - Stateless: Each query is independent, no conversation state
//   - Simple: Fire-and-forget style, no connection management
//   - No interrupts: Cannot interrupt or send follow-up messages
//
// When to use Query():
//   - Simple one-off questions ("What is 2+2?")
//   - Batch processing of independent prompts
//   - Code generation or analysis tasks
//   - Automated scripts and CI/CD pipelines
//   - When you know all inputs upfront
//
// When to use ClaudeSDKClient:
//   - Interactive conversations with follow-ups
//   - Chat applications or REPL-like interfaces
//   - When you need to send messages based on responses
//   - When you need interrupt capabilities
//   - Long-running sessions with state
//
// Example - Simple query:
//
//	messages, err := Query(ctx, "What is the capital of France?", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for msg := range messages {
//	    fmt.Println(msg)
//	}
//
// Example - With options:
//
//	options := &types.ClaudeCodeOptions{
//	    SystemPrompt: stringPtr("You are an expert Python developer"),
//	    CWD: stringPtr("/home/user/project"),
//	}
//	messages, err := Query(ctx, "Create a Python web server", options)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for msg := range messages {
//	    fmt.Println(msg)
//	}
//
// Example - Streaming mode (still unidirectional):
//
//	prompts := make(chan interface{})
//	go func() {
//	    prompts <- "Hello"
//	    prompts <- "How are you?"
//	    close(prompts)
//	}()
//	messages, err := Query(ctx, prompts, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for msg := range messages {
//	    fmt.Println(msg)
//	}
func Query(ctx context.Context, prompt interface{}, options *types.ClaudeCodeOptions) (<-chan types.Message, error) {
	if options == nil {
		options = &types.ClaudeCodeOptions{}
	}
	
	// Set environment variable
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go")
	
	// Create channels
	messages := make(chan types.Message, 100)
	
	// Start query in goroutine
	go func() {
		defer close(messages)
		
		// Create transport
		t := transport.NewSubprocessTransport(prompt, options, "")
		
		// Connect
		if err := t.Connect(ctx); err != nil {
			messages <- &types.SystemMessage{
				Subtype: "error",
				Data: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}
		defer t.Close()
		
		// Create query handler
		isStreaming := false
		if _, ok := prompt.(chan interface{}); ok {
			isStreaming = true
		}
		
		query := internal.NewQuery(
			t,
			isStreaming,
			nil, // No canUseTool for one-shot queries
			nil, // No hooks for one-shot queries
			nil, // No SDK MCP servers for one-shot queries
		)
		
		// Start query
		if err := query.Start(); err != nil {
			messages <- &types.SystemMessage{
				Subtype: "error",
				Data: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}
		defer query.Stop()
		
		// Initialize
		if err := query.Initialize(); err != nil {
			messages <- &types.SystemMessage{
				Subtype: "error",
				Data: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}
		
		// Process messages
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-query.ReceiveMessages():
				if !ok {
					return
				}
				
				msg, err := internal.ParseMessage(data)
				if err != nil {
					messages <- &types.SystemMessage{
						Subtype: "error",
						Data: map[string]interface{}{
							"error": err.Error(),
						},
					}
					continue
				}
				
				messages <- msg
				
				// Check if we got a result message (end of conversation)
				if _, isResult := msg.(*types.ResultMessage); isResult {
					return
				}
			case err, ok := <-query.Errors():
				if !ok {
					return
				}
				
				messages <- &types.SystemMessage{
					Subtype: "error",
					Data: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}
		}
	}()
	
	return messages, nil
}

// QuerySync performs a synchronous query and collects all messages
func QuerySync(ctx context.Context, prompt string, options *types.ClaudeCodeOptions) ([]types.Message, error) {
	msgChan, err := Query(ctx, prompt, options)
	if err != nil {
		return nil, err
	}
	
	var messages []types.Message
	for msg := range msgChan {
		messages = append(messages, msg)
		
		// Check for errors
		if sysMsg, ok := msg.(*types.SystemMessage); ok && sysMsg.Subtype == "error" {
			if errStr, ok := sysMsg.Data["error"].(string); ok {
				return messages, errors.NewCLIConnectionError(errStr, nil)
			}
		}
	}
	
	return messages, nil
}