package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deklarant-ai/backend/internal/chat"
	"deklarant-ai/backend/internal/docs"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
	"deklarant-ai/backend/internal/llm"
)

// newServer — haqiqiy bazalar bilan server yig'adi.
// aiURL bo'sh bo'lmasa, LLM so'rovlari o'sha manzilga (soxta serverga) ketadi.
func newServer(t *testing.T, apiKey, aiURL string) http.Handler {
	t.Helper()
	t.Setenv("ANTHROPIC_API_KEY", apiKey)
	t.Setenv("ANTHROPIC_API_URL", aiURL)

	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Fatal(err)
	}
	lawStore, err := laws.Load("../../data/laws.json")
	if err != nil {
		t.Fatal(err)
	}
	docStore, err := docs.Load("../../data/docs.json")
	if err != nil {
		t.Fatal(err)
	}
	client := llm.New()
	return New(codes, lawStore, chat.New(client, codes, lawStore, docStore), client).Routes()
}

// do — so'rov yuboradi va javobni qaytaradi.
func do(t *testing.T, h http.Handler, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	var out map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &out)
	}
	return w, out
}

func wantStatus(t *testing.T, w *httptest.ResponseRecorder, want int, what string) {
	t.Helper()
	if w.Code != want {
		t.Errorf("%s: status %d; %d kutilgan. Javob: %s", what, w.Code, want, w.Body.String())
	}
}

// ------------------------------------------------------------------ health

func TestHealth(t *testing.T) {
	h := newServer(t, "", "")
	w, out := do(t, h, http.MethodGet, "/api/health", "")
	wantStatus(t, w, http.StatusOK, "health")

	if out["status"] != "ok" {
		t.Errorf("status = %v; \"ok\" kutilgan", out["status"])
	}
	// Kalitsiz ishga tushirilgan — AI o'chiq bo'lishi kerak.
	if out["ai_available"] != false {
		t.Errorf("ai_available = %v; false kutilgan", out["ai_available"])
	}
	if n, _ := out["codes"].(float64); n < 13000 {
		t.Errorf("codes = %v; 13000 dan ko'p kutilgan", out["codes"])
	}
	// Baza kelib chiqishi ko'rsatilishi kerak — foydalanuvchi "qachongi
	// ma'lumot" deb so'raganda shu javob beradi.
	base, ok := out["base"].(map[string]any)
	if !ok || base["rates_as_of"] == nil {
		t.Error("base.rates_as_of yo'q")
	}
	if out["laws"] == nil {
		t.Error("laws meta'si yo'q")
	}
}

func TestHealthWithAIKey(t *testing.T) {
	_, out := do(t, newServer(t, "kalit", ""), http.MethodGet, "/api/health", "")
	if out["ai_available"] != true {
		t.Errorf("kalit bor, lekin ai_available = %v", out["ai_available"])
	}
}

// ------------------------------------------------------------------ hscode

func TestHSSearch(t *testing.T) {
	w, out := do(t, newServer(t, "", ""), http.MethodPost, "/api/hscode/search",
		`{"query":"traktor"}`)
	wantStatus(t, w, http.StatusOK, "hscode qidiruv")

	if out["source"] != "keyword" {
		t.Errorf("source = %v; \"keyword\" kutilgan", out["source"])
	}
	matches, _ := out["matches"].([]any)
	if len(matches) == 0 {
		t.Fatal("natija bo'sh")
	}
	first, _ := matches[0].(map[string]any)
	code, _ := first["code"].(map[string]any)
	if c, _ := code["code"].(string); !strings.HasPrefix(c, "8701") {
		t.Errorf("birinchi kod %v; 8701… kutilgan", code["code"])
	}
}

// AI o'chiq bo'lsa, use_ai so'ralsa ham qidiruv ishlashi kerak.
func TestHSSearchUseAIWithoutKey(t *testing.T) {
	w, out := do(t, newServer(t, "", ""), http.MethodPost, "/api/hscode/search",
		`{"query":"traktor","use_ai":true}`)
	wantStatus(t, w, http.StatusOK, "AI siz qidiruv")
	if out["source"] != "keyword" {
		t.Errorf("source = %v; AI yo'q, \"keyword\" kutilgan", out["source"])
	}
	if out["ai_comment"] != nil {
		t.Error("AI o'chiq, lekin izoh qaytdi")
	}
}

