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

package translator

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/savaki/twin-in-disguise/types"
)

// ToGeminiContents converts Anthropic messages to Gemini contents
func ToGeminiContents(messages []types.AnthropicMessage) ([]*genai.Content, error) {
	customContents, err := ToCustomGeminiContents(messages)
	if err != nil {
		return nil, err
	}

	// Convert custom contents to genai.Content
	// Note: This will lose thought signatures, but they're preserved in the custom version
	var contents []*genai.Content
	for _, cc := range customContents {
		content := &genai.Content{
			Role: cc.Role,
		}

		for _, part := range cc.Parts {
			if part.Text != "" {
				content.Parts = append(content.Parts, genai.Text(part.Text))
			} else if part.FunctionCall != nil {
				content.Parts = append(content.Parts, genai.FunctionCall{
					Name: part.FunctionCall.Name,
					Args: part.FunctionCall.Args,
				})
			} else if part.FunctionResponse != nil {
				content.Parts = append(content.Parts, genai.FunctionResponse{
					Name:     part.FunctionResponse.Name,
					Response: part.FunctionResponse.Response,
				})
			} else if part.InlineData != nil {
				// Add inline data (images) to the content
				// Data is base64 encoded, decode it
				data, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return nil, fmt.Errorf("failed to decode base64 image data: %w", err)
				}
				content.Parts = append(content.Parts, genai.ImageData(part.InlineData.MimeType, data))
			}
		}

		if len(content.Parts) > 0 {
			contents = append(contents, content)
		}
	}

	return contents, nil
}

// ToCustomGeminiContents converts Anthropic messages to custom Gemini contents with thought signature support
func ToCustomGeminiContents(messages []types.AnthropicMessage) ([]types.GeminiContent, error) {
	var contents []types.GeminiContent

	// Build a map of tool_use_id -> tool_name for resolving tool_result blocks
	toolMap := make(map[string]string)
	for _, msg := range messages {
		for _, block := range msg.Content {
			if block.Type == types.ContentTypeToolUse && block.ID != "" && block.Name != "" {
				toolMap[block.ID] = block.Name
			}
		}
	}

	for _, msg := range messages {
		// Map role: assistant -> model
		role := msg.Role
		if role == types.RoleAssistant {
			role = types.RoleModel
		}

		var parts []types.GeminiPart
		for _, block := range msg.Content {
			part, err := convertContentBlockToCustom(block, toolMap)
			if err != nil {
				return nil, err
			}
			if part != nil {
				parts = append(parts, *part)
			}
		}

		if len(parts) > 0 {
			contents = append(contents, types.GeminiContent{
				Role:  role,
				Parts: parts,
			})
		}
	}

	return contents, nil
}

func convertContentBlockToCustom(block types.AnthropicContentBlock, toolMap map[string]string) (*types.GeminiPart, error) {
	switch block.Type {
	case types.ContentTypeText, "":
		if block.Text != "" {
			return &types.GeminiPart{
				Text: block.Text,
			}, nil
		}

	case types.ContentTypeImage:
		if block.Source != nil && block.Source.Data != "" {
			return &types.GeminiPart{
				InlineData: &types.GeminiBlob{
					MimeType: block.Source.MediaType,
					Data:     block.Source.Data,
				},
			}, nil
		}

	case types.ContentTypeToolUse:
		// Function call from assistant
		if block.Name != "" {
			part := &types.GeminiPart{
				FunctionCall: &types.GeminiFunctionCall{
					Name: block.Name,
					Args: block.Input,
				},
			}
			// Include thought signature if present
			if block.ThoughtSignature != "" {
				part.ThoughtSignature = block.ThoughtSignature
			}
			return part, nil
		}

	case types.ContentTypeToolResult:
		// Function response from user
		if block.ToolUseID != "" {
			// Look up the tool name from the tool_use_id
			toolName, ok := toolMap[block.ToolUseID]
			if !ok {
				return nil, fmt.Errorf("tool_result references unknown tool_use_id: %s", block.ToolUseID)
			}

			// Extract the content from the tool_result block
			var content interface{}
			if block.Content != nil {
				content = block.Content
			} else if block.Text != "" {
				content = block.Text
			}

			// Gemini expects the response to be a map[string]interface{}
			response := map[string]interface{}{
				types.ResponseFieldResult: content,
			}

			return &types.GeminiPart{
				FunctionResponse: &types.GeminiFunctionResponse{
					Name:     toolName,
					Response: response,
				},
			}, nil
		}
	}

	return nil, nil
}

