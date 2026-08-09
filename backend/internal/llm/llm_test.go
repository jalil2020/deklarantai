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

// sseServer — Anthropic oqimini taqlid qiladigan soxta server.
func sseServer(t *testing.T, events ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["stream"] != true {
			t.Error("stream:true yuborilmadi")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range events {
			_, _ = w.Write([]byte(e))
		}
	}))
}

// Matn bo'laklari kelib tushishi kerak; boshqa hodisalar e'tiborsiz.
func TestStreamCollectsTextDeltas(t *testing.T) {
	srv := sseServer(t,
		"event: message_start\ndata: {\"type\":\"message_start\"}\n\n",
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Sa\"}}\n\n",
		"event: ping\ndata: {\"type\":\"ping\"}\n\n",
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"lom\"}}\n\n",
		"data: {\"type\":\"message_stop\"}\n\n",
	)
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	var got []string
	err := New().Stream(context.Background(), Full, "s", []Message{{Role: "user", Content: "x"}},
		func(chunk string) error { got = append(got, chunk); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, "") != "Salom" {
		t.Errorf("yig'ilgan matn = %q; \"Salom\" kutilgan", strings.Join(got, ""))
	}
	// Bo'lak-bo'lak kelishi muhim — bitta bo'lakda kelsa oqimning ma'nosi yo'q.
	if len(got) != 2 {
		t.Errorf("bo'laklar soni = %d; 2 kutilgan", len(got))
	}
}

// Oqim ICHIDA kelgan xato yo'qolmasligi kerak.
func TestStreamMidStreamError(t *testing.T) {
	srv := sseServer(t,
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"boshi\"}}\n\n",
		"event: error\ndata: {\"type\":\"error\",\"error\":{\"message\":\"ortiqcha yuklama\"}}\n\n",
	)
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	err := New().Stream(context.Background(), Full, "s", []Message{{Role: "user", Content: "x"}},
		func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "ortiqcha yuklama") {
		t.Errorf("oqim ichidagi xato yo'qoldi: %v", err)
	}
}

// Oqim BOSHLANMASDAN kelgan xato (HTTP status + JSON tana).
func TestStreamPreStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"kalit yaroqsiz"}}`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	err := New().Stream(context.Background(), Full, "s", []Message{{Role: "user", Content: "x"}},
		func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "kalit yaroqsiz") {
		t.Errorf("xato matni yo'qoldi: %v", err)
	}
}

// onChunk xato qaytarsa — oqim to'xtashi kerak (mijoz uzilib ketgan).
func TestStreamStopsOnCallbackError(t *testing.T) {
	srv := sseServer(t,
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"a\"}}\n\n",
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"b\"}}\n\n",
	)
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	n := 0
	err := New().Stream(context.Background(), Full, "s", []Message{{Role: "user", Content: "x"}},
		func(string) error { n++; return errors.New("mijoz ketdi") })
	if err == nil {
		t.Fatal("xato kutilgan edi")
	}
	if n != 1 {
		t.Errorf("callback %d marta chaqirildi; 1 dan keyin to'xtashi kerak edi", n)
	}
}

// Bo'sh javob sukut deb qabul qilinmasligi kerak.
func TestStreamEmptyResponse(t *testing.T) {
	srv := sseServer(t, "data: {\"type\":\"message_stop\"}\n\n")
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	if err := New().Stream(context.Background(), Full, "s", []Message{{Role: "user", Content: "x"}},
		func(string) error { return nil }); err == nil {
		t.Error("bo'sh javobda xato kutilgan edi")
	}
}

func TestStreamWithoutKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if err := New().Stream(context.Background(), Full, "s", nil, func(string) error { return nil }); !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("ErrNoAPIKey kutilgan: %v", err)
	}
}

// GLM sozlanmaganda arzon daraja Claude ning arzon modeliga qaytishi kerak.
func TestModelRouting(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_MODEL", "asosiy")
	t.Setenv("ANTHROPIC_FAST_MODEL", "arzon")
	t.Setenv("GLM_API_KEY", "") // GLM o'chiq
	c := New()

	if got := c.anthropicModel(Full); got != "asosiy" {
		t.Errorf("Full -> %q; \"asosiy\" kutilgan", got)
	}
	if got := c.anthropicModel(Cheap); got != "arzon" {
		t.Errorf("Cheap -> %q; \"arzon\" kutilgan", got)
	}
	if c.useGLM(Cheap) {
		t.Error("GLM kalitisiz ham GLM tanlandi")
	}
}

// GLM sozlanganda arzon daraja unga ketadi, asosiy daraja esa Claude da
// qoladi — bu yo'naltirish siyosatining o'zagi.
func TestGLMRouting(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_MODEL", "asosiy")
	t.Setenv("GLM_API_KEY", "g")
	t.Setenv("GLM_ENABLED", "1")
	t.Setenv("GLM_MODEL", "glm-test")
	c := New()

	if !c.useGLM(Cheap) {
		t.Error("Cheap GLM ga ketmadi")
	}
	if c.useGLM(Full) {
		t.Error("Full GLM ga ketdi — hisob-kitob arzon modelga tushib qoldi")
	}
	if got := c.modelName(Cheap); got != "glm-test" {
		t.Errorf("hisobotdagi model %q; \"glm-test\" kutilgan", got)
	}
	// Anthropic so'roviga GLM nomi tushmasligi kerak.
	if got := c.anthropicModel(Cheap); got == "glm-test" {
		t.Error("Anthropic so'roviga GLM model nomi qo'yildi")
	}
}

// RASM QAT'IY QOIDASI: GLM-5.2 rasmni qabul qilmaydi, shuning uchun
// rasmli xabar arzon daraja so'ralganda ham Full ga majburlanishi kerak.
// Bu llm qatlamida — yo'naltirishda xato bo'lsa ham rasm yo'qolmasin.
func TestImageForcesFullModel(t *testing.T) {
	withImage := []Message{{
		Role: "user", Content: "bu nima?",
		Images: []Image{{MediaType: "image/jpeg", Data: "AAAA"}},
	}}
	if got := pickModel(Cheap, withImage); got != Full {
		t.Error("rasmli xabar arzon modelga yuborildi — GLM uni ko'ra olmaydi")
	}
	textOnly := []Message{{Role: "user", Content: "salom"}}
	if got := pickModel(Cheap, textOnly); got != Cheap {
		t.Error("matnli xabar keraksiz ravishda Full ga ko'tarildi")
	}
}

// GLM ga rasm yetib borsa ham, u jimgina MATNGA aylanib qolmasligi kerak —
// aks holda foydalanuvchi "rasmni ko'rdim" degan taassurot olardi.
func TestGLMRejectsImages(t *testing.T) {
	g := &glmClient{apiKey: "k", model: "glm-test", maxTokens: 100}
	_, err := g.buildRequest("tizim", []Message{{
		Role: "user", Content: "bu nima?",
		Images: []Image{{MediaType: "image/jpeg", Data: "AAAA"}},
	}}, false)
	if err == nil {
		t.Error("GLM rasmli xabarni qabul qildi; xato kutilgan")
	}
}

// Tizim ko'rsatmasi keshlanishi; qisqa matn esa keshlanmasligi kerak.
func TestSystemPromptCaching(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	c := New()

	type sysBlk struct {
		Text         string `json:"text"`
		CacheControl *struct {
			Type string `json:"type"`
		} `json:"cache_control"`
	}
	parse := func(body []byte) []sysBlk {
		var req struct {
			System []sysBlk `json:"system"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		return req.System
	}

	long := strings.Repeat("a", minCacheable+1)
	body, _ := c.buildRequest(Full, long, nil, false)
	sys := parse(body)
	if len(sys) != 1 || sys[0].CacheControl == nil {
		t.Error("uzun ko'rsatma keshlanmadi")
	} else if sys[0].CacheControl.Type != "ephemeral" {
		t.Errorf("cache_control turi = %q", sys[0].CacheControl.Type)
	}

	// Qisqa matnni keshlashning ma'nosi yo'q — API ham rad etishi mumkin.
	body, _ = c.buildRequest(Full, "qisqa", nil, false)
	if sys := parse(body); len(sys) == 1 && sys[0].CacheControl != nil {
		t.Error("qisqa ko'rsatmaga ham kesh qo'yildi")
	}
}

