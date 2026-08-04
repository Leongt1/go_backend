package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Minimal OpenAI chat-completions client (tool calling only, no streaming).
// Hand-rolled on net/http so we control the surface and can point BaseURL at
// a stub server in tests.

type OpenAIClient struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

func NewOpenAIClient(apiKey, model, baseURL string) *OpenAIClient {
	return &OpenAIClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenAIClient) configured() bool { return c.apiKey != "" }

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolDef struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

type functionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []toolDef     `json:"tools,omitempty"`
	// ResponseFormat forces a JSON reply (e.g. {"type":"json_object"}) for the
	// extraction path, which parses the content as structured data.
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// chat runs one tool-calling completion and returns the assistant message.
func (c *OpenAIClient) chat(ctx context.Context, messages []chatMessage, tools []toolDef) (*chatMessage, error) {
	return c.do(ctx, chatRequest{Model: c.model, Messages: messages, Tools: tools})
}

// completeJSON runs one completion with no tools, forcing a JSON-object reply,
// and returns the raw content string for the caller to parse.
func (c *OpenAIClient) completeJSON(ctx context.Context, messages []chatMessage) (string, error) {
	msg, err := c.do(ctx, chatRequest{
		Model:          c.model,
		Messages:       messages,
		ResponseFormat: map[string]string{"type": "json_object"},
	})
	if err != nil {
		return "", err
	}
	return msg.Content, nil
}

func (c *OpenAIClient) do(ctx context.Context, reqBody chatRequest) (*chatMessage, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("openai: unparseable response (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		msg := "unknown error"
		if parsed.Error != nil {
			msg = parsed.Error.Message
		}
		return nil, fmt.Errorf("openai: status %d: %s", resp.StatusCode, msg)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai: empty choices")
	}
	return &parsed.Choices[0].Message, nil
}
