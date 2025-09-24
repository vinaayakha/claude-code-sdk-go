package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/errors"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/transport"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

// Query handles the control protocol and message processing
type Query struct {
	transport       transport.Transport
	isStreamingMode bool
	canUseTool      types.CanUseTool
	hooks           map[types.HookEvent][]types.HookMatcher
	sdkMCPServers   map[string]interface{} // SDK MCP server instances

	reader *bufio.Reader
	ctx    context.Context
	cancel context.CancelFunc

	// Channel for messages
	messages chan map[string]interface{}
	errors   chan error

	// Control state
	initialized   bool
	hookCallbacks map[string]types.HookCallback
	mu            sync.RWMutex
	wg            sync.WaitGroup
}

// NewQuery creates a new Query handler
func NewQuery(
	transport transport.Transport,
	isStreamingMode bool,
	canUseTool types.CanUseTool,
	hooks map[types.HookEvent][]types.HookMatcher,
	sdkMCPServers map[string]interface{},
) *Query {
	ctx, cancel := context.WithCancel(context.Background())

	return &Query{
		transport:       transport,
		isStreamingMode: isStreamingMode,
		canUseTool:      canUseTool,
		hooks:           hooks,
		sdkMCPServers:   sdkMCPServers,
		ctx:             ctx,
		cancel:          cancel,
		messages:        make(chan map[string]interface{}, 100),
		errors:          make(chan error, 10),
		hookCallbacks:   make(map[string]types.HookCallback),
	}
}

// Start begins reading messages from the transport
func (q *Query) Start() error {
	if q.reader == nil {
		q.reader = bufio.NewReader(q.transport.Reader())
	}

	q.wg.Add(1)
	go q.readLoop()

	return nil
}

// Stop stops the query handler
func (q *Query) Stop() {
	q.cancel()
	q.wg.Wait()
	close(q.messages)
	close(q.errors)
}

// Initialize sends the initialization message
func (q *Query) Initialize() error {
	if q.initialized {
		return nil
	}

	// Build hooks map for initialization
	hooksMap := make(map[string]interface{})
	if q.hooks != nil {
		for event, matchers := range q.hooks {
			var matchersList []map[string]interface{}
			for _, matcher := range matchers {
				// Register callbacks
				for _, callback := range matcher.Hooks {
					callbackID := fmt.Sprintf("hook_%s_%d", event, len(q.hookCallbacks))
					q.mu.Lock()
					q.hookCallbacks[callbackID] = callback
					q.mu.Unlock()

					matcherMap := map[string]interface{}{
						"matcher":     matcher.Matcher,
						"callback_id": callbackID,
					}
					matchersList = append(matchersList, matcherMap)
				}
			}
			hooksMap[string(event)] = matchersList
		}
	}

	// Wait for initialization to complete
	// In streaming mode, we don't send an explicit init message
	q.initialized = true
	return nil
}

// ReceiveMessages returns a channel of received messages
func (q *Query) ReceiveMessages() <-chan map[string]interface{} {
	return q.messages
}

// Errors returns the error channel
func (q *Query) Errors() <-chan error {
	return q.errors
}

// Interrupt sends an interrupt request
func (q *Query) Interrupt() error {
	request := types.SDKControlRequest{
		Type:      "control_request",
		RequestID: generateRequestID(),
		Request: types.SDKControlInterruptRequest{
			Subtype: "interrupt",
		},
	}

	return q.sendControlRequest(request)
}

// readLoop continuously reads messages from the transport
func (q *Query) readLoop() {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		default:
			line, err := q.reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					select {
					case q.errors <- errors.NewCLIConnectionError("error reading from transport", err):
					case <-q.ctx.Done():
					}
				}
				return
			}

			if line == "" {
				continue
			}

			var data map[string]interface{}
			if err := json.Unmarshal([]byte(line), &data); err != nil {
				select {
				case q.errors <- errors.NewJSONDecodeError("failed to decode message", line, err):
				case <-q.ctx.Done():
				}
				continue
			}

			// Check if this is a control request
			if msgType, ok := data["type"].(string); ok && msgType == "control_request" {
				go q.handleControlRequest(data)
			} else {
				// Regular message
				select {
				case q.messages <- data:
				case <-q.ctx.Done():
					return
				}
			}
		}
	}
}

// handleControlRequest processes control protocol requests
func (q *Query) handleControlRequest(data map[string]interface{}) {
	requestID, _ := data["request_id"].(string)
	request, ok := data["request"].(map[string]interface{})
	if !ok {
		q.sendErrorResponse(requestID, "invalid request format")
		return
	}

	subtype, _ := request["subtype"].(string)

	switch subtype {
	case "can_use_tool":
		q.handleCanUseTool(requestID, request)
	case "hook_callback":
		q.handleHookCallback(requestID, request)
	case "mcp_message":
		q.handleMCPMessage(requestID, request)
	default:
		q.sendErrorResponse(requestID, fmt.Sprintf("unknown control request subtype: %s", subtype))
	}
}