// Javob uzunligi chegarasi sozlanishi.
func TestMaxTokensFromEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_MAX_TOKENS", "512")
	body, _ := New().buildRequest(Full, "s", nil, false)

	var req struct {
		MaxTokens int `json:"max_tokens"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	if req.MaxTokens != 512 {
		t.Errorf("max_tokens = %d; 512 kutilgan", req.MaxTokens)
	}

	// Noto'g'ri qiymat sukutga qaytishi kerak, 0 ga emas — 0 bo'lsa API
	// so'rovni rad etardi.
	t.Setenv("ANTHROPIC_MAX_TOKENS", "salom")
	body, _ = New().buildRequest(Full, "s", nil, false)
	_ = json.Unmarshal(body, &req)
	if req.MaxTokens != defaultMaxTokens {
		t.Errorf("noto'g'ri qiymatda max_tokens = %d; %d kutilgan", req.MaxTokens, defaultMaxTokens)
	}
}

// SUKUT HOLAT: GLM kaliti bo'lsa ham, ochiq ruxsatsiz ISHLATILMAYDI.
//
// Muhitda tasodifan qolib ketgan GLM_API_KEY jimgina provayderni
// almashtirib yuborardi — foydalanuvchi javob boshqa modeldan
// kelayotganini bilmasdi. Provayder almashinuvi OSHKORA bo'lishi kerak.
func TestGLMOffByDefault(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("GLM_API_KEY", "g") // kalit BOR
	t.Setenv("GLM_ENABLED", "")  // lekin ruxsat YO'Q
	c := New()

	if c.useGLM(Cheap) {
		t.Error("GLM_ENABLED siz ham GLM tanlandi")
	}
	if c.GLMAvailable() {
		t.Error("GLMAvailable() ruxsatsiz true qaytardi")
	}
	// Hamma so'rov Claude ga ketishi kerak.
	if got := c.modelName(Cheap); !strings.HasPrefix(got, "claude") {
		t.Errorf("arzon daraja modeli %q; claude kutilgan", got)
	}
	if got := c.modelName(Full); !strings.HasPrefix(got, "claude") {
		t.Errorf("asosiy model %q; claude kutilgan", got)
	}
}

// Kesh muddati. Sukut — 5 daqiqa (ttl yuborilmaydi); "1h" tanlanganda
// qiymat ham, beta sarlavhasi ham ketishi kerak — biri bo'lmasa API
// so'rovni rad etadi yoki jimgina 5 daqiqaga qaytaradi.
func TestCacheTTL(t *testing.T) {
	type sysBlk struct {
		CacheControl *struct {
			Type string `json:"type"`
			TTL  string `json:"ttl"`
		} `json:"cache_control"`
	}
	ttlOf := func(c *Client) string {
		body, _ := c.buildRequest(Full, strings.Repeat("a", minCacheable+1), nil, false)
		var req struct {
			System []sysBlk `json:"system"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if len(req.System) != 1 || req.System[0].CacheControl == nil {
			t.Fatal("cache_control yo'q")
		}
		return req.System[0].CacheControl.TTL
	}
	betaOf := func(c *Client) string {
		r, err := c.newHTTPRequest(context.Background(), []byte("{}"))
		if err != nil {
			t.Fatal(err)
		}
		return r.Header.Get("anthropic-beta")
	}

	t.Setenv("ANTHROPIC_API_KEY", "k")

	t.Run("sukut", func(t *testing.T) {
		c := New()
		if got := ttlOf(c); got != "" {
			t.Errorf("ttl = %q; bo'sh kutilgan", got)
		}
		if got := betaOf(c); got != "" {
			t.Errorf("keraksiz beta sarlavhasi: %q", got)
		}
	})

	t.Run("1h", func(t *testing.T) {
		t.Setenv("ANTHROPIC_CACHE_TTL", "1h")
		c := New()
		if got := ttlOf(c); got != "1h" {
			t.Errorf("ttl = %q; \"1h\" kutilgan", got)
		}
		if got := betaOf(c); got != extendedCacheBeta {
			t.Errorf("beta sarlavhasi = %q; %q kutilgan", got, extendedCacheBeta)
		}
	})

	// Noma'lum qiymat API xatosiga olib kelardi — sukutga qaytamiz.
	t.Run("notogri qiymat", func(t *testing.T) {
		t.Setenv("ANTHROPIC_CACHE_TTL", "2h")
		c := New()
		if got := ttlOf(c); got != "" {
			t.Errorf("ttl = %q; noma'lum qiymat e'tiborsiz qolishi kerak", got)
		}
	})
}

