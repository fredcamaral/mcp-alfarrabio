package sampling

import "encoding/json"

// SamplingMessage represents a message in the sampling context
type SamplingMessage struct {
	Role    string                 `json:"role"`
	Content SamplingMessageContent `json:"content"`
}

// SamplingMessageContent can be either a string or structured content
type SamplingMessageContent struct {
	Type string          `json:"type,omitempty"`
	Text string          `json:"text,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// ModelPreferences defines model-specific preferences for sampling
type ModelPreferences struct {
	Hints              []ModelHint            `json:"hints,omitempty"`
	CostPriority       float64                `json:"costPriority,omitempty"`
	SpeedPriority      float64                `json:"speedPriority,omitempty"`
	IntelligencePriority float64              `json:"intelligencePriority,omitempty"`
}

// ModelHint provides hints about model selection
type ModelHint struct {
	Name string `json:"name,omitempty"`
}

// CreateMessageRequest is the request structure for sampling/createMessage
type CreateMessageRequest struct {
	Messages         []SamplingMessage `json:"messages"`
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
	SystemPrompt     *string           `json:"systemPrompt,omitempty"`
	IncludeContext   string            `json:"includeContext,omitempty"`
	Temperature      *float64          `json:"temperature,omitempty"`
	MaxTokens        int               `json:"maxTokens"`
	StopSequences    []string          `json:"stopSequences,omitempty"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
}

// CreateMessageResponse is the response structure for sampling/createMessage
type CreateMessageResponse struct {
	Role    string                 `json:"role"`
	Content SamplingMessageContent `json:"content"`
	Model   string                 `json:"model"`
	StopReason string              `json:"stopReason,omitempty"`
}