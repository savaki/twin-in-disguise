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
	"context"
	"os"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/savaki/twin-in-disguise/types"
	"google.golang.org/api/option"
)

func TestToGeminiContents_SingleMessage(t *testing.T) {
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "Hello!"},
			},
		},
	}

	contents, err := ToGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	if len(contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(contents))
	}

	if contents[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", contents[0].Role)
	}

	if len(contents[0].Parts) != 1 {
		t.Errorf("expected 1 part, got %d", len(contents[0].Parts))
	}
}

func TestToGeminiContents_MultiTurn(t *testing.T) {
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "Hello!"},
			},
		},
		{
			Role: "assistant",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "Hi there!"},
			},
		},
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "How are you?"},
			},
		},
	}

	contents, err := ToGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	if len(contents) != 3 {
		t.Errorf("expected 3 contents, got %d", len(contents))
	}

	expectedRoles := []string{"user", "model", "user"}
	for i, content := range contents {
		if content.Role != expectedRoles[i] {
			t.Errorf("content[%d]: expected role '%s', got '%s'", i, expectedRoles[i], content.Role)
		}
	}
}

func TestToGeminiContents_MultipleContentBlocks(t *testing.T) {
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "First part"},
				{Type: "text", Text: "Second part"},
			},
		},
	}

	contents, err := ToGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	if len(contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(contents))
	}

	if len(contents[0].Parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(contents[0].Parts))
	}
}

func TestToGeminiContents_WithImage(t *testing.T) {
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "What's in this image?"},
				{
					Type: "image",
					Source: &types.AnthropicImageSource{
						Type:      "base64",
						MediaType: "image/png",
						Data:      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
					},
				},
			},
		},
	}

	contents, err := ToGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	if len(contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(contents))
	}

	if len(contents[0].Parts) != 2 {
		t.Errorf("expected 2 parts (text + image), got %d", len(contents[0].Parts))
	}
}

func TestToGeminiTools(t *testing.T) {
	anthropicTools := []types.AnthropicTool{
		{
			Name:        "get_weather",
			Description: "Get the current weather",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city name",
					},
				},
				"required": []interface{}{"location"},
			},
		},
	}

	tools, err := ToGeminiTools(anthropicTools)
	if err != nil {
		t.Fatalf("ToGeminiTools failed: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	if len(tools[0].FunctionDeclarations) != 1 {
		t.Errorf("expected 1 function declaration, got %d", len(tools[0].FunctionDeclarations))
	}

	fn := tools[0].FunctionDeclarations[0]
	if fn.Name != "get_weather" {
		t.Errorf("expected function name 'get_weather', got '%s'", fn.Name)
	}
}

func TestToAnthropicResponse_Live(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping live test: GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.0-flash")
	resp, err := model.GenerateContent(ctx, genai.Text("Say 'test' and nothing else"))
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	anthropicResp, err := ToAnthropicResponse(resp, "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("ToAnthropicResponse failed: %v", err)
	}

	if anthropicResp.Type != "message" {
		t.Errorf("expected type 'message', got '%s'", anthropicResp.Type)
	}

	if anthropicResp.Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", anthropicResp.Role)
	}

	if len(anthropicResp.Content) == 0 {
		t.Error("expected content, got empty array")
	}

	if anthropicResp.Content[0].Type != "text" {
		t.Errorf("expected content type 'text', got '%s'", anthropicResp.Content[0].Type)
	}

	if anthropicResp.Usage.InputTokens == 0 {
		t.Error("expected input tokens > 0")
	}

	if anthropicResp.Usage.OutputTokens == 0 {
		t.Error("expected output tokens > 0")
	}

	t.Logf("Response text: %s", anthropicResp.Content[0].Text)
	t.Logf("Usage: input=%d, output=%d", anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens)
}

func TestRoundTrip_Live(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping live test: GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	// Start with Anthropic request
	anthropicMessages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "Say 'roundtrip test' and nothing else"},
			},
		},
	}

	// Convert to Gemini
	geminiContents, err := ToGeminiContents(anthropicMessages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	// Call Gemini
	model := client.GenerativeModel("gemini-2.0-flash")
	geminiResp, err := model.GenerateContent(ctx, geminiContents[0].Parts...)
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	// Convert back to Anthropic
	anthropicResp, err := ToAnthropicResponse(geminiResp, "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("ToAnthropicResponse failed: %v", err)
	}

	// Verify
	if anthropicResp.Type != "message" {
		t.Errorf("expected type 'message', got '%s'", anthropicResp.Type)
	}

	if len(anthropicResp.Content) == 0 {
		t.Fatal("expected content, got empty array")
	}

	t.Logf("Roundtrip successful! Response: %s", anthropicResp.Content[0].Text)
}