// Uzunlik chegarasida kesilgan javob OCHIQ belgilanishi kerak.
//
// NEGA TEST: chegaraga urilgan javob jimgina kesiladi — API xato
// bermaydi, matn gap o'rtasida tugaydi. Jonli sinovda (Android,
// 2026-08-09) boj hisobi aynan shunday kesildi va oxirida turgan
// "kelib chiqish sertifikati bo'lmasa boj IKKI BAROBAR" ogohlantirishi
// yo'qoldi. Foydalanuvchi javobni to'liq deb o'qib, kam to'lov
// hisoblab qolardi.
func TestCompleteMarksTruncation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"stop_reason":"max_tokens",
			"content":[{"type":"text","text":"boj 5 957 820 so'm, kelib chiqish"}]}`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	got, err := New().Complete(context.Background(), "s", []Message{{Role: "user", Content: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "TO'LIQ EMAS") {
		t.Errorf("kesilgan javob belgilanmadi:\n%s", got)
	}
	// Kelgan matn YO'QOLMASLIGI kerak.
	if !strings.Contains(got, "5 957 820") {
		t.Error("javob matni yo'qoldi")
	}
}

// Oddiy tugagan javobga ogohlantirish QO'SHILMASLIGI kerak.
func TestCompleteNoNoteWhenComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"stop_reason":"end_turn",
			"content":[{"type":"text","text":"tugadi"}]}`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	got, _ := New().Complete(context.Background(), "s", []Message{{Role: "user", Content: "x"}})
	if strings.Contains(got, "TO'LIQ EMAS") {
		t.Errorf("to'liq javobga ortiqcha ogohlantirish qo'shildi:\n%s", got)
	}
}

