package examples

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

func examples() {
	ctx := context.Background()

	// Create client with options
	options := &types.ClaudeCodeOptions{
		SystemPrompt: stringPtr("You are a helpful assistant."),
		AllowedTools: []string{"Read", "Write", "Edit", "Bash"},
	}

	client := claudecode.NewClaudeSDKClient(options)

	// Connect with initial prompt
	fmt.Println("Connecting to Claude...")
	err := client.Connect(ctx, "Hello! I'm ready to help you with coding tasks.")
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer client.Close()

	// Start message handler
	go handleMessages(client)

	// Interactive loop
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\nConnected! Type your messages (or 'quit' to exit):")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "quit" || input == "exit" {
			break
		}

		if input == "" {
			continue
		}

		// Send message
		err := client.SendMessage(input, "default")
		if err != nil {
			fmt.Printf("Error sending message: %v\n", err)
		}
	}

	fmt.Println("\nGoodbye!")
}

func handleMessages(client *claudecode.ClaudeSDKClient) {
	for msg := range client.Messages() {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			fmt.Println("\nAssistant:")
			for _, block := range m.Content {
				switch b := block.(type) {
				case *types.TextBlock:
					fmt.Println(b.Text)
				case *types.ToolUseBlock:
					fmt.Printf("[Tool: %s]\n", b.Name)
				case *types.ThinkingBlock:
					// Optionally show thinking
				}
			}
			fmt.Print("\n> ")

		case *types.UserMessage:
			// This might be tool results

		case *types.SystemMessage:
			if m.Subtype == "error" {
				fmt.Printf("\nSystem error: %v\n> ", m.Data["error"])
			}

		case *types.ResultMessage:
			fmt.Printf("\n[Session %s completed in %dms]\n> ", m.SessionID, m.DurationMS)
		}
	}
}

func stringPtr(s string) *string {
	return &s
}
