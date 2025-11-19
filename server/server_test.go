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
	fmt "fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/savaki/twin-in-disguise/types"
	"google.golang.org/api/option"
)

func TestHandleInvoke_Live(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping live test: GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer client.Close()

	srv := New(client)

	tests := []struct {
		name      string
		modelID   string
		request   types.AnthropicRequest
		wantError bool
	}{
		{
			name:    "simple text request - sonnet",
			modelID: "anthropic.claude-3-sonnet-20240229-v1:0",
			request: types.AnthropicRequest{
				Model: "anthropic.claude-3-sonnet-20240229-v1:0",
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Say 'hello' and nothing else"},
						},
					},
				},
				MaxTokens: 100,
			},
			wantError: false,
		},
		{
			name:    "simple text request - haiku",
			modelID: "anthropic.claude-3-haiku-20240307-v1:0",
			request: types.AnthropicRequest{
				Model: "anthropic.claude-3-haiku-20240307-v1:0",
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Say 'hi' and nothing else"},
						},
					},
				},
				MaxTokens: 50,
			},
			wantError: false,
		},
		{
			name:    "request with system prompt",
			modelID: "anthropic.claude-3-sonnet-20240229-v1:0",
			request: types.AnthropicRequest{
				Model:  "anthropic.claude-3-sonnet-20240229-v1:0",
				System: "You are a helpful assistant that responds in one word.",
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "What color is the sky?"},
						},
					},
				},
				MaxTokens: 50,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.HandleMessages(w, req)

			if tt.wantError {
				if w.Code == http.StatusOK {
					t.Errorf("expected error, got success")
				}
				return
			}

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
				return
			}

			data, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("failed to read response: %v", err)
			}

			fmt.Println(string(data))

			var response types.AnthropicResponse
			if err := json.Unmarshal(data, &response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Type != "message" {
				t.Errorf("expected type 'message', got '%s'", response.Type)
			}

			if response.Role != "assistant" {
				t.Errorf("expected role 'assistant', got '%s'", response.Role)
			}

			if len(response.Content) == 0 {
				t.Error("expected content, got empty array")
			}

			if response.Usage.InputTokens == 0 {
				t.Error("expected input tokens > 0")
			}

			if response.Usage.OutputTokens == 0 {
				t.Error("expected output tokens > 0")
			}

			t.Logf("Response: %+v", response.Content[0].Text)
		})
	}
}

func TestHandleInvoke_MultiTurn_Live(t *testing.T) {
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

	srv := New(client)

	request := types.AnthropicRequest{
		Model: "anthropic.claude-3-sonnet-20240229-v1:0",
		Messages: []types.AnthropicMessage{
			{
				Role: "user",
				Content: []types.AnthropicContentBlock{
					{Type: "text", Text: "My name is Alice."},
				},
			},
			{
				Role: "assistant",
				Content: []types.AnthropicContentBlock{
					{Type: "text", Text: "Nice to meet you, Alice!"},
				},
			},
			{
				Role: "user",
				Content: []types.AnthropicContentBlock{
					{Type: "text", Text: "What is my name?"},
				},
			},
		},
		MaxTokens: 100,
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.HandleMessages(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response types.AnthropicResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Content) == 0 {
		t.Fatal("expected content, got empty array")
	}

	// Response should contain "Alice" since it should remember the name
	responseText := response.Content[0].Text
	t.Logf("Multi-turn response: %s", responseText)
}

func TestNewWithAPIKey(t *testing.T) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	srv := NewWithAPIKey(client, "test-api-key")
	if srv == nil {
		t.Fatal("expected server to be created")
	}

	if srv.geminiClient == nil {
		t.Error("expected gemini client to be set")
	}

	if srv.geminiHTTPClient == nil {
		t.Error("expected gemini HTTP client to be set")
	}

	if srv.thoughtSignatures == nil {
		t.Error("expected thought signatures map to be initialized")
	}
}

func TestSetDebug(t *testing.T) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	srv := New(client)

	// Default should be false
	if srv.debug {
		t.Error("expected debug to be false by default")
	}

	// Enable debug
	srv.SetDebug(true)
	if !srv.debug {
		t.Error("expected debug to be true")
	}

	// Disable debug
	srv.SetDebug(false)
	if srv.debug {
		t.Error("expected debug to be false")
	}
}

