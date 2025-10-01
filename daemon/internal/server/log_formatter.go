package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ClaudeMessage represents the various message types from Claude Code
type ClaudeMessage struct {
	Type                string      `json:"type"`
	Message             interface{} `json:"message,omitempty"`
	Event               interface{} `json:"event,omitempty"`
	ParentToolUseID     *string     `json:"parent_tool_use_id"`
	SessionID           string      `json:"session_id"`
	UUID                string      `json:"uuid"`
}

// AssistantMessage represents an assistant message
type AssistantMessage struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Model   string `json:"model"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents different types of content in messages
type ContentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// StreamEvent represents stream events
type StreamEvent struct {
	Type  string `json:"type"`
	Index *int   `json:"index,omitempty"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"delta,omitempty"`
}

// UserMessage represents user input
type UserMessage struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SystemMessage represents system initialization
type SystemMessage struct {
	Type                string                 `json:"type"`
	CurrentWorkingDir   string                 `json:"current_working_dir"`
	Tools               []string               `json:"tools"`
	PermissionMode      string                 `json:"permission_mode"`
	APIKeySource        string                 `json:"api_key_source"`
	Model               string                 `json:"model"`
	AdditionalMetadata  map[string]interface{} `json:"additional_metadata"`
}

// ResultMessage represents execution results
type ResultMessage struct {
	Type             string  `json:"type"`
	DurationMs       int64   `json:"duration_ms"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	TotalCost        float64 `json:"total_cost"`
	PermissionDenial *string `json:"permission_denial,omitempty"`
}

// FormatClaudeOutput formats a Claude JSON output line into human-readable format for daemon logs
func FormatClaudeOutput(line string, taskPrompt string) string {
	// Skip empty lines
	if strings.TrimSpace(line) == "" {
		return ""
	}

	// Try to parse as Claude message
	var claudeMsg ClaudeMessage
	if err := json.Unmarshal([]byte(line), &claudeMsg); err != nil {
		// If it's not JSON, return as plain text
		return fmt.Sprintf("💬 %s", line)
	}

	switch claudeMsg.Type {
	case "user":
		return formatUserMessage(claudeMsg, taskPrompt)
	case "assistant":
		return formatAssistantMessage(claudeMsg)
	case "stream_event":
		return formatStreamEvent(claudeMsg)
	case "system":
		return formatSystemMessage(claudeMsg)
	case "result":
		return formatResultMessage(claudeMsg)
	default:
		// Fallback for unknown types
		return fmt.Sprintf("❓ Unknown message type: %s", claudeMsg.Type)
	}
}

// formatUserMessage formats user input messages
func formatUserMessage(msg ClaudeMessage, taskPrompt string) string {
	var output strings.Builder

	output.WriteString("👤 **Task Started**")
	if taskPrompt != "" {
		output.WriteString(fmt.Sprintf(" - **Prompt**: %s", taskPrompt))
	}

	// Try to parse the message content
	if msg.Message != nil {
		msgBytes, _ := json.Marshal(msg.Message)
		var userMsg UserMessage
		if err := json.Unmarshal(msgBytes, &userMsg); err == nil && userMsg.Content != "" {
			output.WriteString(fmt.Sprintf(" - **Request**: %s", userMsg.Content))
		}
	}

	return output.String()
}

// formatAssistantMessage formats assistant messages
func formatAssistantMessage(msg ClaudeMessage) string {
	var output strings.Builder

	// Try to parse the message content
	if msg.Message != nil {
		msgBytes, _ := json.Marshal(msg.Message)
		var assistantMsg AssistantMessage
		if err := json.Unmarshal(msgBytes, &assistantMsg); err != nil {
			return "❓ Could not parse assistant message"
		}

		for _, content := range assistantMsg.Content {
			switch content.Type {
			case "text":
				if content.Text != "" {
					output.WriteString(fmt.Sprintf("🤖 %s", content.Text))
				}
			case "tool_use":
				output.WriteString(fmt.Sprintf("🛠️ **Using %s**", content.Name))
				if inputStr := formatToolInput(content.Input); inputStr != "" {
					output.WriteString(fmt.Sprintf(" - %s", inputStr))
				}
			}
		}
	}

	return output.String()
}