// ToGeminiTools converts Anthropic tools to Gemini tools
func ToGeminiTools(tools []types.AnthropicTool) ([]*genai.Tool, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	var functionDecls []*genai.FunctionDeclaration

	for _, tool := range tools {
		schema := &genai.Schema{
			Type: genai.TypeObject,
		}

		// Extract properties
		if props, ok := tool.InputSchema[types.SchemaFieldProperties].(map[string]interface{}); ok {
			schema.Properties = make(map[string]*genai.Schema)
			for propName, propVal := range props {
				if propMap, ok := propVal.(map[string]interface{}); ok {
					schema.Properties[propName] = convertJSONSchemaToGemini(propMap)
				}
			}
		}

		// Extract required fields
		if required, ok := tool.InputSchema[types.SchemaFieldRequired].([]interface{}); ok {
			schema.Required = make([]string, len(required))
			for i, r := range required {
				if s, ok := r.(string); ok {
					schema.Required[i] = s
				}
			}
		}

		functionDecls = append(functionDecls, &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  schema,
		})
	}

	return []*genai.Tool{{FunctionDeclarations: functionDecls}}, nil
}

// CleanSchemaForGemini removes fields that Gemini doesn't support from a JSON schema
func CleanSchemaForGemini(schema map[string]interface{}) map[string]interface{} {
	if schema == nil {
		return nil
	}

	cleaned := make(map[string]interface{})

	for key, value := range schema {
		// Skip fields that Gemini doesn't support
		if key == types.SchemaFieldDollarSchema || key == types.SchemaFieldAdditionalProperties {
			continue
		}

		// Recursively clean nested objects
		switch v := value.(type) {
		case map[string]interface{}:
			cleaned[key] = CleanSchemaForGemini(v)

		case []interface{}:
			// Clean array elements if they're objects
			cleanedArray := make([]interface{}, len(v))
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					cleanedArray[i] = CleanSchemaForGemini(itemMap)
				} else {
					cleanedArray[i] = item
				}
			}
			cleaned[key] = cleanedArray

		default:
			cleaned[key] = value
		}
	}

	return cleaned
}

func convertJSONSchemaToGemini(schema map[string]interface{}) *genai.Schema {
	result := &genai.Schema{}

	// Map type
	if typeStr, ok := schema[types.SchemaFieldType].(string); ok {
		switch strings.ToLower(typeStr) {
		case types.SchemaTypeString:
			result.Type = genai.TypeString
		case types.SchemaTypeNumber:
			result.Type = genai.TypeNumber
		case types.SchemaTypeInteger:
			result.Type = genai.TypeInteger
		case types.SchemaTypeBoolean:
			result.Type = genai.TypeBoolean
		case types.SchemaTypeArray:
			result.Type = genai.TypeArray
		case types.SchemaTypeObject:
			result.Type = genai.TypeObject
		}
	}

	// Map description
	if desc, ok := schema[types.SchemaFieldDescription].(string); ok {
		result.Description = desc
	}

	// Map enum
	if enum, ok := schema[types.SchemaFieldEnum].([]interface{}); ok {
		result.Enum = make([]string, len(enum))
		for i, e := range enum {
			if s, ok := e.(string); ok {
				result.Enum[i] = s
			}
		}
	}

	// Map items (for arrays)
	if items, ok := schema[types.SchemaFieldItems].(map[string]interface{}); ok {
		result.Items = convertJSONSchemaToGemini(items)
	}

	// Map nested properties (for objects)
	if props, ok := schema[types.SchemaFieldProperties].(map[string]interface{}); ok {
		result.Properties = make(map[string]*genai.Schema)
		for propName, propVal := range props {
			if propMap, ok := propVal.(map[string]interface{}); ok {
				result.Properties[propName] = convertJSONSchemaToGemini(propMap)
			}
		}
	}

	// Map required fields (for objects)
	if required, ok := schema[types.SchemaFieldRequired].([]interface{}); ok {
		result.Required = make([]string, len(required))
		for i, r := range required {
			if s, ok := r.(string); ok {
				result.Required[i] = s
			}
		}
	}

	// NOTE: We explicitly do NOT map "$schema" or "additionalProperties" as Gemini doesn't support them
	// and will return a 400 error if they are present

	return result
}

