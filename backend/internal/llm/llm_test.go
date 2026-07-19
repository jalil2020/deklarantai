package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Matnli xabar oddiy string sifatida kodlanishi kerak.
func TestBuildContentTextOnly(t *testing.T) {
	c := buildContent(Message{Role: "user", Content: "salom"})
	if s, ok := c.(string); !ok || s != "salom" {
		t.Fatalf("matnli content string bo'lishi kerak, oldik: %#v", c)
	}
}

// Rasmli xabar bloklar ro'yxati sifatida kodlanishi kerak.
func TestBuildContentWithImage(t *testing.T) {
	c := buildContent(Message{
		Role:    "user",
		Content: "bu nima?",
		Images:  []Image{{MediaType: "image/jpeg", Data: "QUJD"}},
	})
	blocks, ok := c.([]any)
	if !ok {
		t.Fatalf("rasmli content ro'yxat bo'lishi kerak, oldik: %#v", c)
	}
	if len(blocks) != 2 {
		t.Fatalf("2 blok kutilgan (rasm + matn), oldik: %d", len(blocks))
	}

	// JSON strukturasi Anthropic API formatiga mos kelishini tekshiramiz.
	raw, _ := json.Marshal(blocks[0])
	var img map[string]any
	_ = json.Unmarshal(raw, &img)
	if img["type"] != "image" {
		t.Errorf("birinchi blok turi 'image' bo'lishi kerak: %v", img["type"])
	}
	src, _ := img["source"].(map[string]any)
	if src["type"] != "base64" || src["media_type"] != "image/jpeg" || src["data"] != "QUJD" {
		t.Errorf("rasm source noto'g'ri: %v", src)
	}
}

// Manzil muhit o'zgaruvchisidan olinishi — korporativ shlyuz orqali
// ishlatish va testda soxta server uchun kerak.
func TestNewReadsEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_MODEL", "test-model")
	t.Setenv("ANTHROPIC_API_URL", "http://example.invalid/v1")

	c := New()
	if c.model != "test-model" || c.url != "http://example.invalid/v1" {
		t.Errorf("model=%q url=%q", c.model, c.url)
	}
	if !c.Available() {
		t.Error("kalit berilgan, Available() false qaytardi")
	}
}

// Sozlanmagan bo'lsa — rasmiy manzil va standart model.
func TestNewDefaults(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_MODEL", "")
	t.Setenv("ANTHROPIC_API_URL", "")

	c := New()
	if c.url != defaultAPIURL || c.model != defaultModel {
		t.Errorf("url=%q model=%q", c.url, c.model)
	}
	if c.Available() {
		t.Error("kalitsiz Available() true qaytardi")
	}
}

// Kalitsiz so'rov yuborilmasligi kerak — tarmoqqa chiqmasdan xato.
func TestCompleteWithoutKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, err := New().Complete(context.Background(), "s", nil); !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("ErrNoAPIKey kutilgan, oldik: %v", err)
	}
}

// Javobdagi bir nechta matn bloki birlashtirilishi kerak.
func TestCompleteJoinsTextBlocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Error("anthropic-version sarlavhasi yo'q")
		}
		_, _ = w.Write([]byte(`{"content":[
			{"type":"text","text":"birinchi "},
			{"type":"thinking","text":"E'TIBORSIZ"},
			{"type":"text","text":"ikkinchi"}]}`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	got, err := New().Complete(context.Background(), "tizim", []Message{{Role: "user", Content: "salom"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "birinchi ikkinchi" {
		t.Errorf("javob = %q; \"birinchi ikkinchi\" kutilgan", got)
	}
}

// API xatosi status koddan oldin tekshirilishi va matni saqlanishi kerak.
func TestCompleteAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"limit oshdi"}}`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	_, err := New().Complete(context.Background(), "s", []Message{{Role: "user", Content: "x"}})
	if err == nil || !strings.Contains(err.Error(), "limit oshdi") {
		t.Errorf("xato matni yo'qoldi: %v", err)
	}
}

// JSON bo'lmagan javob tushunarli xato berishi kerak.
func TestCompleteBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>502 Bad Gateway</html>`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	_, err := New().Complete(context.Background(), "s", []Message{{Role: "user", Content: "x"}})
	if err == nil || !strings.Contains(err.Error(), "javobni o'qishda xato") {
		t.Errorf("aniq xato kutilgan, oldik: %v", err)
	}
}