func TestToGeminiContents_WithToolUseAndResult(t *testing.T) {
	// Simulate a multi-turn conversation with tool use
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "What's the weather in San Francisco?"},
			},
		},
		{
			Role: "assistant",
			Content: []types.AnthropicContentBlock{
				{
					Type: "tool_use",
					ID:   "toolu_123",
					Name: "get_weather",
					Input: map[string]interface{}{
						"location": "San Francisco, CA",
					},
				},
			},
		},
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: "toolu_123",
					Content:   "72 degrees and sunny",
				},
			},
		},
	}

	contents, err := ToGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	if len(contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(contents))
	}

	// Verify the tool_use was converted correctly
	if len(contents[1].Parts) != 1 {
		t.Fatalf("expected 1 part in assistant message, got %d", len(contents[1].Parts))
	}

	functionCall, ok := contents[1].Parts[0].(genai.FunctionCall)
	if !ok {
		t.Fatalf("expected FunctionCall, got %T", contents[1].Parts[0])
	}

	if functionCall.Name != "get_weather" {
		t.Errorf("expected function name 'get_weather', got '%s'", functionCall.Name)
	}

	// Verify the tool_result was converted correctly
	if len(contents[2].Parts) != 1 {
		t.Fatalf("expected 1 part in tool result message, got %d", len(contents[2].Parts))
	}

	functionResponse, ok := contents[2].Parts[0].(genai.FunctionResponse)
	if !ok {
		t.Fatalf("expected FunctionResponse, got %T", contents[2].Parts[0])
	}

	if functionResponse.Name != "get_weather" {
		t.Errorf("expected function response name 'get_weather', got '%s'", functionResponse.Name)
	}

	// Verify the response content is wrapped in a result key
	if result, ok := functionResponse.Response["result"].(string); !ok || result != "72 degrees and sunny" {
		t.Errorf("expected result '72 degrees and sunny', got '%v'", functionResponse.Response["result"])
	}
}

func TestToGeminiContents_ToolResultWithComplexContent(t *testing.T) {
	// Test tool_result with array content
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: "Read the file"},
			},
		},
		{
			Role: "assistant",
			Content: []types.AnthropicContentBlock{
				{
					Type: "tool_use",
					ID:   "toolu_456",
					Name: "read_file",
					Input: map[string]interface{}{
						"path": "test.txt",
					},
				},
			},
		},
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: "toolu_456",
					Content: []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "File contents here",
						},
					},
				},
			},
		},
	}

	contents, err := ToGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	if len(contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(contents))
	}

	// Verify the tool_result was converted correctly
	functionResponse, ok := contents[2].Parts[0].(genai.FunctionResponse)
	if !ok {
		t.Fatalf("expected FunctionResponse, got %T", contents[2].Parts[0])
	}

	if functionResponse.Name != "read_file" {
		t.Errorf("expected function response name 'read_file', got '%s'", functionResponse.Name)
	}

	// Content should be the array wrapped in result key
	contentArray, ok := functionResponse.Response["result"].([]interface{})
	if !ok {
		t.Errorf("expected array content in result, got %T", functionResponse.Response["result"])
	}

	if len(contentArray) != 1 {
		t.Errorf("expected 1 content block, got %d", len(contentArray))
	}
}

