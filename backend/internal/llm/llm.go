// Package llm Anthropic Claude Messages API bilan ishlaydigan minimal klient.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	apiURL       = "https://api.anthropic.com/v1/messages"
	apiVersion   = "2023-06-01"
	defaultModel = "claude-opus-4-8"
)

// ErrNoAPIKey — API kaliti sozlanmagan.
var ErrNoAPIKey = errors.New("ANTHROPIC_API_KEY sozlanmagan")

// Client — Claude API klienti.
type Client struct {
	apiKey string
	model  string
	http   *http.Client
}

// New — muhit o'zgaruvchilaridan klient yaratadi.
// ANTHROPIC_API_KEY majburiy; ANTHROPIC_MODEL ixtiyoriy.
func New() *Client {
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = defaultModel
	}
	return &Client{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		model:  model,
		http:   &http.Client{Timeout: 60 * time.Second},
	}
}

// Available — API kaliti mavjudligini bildiradi.
func (c *Client) Available() bool {
	return c.apiKey != ""
}

// Message — suhbat xabari.
type Message struct {
	Role    string `json:"role"` // "user" yoki "assistant"
	Content string `json:"content"`
}

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Complete — tizim ko'rsatmasi va xabarlar tarixidan javob oladi.
func (c *Client) Complete(ctx context.Context, system string, history []Message) (string, error) {
	if !c.Available() {
		return "", ErrNoAPIKey
	}

	msgs := make([]apiMessage, 0, len(history))
	for _, m := range history {
		msgs = append(msgs, apiMessage{Role: m.Role, Content: m.Content})
	}

	body, err := json.Marshal(apiRequest{
		Model:     c.model,
		MaxTokens: 1024,
		System:    system,
		Messages:  msgs,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var out apiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("javobni o'qishda xato: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("API xatosi: %s", out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API status %d: %s", resp.StatusCode, string(raw))
	}

	var text string
	for _, blk := range out.Content {
		if blk.Type == "text" {
			text += blk.Text
		}
	}
	return text, nil
}