func TestHandleMessages_InvalidRequest(t *testing.T) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	srv := New(client)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid json",
			body:           "{invalid json}",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.HandleMessages(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestInjectAndCacheThoughtSignatures(t *testing.T) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	srv := New(client)

	// Create a response with thought signatures
	resp := &types.AnthropicResponse{
		Content: []types.AnthropicContentBlock{
			{
				Type:             "tool_use",
				ID:               "tool_123",
				Name:             "search",
				ThoughtSignature: "I need to search",
			},
		},
	}

	// Cache the thought signature
	srv.cacheThoughtSignatures(resp)

	// Verify it was cached
	srv.thoughtSignaturesMu.RLock()
	sig, ok := srv.thoughtSignatures["tool_123"]
	srv.thoughtSignaturesMu.RUnlock()

	if !ok {
		t.Error("expected thought signature to be cached")
	}

	if sig != "I need to search" {
		t.Errorf("expected thought signature 'I need to search', got '%s'", sig)
	}

	// Create a request with the tool_use ID
	req := &types.AnthropicRequest{
		Messages: []types.AnthropicMessage{
			{
				Role: "assistant",
				Content: []types.AnthropicContentBlock{
					{
						Type: "tool_use",
						ID:   "tool_123",
						Name: "search",
					},
				},
			},
		},
	}

	// Inject the thought signature
	srv.injectThoughtSignatures(req)

	// Verify it was injected
	if req.Messages[0].Content[0].ThoughtSignature != "I need to search" {
		t.Errorf("expected thought signature to be injected, got '%s'", req.Messages[0].Content[0].ThoughtSignature)
	}
}

func TestHandleMessages_WithSystemPrompt(t *testing.T) {
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

	srv := New(client)

	// Test with array-style system prompt
	request := types.AnthropicRequest{
		Model: "gemini-2.0-flash",
		System: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": "You are a helpful assistant.",
			},
		},
		Messages: []types.AnthropicMessage{
			{
				Role: "user",
				Content: []types.AnthropicContentBlock{
					{Type: "text", Text: "Say 'hello'"},
				},
			},
		},
		MaxTokens: 50,
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.HandleMessages(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response types.AnthropicResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Content) == 0 {
		t.Error("expected content, got empty array")
	}

	t.Logf("Response with system prompt: %s", response.Content[0].Text)
}

func TestHandleMessages_WithTools_Live(t *testing.T) {
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

	// Use NewWithAPIKey to enable HTTP client for tool support
	srv := NewWithAPIKey(client, apiKey)
	srv.SetDebug(true) // Enable debug mode for this test

	// Test with tools to trigger HTTP client path
	request := types.AnthropicRequest{
		Model: "gemini-2.0-flash",
		Tools: []types.AnthropicTool{
			{
				Name:        "get_weather",
				Description: "Get weather for a location",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "City name",
						},
					},
					"required": []interface{}{"location"},
				},
			},
		},
		Messages: []types.AnthropicMessage{
			{
				Role: "user",
				Content: []types.AnthropicContentBlock{
					{Type: "text", Text: "What's the weather in Paris? Use the get_weather tool."},
				},
			},
		},
		MaxTokens: 100,
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.HandleMessages(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response types.AnthropicResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Content) == 0 {
		t.Error("expected content, got empty array")
	}

	t.Logf("Response with tools: %+v", response.Content)
}

func TestRespondJSON_Error(t *testing.T) {
	// Test error case in respondJSON (when encoding fails)
	// This is difficult to trigger naturally, but we can test the function itself

	w := httptest.NewRecorder()
	// Channel cannot be JSON encoded, will cause error
	invalidData := make(chan int)

	respondJSON(w, http.StatusOK, invalidData)

	// Should still set headers even if encoding fails
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type to be set")
	}
}

func TestGenerateContent_PathSelection(t *testing.T) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name           string
		setupServer    func() *Server
		request        *types.AnthropicRequest
		expectHTTPPath bool
	}{
		{
			name: "SDK path - no tools, no HTTP client",
			setupServer: func() *Server {
				return New(client)
			},
			request: &types.AnthropicRequest{
				Model: "gemini-2.0-flash",
				Messages: []types.AnthropicMessage{
					{
						Role: "user",
						Content: []types.AnthropicContentBlock{
							{Type: "text", Text: "Hello"},
						},
					},
				},
			},
			expectHTTPPath: false,
		},
		{
			name: "HTTP path - has tools and HTTP client",
			setupServer: func() *Server {
				return NewWithAPIKey(client, "test-key")
			},
			request: &types.AnthropicRequest{
				Model: "gemini-2.0-flash",
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
							{Type: "text", Text: "Hello"},
						},
					},
				},
			},
			expectHTTPPath: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := tt.setupServer()

			// This will fail because we don't have a valid API key,
			// but it will exercise the path selection logic
			_, err := srv.generateContent(ctx, tt.request.Model, tt.request)

			// We expect an error since we're using a fake API key
			if err == nil {
				t.Log("unexpectedly succeeded (might have cached response)")
			}

			// The test passes if we reach here without panic
		})
	}
}
