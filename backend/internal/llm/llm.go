// Package llm Anthropic Claude Messages API bilan ishlaydigan minimal klient.
// Matn va rasm (vision) xabarlarini qo'llab-quvvatlaydi.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultAPIURL = "https://api.anthropic.com/v1/messages"
	apiVersion    = "2023-06-01"
	defaultModel  = "claude-opus-4-8"
)

// ErrNoAPIKey — API kaliti sozlanmagan.
var ErrNoAPIKey = errors.New("ANTHROPIC_API_KEY sozlanmagan")

// Client — Claude API klienti.
type Client struct {
	apiKey string
	model  string
	url    string
	http   *http.Client
}

// New — muhit o'zgaruvchilaridan klient yaratadi.
//
//	ANTHROPIC_API_KEY — majburiy (bo'lmasa chat o'chiq)
//	ANTHROPIC_MODEL   — ixtiyoriy
//	ANTHROPIC_API_URL — ixtiyoriy: so'rovlar korporativ shlyuz orqali
//	                    yuborilsa yoki testda soxta server ishlatilsa
func New() *Client {
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = defaultModel
	}
	url := os.Getenv("ANTHROPIC_API_URL")
	if url == "" {
		url = defaultAPIURL
	}
	return &Client{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		model:  model,
		url:    url,
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
	Stream    bool         `json:"stream,omitempty"`
}

const maxTokens = 2048

// buildRequest — so'rov tanasini yig'adi. Complete va Stream uchun umumiy,
// shunda ikkalasi bir xil model, limit va kontekst bilan ishlaydi.
func (c *Client) buildRequest(system string, history []Message, stream bool) ([]byte, error) {
	msgs := make([]apiMessage, 0, len(history))
	for _, m := range history {
		msgs = append(msgs, apiMessage{Role: m.Role, Content: buildContent(m)})
	}
	return json.Marshal(apiRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  msgs,
		Stream:    stream,
	})
}

// newHTTPRequest — sarlavhalari qo'yilgan POST so'rov.
func (c *Client) newHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	return req, nil
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

	body, err := c.buildRequest(system, history, false)
	if err != nil {
		return "", err
	}
	req, err := c.newHTTPRequest(ctx, body)
	if err != nil {
		return "", err
	}

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

// ---------------------------------------------------------------- streaming

// StreamFunc — javobning har bir bo'lagi kelganda chaqiriladi.
// Xato qaytarsa, oqim to'xtatiladi (masalan foydalanuvchi ketib qolgan).
type StreamFunc func(chunk string) error

// streamEvent — SSE hodisasining bizga kerakli qismi.
//
// Anthropic oqimi bir necha turdagi hodisa yuboradi (message_start,
// content_block_start, ping, message_stop...). Bizga faqat matn bo'laklari
// kerak: type="content_block_delta", delta.type="text_delta".
type streamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Stream — javobni bo'lak-bo'lak qaytaradi.
//
// NEGA KERAK: to'liq javob 23–49 soniya oladi. Foydalanuvchi shuncha vaqt
// bo'sh ekranga qarab turmasligi uchun matn yozilayotganda ko'rsatiladi.
//
// Xatolar ikki joyda chiqishi mumkin: oqim BOSHLANISHIDAN oldin (HTTP
// status va JSON xato tanasi) va oqim ICHIDA (event: error). Ikkalasi ham
// qayta ishlanadi — aks holda foydalanuvchi yarim javob olib, nima
// bo'lganini bilmay qolardi.
func (c *Client) Stream(ctx context.Context, system string, history []Message, onChunk StreamFunc) error {
	if !c.Available() {
		return ErrNoAPIKey
	}
	body, err := c.buildRequest(system, history, true)
	if err != nil {
		return err
	}
	req, err := c.newHTTPRequest(ctx, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Oqim boshlanmasdan xato qaytgan bo'lsa — tana oddiy JSON.
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		var out apiResponse
		if json.Unmarshal(raw, &out) == nil && out.Error != nil {
			return fmt.Errorf("API xatosi: %s", out.Error.Message)
		}
		return fmt.Errorf("API status %d: %s", resp.StatusCode, string(raw))
	}

	sc := bufio.NewScanner(resp.Body)
	// Bo'laklar kichik, lekin bitta hodisa uzun bo'lib qolsa oqim
	// jim uzilib qolmasligi uchun buferni kengaytiramiz.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var got bool
	for sc.Scan() {
		line := sc.Text()
		// SSE: bizga faqat "data:" qatorlari kerak, "event:" va bo'sh
		// qatorlar tashlab yuboriladi.
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		if data == "[DONE]" {
			break
		}
		var ev streamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue // noma'lum hodisa — e'tiborsiz qoldiramiz
		}
		if ev.Error != nil {
			return fmt.Errorf("API xatosi: %s", ev.Error.Message)
		}
		if ev.Type != "content_block_delta" || ev.Delta.Type != "text_delta" || ev.Delta.Text == "" {
			continue
		}
		got = true
		if err := onChunk(ev.Delta.Text); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("oqim uzildi: %w", err)
	}
	// Hech narsa kelmasa, chaqiruvchi buni sukut deb o'ylamasligi kerak.
	if !got {
		return errors.New("API bo'sh javob qaytardi")
	}
	return nil
}
