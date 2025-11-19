// Copyright 2025 Matt Ho
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import "encoding/json"

// AnthropicRequest represents an Anthropic API request
type AnthropicRequest struct {
	Messages  []AnthropicMessage `json:"messages"`
	System    interface{}        `json:"system,omitempty"` // Can be string or array of content blocks
	MaxTokens int                `json:"max_tokens,omitempty"`
	Tools     []AnthropicTool    `json:"tools,omitempty"`
	Model     string             `json:"model,omitempty"`
}

// AnthropicMessage represents a message in the conversation
type AnthropicMessage struct {
	Role    string                  `json:"role"`
	Content []AnthropicContentBlock `json:"content"`
}

// UnmarshalJSON implements custom JSON unmarshaling for AnthropicMessage
// to handle content as either a string or an array of content blocks
func (m *AnthropicMessage) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with the same fields
	type Alias AnthropicMessage
	aux := &struct {
		Content interface{} `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle content field
	switch v := aux.Content.(type) {
	case string:
		// Content is a string, wrap it in a text block
		m.Content = []AnthropicContentBlock{
			{
				Type: ContentTypeText,
				Text: v,
			},
		}
	case []interface{}:
		// Content is an array, unmarshal normally
		contentBytes, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(contentBytes, &m.Content); err != nil {
			return err
		}
	}

	return nil
}

// AnthropicContentBlock represents a block of content (text, image, tool use, etc.)
type AnthropicContentBlock struct {
	Type             string                 `json:"type,omitempty"`
	Text             string                 `json:"text,omitempty"`
	Source           *AnthropicImageSource  `json:"source,omitempty"`
	ID               string                 `json:"id,omitempty"`
	Name             string                 `json:"name,omitempty"`
	Input            map[string]interface{} `json:"input,omitempty"`
	ThoughtSignature string                 `json:"thought_signature,omitempty"` // For tool use blocks
	ToolUseID        string                 `json:"tool_use_id,omitempty"`       // For tool_result blocks
	Content          interface{}            `json:"content,omitempty"`           // For tool_result blocks - can be string or array
}

// AnthropicImageSource represents an embedded image
type AnthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// AnthropicTool represents a function/tool definition
type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// AnthropicResponse represents an Anthropic API response
type AnthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []AnthropicContentBlock `json:"content"`
	Usage      AnthropicUsage          `json:"usage"`
	Model      string                  `json:"model"`
	StopReason string                  `json:"stop_reason,omitempty"`
}

// AnthropicUsage represents token usage statistics
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
