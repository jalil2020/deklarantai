package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// raw — do() dan farqli: mijoz belgisini O'ZIMIZ boshqaramiz.
func raw(t *testing.T, h http.Handler, method, path, body, key string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		r.Header.Set(clientHeader, key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

const chatBody = `{"messages":[{"role":"user","content":"salom"}]}`

// Eng muhim tekshiruv: belgisiz chat endpointi PUL SARFLAMASLIGI kerak.
func TestChatRequiresClient(t *testing.T) {
	h := newServer(t, "sk-test", "")

	for _, path := range []string{"/api/chat", "/api/chat/stream"} {
		w := raw(t, h, http.MethodPost, path, chatBody, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s belgisiz: status %d; 401 kutilgan", path, w.Code)
		}
		// Xato matni mijozga NIMA QILISHNI aytishi kerak.
		if !strings.Contains(w.Body.String(), "/api/session") {
			t.Errorf("%s: xato matni yo'lni ko'rsatmaydi: %s", path, w.Body.String())
		}
	}
}

func TestChatRejectsWrongKey(t *testing.T) {
	h := newServer(t, "sk-test", "")
	w := raw(t, h, http.MethodPost, "/api/chat", chatBody, "notoshri-kalit")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("noto'g'ri kalit: status %d; 401 kutilgan", w.Code)
	}
}

// Anonim token — brauzer uchun yo'l. Olingandan keyin chat ochilishi kerak.
func TestSessionTokenOpensChat(t *testing.T) {
	h := newServer(t, "", "") // AI kalitisiz: 503 gacha yetsa, belgi ishladi

	w := raw(t, h, http.MethodPost, "/api/session", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("/api/session: status %d; 200 kutilgan", w.Code)
	}
	var out struct {
		Token  string `json:"token"`
		Header string `json:"header"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Token == "" {
		t.Fatal("token bo'sh")
	}
	if out.Header != clientHeader {
		t.Errorf("sarlavha nomi %q; %q kutilgan", out.Header, clientHeader)
	}

	// 401 EMAS — ya'ni token tan olindi. 503 chiqadi, chunki
	// ANTHROPIC_API_KEY yo'q; bu boshqa bosqich.
	got := raw(t, h, http.MethodPost, "/api/chat", chatBody, out.Token)
	if got.Code == http.StatusUnauthorized {
		t.Errorf("token bilan ham 401: %s", got.Body.String())
	}
}

// Qidiruv, boj va qonunlar LLM ga bormaydi — ular ochiq qolishi kerak,
// aks holda bepul qism ham yopilib qolardi.
func TestFreeEndpointsStayOpen(t *testing.T) {
	h := newServer(t, "", "")

	cases := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/api/health", ""},
		{http.MethodPost, "/api/hscode/search", `{"query":"noutbuk"}`},
		{http.MethodPost, "/api/duty/calculate", `{"customs_value":1000000,"import_duty":10,"vat":12}`},
		{http.MethodGet, "/api/hscode/browse", ""},
		{http.MethodGet, "/api/laws/browse", ""},
	}
	for _, c := range cases {
		w := raw(t, h, c.method, c.path, c.body, "")
		if w.Code == http.StatusUnauthorized {
			t.Errorf("%s belgisiz 401 qaytardi — bu endpoint ochiq bo'lishi kerak", c.path)
		}
	}
}

// ---- token mexanikasi ----

func newTestAuth(t *testing.T) *clientAuth {
	t.Helper()
	t.Setenv("API_KEYS", "mobil:mobil-siri, hamkor:hamkor-siri")
	t.Setenv("CLIENT_TOKEN_SECRET", "sinov-siri")
	return newClientAuth()
}

func TestIdentifyAPIKeys(t *testing.T) {
	a := newTestAuth(t)

	// Ro'yxatdagi kalit o'z nomi bilan tanilishi kerak — statistika
	// keyinchalik shu nom bo'yicha ajratiladi.
	for key, want := range map[string]string{
		"mobil-siri":  "mobil",
		"hamkor-siri": "hamkor",
	} {
		got, ok := a.identify(key)
		if !ok || got != want {
			t.Errorf("kalit %q → (%q, %v); (%q, true) kutilgan", key, got, ok, want)
		}
	}

	for _, bad := range []string{"", "mobil", "mobil-sir", "mobil-sirii", "hamkor:hamkor-siri"} {
		if _, ok := a.identify(bad); ok {
			t.Errorf("%q tan olindi — olinmasligi kerak edi", bad)
		}
	}
}

func TestTokenLifecycle(t *testing.T) {
	a := newTestAuth(t)
	tok, exp := a.issue()

	name, ok := a.identify(tok)
	if !ok || name != anonName {
		t.Fatalf("yangi token → (%q, %v); (%q, true) kutilgan", name, ok, anonName)
	}

	// Muddati o'tgach tan olinmasligi kerak.
	a.now = func() time.Time { return exp.Add(time.Second) }
	if _, ok := a.identify(tok); ok {
		t.Error("muddati o'tgan token tan olindi")
	}
}

// Imzo — tokenning butun ma'nosi. Buzilgan token o'tib ketsa, kim
// xohlasa o'zi token yozib olardi.
func TestTokenRejectsTampering(t *testing.T) {
	a := newTestAuth(t)
	tok, _ := a.issue()
	i := strings.LastIndexByte(tok, '.')
	payload, sig := tok[:i], tok[i+1:]

	bad := []string{
		"v1.99999999999." + sig,         // muddat cho'zilgan, imzo eski
		payload + ".",                   // imzosiz
		payload + "." + sig + "x",       // imzo o'zgartirilgan
		"v2." + payload[3:] + "." + sig, // versiya boshqa
		payload,                         // umuman imzo yo'q
		"",
	}
	for _, tok := range bad {
		if a.validToken(tok) {
			t.Errorf("buzilgan token qabul qilindi: %q", tok)
		}
	}

	// Boshqa sir bilan imzolangan token ham o'tmasligi kerak — bir necha
	// nusxa ishlaganda sir bir xil bo'lishi shartligi shundan.
	t.Setenv("CLIENT_TOKEN_SECRET", "boshqa-sir")
	other := newClientAuth()
	if other.validToken(tok) {
		t.Error("boshqa sir bilan yasalgan token qabul qilindi")
	}
}

func TestParseAPIKeys(t *testing.T) {
	got := parseAPIKeys(" mobil:siri , ,buzuq, :sir, nom:, web:ikkinchi ")
	want := []apiKey{{"mobil", "siri"}, {"web", "ikkinchi"}}

	if len(got) != len(want) {
		t.Fatalf("kalitlar soni %d; %d kutilgan: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("kalit %d: %+v; %+v kutilgan", i, got[i], want[i])
		}
	}
}