// AI izohi qo'shilishi.
func TestHSSearchWithAIComment(t *testing.T) {
	srv := fakeAI(t, `{"content":[{"type":"text","text":"8701 mos keladi"}]}`, http.StatusOK)
	defer srv.Close()

	w, out := do(t, newServer(t, "kalit", srv.URL), http.MethodPost, "/api/hscode/search",
		`{"query":"traktor","use_ai":true}`)
	wantStatus(t, w, http.StatusOK, "AI izohli qidiruv")

	if out["source"] != "ai" {
		t.Errorf("source = %v; \"ai\" kutilgan", out["source"])
	}
	if out["ai_comment"] != "8701 mos keladi" {
		t.Errorf("ai_comment = %v", out["ai_comment"])
	}
}

// AI xato bersa, qidiruv baribir natija qaytarishi kerak — izohsiz.
// Bu muhim: AI nosozligi butun qidiruvni yiqitmasligi kerak.
func TestHSSearchAIFailureIsNotFatal(t *testing.T) {
	srv := fakeAI(t, `{"error":{"message":"xato"}}`, http.StatusInternalServerError)
	defer srv.Close()

	w, out := do(t, newServer(t, "kalit", srv.URL), http.MethodPost, "/api/hscode/search",
		`{"query":"traktor","use_ai":true}`)
	wantStatus(t, w, http.StatusOK, "AI xato bergan qidiruv")

	if matches, _ := out["matches"].([]any); len(matches) == 0 {
		t.Error("AI xato berdi va natija ham yo'qoldi")
	}
	if out["source"] != "keyword" {
		t.Errorf("source = %v; AI ishlamadi, \"keyword\" kutilgan", out["source"])
	}
}

