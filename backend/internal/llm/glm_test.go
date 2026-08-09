package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// glmServer — soxta OpenAI-mos server. So'rov tanasini testga qaytaradi.
func glmServer(t *testing.T, handler func(w http.ResponseWriter, body map[string]any)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer g" {
			t.Errorf("Authorization = %q; \"Bearer g\" kutilgan", got)
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		handler(w, body)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("GLM_API_KEY", "g")
	t.Setenv("GLM_ENABLED", "1")
	t.Setenv("GLM_MODEL", "glm-5.2")
	t.Setenv("GLM_API_URL", srv.URL)
	return New()
}

// Tizim ko'rsatmasi OpenAI formatida BIRINCHI xabar bo'lishi kerak —
// Anthropic dagi alohida "system" maydoni bu yerda yo'q.
func TestGLMSystemBecomesFirstMessage(t *testing.T) {
	var seen map[string]any
	c := glmServer(t, func(w http.ResponseWriter, body map[string]any) {
		seen = body
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"javob"}}],
			"usage":{"prompt_tokens":100,"completion_tokens":20,
			         "prompt_tokens_details":{"cached_tokens":40}}}`))
	})

	got, err := c.CompleteWith(context.Background(), Cheap, "TIZIM KO'RSATMASI",
		[]Message{{Role: "user", Content: "savol"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "javob" {
		t.Errorf("javob = %q", got)
	}
	if seen["model"] != "glm-5.2" {
		t.Errorf("model = %v; glm-5.2 kutilgan", seen["model"])
	}
	msgs, _ := seen["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("xabarlar soni %d; 2 kutilgan (system + user)", len(msgs))
	}
	first, _ := msgs[0].(map[string]any)
	if first["role"] != "system" || first["content"] != "TIZIM KO'RSATMASI" {
		t.Errorf("birinchi xabar system emas: %v", first)
	}
}

// Sarf hisoboti Anthropic ma'nosiga keltirilishi kerak: OpenAI da
// prompt_tokens kesh ICHIDA, bizda esa kirish keshdan tashqari qism.
func TestGLMUsageNormalized(t *testing.T) {
	var got Usage
	OnUsage = func(u Usage) { got = u }
	defer func() { OnUsage = nil }()

	c := glmServer(t, func(w http.ResponseWriter, _ map[string]any) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"x"}}],
			"usage":{"prompt_tokens":100,"completion_tokens":20,
			         "prompt_tokens_details":{"cached_tokens":40}}}`))
	})
	if _, err := c.CompleteWith(context.Background(), Cheap, "s",
		[]Message{{Role: "user", Content: "q"}}); err != nil {
		t.Fatal(err)
	}

	if got.InputTokens != 60 {
		t.Errorf("kirish = %d; 60 kutilgan (100 − 40 kesh)", got.InputTokens)
	}
	if got.CacheRead != 40 {
		t.Errorf("kesh o'qildi = %d; 40 kutilgan", got.CacheRead)
	}
	if got.OutputTokens != 20 {
		t.Errorf("chiqish = %d; 20 kutilgan", got.OutputTokens)
	}
	if got.Model != "glm-5.2" {
		t.Errorf("model = %q; glm-5.2 kutilgan", got.Model)
	}
}

// Oqim: matn bo'laklari yig'iladi, ichki mulohaza (reasoning_content)
// foydalanuvchiga KO'RSATILMAYDI.
func TestGLMStreamSkipsReasoning(t *testing.T) {
	c := glmServer(t, func(w http.ResponseWriter, body map[string]any) {
		// stream_options bo'lmasa sarf umuman kelmaydi.
		if body["stream_options"] == nil {
			t.Error("stream_options yo'q — token sarfi ko'rinmay qoladi")
		}
		for _, line := range []string{
			`data: {"choices":[{"delta":{"reasoning_content":"ICHKI MULOHAZA"}}]}`,
			`data: {"choices":[{"delta":{"content":"Boj "}}]}`,
			`data: {"choices":[{"delta":{"content":"30%"}}]}`,
			`data: {"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
			`data: [DONE]`,
		} {
			_, _ = w.Write([]byte(line + "\n"))
		}
	})

	var sb strings.Builder
	err := c.Stream(context.Background(), Cheap, "s",
		[]Message{{Role: "user", Content: "q"}},
		func(chunk string) error { sb.WriteString(chunk); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if sb.String() != "Boj 30%" {
		t.Errorf("javob = %q; \"Boj 30%%\" kutilgan", sb.String())
	}
	if strings.Contains(sb.String(), "ICHKI") {
		t.Error("ichki mulohaza foydalanuvchiga chiqib ketdi")
	}
}

// GLM xatosi yashirilmasligi kerak.
func TestGLMErrorSurfaces(t *testing.T) {
	c := glmServer(t, func(w http.ResponseWriter, _ map[string]any) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"limit oshdi"}}`))
	})
	_, err := c.CompleteWith(context.Background(), Cheap, "s",
		[]Message{{Role: "user", Content: "q"}})
	if err == nil || !strings.Contains(err.Error(), "limit oshdi") {
		t.Errorf("xato matni yo'qoldi: %v", err)
	}
}

// RASM: arzon daraja so'ralsa ham, rasmli xabar GLM ga TUSHMASLIGI kerak.
// Soxta GLM serveri chaqirilsa — test yiqiladi.
func TestImageNeverReachesGLM(t *testing.T) {
	called := false
	c := glmServer(t, func(w http.ResponseWriter, _ map[string]any) {
		called = true
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"x"}}]}`))
	})
	// Anthropic tomoni ham soxta bo'lsin.
	anth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"Claude javobi"}]}`))
	}))
	defer anth.Close()
	c.url = anth.URL

	got, err := c.CompleteWith(context.Background(), Cheap, "s", []Message{{
		Role: "user", Content: "bu nima?",
		Images: []Image{{MediaType: "image/jpeg", Data: "AAAA"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("rasmli xabar GLM ga yuborildi — u rasmni ko'ra olmaydi")
	}
	if got != "Claude javobi" {
		t.Errorf("javob = %q; Claude dan kutilgan", got)
	}
}
