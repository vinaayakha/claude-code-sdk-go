package types_test

import (
	"encoding/json"
	"testing"

	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

func TestMessageTypes(t *testing.T) {
	// Test UserMessage
	userMsg := &types.UserMessage{
		Content: "Hello, Claude!",
	}

	if userMsg.GetType() != types.MessageTypeUser {
		t.Errorf("Expected message type %s, got %s", types.MessageTypeUser, userMsg.GetType())
	}

	// Test AssistantMessage
	assistantMsg := &types.AssistantMessage{
		Model: "claude-3",
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "Hello!"},
		},
	}

	if assistantMsg.GetType() != types.MessageTypeAssistant {
		t.Errorf("Expected message type %s, got %s", types.MessageTypeAssistant, assistantMsg.GetType())
	}
}

func TestClaudeCodeOptionsJSON(t *testing.T) {
	// Test marshaling with MCP servers
	options := &types.ClaudeCodeOptions{
		SystemPrompt: stringPtr("Test prompt"),
		AllowedTools: []string{"Read", "Write"},
		MCPServers: map[string]types.MCPServerConfig{
			"test": types.MCPStdioServerConfig{
				Type:    "stdio",
				Command: "test-server",
				Args:    []string{"--arg1", "value1"},
			},
		},
	}

	data, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("Failed to marshal options: %v", err)
	}

	// Test unmarshaling
	var decoded types.ClaudeCodeOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal options: %v", err)
	}

	if *decoded.SystemPrompt != "Test prompt" {
		t.Errorf("Expected system prompt 'Test prompt', got %s", *decoded.SystemPrompt)
	}

	if len(decoded.AllowedTools) != 2 {
		t.Errorf("Expected 2 allowed tools, got %d", len(decoded.AllowedTools))
	}
}

func TestContentBlockTypes(t *testing.T) {
	blocks := []types.ContentBlock{
		&types.TextBlock{Text: "Hello"},
		&types.ThinkingBlock{Thinking: "Hmm...", Signature: "sig"},
		&types.ToolUseBlock{ID: "123", Name: "Read", Input: map[string]interface{}{"file": "test.txt"}},
		&types.ToolResultBlock{ToolUseID: "123", Content: "File contents"},
	}

	// Verify we can store different block types
	if len(blocks) != 4 {
		t.Errorf("Expected 4 blocks, got %d", len(blocks))
	}
}

func stringPtr(s string) *string {
	return &s
}
