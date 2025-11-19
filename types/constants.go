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

// Content block types
const (
	ContentTypeText       = "text"
	ContentTypeImage      = "image"
	ContentTypeToolUse    = "tool_use"
	ContentTypeToolResult = "tool_result"
)

// Role types
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleModel     = "model"
)

// Response types
const (
	ResponseTypeMessage = "message"
	StopReasonEndTurn   = "end_turn"
)

// JSON Schema field names
const (
	SchemaFieldType                 = "type"
	SchemaFieldProperties           = "properties"
	SchemaFieldRequired             = "required"
	SchemaFieldDescription          = "description"
	SchemaFieldEnum                 = "enum"
	SchemaFieldItems                = "items"
	SchemaFieldDollarSchema         = "$schema"
	SchemaFieldAdditionalProperties = "additionalProperties"
)

// JSON Schema type values
const (
	SchemaTypeString  = "string"
	SchemaTypeNumber  = "number"
	SchemaTypeInteger = "integer"
	SchemaTypeBoolean = "boolean"
	SchemaTypeArray   = "array"
	SchemaTypeObject  = "object"
)

// Response field names
const (
	ResponseFieldResult = "result"
	ResponseFieldError  = "error"
)
