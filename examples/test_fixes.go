package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

func main() {
	fmt.Println("Testing Claude Code SDK fixes...")

	// Test 1: Simple query (non-streaming mode)
	fmt.Println("\n=== Test 1: Simple Query ===")
	testSimpleQuery()

	// Test 2: Interactive client (streaming mode)
	fmt.Println("\n=== Test 2: Interactive Client ===")
	testInteractiveClient()

	// Test 3: Concurrent operations
	fmt.Println("\n=== Test 3: Concurrent Operations ===")
	testConcurrentOperations()
}

func testSimpleQuery() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := claudecode.Query(ctx, "What is 2+2?", nil)
	if err != nil {
		log.Printf("Error in simple query: %v", err)
		return
	}

	messageCount := 0
	for msg := range messages {
		messageCount++
		switch m := msg.(type) {
		case *types.AssistantMessage:
			fmt.Println("Got assistant message")
			for _, block := range m.Content {
				if text, ok := block.(*types.TextBlock); ok {
					fmt.Printf("Response: %s\n", text.Text)
				}
			}
		case *types.ResultMessage:
			fmt.Printf("Query completed in %dms\n", m.DurationMS)
		case *types.SystemMessage:
			if m.Subtype == "error" {
				fmt.Printf("System error: %v\n", m.Data["error"])
			}
		}
	}

	if messageCount == 0 {
		fmt.Println("No messages received")
	}
}

func testInteractiveClient() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := claudecode.NewClaudeSDKClient(nil)

	// Connect with initial prompt
	err := client.Connect(ctx, "Hello! Ready to test.")
	if err != nil {
		log.Printf("Error connecting: %v", err)
		return
	}
	defer client.Close()

	// Start message handler
	done := make(chan bool)
	go func() {
		for msg := range client.Messages() {
			switch m := msg.(type) {
			case *types.AssistantMessage:
				fmt.Println("Got assistant response")
			case *types.ResultMessage:
				fmt.Printf("Turn completed in %dms\n", m.DurationMS)
				done <- true
			}
		}
	}()

	// Send a test message
	err = client.SendMessage("What is 3+3?", "test-session")
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}

	// Wait for completion or timeout
	select {
	case <-done:
		fmt.Println("Interactive test completed")
	case <-ctx.Done():
		fmt.Println("Interactive test timed out")
	}
}

func testConcurrentOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run multiple queries concurrently
	for i := 0; i < 3; i++ {
		go func(id int) {
			prompt := fmt.Sprintf("What is %d+%d?", id, id)
			messages, err := claudecode.Query(ctx, prompt, nil)
			if err != nil {
				log.Printf("Query %d error: %v", id, err)
				return
			}

			for msg := range messages {
				if _, ok := msg.(*types.ResultMessage); ok {
					fmt.Printf("Query %d completed\n", id)
				}
			}
		}(i)
	}

	// Wait a bit for queries to complete
	time.Sleep(5 * time.Second)
	fmt.Println("Concurrent test completed")
}