// ToAnthropicResponse converts a Gemini response to Anthropic format
func ToAnthropicResponse(resp *genai.GenerateContentResponse, model string) (*types.AnthropicResponse, error) {
	anthropicResp := &types.AnthropicResponse{
		ID:    uuid.New().String(),
		Type:  types.ResponseTypeMessage,
		Role:  types.RoleAssistant,
		Model: model,
	}

	// Extract content from first candidate
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				block := convertGeminiPart(part)
				if block != nil {
					anthropicResp.Content = append(anthropicResp.Content, *block)
				}
			}
		}

		// Map stop reason
		if candidate.FinishReason != 0 {
			anthropicResp.StopReason = types.StopReasonEndTurn
		}
	}

	// Map usage metadata
	if resp.UsageMetadata != nil {
		anthropicResp.Usage = types.AnthropicUsage{
			InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
			OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		}
	}

	return anthropicResp, nil
}

func convertGeminiPart(part genai.Part) *types.AnthropicContentBlock {
	switch p := part.(type) {
	case genai.Text:
		return &types.AnthropicContentBlock{
			Type: types.ContentTypeText,
			Text: string(p),
		}

	case genai.FunctionCall:
		return &types.AnthropicContentBlock{
			Type:  types.ContentTypeToolUse,
			ID:    uuid.New().String(),
			Name:  p.Name,
			Input: p.Args,
		}
	}

	return nil
}

// convertCustomGeminiPart converts a custom Gemini part (with thought signature support) to Anthropic format
func convertCustomGeminiPart(part types.GeminiPart) *types.AnthropicContentBlock {
	if part.Text != "" {
		return &types.AnthropicContentBlock{
			Type: types.ContentTypeText,
			Text: part.Text,
		}
	}

	if part.FunctionCall != nil {
		block := &types.AnthropicContentBlock{
			Type:  types.ContentTypeToolUse,
			ID:    uuid.New().String(),
			Name:  part.FunctionCall.Name,
			Input: part.FunctionCall.Args,
		}
		// Preserve thought signature
		if part.ThoughtSignature != "" {
			block.ThoughtSignature = part.ThoughtSignature
		}
		return block
	}

	return nil
}

// ToAnthropicResponseFromCustom converts a custom Gemini response to Anthropic format
func ToAnthropicResponseFromCustom(resp *GenerateContentResponse, model string) (*types.AnthropicResponse, error) {
	anthropicResp := &types.AnthropicResponse{
		ID:    uuid.New().String(),
		Type:  types.ResponseTypeMessage,
		Role:  types.RoleAssistant,
		Model: model,
	}

	// Extract content from first candidate
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				block := convertCustomGeminiPart(part)
				if block != nil {
					anthropicResp.Content = append(anthropicResp.Content, *block)
				}
			}
		}

		// Map stop reason
		if candidate.FinishReason != "" {
			anthropicResp.StopReason = types.StopReasonEndTurn
		}
	}

	// Map usage metadata
	if resp.UsageMetadata != nil {
		anthropicResp.Usage = types.AnthropicUsage{
			InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
			OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		}
	}

	return anthropicResp, nil
}
