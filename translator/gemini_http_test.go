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
	"testing"

	"github.com/savaki/twin-in-disguise/types"
)

func TestNewGeminiHTTPClient(t *testing.T) {
	client := NewGeminiHTTPClient("test-api-key")
	if client == nil {
		t.Fatal("expected client to be created")
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("expected apiKey 'test-api-key', got '%s'", client.apiKey)
	}

	if client.baseURL != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("unexpected baseURL: %s", client.baseURL)
	}
}

func TestGeminiHTTPClient_GenerateContent_InvalidAPIKey(t *testing.T) {
	client := NewGeminiHTTPClient("invalid-key")

	req := &GenerateContentRequest{
		Contents: []types.GeminiContent{
			{
				Role: "user",
				Parts: []types.GeminiPart{
					{Text: "Hello"},
				},
			},
		},
	}

	ctx := context.Background()
	_, err := client.GenerateContent(ctx, "gemini-2.0-flash", req)

	// Should get an error (likely 400 or 401)
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}