func TestHSSearchBadInput(t *testing.T) {
	h := newServer(t, "", "")
	cases := []struct {
		name, body string
		want       int
	}{
		{"bo'sh so'rov", `{"query":"   "}`, http.StatusBadRequest},
		{"buzuq JSON", `{"query":`, http.StatusBadRequest},
		{"noma'lum maydon", `{"query":"traktor","xyz":1}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		w, out := do(t, h, http.MethodPost, "/api/hscode/search", c.body)
		wantStatus(t, w, c.want, c.name)
		if out["error"] == nil {
			t.Errorf("%s: xato matni yo'q", c.name)
		}
	}
}

// ------------------------------------------------------------------ duty

// manba bilan solishtirilgan holat HTTP orqali ham to'g'ri chiqishi kerak.
func TestDutyCalculate(t *testing.T) {
	w, out := do(t, newServer(t, "", ""), http.MethodPost, "/api/duty/calculate",
		`{"date":"2026-07-19T00:00:00Z","invoice":1230000,"transport":25000,
		  "currency_rate":12093.35,"usd_rate":12093.35,"import_duty":0,"vat":12}`)
	wantStatus(t, w, http.StatusOK, "boj hisoblash")

	if cv, _ := out["customs_value"].(float64); cv != 15_177_154_250 {
		t.Errorf("bojxona qiymati = %v; 15 177 154 250 kutilgan", out["customs_value"])
	}
	if total, _ := out["total"].(float64); total != 1_831_558_510 {
		t.Errorf("jami = %v; 1 831 558 510 kutilgan", out["total"])
	}
}

func TestDutyNegativeValue(t *testing.T) {
	w, out := do(t, newServer(t, "", ""), http.MethodPost, "/api/duty/calculate",
		`{"customs_value":-100}`)
	wantStatus(t, w, http.StatusBadRequest, "manfiy qiymat")
	if out["error"] == nil {
		t.Error("xato matni yo'q")
	}
}

// ------------------------------------------------------------------ chat

// Kalit yo'q bo'lsa, aniq 503 va tushunarli xabar qaytishi kerak.
func TestChatWithoutAPIKey(t *testing.T) {
	w, out := do(t, newServer(t, "", ""), http.MethodPost, "/api/chat",
		`{"messages":[{"role":"user","content":"salom"}]}`)
	wantStatus(t, w, http.StatusServiceUnavailable, "kalitsiz chat")

	msg, _ := out["error"].(string)
	if !strings.Contains(msg, "ANTHROPIC_API_KEY") {
		t.Errorf("xato xabari kalit haqida aytmaydi: %q", msg)
	}
}

func TestChatSuccess(t *testing.T) {
	srv := fakeAI(t, `{"content":[{"type":"text","text":"Salom! Yordam beraman."}]}`, http.StatusOK)
	defer srv.Close()

	w, out := do(t, newServer(t, "kalit", srv.URL), http.MethodPost, "/api/chat",
		`{"messages":[{"role":"user","content":"traktor bojini hisobla"}]}`)
	wantStatus(t, w, http.StatusOK, "chat")

	if out["reply"] != "Salom! Yordam beraman." {
		t.Errorf("reply = %v", out["reply"])
	}
}

// AI nosozligi 502 bo'lib qaytishi va sabab ko'rinishi kerak.
func TestChatAIError(t *testing.T) {
	srv := fakeAI(t, `{"error":{"message":"model band"}}`, http.StatusInternalServerError)
	defer srv.Close()

	w, out := do(t, newServer(t, "kalit", srv.URL), http.MethodPost, "/api/chat",
		`{"messages":[{"role":"user","content":"salom"}]}`)
	wantStatus(t, w, http.StatusBadGateway, "AI xatosi")

	if msg, _ := out["error"].(string); !strings.Contains(msg, "model band") {
		t.Errorf("xato sababi yo'qoldi: %q", msg)
	}
}

func TestChatBadInput(t *testing.T) {
	h := newServer(t, "kalit", "")
	for name, body := range map[string]string{
		"xabarsiz":   `{"messages":[]}`,
		"buzuq JSON": `{"messages":`,
	} {
		w, out := do(t, h, http.MethodPost, "/api/chat", body)
		wantStatus(t, w, http.StatusBadRequest, name)
		if out["error"] == nil {
			t.Errorf("%s: xato matni yo'q", name)
		}
	}
}

// Rasmli xabar API ga yetib borishi kerak (vision yo'li).
func TestChatWithImage(t *testing.T) {
	var gotImage bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content any `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) > 0 {
			if blocks, ok := req.Messages[0].Content.([]any); ok {
				for _, b := range blocks {
					if m, ok := b.(map[string]any); ok && m["type"] == "image" {
						gotImage = true
					}
				}
			}
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"rasmni ko'rdim"}]}`))
	}))
	defer srv.Close()

	w, _ := do(t, newServer(t, "kalit", srv.URL), http.MethodPost, "/api/chat",
		`{"messages":[{"role":"user","content":"bu nima?",
		  "images":[{"media_type":"image/jpeg","data":"AAAA"}]}]}`)
	wantStatus(t, w, http.StatusOK, "rasmli chat")

	if !gotImage {
		t.Error("rasm API so'roviga qo'shilmadi")
	}
}

// ------------------------------------------------------------------ CORS va yo'llar

func TestCORSHeaders(t *testing.T) {
	w, _ := do(t, newServer(t, "", ""), http.MethodGet, "/api/health", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS sarlavhasi = %q", got)
	}
}

// Brauzer preflight so'rovi 204 bilan javob olishi kerak.
func TestCORSPreflight(t *testing.T) {
	w, _ := do(t, newServer(t, "", ""), http.MethodOptions, "/api/chat", "")
	wantStatus(t, w, http.StatusNoContent, "preflight")
	if got := w.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Errorf("ruxsat etilgan metodlar = %q", got)
	}
}

func TestRouting(t *testing.T) {
	h := newServer(t, "", "")
	cases := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/yoq", http.StatusNotFound},
		{http.MethodPost, "/api/health", http.StatusMethodNotAllowed}, // health faqat GET
		{http.MethodGet, "/api/chat", http.StatusMethodNotAllowed},    // chat faqat POST
	}
	for _, c := range cases {
		w, _ := do(t, h, c.method, c.path, "")
		wantStatus(t, w, c.want, c.method+" "+c.path)
	}
}

// ------------------------------------------------------------------ yordamchi

func fakeAI(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}
