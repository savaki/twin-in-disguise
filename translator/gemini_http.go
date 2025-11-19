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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/savaki/twin-in-disguise/types"
)

// GeminiHTTPClient makes direct HTTP calls to the Gemini API with support for thought signatures
type GeminiHTTPClient struct {
	apiKey  string
	baseURL string
}

// NewGeminiHTTPClient creates a new HTTP client for the Gemini API
func NewGeminiHTTPClient(apiKey string) *GeminiHTTPClient {
	return &GeminiHTTPClient{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
	}
}

// GenerateContentRequest represents a request to the Gemini API
type GenerateContentRequest struct {
	Contents          []types.GeminiContent `json:"contents"`
	Tools             []GeminiToolWrapper   `json:"tools,omitempty"`
	SystemInstruction *types.GeminiContent  `json:"systemInstruction,omitempty"`
	GenerationConfig  *GenerationConfig     `json:"generationConfig,omitempty"`
}

// GeminiToolWrapper wraps function declarations
type GeminiToolWrapper struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations"`
}

// FunctionDeclaration represents a function/tool declaration
type FunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// GenerationConfig represents generation configuration
type GenerationConfig struct {
	MaxOutputTokens *int32   `json:"maxOutputTokens,omitempty"`
	Temperature     *float32 `json:"temperature,omitempty"`
}

// GenerateContentResponse represents a response from the Gemini API
type GenerateContentResponse struct {
	Candidates    []Candidate    `json:"candidates,omitempty"`
	UsageMetadata *UsageMetadata `json:"usageMetadata,omitempty"`
}

// Candidate represents a response candidate
type Candidate struct {
	Content      *types.GeminiContent `json:"content,omitempty"`
	FinishReason string               `json:"finishReason,omitempty"`
}

// UsageMetadata represents usage statistics
type UsageMetadata struct {
	PromptTokenCount     int32 `json:"promptTokenCount"`
	CandidatesTokenCount int32 `json:"candidatesTokenCount"`
	TotalTokenCount      int32 `json:"totalTokenCount"`
}

// GenerateContent makes a generateContent API call with thought signature support
func (c *GeminiHTTPClient) GenerateContent(ctx context.Context, model string, req *GenerateContentRequest) (*GenerateContentResponse, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, model, c.apiKey)

	// Marshal request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Make request
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini API error: %s (status %d): %s", httpResp.Status, httpResp.StatusCode, string(respBody))
	}

	// Unmarshal response
	var geminiResp GenerateContentResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &geminiResp, nil
}
