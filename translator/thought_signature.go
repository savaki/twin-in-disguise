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

	"github.com/google/generative-ai-go/genai"
	"github.com/savaki/twin-in-disguise/types"
)

// InjectThoughtSignatures takes genai.Content and injects thought signatures based on
// the original Anthropic messages
func InjectThoughtSignatures(contents []*genai.Content, messages []types.AnthropicMessage) ([]*genai.Content, error) {
	// Create a map of message indices to thought signatures
	// We need to correlate assistant messages with tool_use blocks that have thought signatures
	thoughtSigs := make(map[int]map[int]string) // messageIdx -> partIdx -> signature

	msgIdx := 0
	for _, msg := range messages {
		if msg.Role == "assistant" {
			partIdx := 0
			for _, block := range msg.Content {
				if block.Type == "tool_use" && block.ThoughtSignature != "" {
					if thoughtSigs[msgIdx] == nil {
						thoughtSigs[msgIdx] = make(map[int]string)
					}
					thoughtSigs[msgIdx][partIdx] = block.ThoughtSignature
				}
				partIdx++
			}
		}
		msgIdx++
	}

	// If no thought signatures found, return contents as-is
	if len(thoughtSigs) == 0 {
		return contents, nil
	}

	// We need to work with JSON to inject thought signatures
	// Marshal contents to JSON
	jsonBytes, err := json.Marshal(contents)
	if err != nil {
		return nil, err
	}

	// Unmarshal to our custom type that supports thought signatures
	var customContents []types.GeminiContent
	if err := json.Unmarshal(jsonBytes, &customContents); err != nil {
		return nil, err
	}

	// Inject thought signatures
	for contentIdx, content := range customContents {
		if sigs, ok := thoughtSigs[contentIdx]; ok {
			for partIdx, sig := range sigs {
				if partIdx < len(content.Parts) {
					customContents[contentIdx].Parts[partIdx].ThoughtSignature = sig
				}
			}
		}
	}

	// Convert back to genai.Content
	// We'll use JSON round-tripping since genai types should preserve unknown fields
	jsonBytes, err = json.Marshal(customContents)
	if err != nil {
		return nil, err
	}

	var result []*genai.Content
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}
