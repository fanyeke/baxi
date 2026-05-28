package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"baxi/internal/config"
)

// ProviderError wraps errors from LLM providers with metadata.
type ProviderError struct {
	Err       error
	Provider  string
	Model     string
	LatencyMs int64
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s[%s]: %v", e.Provider, e.Model, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// OpenAICompatibleProvider implements DecisionProvider using an OpenAI-compatible API.
type OpenAICompatibleProvider struct {
	client         openai.Client
	model          string
	temperature    float64
	maxTokens      int
	timeout        time.Duration
	promptRegistry *PromptRegistry
}

// NewOpenAIProvider creates a new OpenAICompatibleProvider.
func NewOpenAIProvider(cfg *config.Config, registry *PromptRegistry) (*OpenAICompatibleProvider, error) {
	if cfg.LLMAPIKey == "" {
		return nil, errors.New("LLM_API_KEY is required")
	}

	clientOpts := []option.RequestOption{
		option.WithAPIKey(cfg.LLMAPIKey),
	}
	if cfg.LLMAPIBase != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(cfg.LLMAPIBase))
	}

	return &OpenAICompatibleProvider{
		client:         openai.NewClient(clientOpts...),
		model:          cfg.LLMModel,
		temperature:    cfg.LLMTemperature,
		maxTokens:      cfg.LLMMaxTokens,
		timeout:        time.Duration(cfg.LLMTimeoutSeconds) * time.Second,
		promptRegistry: registry,
	}, nil
}

// Name returns the provider name.
func (p *OpenAICompatibleProvider) Name() string {
	return "openai_compatible"
}

func (p *OpenAICompatibleProvider) ModelName() string {
	return p.model
}

// GenerateDecision implements DecisionProvider.GenerateDecision.
func (p *OpenAICompatibleProvider) GenerateDecision(ctx context.Context, input LLMSafeContext) (*DecisionOutput, error) {
	// Load prompt templates
	systemPrompt, err := p.promptRegistry.Load("decision_support")
	if err != nil {
		return nil, fmt.Errorf("load prompt: %w", err)
	}

	// Render user prompt with context
	contextJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal context: %w", err)
	}

	userPrompt, err := p.promptRegistry.RenderUserPrompt("decision_support", UserPromptData{
		ContextJSON:      string(contextJSON),
		AllowedActions:   input.AllowedActions,
		ForbiddenActions: input.ForbiddenActions,
	})
	if err != nil {
		return nil, fmt.Errorf("render user prompt: %w", err)
	}

	// Call API with timeout
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	startTime := time.Now()

	chatResp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(p.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt.SystemPrompt),
			openai.UserMessage(userPrompt),
		},
		Temperature: openai.Float(p.temperature),
		MaxTokens:   openai.Int(int64(p.maxTokens)),
		Seed:        openai.Int(42), // deterministic for replay
	})

	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		return nil, &ProviderError{
			Err:       err,
			Provider:  p.Name(),
			Model:     p.model,
			LatencyMs: latencyMs,
		}
	}

	// Check empty choices
	if len(chatResp.Choices) == 0 {
		return nil, &ProviderError{
			Err:       errors.New("empty choices from OpenAI API"),
			Provider:  p.Name(),
			Model:     p.model,
			LatencyMs: latencyMs,
		}
	}

	// Check for refusal
	if chatResp.Choices[0].Message.Refusal != "" {
		return nil, &ProviderError{
			Err:       fmt.Errorf("model refusal: %s", chatResp.Choices[0].Message.Refusal),
			Provider:  p.Name(),
			Model:     p.model,
			LatencyMs: latencyMs,
		}
	}

	content := chatResp.Choices[0].Message.Content

	// Parse JSON from response
	var output DecisionOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	// Populate schema_version if missing (assume v1 for OpenAI responses)
	if output.SchemaVersion == "" {
		output.SchemaVersion = "decision_output.v1"
	}

	// Always enforce requires_human_review
	output.RequiresHumanReview = true

	return &output, nil
}

// GenerateDecisionRaw returns both the parsed DecisionOutput and the raw response string.
func (p *OpenAICompatibleProvider) GenerateDecisionRaw(ctx context.Context, input LLMSafeContext) (*DecisionOutput, string, error) {
	systemPrompt, err := p.promptRegistry.Load("decision_support")
	if err != nil {
		return nil, "", fmt.Errorf("load prompt: %w", err)
	}

	contextJSON, err := json.Marshal(input)
	if err != nil {
		return nil, "", fmt.Errorf("marshal context: %w", err)
	}

	userPrompt, err := p.promptRegistry.RenderUserPrompt("decision_support", UserPromptData{
		ContextJSON:      string(contextJSON),
		AllowedActions:   input.AllowedActions,
		ForbiddenActions: input.ForbiddenActions,
	})
	if err != nil {
		return nil, "", fmt.Errorf("render user prompt: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	startTime := time.Now()

	chatResp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(p.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt.SystemPrompt),
			openai.UserMessage(userPrompt),
		},
		Temperature: openai.Float(p.temperature),
		MaxTokens:   openai.Int(int64(p.maxTokens)),
		Seed:        openai.Int(42),
	})

	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		return nil, "", &ProviderError{
			Err:       err,
			Provider:  p.Name(),
			Model:     p.model,
			LatencyMs: latencyMs,
		}
	}

	if len(chatResp.Choices) == 0 {
		return nil, "", &ProviderError{
			Err:       errors.New("empty choices from OpenAI API"),
			Provider:  p.Name(),
			Model:     p.model,
			LatencyMs: latencyMs,
		}
	}

	if chatResp.Choices[0].Message.Refusal != "" {
		return nil, "", &ProviderError{
			Err:       fmt.Errorf("model refusal: %s", chatResp.Choices[0].Message.Refusal),
			Provider:  p.Name(),
			Model:     p.model,
			LatencyMs: latencyMs,
		}
	}

	content := chatResp.Choices[0].Message.Content

	var output DecisionOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		return nil, "", fmt.Errorf("parse LLM response: %w", err)
	}

	if output.SchemaVersion == "" {
		output.SchemaVersion = "decision_output.v1"
	}
	output.RequiresHumanReview = true

	return &output, content, nil
}
