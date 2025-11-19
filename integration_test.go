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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/savaki/twin-in-disguise/server"
	"github.com/savaki/twin-in-disguise/types"
	"google.golang.org/api/option"
)

func TestIntegration_FullProxy_Live(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping live test: GEMINI_API_KEY not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create Gemini client
	geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("failed to create Gemini client: %v", err)
	}
	defer geminiClient.Close()

	// Create server
	srv := server.New(geminiClient)

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", srv.HandleMessages)

	// Use a fixed port for testing
	port := 18080
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start server in background
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	defer httpServer.Shutdown(ctx)

	// Wait for server to be ready
	time.Sleep(500 * time.Millisecond)

	baseURL := fmt.Sprintf("http://localhost:%d", port)

	t.Run("invoke with gemini model", func(t *testing.T) {
		req := types.AnthropicRequest{
			Model: "gemini-2.0-flash",
			Messages: []types.AnthropicMessage{
				{
					Role: "user",
					Content: []types.AnthropicContentBlock{
						{Type: "text", Text: "Say 'integration test' and nothing else"},
					},
				},
			},
			MaxTokens: 100,
		}

		body, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			baseURL+"/v1/messages",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("failed to call invoke: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
			return
		}

		var anthropicResp types.AnthropicResponse
		if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if anthropicResp.Type != "message" {
			t.Errorf("expected type 'message', got '%s'", anthropicResp.Type)
		}

		if len(anthropicResp.Content) == 0 {
			t.Error("expected content, got empty array")
		}

		t.Logf("Response: %s", anthropicResp.Content[0].Text)
	})

	t.Run("invoke with another model", func(t *testing.T) {
		req := types.AnthropicRequest{
			Model: "gemini-2.0-flash",
			Messages: []types.AnthropicMessage{
				{
					Role: "user",
					Content: []types.AnthropicContentBlock{
						{Type: "text", Text: "Say 'another test' and nothing else"},
					},
				},
			},
			MaxTokens: 50,
		}

		body, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			baseURL+"/v1/messages",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("failed to call invoke: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
			return
		}

		var anthropicResp types.AnthropicResponse
		if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		t.Logf("Haiku response: %s", anthropicResp.Content[0].Text)
	})

	t.Run("invoke with system prompt", func(t *testing.T) {
		req := types.AnthropicRequest{
			Model:  "gemini-2.0-flash",
			System: "You are a helpful assistant that responds in exactly one word.",
			Messages: []types.AnthropicMessage{
				{
					Role: "user",
					Content: []types.AnthropicContentBlock{
						{Type: "text", Text: "What color is the sky?"},
					},
				},
			},
			MaxTokens: 50,
		}

		body, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		resp, err := http.Post(
			baseURL+"/v1/messages",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("failed to call invoke: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
			return
		}

		var anthropicResp types.AnthropicResponse
		if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		t.Logf("System prompt response: %s", anthropicResp.Content[0].Text)
	})
}