// handleCanUseTool processes tool permission requests
func (q *Query) handleCanUseTool(requestID string, request map[string]interface{}) {
	if q.canUseTool == nil {
		q.sendSuccessResponse(requestID, map[string]interface{}{
			"behavior": "allow",
		})
		return
	}

	toolName, _ := request["tool_name"].(string)
	input, _ := request["input"].(map[string]interface{})

	// Build context
	ctx := &types.ToolPermissionContext{
		Suggestions: []types.PermissionUpdate{},
	}

	// Extract suggestions if present
	if suggestions, ok := request["permission_suggestions"].([]interface{}); ok {
		for _, s := range suggestions {
			if _, ok := s.(map[string]interface{}); ok {
				// Parse suggestion into PermissionUpdate
				// TODO: full implementation would parse all fields
				ctx.Suggestions = append(ctx.Suggestions, types.PermissionUpdate{})
			}
		}
	}

	// Call the callback
	result, err := q.canUseTool(toolName, input, ctx)
	if err != nil {
		q.sendErrorResponse(requestID, err.Error())
		return
	}

	// Convert result to response
	var response map[string]interface{}
	switch r := result.(type) {
	case *types.PermissionResultAllow:
		response = map[string]interface{}{
			"behavior": string(r.Behavior),
		}
		if r.UpdatedInput != nil {
			response["updated_input"] = r.UpdatedInput
		}
		if r.UpdatedPermissions != nil {
			response["updated_permissions"] = r.UpdatedPermissions
		}
	case *types.PermissionResultDeny:
		response = map[string]interface{}{
			"behavior": string(r.Behavior),
			"message":  r.Message,
		}
		if r.Interrupt {
			response["interrupt"] = true
		}
	default:
		response = map[string]interface{}{
			"behavior": "allow",
		}
	}

	q.sendSuccessResponse(requestID, response)
}

// handleHookCallback processes hook callbacks
func (q *Query) handleHookCallback(requestID string, request map[string]interface{}) {
	callbackID, _ := request["callback_id"].(string)
	input, _ := request["input"].(map[string]interface{})
	toolUseID, _ := request["tool_use_id"].(string)

	q.mu.RLock()
	callback, exists := q.hookCallbacks[callbackID]
	q.mu.RUnlock()

	if !exists {
		q.sendErrorResponse(requestID, fmt.Sprintf("callback not found: %s", callbackID))
		return
	}

	ctx := &types.HookContext{}
	var toolUseIDPtr *string
	if toolUseID != "" {
		toolUseIDPtr = &toolUseID
	}

	output, err := callback(input, toolUseIDPtr, ctx)
	if err != nil {
		q.sendErrorResponse(requestID, err.Error())
		return
	}

	response := make(map[string]interface{})
	if output != nil {
		if output.Decision != nil {
			response["decision"] = string(*output.Decision)
		}
		if output.SystemMessage != nil {
			response["systemMessage"] = *output.SystemMessage
		}
		if output.HookSpecificOutput != nil {
			response["hookSpecificOutput"] = output.HookSpecificOutput
		}
	}

	q.sendSuccessResponse(requestID, response)
}

// handleMCPMessage processes MCP server messages
func (q *Query) handleMCPMessage(requestID string, request map[string]interface{}) {
	serverName, _ := request["server_name"].(string)

	_, exists := q.sdkMCPServers[serverName]
	if !exists {
		q.sendErrorResponse(requestID, fmt.Sprintf("SDK MCP server not found: %s", serverName))
		return
	}

	// TODO: Implement MCP message handling
	// This would involve calling the appropriate method on the MCP server instance

	q.sendSuccessResponse(requestID, map[string]interface{}{
		"result": "not implemented",
	})
}

// sendControlRequest sends a control request
func (q *Query) sendControlRequest(request types.SDKControlRequest) error {
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	return q.transport.Write(data)
}

// sendSuccessResponse sends a success control response
func (q *Query) sendSuccessResponse(requestID string, response map[string]interface{}) {
	resp := types.SDKControlResponse{
		Type: "control_response",
		Response: types.ControlResponse{
			Subtype:   "success",
			RequestID: requestID,
			Response:  response,
		},
	}

	if data, err := json.Marshal(resp); err == nil {
		q.transport.Write(append(data, '\n'))
	}
}

// sendErrorResponse sends an error control response
func (q *Query) sendErrorResponse(requestID string, errorMsg string) {
	resp := types.SDKControlResponse{
		Type: "control_response",
		Response: types.ControlErrorResponse{
			Subtype:   "error",
			RequestID: requestID,
			Error:     errorMsg,
		},
	}

	if data, err := json.Marshal(resp); err == nil {
		q.transport.Write(append(data, '\n'))
	}
}

// generateRequestID generates a unique request ID
var requestCounter int
var requestCounterMu sync.Mutex

func generateRequestID() string {
	requestCounterMu.Lock()
	defer requestCounterMu.Unlock()
	requestCounter++
	return fmt.Sprintf("req_%d", requestCounter)
}