func TestToGeminiContents_ToolResultMissingToolUseID(t *testing.T) {
	// Test that we get an error when tool_result references unknown tool_use_id
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: "unknown_id",
					Content:   "some result",
				},
			},
		},
	}

	_, err := ToGeminiContents(messages)
	if err == nil {
		t.Fatal("expected error for unknown tool_use_id, got nil")
	}

	expectedError := "tool_result references unknown tool_use_id: unknown_id"
	if err.Error() != expectedError {
		t.Errorf("expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestConvertJSONSchemaToGemini(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected *genai.Schema
	}{
		{
			name: "string type",
			schema: map[string]interface{}{
				"type":        "string",
				"description": "A string field",
			},
			expected: &genai.Schema{
				Type:        genai.TypeString,
				Description: "A string field",
			},
		},
		{
			name: "number type",
			schema: map[string]interface{}{
				"type": "number",
			},
			expected: &genai.Schema{
				Type: genai.TypeNumber,
			},
		},
		{
			name: "integer type",
			schema: map[string]interface{}{
				"type": "integer",
			},
			expected: &genai.Schema{
				Type: genai.TypeInteger,
			},
		},
		{
			name: "boolean type",
			schema: map[string]interface{}{
				"type": "boolean",
			},
			expected: &genai.Schema{
				Type: genai.TypeBoolean,
			},
		},
		{
			name: "array type with items",
			schema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			expected: &genai.Schema{
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
			},
		},
		{
			name: "object type with properties",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
					"age": map[string]interface{}{
						"type": "integer",
					},
				},
				"required": []interface{}{"name"},
			},
			expected: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"name": {Type: genai.TypeString},
					"age":  {Type: genai.TypeInteger},
				},
				Required: []string{"name"},
			},
		},
		{
			name: "enum type",
			schema: map[string]interface{}{
				"type": "string",
				"enum": []interface{}{"red", "green", "blue"},
			},
			expected: &genai.Schema{
				Type: genai.TypeString,
				Enum: []string{"red", "green", "blue"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertJSONSchemaToGemini(tt.schema)
			if result.Type != tt.expected.Type {
				t.Errorf("type: expected %v, got %v", tt.expected.Type, result.Type)
			}
			if result.Description != tt.expected.Description {
				t.Errorf("description: expected %v, got %v", tt.expected.Description, result.Description)
			}
		})
	}
}

func TestToAnthropicResponse(t *testing.T) {
	tests := []struct {
		name     string
		resp     *genai.GenerateContentResponse
		model    string
		validate func(*testing.T, *types.AnthropicResponse)
	}{
		{
			name: "text response",
			resp: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []genai.Part{
								genai.Text("Hello, world!"),
							},
						},
						FinishReason: genai.FinishReasonStop,
					},
				},
				UsageMetadata: &genai.UsageMetadata{
					PromptTokenCount:     10,
					CandidatesTokenCount: 5,
				},
			},
			model: "gemini-2.0-flash",
			validate: func(t *testing.T, resp *types.AnthropicResponse) {
				if resp.Type != "message" {
					t.Errorf("expected type 'message', got '%s'", resp.Type)
				}
				if resp.Role != "assistant" {
					t.Errorf("expected role 'assistant', got '%s'", resp.Role)
				}
				if resp.Model != "gemini-2.0-flash" {
					t.Errorf("expected model 'gemini-2.0-flash', got '%s'", resp.Model)
				}
				if len(resp.Content) != 1 {
					t.Fatalf("expected 1 content block, got %d", len(resp.Content))
				}
				if resp.Content[0].Type != "text" {
					t.Errorf("expected content type 'text', got '%s'", resp.Content[0].Type)
				}
				if resp.Content[0].Text != "Hello, world!" {
					t.Errorf("expected text 'Hello, world!', got '%s'", resp.Content[0].Text)
				}
				if resp.Usage.InputTokens != 10 {
					t.Errorf("expected input tokens 10, got %d", resp.Usage.InputTokens)
				}
				if resp.Usage.OutputTokens != 5 {
					t.Errorf("expected output tokens 5, got %d", resp.Usage.OutputTokens)
				}
				if resp.StopReason != "end_turn" {
					t.Errorf("expected stop reason 'end_turn', got '%s'", resp.StopReason)
				}
			},
		},
		{
			name: "function call response",
			resp: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []genai.Part{
								genai.FunctionCall{
									Name: "get_weather",
									Args: map[string]interface{}{
										"location": "San Francisco",
									},
								},
							},
						},
						FinishReason: genai.FinishReasonStop,
					},
				},
			},
			model: "gemini-2.0-flash",
			validate: func(t *testing.T, resp *types.AnthropicResponse) {
				if len(resp.Content) != 1 {
					t.Fatalf("expected 1 content block, got %d", len(resp.Content))
				}
				if resp.Content[0].Type != "tool_use" {
					t.Errorf("expected content type 'tool_use', got '%s'", resp.Content[0].Type)
				}
				if resp.Content[0].Name != "get_weather" {
					t.Errorf("expected name 'get_weather', got '%s'", resp.Content[0].Name)
				}
			},
		},
		{
			name: "empty response",
			resp: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{},
			},
			model: "gemini-2.0-flash",
			validate: func(t *testing.T, resp *types.AnthropicResponse) {
				if len(resp.Content) != 0 {
					t.Errorf("expected 0 content blocks, got %d", len(resp.Content))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ToAnthropicResponse(tt.resp, tt.model)
			if err != nil {
				t.Fatalf("ToAnthropicResponse failed: %v", err)
			}
			tt.validate(t, resp)
		})
	}
}

