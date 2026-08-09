package llm

// GLM (Z.ai) provayderi — OpenAI-mos Chat Completions API.
//
// NEGA ALOHIDA FAYL: Z.ai ning GLM-5.2 uchun rasmiy endpointi Anthropic
// emas, OpenAI formatida ishlaydi (api.z.ai/api/paas/v4/chat/completions).
// Ya'ni ANTHROPIC_API_URL ni almashtirish bilan qutulib bo'lmaydi —
// so'rov va javob tuzilishi boshqacha:
//
//	Anthropic                     OpenAI-mos
//	system: [{type,text}]     →   messages[0].role = "system"
//	content: [bloklar]        →   content: matn
//	usage.input_tokens        →   usage.prompt_tokens (kesh ICHIDA)
//	content_block_delta       →   choices[0].delta.content
//
// ⚠️ GLM-5.2 FAQAT MATN. Rasmni qabul qilmaydi (Z.ai hujjati: "Input
// Modalities: Text"). Shuning uchun rasmli xabar bu provayderga
// YUBORILMAYDI — buni llm.go dagi pickModel qat'iy ta'minlaydi.

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
	defaultGLMURL   = "https://api.z.ai/api/paas/v4/chat/completions"
	defaultGLMModel = "glm-5.2"
	// 1M kontekstda birinchi belgi kechikishi Claude ga qaraganda uzunroq —
	// Z.ai hujjati uzoqroq timeout tavsiya qiladi.
	defaultGLMTimeout = 180 * time.Second
)

// glmClient — OpenAI-mos xizmat klienti.
type glmClient struct {
	apiKey    string
	enabled   bool // GLM_ENABLED=1 — ochiq ruxsat
	model     string
	url       string
	maxTokens int
	http      *http.Client
}

// newGLM — muhit o'zgaruvchilaridan GLM klienti.
//
//	GLM_ENABLED — "1" bo'lmasa provayder O'CHIQ (sukut: hammasi Claude da)
//	GLM_API_KEY — majburiy, yoqilgan bo'lsa
//	GLM_MODEL   — ixtiyoriy, sukut glm-5.2
//	GLM_API_URL — ixtiyoriy (testda soxta server uchun ham)
func newGLM(maxTokens int) *glmClient {
	model := os.Getenv("GLM_MODEL")
	if model == "" {
		model = defaultGLMModel
	}
	url := os.Getenv("GLM_API_URL")
	if url == "" {
		url = defaultGLMURL
	}
	timeout := defaultGLMTimeout
	if s := envInt("GLM_TIMEOUT_SECONDS", 0); s > 0 {
		timeout = time.Duration(s) * time.Second
	}
	return &glmClient{
		apiKey:    os.Getenv("GLM_API_KEY"),
		enabled:   os.Getenv("GLM_ENABLED") == "1",
		model:     model,
		url:       url,
		maxTokens: maxTokens,
		http:      &http.Client{Timeout: timeout},
	}
}

// available — GLM ishlatiladimi.
//
// IKKI shart: kalit VA ochiq ruxsat (GLM_ENABLED=1).
//
// NEGA IKKITA: sukut bo'yicha HAMMA so'rov Claude ga ketadi. Faqat kalit
// yetarli bo'lganda, muhitda tasodifan qolib ketgan GLM_API_KEY jimgina
// provayderni almashtirib yuborardi — foydalanuvchi esa javob boshqa
// modeldan kelayotganini bilmasdi. Bunday almashinuv OSHKORA bo'lishi
// kerak, ayniqsa javob yuridik oqibatga ega bo'lganda.
func (g *glmClient) available() bool {
	return g != nil && g.apiKey != "" && g.enabled
}

// ---------------------------------------------------------------- so'rov

type glmRequest struct {
	Model         string         `json:"model"`
	Messages      []glmMessage   `json:"messages"`
	MaxTokens     int            `json:"max_tokens"`
	Stream        bool           `json:"stream,omitempty"`
	StreamOptions *glmStreamOpts `json:"stream_options,omitempty"`
}

// StreamOptions — usage ni oqimning oxirgi bo'lagida olish uchun.
// Ularsiz OpenAI-mos oqim token sarfini UMUMAN qaytarmaydi va xarajat
// ko'rinmay qoladi.
type glmStreamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

type glmMessage struct {
	Role    string `json:"role"` // "system" | "user" | "assistant"
	Content string `json:"content"`
}

