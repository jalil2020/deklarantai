package llm

import (
	"encoding/json"
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
