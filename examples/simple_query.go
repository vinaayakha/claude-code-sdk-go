package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

func main() {
	ctx := context.Background()

	// Example 1: Simple query
	fmt.Println("=== Simple Query ===")
	messages, err := claudecode.Query(ctx, "What is the capital of France?", nil)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range messages {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			fmt.Println("Assistant:")
			for _, block := range m.Content {
				if text, ok := block.(*types.TextBlock); ok {
					fmt.Println(text.Text)
				}
			}
		case *types.ResultMessage:
			fmt.Printf("\nSession: %s, Duration: %dms\n", m.SessionID, m.DurationMS)
		}
	}

	// Example 2: Query with options
	fmt.Println("\n=== Query with Options ===")
	options := &types.ClaudeCodeOptions{
		SystemPrompt: stringPtr("You are a helpful coding assistant."),
		AllowedTools: []string{"Read", "Write", "Edit"},
	}

	messages2, err := claudecode.Query(ctx, "Create a simple Python hello world script", options)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range messages2 {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			fmt.Println("Assistant response received")
			// Handle tool uses
			for _, block := range m.Content {
				switch b := block.(type) {
				case *types.ToolUseBlock:
					fmt.Printf("Tool use: %s\n", b.Name)
				case *types.TextBlock:
					fmt.Println(b.Text)
				}
			}
		case *types.UserMessage:
			// Tool results from Claude
		case *types.ResultMessage:
			fmt.Printf("\nCompleted in %dms\n", m.DurationMS)
		}
	}
}

func stringPtr(s string) *string {
	return &s
}
