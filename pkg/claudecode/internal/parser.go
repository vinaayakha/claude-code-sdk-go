package internal

import (
	"encoding/json"
	"fmt"

	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/errors"
	"github.com/vinaayakha/claude-code-sdk-go/pkg/claudecode/types"
)

// ParseMessage parses a raw message into the appropriate typed message
func ParseMessage(data map[string]interface{}) (types.Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, errors.NewMessageParseError("message missing 'type' field", data)
	}

	switch msgType {
	case types.MessageTypeUser:
		return parseUserMessage(data)
	case types.MessageTypeAssistant:
		return parseAssistantMessage(data)
	case types.MessageTypeSystem:
		return parseSystemMessage(data)
	case types.MessageTypeResult:
		return parseResultMessage(data)
	case types.MessageTypeStream:
		return parseStreamEvent(data)
	default:
		return nil, errors.NewMessageParseError(fmt.Sprintf("unknown message type: %s", msgType), data)
	}
}

func parseUserMessage(data map[string]interface{}) (*types.UserMessage, error) {
	msg := &types.UserMessage{}

	// Parse content - can be string or array of content blocks
	if content, ok := data["content"]; ok {
		switch v := content.(type) {
		case string:
			msg.Content = v
		case []interface{}:
			blocks := make([]types.ContentBlock, 0, len(v))
			for _, block := range v {
				if blockMap, ok := block.(map[string]interface{}); ok {
					parsed, err := parseContentBlock(blockMap)
					if err != nil {
						return nil, err
					}
					blocks = append(blocks, parsed)
				}
			}
			msg.Content = blocks
		default:
			return nil, errors.NewMessageParseError("invalid content type in user message", content)
		}
	}

	// Parse parent_tool_use_id
	if parentID, ok := data["parent_tool_use_id"].(string); ok {
		msg.ParentToolUseID = &parentID
	}

	return msg, nil
}

func parseAssistantMessage(data map[string]interface{}) (*types.AssistantMessage, error) {
	msg := &types.AssistantMessage{}

	// Parse model
	if model, ok := data["model"].(string); ok {
		msg.Model = model
	} else {
		return nil, errors.NewMessageParseError("assistant message missing 'model' field", data)
	}

	// Parse content blocks
	if content, ok := data["content"].([]interface{}); ok {
		blocks := make([]types.ContentBlock, 0, len(content))
		for _, block := range content {
			if blockMap, ok := block.(map[string]interface{}); ok {
				parsed, err := parseContentBlock(blockMap)
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, parsed)
			}
		}
		msg.Content = blocks
	} else {
		return nil, errors.NewMessageParseError("assistant message missing or invalid 'content' field", data)
	}

	// Parse parent_tool_use_id
	if parentID, ok := data["parent_tool_use_id"].(string); ok {
		msg.ParentToolUseID = &parentID
	}

	return msg, nil
}

func parseSystemMessage(data map[string]interface{}) (*types.SystemMessage, error) {
	msg := &types.SystemMessage{}

	// Parse subtype
	if subtype, ok := data["subtype"].(string); ok {
		msg.Subtype = subtype
	} else {
		return nil, errors.NewMessageParseError("system message missing 'subtype' field", data)
	}

	// Parse data
	if msgData, ok := data["data"].(map[string]interface{}); ok {
		msg.Data = msgData
	} else {
		msg.Data = make(map[string]interface{})
	}

	return msg, nil
}

func parseResultMessage(data map[string]interface{}) (*types.ResultMessage, error) {
	msg := &types.ResultMessage{}

	// Parse required fields
	if subtype, ok := data["subtype"].(string); ok {
		msg.Subtype = subtype
	} else {
		return nil, errors.NewMessageParseError("result message missing 'subtype' field", data)
	}

	// Parse numeric fields with type conversion
	msg.DurationMS = getIntField(data, "duration_ms", 0)
	msg.DurationAPIMS = getIntField(data, "duration_api_ms", 0)
	msg.NumTurns = getIntField(data, "num_turns", 0)

	// Parse boolean
	if isError, ok := data["is_error"].(bool); ok {
		msg.IsError = isError
	}

	// Parse session_id
	if sessionID, ok := data["session_id"].(string); ok {
		msg.SessionID = sessionID
	} else {
		return nil, errors.NewMessageParseError("result message missing 'session_id' field", data)
	}

	// Parse optional fields
	if cost, ok := data["total_cost_usd"].(float64); ok {
		msg.TotalCostUSD = &cost
	}

	if usage, ok := data["usage"].(map[string]interface{}); ok {
		msg.Usage = usage
	}

	if result, ok := data["result"].(string); ok {
		msg.Result = &result
	}

	return msg, nil
}