// formatStreamEvent formats streaming events
func formatStreamEvent(msg ClaudeMessage) string {
	if msg.Event != nil {
		eventBytes, _ := json.Marshal(msg.Event)
		var streamEvent StreamEvent
		if err := json.Unmarshal(eventBytes, &streamEvent); err == nil {
			switch streamEvent.Type {
			case "content_block_start":
				return "⏳ Claude is thinking..."
			case "content_block_delta":
				if streamEvent.Delta != nil && streamEvent.Delta.Text != "" {
					return streamEvent.Delta.Text
				}
			case "content_block_stop":
				return "✅ Response complete"
			case "message_start":
				return "🚀 **Claude Started Working**"
			case "message_stop":
				return "🏁 **Claude Finished**"
			}
		}
	}
	return ""
}

// formatSystemMessage formats system messages
func formatSystemMessage(msg ClaudeMessage) string {
	if msg.Message != nil {
		msgBytes, _ := json.Marshal(msg.Message)
		var sysMsg SystemMessage
		if err := json.Unmarshal(msgBytes, &sysMsg); err == nil {
			var output strings.Builder
			output.WriteString("⚙️ **System Initialized**")
			if sysMsg.CurrentWorkingDir != "" {
				output.WriteString(fmt.Sprintf(" - Working Directory: %s", sysMsg.CurrentWorkingDir))
			}
			if sysMsg.Model != "" {
				output.WriteString(fmt.Sprintf(" - Model: %s", sysMsg.Model))
			}
			if len(sysMsg.Tools) > 0 {
				output.WriteString(fmt.Sprintf(" - Tools: %s", strings.Join(sysMsg.Tools, ", ")))
			}
			return output.String()
		}
	}
	return "⚙️ System message received"
}

// formatResultMessage formats result messages
func formatResultMessage(msg ClaudeMessage) string {
	if msg.Message != nil {
		msgBytes, _ := json.Marshal(msg.Message)
		var resultMsg ResultMessage
		if err := json.Unmarshal(msgBytes, &resultMsg); err == nil {
			var output strings.Builder

			duration := time.Duration(resultMsg.DurationMs) * time.Millisecond
			output.WriteString("📊 **Task Summary**")
			output.WriteString(fmt.Sprintf(" - Duration: %s", duration.String()))

			if resultMsg.InputTokens > 0 || resultMsg.OutputTokens > 0 {
				output.WriteString(fmt.Sprintf(" - Tokens: %d input, %d output",
					resultMsg.InputTokens, resultMsg.OutputTokens))
			}

			if resultMsg.TotalCost > 0 {
				output.WriteString(fmt.Sprintf(" - Cost: $%.4f", resultMsg.TotalCost))
			}

			if resultMsg.PermissionDenial != nil {
				output.WriteString(fmt.Sprintf(" - Permission Denied: %s", *resultMsg.PermissionDenial))
			}

			return output.String()
		}
	}
	return "📊 Task completed"
}

// formatToolInput formats tool input parameters in a readable way
func formatToolInput(input map[string]interface{}) string {
	if len(input) == 0 {
		return ""
	}

	var parts []string
	for key, value := range input {
		switch v := value.(type) {
		case string:
			if len(v) > 100 {
				parts = append(parts, fmt.Sprintf("%s: %s...", key, v[:100]))
			} else {
				parts = append(parts, fmt.Sprintf("%s: %s", key, v))
			}
		case []interface{}:
			parts = append(parts, fmt.Sprintf("%s: [%d items]", key, len(v)))
		case map[string]interface{}:
			parts = append(parts, fmt.Sprintf("%s: {...}", key))
		default:
			parts = append(parts, fmt.Sprintf("%s: %v", key, value))
		}
	}

	return strings.Join(parts, ", ")
}