// Oqimda tugash sababi message_delta hodisasida keladi.
func TestStreamMarksTruncation(t *testing.T) {
	const nl = "\n\n"
	srv := sseServer(t,
		"data: "+`{"type":"content_block_delta","delta":{"type":"text_delta","text":"boj hisobi"}}`+nl,
		"data: "+`{"type":"message_delta","delta":{"stop_reason":"max_tokens"},"usage":{"output_tokens":2048}}`+nl,
	)
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	var b strings.Builder
	err := New().Stream(context.Background(), Full, "s",
		[]Message{{Role: "user", Content: "x"}},
		func(chunk string) error { b.WriteString(chunk); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "TO'LIQ EMAS") {
		t.Errorf("oqimda kesilish belgilanmadi:\n%s", b.String())
	}
}

// Sukut holatda HAMMA so'rov asosiy modelga ketishi kerak.
//
// NEGA TEST: bir marta ma'lumot savollari jimgina Sonnet ga o'tib
// ketgan va foydalanuvchi javob sifati pasayganini sezgan. Endi
// arzon darajalar ochiq tanlov: muhitda yoqilmagan bo'lsa, uchala
// daraja ham ANTHROPIC_MODEL ni qaytaradi.
func TestAllTiersUseMainModelByDefault(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_MODEL", "asosiy")
	t.Setenv("ANTHROPIC_MID_MODEL", "")
	t.Setenv("ANTHROPIC_FAST_MODEL", "")

	c := New()
	for _, m := range []Model{Full, Mid, Cheap} {
		if got := c.anthropicModel(m); got != "asosiy" {
			t.Errorf("%v darajasi %q modelini tanladi; \"asosiy\" kutilgan", m, got)
		}
	}
	if got := c.Tiers(); got != "hamma so'rov: asosiy" {
		t.Errorf("Tiers() = %q", got)
	}
}

// Yoqilgan daraja jurnalda OCHIQ ko'rinishi kerak.
func TestTiersReportsOverrides(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_MODEL", "asosiy")
	t.Setenv("ANTHROPIC_MID_MODEL", "orta")
	t.Setenv("ANTHROPIC_FAST_MODEL", "arzon")

	got := New().Tiers()
	for _, want := range []string{"asosiy", "orta", "arzon", "⚠️"} {
		if !strings.Contains(got, want) {
			t.Errorf("Tiers() = %q; %q kutilgan", got, want)
		}
	}
}

// GLM yoqilgan bo'lsa, Tiers() buni AYTISHI shart.
//
// NEGA TEST: GLM tekshiruvi anthropicModel dan OLDIN turadi, ya'ni
// ANTHROPIC_FAST_MODEL bo'sh bo'lsa ham arzon darajani o'ziga oladi.
// Auditda topilgan kamchilik: shu holatda satr "hamma so'rov: opus"
// deb yozardi — adashtirish uchun qo'yilgan satrning o'zi adashtirardi.
func TestTiersReportsGLM(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_MODEL", "asosiy")
	t.Setenv("ANTHROPIC_MID_MODEL", "")
	t.Setenv("ANTHROPIC_FAST_MODEL", "")
	t.Setenv("GLM_ENABLED", "1")
	t.Setenv("GLM_API_KEY", "glm-kalit")
	t.Setenv("GLM_MODEL", "glm-test")

	got := New().Tiers()
	if !strings.Contains(got, "glm-test") || !strings.Contains(got, "⚠️") {
		t.Errorf("Tiers() = %q; GLM ko'rsatilishi kerak edi", got)
	}
	if strings.Contains(got, "hamma so'rov") {
		t.Errorf("Tiers() = %q; GLM yoqiq holatda \"hamma so'rov\" deyish noto'g'ri", got)
	}
}

// GLM o'chiq bo'lsa (kalit bor, lekin GLM_ENABLED yo'q) — satr toza.
func TestTiersIgnoresDisabledGLM(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_MODEL", "asosiy")
	t.Setenv("ANTHROPIC_MID_MODEL", "")
	t.Setenv("ANTHROPIC_FAST_MODEL", "")
	t.Setenv("GLM_ENABLED", "")
	t.Setenv("GLM_API_KEY", "glm-kalit")

	if got := New().Tiers(); got != "hamma so'rov: asosiy" {
		t.Errorf("Tiers() = %q; GLM o'chiq, ta'sir qilmasligi kerak", got)
	}
}
