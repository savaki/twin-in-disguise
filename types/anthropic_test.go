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

import (
	"encoding/json"
	"testing"
)

func TestAnthropicRequest_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "content as string",
			json: `{
				"model": "gemini-2.0-flash",
				"max_tokens": 1,
				"messages": [
					{
						"role": "user",
						"content": "count"
					}
				]
			}`,
			wantErr: false,
		},
		{
			name: "content as array",
			json: `{
				"model": "gemini-2.0-flash",
				"max_tokens": 1,
				"messages": [
					{
						"role": "user",
						"content": [
							{
								"type": "text",
								"text": "count"
							}
						]
					}
				]
			}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req AnthropicRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if len(req.Messages) != 1 {
					t.Errorf("expected 1 message, got %d", len(req.Messages))
				}
				if len(req.Messages[0].Content) != 1 {
					t.Errorf("expected 1 content block, got %d", len(req.Messages[0].Content))
				}
				if req.Messages[0].Content[0].Text != "count" {
					t.Errorf("expected content text 'count', got '%s'", req.Messages[0].Content[0].Text)
				}
			}
		})
	}
}

func TestAnthropicMessage_MarshalJSON(t *testing.T) {
	msg := AnthropicMessage{
		Role: "user",
		Content: []AnthropicContentBlock{
			{
				Type: "text",
				Text: "Hello",
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded AnthropicMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Role != msg.Role {
		t.Errorf("expected role '%s', got '%s'", msg.Role, decoded.Role)
	}

	if len(decoded.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(decoded.Content))
	}

	if decoded.Content[0].Text != "Hello" {
		t.Errorf("expected text 'Hello', got '%s'", decoded.Content[0].Text)
	}
}