func parseStreamEvent(data map[string]interface{}) (*types.StreamEvent, error) {
	msg := &types.StreamEvent{}

	// Parse required fields
	if uuid, ok := data["uuid"].(string); ok {
		msg.UUID = uuid
	} else {
		return nil, errors.NewMessageParseError("stream event missing 'uuid' field", data)
	}

	if sessionID, ok := data["session_id"].(string); ok {
		msg.SessionID = sessionID
	} else {
		return nil, errors.NewMessageParseError("stream event missing 'session_id' field", data)
	}

	if event, ok := data["event"].(map[string]interface{}); ok {
		msg.Event = event
	} else {
		return nil, errors.NewMessageParseError("stream event missing 'event' field", data)
	}

	// Parse parent_tool_use_id
	if parentID, ok := data["parent_tool_use_id"].(string); ok {
		msg.ParentToolUseID = &parentID
	}

	return msg, nil
}

func parseContentBlock(data map[string]interface{}) (types.ContentBlock, error) {
	// Determine block type
	if _, ok := data["text"]; ok {
		return parseTextBlock(data)
	} else if _, ok := data["thinking"]; ok {
		return parseThinkingBlock(data)
	} else if _, ok := data["name"]; ok {
		return parseToolUseBlock(data)
	} else if _, ok := data["tool_use_id"]; ok {
		return parseToolResultBlock(data)
	}

	return nil, errors.NewMessageParseError("unknown content block type", data)
}

func parseTextBlock(data map[string]interface{}) (*types.TextBlock, error) {
	block := &types.TextBlock{}

	if text, ok := data["text"].(string); ok {
		block.Text = text
	} else {
		return nil, errors.NewMessageParseError("text block missing 'text' field", data)
	}

	return block, nil
}

func parseThinkingBlock(data map[string]interface{}) (*types.ThinkingBlock, error) {
	block := &types.ThinkingBlock{}

	if thinking, ok := data["thinking"].(string); ok {
		block.Thinking = thinking
	} else {
		return nil, errors.NewMessageParseError("thinking block missing 'thinking' field", data)
	}

	if signature, ok := data["signature"].(string); ok {
		block.Signature = signature
	} else {
		return nil, errors.NewMessageParseError("thinking block missing 'signature' field", data)
	}

	return block, nil
}

func parseToolUseBlock(data map[string]interface{}) (*types.ToolUseBlock, error) {
	block := &types.ToolUseBlock{}

	if id, ok := data["id"].(string); ok {
		block.ID = id
	} else {
		return nil, errors.NewMessageParseError("tool use block missing 'id' field", data)
	}

	if name, ok := data["name"].(string); ok {
		block.Name = name
	} else {
		return nil, errors.NewMessageParseError("tool use block missing 'name' field", data)
	}

	if input, ok := data["input"].(map[string]interface{}); ok {
		block.Input = input
	} else {
		block.Input = make(map[string]interface{})
	}

	return block, nil
}

func parseToolResultBlock(data map[string]interface{}) (*types.ToolResultBlock, error) {
	block := &types.ToolResultBlock{}

	if toolUseID, ok := data["tool_use_id"].(string); ok {
		block.ToolUseID = toolUseID
	} else {
		return nil, errors.NewMessageParseError("tool result block missing 'tool_use_id' field", data)
	}

	// Content can be string or array
	if content, ok := data["content"]; ok {
		block.Content = content
	}

	if isError, ok := data["is_error"].(bool); ok {
		block.IsError = &isError
	}

	return block, nil
}

// Helper function to get int field with type conversion
func getIntField(data map[string]interface{}, key string, defaultVal int) int {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case json.Number:
			if i, err := v.Int64(); err == nil {
				return int(i)
			}
		}
	}
	return defaultVal
}
