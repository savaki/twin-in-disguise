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
	"testing"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func TestHandleMessages_SystemPromptArray(t *testing.T) {
	// Mock Gemini client (this is a bit tricky without a real API key or a better mock,
	// but we can at least test the request parsing part which was failing)
	// For now, we'll rely on the fact that if parsing fails, it returns 400.
	// If parsing succeeds but generation fails (due to no API key), it returns 500.
	// So we expect 500, not 400.

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey("dummy"))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	server := New(client)

	// Create a request with system prompt as an array
	reqBody := map[string]interface{}{
		"model": "claude-3-opus-20240229",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Hello",
					},
				},
			},
		},
		"system": []map[string]interface{}{
			{
				"type": "text",
				"text": "You are a helpful assistant.",
			},
		},
		"max_tokens": 1024,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	server.HandleMessages(w, req)

	resp := w.Result()

	// We expect 500 because the dummy API key will fail generation,
	// but we definitely DO NOT want 400 (Bad Request) which would mean parsing failed.
	if resp.StatusCode == http.StatusBadRequest {
		t.Errorf("Expected status != 400, got 400. Parsing failed.")
	}
}
