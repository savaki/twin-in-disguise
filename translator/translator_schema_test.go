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
	"encoding/json"
	"testing"
)

func TestCleanSchemaForGemini(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "removes $schema and additionalProperties at top level",
			input: map[string]interface{}{
				"$schema":              "http://json-schema.org/draft-07/schema#",
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		{
			name: "recursively removes from nested objects",
			input: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"nested": map[string]interface{}{
						"type":                 "object",
						"additionalProperties": false,
						"properties": map[string]interface{}{
							"field": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"nested": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"field": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
		},
		{
			name: "preserves required fields",
			input: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"name"},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []interface{}{"name"},
			},
		},
		{
			name: "handles arrays with nested schemas",
			input: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "integer",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "integer",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanSchemaForGemini(tt.input)

			// Compare as JSON to handle deep equality
			resultJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal result: %v", err)
			}

			expectedJSON, err := json.Marshal(tt.expected)
			if err != nil {
				t.Fatalf("failed to marshal expected: %v", err)
			}

			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("CleanSchemaForGemini() mismatch\nGot:  %s\nWant: %s", resultJSON, expectedJSON)
			}
		})
	}
}

func TestCleanSchemaForGemini_NilInput(t *testing.T) {
	result := CleanSchemaForGemini(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}