func TestToAnthropicResponseFromCustom(t *testing.T) {
	tests := []struct {
		name     string
		resp     *GenerateContentResponse
		model    string
		validate func(*testing.T, *types.AnthropicResponse)
	}{
		{
			name: "text response",
			resp: &GenerateContentResponse{
				Candidates: []Candidate{
					{
						Content: &types.GeminiContent{
							Parts: []types.GeminiPart{
								{Text: "Hello from custom!"},
							},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: &UsageMetadata{
					PromptTokenCount:     20,
					CandidatesTokenCount: 10,
				},
			},
			model: "gemini-2.0-flash",
			validate: func(t *testing.T, resp *types.AnthropicResponse) {
				if resp.Type != "message" {
					t.Errorf("expected type 'message', got '%s'", resp.Type)
				}
				if len(resp.Content) != 1 {
					t.Fatalf("expected 1 content block, got %d", len(resp.Content))
				}
				if resp.Content[0].Text != "Hello from custom!" {
					t.Errorf("expected text 'Hello from custom!', got '%s'", resp.Content[0].Text)
				}
				if resp.Usage.InputTokens != 20 {
					t.Errorf("expected input tokens 20, got %d", resp.Usage.InputTokens)
				}
			},
		},
		{
			name: "function call with thought signature",
			resp: &GenerateContentResponse{
				Candidates: []Candidate{
					{
						Content: &types.GeminiContent{
							Parts: []types.GeminiPart{
								{
									FunctionCall: &types.GeminiFunctionCall{
										Name: "search",
										Args: map[string]interface{}{"query": "test"},
									},
									ThoughtSignature: "thinking...",
								},
							},
						},
						FinishReason: "STOP",
					},
				},
			},
			model: "gemini-2.0-flash",
			validate: func(t *testing.T, resp *types.AnthropicResponse) {
				if len(resp.Content) != 1 {
					t.Fatalf("expected 1 content block, got %d", len(resp.Content))
				}
				if resp.Content[0].Type != "tool_use" {
					t.Errorf("expected content type 'tool_use', got '%s'", resp.Content[0].Type)
				}
				if resp.Content[0].ThoughtSignature != "thinking..." {
					t.Errorf("expected thought signature 'thinking...', got '%s'", resp.Content[0].ThoughtSignature)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ToAnthropicResponseFromCustom(tt.resp, tt.model)
			if err != nil {
				t.Fatalf("ToAnthropicResponseFromCustom failed: %v", err)
			}
			tt.validate(t, resp)
		})
	}
}

func TestToGeminiContents_InvalidBase64Image(t *testing.T) {
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{
					Type: "image",
					Source: &types.AnthropicImageSource{
						Type:      "base64",
						MediaType: "image/png",
						Data:      "!!!invalid base64!!!",
					},
				},
			},
		},
	}

	_, err := ToGeminiContents(messages)
	if err == nil {
		t.Fatal("expected error for invalid base64 data, got nil")
	}
}

func TestToCustomGeminiContents_WithThoughtSignature(t *testing.T) {
	messages := []types.AnthropicMessage{
		{
			Role: "assistant",
			Content: []types.AnthropicContentBlock{
				{
					Type:             "tool_use",
					ID:               "tool_123",
					Name:             "search",
					Input:            map[string]interface{}{"query": "test"},
					ThoughtSignature: "I need to search for this",
				},
			},
		},
	}

	contents, err := ToCustomGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToCustomGeminiContents failed: %v", err)
	}

	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	if len(contents[0].Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(contents[0].Parts))
	}

	if contents[0].Parts[0].ThoughtSignature != "I need to search for this" {
		t.Errorf("expected thought signature to be preserved, got '%s'", contents[0].Parts[0].ThoughtSignature)
	}
}

func TestToGeminiTools_Empty(t *testing.T) {
	tools, err := ToGeminiTools(nil)
	if err != nil {
		t.Fatalf("ToGeminiTools failed: %v", err)
	}

	if tools != nil {
		t.Errorf("expected nil tools for empty input, got %v", tools)
	}

	tools, err = ToGeminiTools([]types.AnthropicTool{})
	if err != nil {
		t.Fatalf("ToGeminiTools failed: %v", err)
	}

	if tools != nil {
		t.Errorf("expected nil tools for empty array, got %v", tools)
	}
}

