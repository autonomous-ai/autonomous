package domain

import "strings"

type LLMProvider struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

var ListProviders = []LLMProvider{
	{Key: "openai", Name: "OpenAI"},
	{Key: "anthropic", Name: "Anthropic"},
	{Key: "openai-codex", Name: "OpenAI Codex"},
	{Key: "opencode", Name: "OpenCode Zen"},
	{Key: "google", Name: "Google Gemini API"},
	{Key: "google-vertex", Name: "Google Vertex AI"},
	{Key: "google-antigravity", Name: "Google Antigravity"},
	{Key: "google-gemini-cli", Name: "Google Gemini CLI"},
	{Key: "zai", Name: "Z.AI (GLM Models)"},
	{Key: "vercel-ai-gateway", Name: "Vercel AI Gateway"},
	{Key: "openrouter", Name: "OpenRouter"},
	{Key: "xai", Name: "xAI"},
	{Key: "groq", Name: "Groq"},
	{Key: "cerebras", Name: "Cerebras"},
	{Key: "mistral", Name: "Mistral AI"},
	{Key: "github-copilot", Name: "GitHub Copilot"},
}

type LLMModelCapabilities struct {
	SupportsReasoning       bool `json:"supportsReasoning"`
	SupportsVision          bool `json:"supportsVision"`
	SupportsFunctionCalling bool `json:"supportsFunctionCalling"`
}

type LLMModel struct {
	Key           string                `json:"key"`
	Name          string                `json:"name"`
	Reasoning     bool                  `json:"reasoning"`
	Input         []string              `json:"input"`
	ContextWindow *int                  `json:"contextWindow"`
	MaxTokens     *int                  `json:"maxTokens"`
	Privacy       string                `json:"privacy"`
	Capabilities  *LLMModelCapabilities `json:"capabilities"`
}

// OpenClawAPIType returns the OpenClaw provider api type from raw substring check on Key and Name.
// e.g. "claude" -> "anthropic-messages", "gpt" -> "openai-completions", unknown -> "openai-completions".
func (m LLMModel) OpenClawAPIType() string {
	raw := strings.ToLower(m.Key + " " + m.Name)
	if strings.Contains(raw, "claude") {
		return "anthropic-messages"
	}
	if strings.Contains(raw, "gpt") {
		return "openai-completions"
	}
	return "openai-completions"
}

type LLMModelsListResponse struct {
	Count  int        `json:"count"`
	Models []LLMModel `json:"models"`
}
