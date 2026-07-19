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
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIURL = "https://api.anthropic.com/v1/messages"
	apiVersion    = "2023-06-01"
	defaultModel  = "claude-opus-4-8"
	// Arzon model — faqat bazaga aloqasi yo'q qisqa savollar uchun.
	defaultFastModel = "claude-haiku-4-5-20251001"
	defaultMaxTokens = 2048
)

// ErrNoAPIKey — API kaliti sozlanmagan.
var ErrNoAPIKey = errors.New("ANTHROPIC_API_KEY sozlanmagan")

// Client — Claude API klienti.
type Client struct {
	apiKey    string
	model     string
	fastModel string // qisqa, bazaga aloqasi yo'q savollar uchun
	url       string
	maxTokens int
	http      *http.Client
}

// Model — javob uchun qaysi modelni ishlatish.
type Model int

const (
	// Full — asosiy model. Hisob-kitob, kod, qonun — hamma jiddiy savol.
	Full Model = iota
	// Fast — arzon model. FAQAT bazadan hech narsa topilmagan qisqa
	// savollar uchun (salomlashish, "nima qila olasan").
	Fast
)

func (c *Client) modelFor(m Model) string {
	if m == Fast && c.fastModel != "" {
		return c.fastModel
	}
	return c.model
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
	fast := os.Getenv("ANTHROPIC_FAST_MODEL")
	if fast == "" {
		fast = defaultFastModel
	}
	return &Client{
		apiKey:    os.Getenv("ANTHROPIC_API_KEY"),
		model:     model,
		fastModel: fast,
		url:       url,
		maxTokens: envInt("ANTHROPIC_MAX_TOKENS", defaultMaxTokens),
		http:      &http.Client{Timeout: 120 * time.Second},
	}
}

// envInt — muhit o'zgaruvchisidan musbat butun son; noto'g'ri bo'lsa sukut.
func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
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
	System    []systemMsg  `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
	Stream    bool         `json:"stream,omitempty"`
}

// systemMsg — tizim ko'rsatmasi bloki. Massiv ko'rinishi kesh uchun zarur:
// oddiy matn bo'lsa, cache_control ni biriktirib bo'lmaydi.
type systemMsg struct {
	Type         string        `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// Tizim ko'rsatmasi ~6 000 belgi va HAR SO'ROVDA bir xil. Kesh bo'lmasa,
// u har safar qaytadan to'lanadi. Kesh o'qish narxi asl narxning ~10%i.
//
// Kesh ATAYLAB shu yerda: ko'rsatma barqaror, retrieval bloklari esa
// oxirgi FOYDALANUVCHI xabariga qo'shiladi — ular o'zgaruvchan bo'lgani
// uchun keshga tushmaydi va keshni buzmaydi ham.
const minCacheable = 2048 // belgi; bundan qisqa matnni keshlashning ma'nosi yo'q

// buildRequest — so'rov tanasini yig'adi. Complete va Stream uchun umumiy,
// shunda ikkalasi bir xil model, limit va kontekst bilan ishlaydi.
func (c *Client) buildRequest(model Model, system string, history []Message, stream bool) ([]byte, error) {
	msgs := make([]apiMessage, 0, len(history))
	for _, m := range history {
		msgs = append(msgs, apiMessage{Role: m.Role, Content: buildContent(m)})
	}

	var sys []systemMsg
	if system != "" {
		blk := systemMsg{Type: "text", Text: system}
		if len(system) >= minCacheable {
			blk.CacheControl = &cacheControl{Type: "ephemeral"}
		}
		sys = []systemMsg{blk}
	}

	return json.Marshal(apiRequest{
		Model:     c.modelFor(model),
		MaxTokens: c.maxTokens,
		System:    sys,
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
	Usage apiUsage `json:"usage"`
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

	body, err := c.buildRequest(Full, system, history, false)
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

	c.reportUsage(Full, out.Usage)

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
	Type    string   `json:"type"`
	Usage   apiUsage `json:"usage"`
	Message *struct {
		Usage apiUsage `json:"usage"`
	} `json:"message"`
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
func (c *Client) Stream(ctx context.Context, model Model, system string, history []Message, onChunk StreamFunc) error {
	if !c.Available() {
		return ErrNoAPIKey
	}
	body, err := c.buildRequest(model, system, history, true)
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

	// Oqimda sarf ikki joyda keladi: message_start da kirish tokenlari
	// (kesh ma'lumoti ham shu yerda), message_delta da chiqish tokenlari.
	var usage apiUsage
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
		if ev.Message != nil && ev.Message.Usage.InputTokens > 0 {
			usage.InputTokens = ev.Message.Usage.InputTokens
			usage.CacheCreationInputTokens = ev.Message.Usage.CacheCreationInputTokens
			usage.CacheReadInputTokens = ev.Message.Usage.CacheReadInputTokens
		}
		if ev.Usage.OutputTokens > 0 {
			usage.OutputTokens = ev.Usage.OutputTokens
		}
		if ev.Type != "content_block_delta" || ev.Delta.Type != "text_delta" || ev.Delta.Text == "" {
			continue
		}
		got = true
		if err := onChunk(ev.Delta.Text); err != nil {
			return err
		}
	}
	c.reportUsage(model, usage)

	if err := sc.Err(); err != nil {
		return fmt.Errorf("oqim uzildi: %w", err)
	}
	// Hech narsa kelmasa, chaqiruvchi buni sukut deb o'ylamasligi kerak.
	if !got {
		return errors.New("API bo'sh javob qaytardi")
	}
	return nil
}

// ---------------------------------------------------------------- hisob

// Usage — bitta so'rovning token sarfi.
//
// NEGA KERAK: xarajatni ko'rmasdan boshqarib bo'lmaydi. Ayniqsa
// CacheRead — kesh ishlayotganini shu ko'rsatadi. Kesh ishlamasa,
// CacheRead doim 0 bo'ladi va tizim ko'rsatmasi har safar to'liq
// to'lanadi.
type Usage struct {
	Model        string
	InputTokens  int
	OutputTokens int
	CacheWrite   int // keshga yozilgan (birinchi so'rovda)
	CacheRead    int // keshdan o'qilgan (arzon)
}

// OnUsage — har so'rovdan keyin chaqiriladi (nil bo'lishi mumkin).
// main.go da jurnalga yozish uchun o'rnatiladi.
var OnUsage func(Usage)

type apiUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

func (c *Client) reportUsage(model Model, u apiUsage) {
	if OnUsage == nil {
		return
	}
	OnUsage(Usage{
		Model:        c.modelFor(model),
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		CacheWrite:   u.CacheCreationInputTokens,
		CacheRead:    u.CacheReadInputTokens,
	})
}
