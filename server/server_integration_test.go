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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/savaki/twin-in-disguise/types"
	"google.golang.org/api/option"
)

// TestIntegration_GenerateContentWithSDK tests the SDK path with various scenarios
func TestIntegration_GenerateContentWithSDK(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	srv := New(client)

	tests := []struct {
		name    string
		request types.AnthropicRequest
	}{
		{
			name: "simple text - single turn",
			request: types.AnthropicRequest{
				Model: "gemini-2.0-flash",
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Say 'test' and nothing else"},
						},
					},
				},
				MaxTokens: 10,
			},
		},
		{
			name: "multi-turn conversation",
			request: types.AnthropicRequest{
				Model: "gemini-2.0-flash",
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Hi"},
						},
					},
					{
						Role: "assistant",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Hello!"},
						},
					},
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Say 'bye'"},
						},
					},
				},
				MaxTokens: 10,
			},
		},
		{
			name: "with string system prompt",
			request: types.AnthropicRequest{
				Model:  "gemini-2.0-flash",
				System: "You are brief.",
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Say hi"},
						},
					},
				},
				MaxTokens: 10,
			},
		},
		{
			name: "with array system prompt",
			request: types.AnthropicRequest{
				Model: "gemini-2.0-flash",
				System: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "You are brief.",
					},
					map[string]interface{}{
						"type": "text",
						"text": "Keep it short.",
					},
				},
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Say hi"},
						},
					},
				},
				MaxTokens: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := srv.generateContentWithSDK(ctx, tt.request.Model, &tt.request)
			if err != nil {
				t.Errorf("generateContentWithSDK failed: %v", err)
			}
		})
	}
}

// TestIntegration_GenerateContentWithHTTP tests the HTTP path
func TestIntegration_GenerateContentWithHTTP(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	srv := NewWithAPIKey(client, apiKey)

	tests := []struct {
		name    string
		request types.AnthropicRequest
	}{
		{
			name: "simple with tools",
			request: types.AnthropicRequest{
				Model: "gemini-2.0-flash",
				Tools: []types.AnthropicTool{
					{
						Name:        "get_time",
						Description: "Get current time",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"timezone": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Just say 'hello', don't use tools"},
						},
					},
				},
				MaxTokens: 10,
			},
		},
		{
			name: "with string system and tools",
			request: types.AnthropicRequest{
				Model:  "gemini-2.0-flash",
				System: "You are brief.",
				Tools: []types.AnthropicTool{
					{
						Name: "test_tool",
						InputSchema: map[string]interface{}{
							"type": "object",
						},
					},
				},
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Say hi"},
						},
					},
				},
				MaxTokens: 10,
			},
		},
		{
			name: "with array system and tools",
			request: types.AnthropicRequest{
				Model: "gemini-2.0-flash",
				System: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "You are brief.",
					},
				},
				Tools: []types.AnthropicTool{
					{
						Name: "test_tool",
						InputSchema: map[string]interface{}{
							"type": "object",
						},
					},
				},
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Say hi"},
						},
					},
				},
				MaxTokens: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := srv.generateContentWithHTTP(ctx, tt.request.Model, &tt.request)
			if err != nil {
				t.Errorf("generateContentWithHTTP failed: %v", err)
			}
		})
	}
}

// TestIntegration_HandleMessagesDebugMode tests debug logging
func TestIntegration_HandleMessagesDebugMode(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	srv := New(client)
	srv.SetDebug(true)

	request := types.AnthropicRequest{
		Model: "gemini-2.0-flash",
		Messages: []types.AnthropicMessage{
			{
				Role: "user",
				Content: []types.AnthropicContentBlock{
					{Type: "text", Text: "Hi"},
				},
			},
		},
		MaxTokens: 10,
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.HandleMessages(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