func TestToGeminiContents_EmptyTextBlock(t *testing.T) {
	messages := []types.AnthropicMessage{
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{Type: "text", Text: ""}, // Empty text
				{Type: "text", Text: "Actual text"},
			},
		},
	}

	contents, err := ToGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	// Should only have the non-empty text part
	if len(contents[0].Parts) != 1 {
		t.Errorf("expected 1 part (empty text should be skipped), got %d", len(contents[0].Parts))
	}
}

func TestToGeminiContents_ToolResultWithTextFallback(t *testing.T) {
	messages := []types.AnthropicMessage{
		{
			Role: "assistant",
			Content: []types.AnthropicContentBlock{
				{
					Type:  "tool_use",
					ID:    "tool_456",
					Name:  "read_file",
					Input: map[string]interface{}{"path": "test.txt"},
				},
			},
		},
		{
			Role: "user",
			Content: []types.AnthropicContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: "tool_456",
					Text:      "fallback text", // Using old Text field instead of Content
				},
			},
		},
	}

	contents, err := ToGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToGeminiContents failed: %v", err)
	}

	if len(contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(contents))
	}

	// Verify tool result was converted
	if len(contents[1].Parts) != 1 {
		t.Fatalf("expected 1 part in tool result, got %d", len(contents[1].Parts))
	}
}

func TestConvertGeminiPart_UnsupportedType(t *testing.T) {
	// Test with an unsupported Part type (e.g., Blob)
	blob := genai.Blob{
		MIMEType: "image/png",
		Data:     []byte("test data"),
	}

	result := convertGeminiPart(blob)
	if result != nil {
		t.Errorf("expected nil for unsupported Blob type, got %+v", result)
	}
}

func TestConvertCustomGeminiPart_NoPart(t *testing.T) {
	// Test with empty GeminiPart
	part := types.GeminiPart{}

	result := convertCustomGeminiPart(part)
	if result != nil {
		t.Errorf("expected nil for empty part, got %+v", result)
	}
}

func TestToAnthropicResponse_NoUsageMetadata(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []genai.Part{
						genai.Text("Test"),
					},
				},
			},
		},
		// No UsageMetadata
	}

	anthropicResp, err := ToAnthropicResponse(resp, "test-model")
	if err != nil {
		t.Fatalf("ToAnthropicResponse failed: %v", err)
	}

	// Should have zero usage
	if anthropicResp.Usage.InputTokens != 0 {
		t.Errorf("expected 0 input tokens, got %d", anthropicResp.Usage.InputTokens)
	}
	if anthropicResp.Usage.OutputTokens != 0 {
		t.Errorf("expected 0 output tokens, got %d", anthropicResp.Usage.OutputTokens)
	}
}

func TestToAnthropicResponse_NoFinishReason(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []genai.Part{
						genai.Text("Test"),
					},
				},
				// FinishReason is 0 (unset)
			},
		},
	}

	anthropicResp, err := ToAnthropicResponse(resp, "test-model")
	if err != nil {
		t.Fatalf("ToAnthropicResponse failed: %v", err)
	}

	// StopReason should be empty
	if anthropicResp.StopReason != "" {
		t.Errorf("expected empty stop reason, got '%s'", anthropicResp.StopReason)
	}
}

func TestToAnthropicResponseFromCustom_NoFinishReason(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{
			{
				Content: &types.GeminiContent{
					Parts: []types.GeminiPart{
						{Text: "Test"},
					},
				},
				// FinishReason is empty
			},
		},
	}

	anthropicResp, err := ToAnthropicResponseFromCustom(resp, "test-model")
	if err != nil {
		t.Fatalf("ToAnthropicResponseFromCustom failed: %v", err)
	}

	// StopReason should be empty
	if anthropicResp.StopReason != "" {
		t.Errorf("expected empty stop reason, got '%s'", anthropicResp.StopReason)
	}
}

func TestToCustomGeminiContents_EmptyMessages(t *testing.T) {
	var messages []types.AnthropicMessage

	contents, err := ToCustomGeminiContents(messages)
	if err != nil {
		t.Fatalf("ToCustomGeminiContents failed: %v", err)
	}

	if len(contents) != 0 {
		t.Errorf("expected 0 contents, got %d", len(contents))
	}
}

func TestCleanSchemaForGemini_NilSchema(t *testing.T) {
	result := CleanSchemaForGemini(nil)
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}
