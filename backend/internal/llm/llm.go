// Package llm Anthropic Claude Messages API bilan ishlaydigan minimal klient.
// Matn va rasm (vision) xabarlarini qo'llab-quvvatlaydi.
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
		http:   &http.Client{Timeout: 120 * time.Second},
	}
}

// Available — API kaliti mavjudligini bildiradi.
func (c *Client) Available() bool {
	return c.apiKey != ""
}

// Image — suhbatga biriktirilgan rasm (base64, "data:" prefiksisiz).
type Image struct {
	MediaType string `json:"media_type"` // masalan "image/jpeg", "image/png"
	Data      string `json:"data"`       // base64 kodlangan tarkib
}

// Message — suhbat xabari (matn va ixtiyoriy rasmlar).
type Message struct {
	Role    string  `json:"role"` // "user" yoki "assistant"
	Content string  `json:"content"`
	Images  []Image `json:"images,omitempty"`
}

// DIQQAT: `temperature` bu yerda ATAYLAB yo'q.
//
// Javob barqarorligini oshirish uchun temperature=0.2 qo'yib ko'rildi, ammo
// API xato qaytardi: "`temperature` is deprecated for this model"
// (claude-opus-4-8). Shuning uchun javob turg'unligi so'rov parametri bilan
// emas, kontekstni aniq yozish bilan ta'minlanadi — masalan aksiz bo'shlig'i
// chat.formatMatches da ochiq ogohlantirish sifatida chiqariladi.
type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
}

// apiMessage.Content matn (string) yoki bloklar ro'yxati (any) bo'lishi mumkin.
type apiMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type textBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

type imageBlock struct {
	Type   string      `json:"type"` // "image"
	Source imageSource `json:"source"`
}

type imageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
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

// buildContent — xabarni API formatiga o'giradi.
// Rasm bo'lmasa oddiy matn (string), aks holda bloklar ro'yxati qaytariladi.
func buildContent(m Message) any {
	if len(m.Images) == 0 {
		return m.Content
	}
	blocks := make([]any, 0, len(m.Images)+1)
	for _, img := range m.Images {
		blocks = append(blocks, imageBlock{
			Type: "image",
			Source: imageSource{
				Type:      "base64",
				MediaType: img.MediaType,
				Data:      img.Data,
			},
		})
	}
	if m.Content != "" {
		blocks = append(blocks, textBlock{Type: "text", Text: m.Content})
	}
	return blocks
}

// Complete — tizim ko'rsatmasi va xabarlar tarixidan javob oladi.
func (c *Client) Complete(ctx context.Context, system string, history []Message) (string, error) {
	if !c.Available() {
		return "", ErrNoAPIKey
	}

	msgs := make([]apiMessage, 0, len(history))
	for _, m := range history {
		msgs = append(msgs, apiMessage{Role: m.Role, Content: buildContent(m)})
	}

	body, err := json.Marshal(apiRequest{
		Model:     c.model,
		MaxTokens: 2048,
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