func (g *glmClient) buildRequest(system string, history []Message, stream bool) ([]byte, error) {
	msgs := make([]glmMessage, 0, len(history)+1)
	if system != "" {
		msgs = append(msgs, glmMessage{Role: "system", Content: system})
	}
	for _, m := range history {
		// Rasm bu yerga yetib kelmasligi kerak (pickModel to'sadi), lekin
		// himoya sifatida: rasmli xabar jimgina MATNGA aylanib qolmasin —
		// bu foydalanuvchiga "rasmni ko'rdim" degan taassurot berardi.
		if len(m.Images) > 0 {
			return nil, errors.New("GLM rasmni qo'llab-quvvatlamaydi")
		}
		msgs = append(msgs, glmMessage{Role: m.Role, Content: m.Content})
	}

	req := glmRequest{
		Model:     g.model,
		Messages:  msgs,
		MaxTokens: g.maxTokens,
		Stream:    stream,
	}
	if stream {
		req.StreamOptions = &glmStreamOpts{IncludeUsage: true}
	}
	return json.Marshal(req)
}

func (g *glmClient) newHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// OpenAI-mos xizmatlar Bearer sxemasini ishlatadi (x-api-key emas).
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	return req, nil
}

// ---------------------------------------------------------------- javob

type glmUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	Details          struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
}

// toAPIUsage — OpenAI hisobini Anthropic ma'nosiga keltiradi.
//
// MUHIM FARQ: OpenAI da prompt_tokens keshlangan tokenlarni HAM o'z ichiga
// oladi, Anthropic da esa input_tokens keshdan tashqari qismi. Ayirmasak,
// admin panelidagi "kirish" ustuni ikki provayderda boshqa narsani
// anglatardi va taqqoslash noto'g'ri chiqardi.
func (u glmUsage) toAPIUsage() apiUsage {
	in := u.PromptTokens - u.Details.CachedTokens
	if in < 0 {
		in = 0
	}
	return apiUsage{
		InputTokens:          in,
		OutputTokens:         u.CompletionTokens,
		CacheReadInputTokens: u.Details.CachedTokens,
		// Kesh yozish: Z.ai da kesh avtomatik va alohida hisoblanmaydi.
	}
}

type glmResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage glmUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Complete — to'liq javob.
func (g *glmClient) Complete(ctx context.Context, system string, history []Message) (string, error) {
	body, err := g.buildRequest(system, history, false)
	if err != nil {
		return "", err
	}
	req, err := g.newHTTPRequest(ctx, body)
	if err != nil {
		return "", err
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var out glmResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("GLM javobini o'qishda xato: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("GLM xatosi: %s", out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GLM status %d: %s", resp.StatusCode, string(raw))
	}
	reportUsage(g.model, out.Usage.toAPIUsage())

	if len(out.Choices) == 0 {
		return "", errors.New("GLM bo'sh javob qaytardi")
	}
	return out.Choices[0].Message.Content, nil
}

// glmStreamEvent — SSE bo'lagining bizga kerakli qismi.
type glmStreamEvent struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
			// ReasonContent — GLM ning ichki mulohazasi. Foydalanuvchiga
			// KO'RSATILMAYDI: u javob emas, o'ylash jarayoni.
			ReasonContent string `json:"reasoning_content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *glmUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Stream — javobni bo'lak-bo'lak qaytaradi.
func (g *glmClient) Stream(ctx context.Context, system string, history []Message, onChunk StreamFunc) error {
	body, err := g.buildRequest(system, history, true)
	if err != nil {
		return err
	}
	req, err := g.newHTTPRequest(ctx, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		var out glmResponse
		if json.Unmarshal(raw, &out) == nil && out.Error != nil {
			return fmt.Errorf("GLM xatosi: %s", out.Error.Message)
		}
		return fmt.Errorf("GLM status %d: %s", resp.StatusCode, string(raw))
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var usage apiUsage
	var got bool
	for sc.Scan() {
		data, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		if data == "[DONE]" {
			break
		}
		var ev glmStreamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}
		if ev.Error != nil {
			return fmt.Errorf("GLM xatosi: %s", ev.Error.Message)
		}
		if ev.Usage != nil {
			usage = ev.Usage.toAPIUsage()
		}
		for _, ch := range ev.Choices {
			if ch.Delta.Content == "" {
				continue
			}
			got = true
			if err := onChunk(ch.Delta.Content); err != nil {
				return err
			}
		}
	}
	reportUsage(g.model, usage)

	if err := sc.Err(); err != nil {
		return fmt.Errorf("GLM oqimi uzildi: %w", err)
	}
	if !got {
		return errors.New("GLM bo'sh javob qaytardi")
	}
	return nil
}
