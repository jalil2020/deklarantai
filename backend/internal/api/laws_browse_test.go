package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deklarant-ai/backend/internal/chat"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
	"deklarant-ai/backend/internal/llm"
)

// lawsServer — qonun korpusi ulangan server.
func lawsServer(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("ANTHROPIC_API_KEY", "")
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Fatal(err)
	}
	lawStore, err := laws.Load("../../data/laws.json")
	if err != nil {
		t.Fatal(err)
	}
	client := llm.New()
	return New(codes, lawStore, chat.New(client, codes, lawStore, nil, nil), client, nil, nil, nil).Routes()
}

func getLaws(t *testing.T, h http.Handler, query string) lawsBrowseResponse {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/laws/browse"+query, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("laws/browse%s: status %d, %s", query, w.Code, w.Body.String())
	}
	var out lawsBrowseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("javobni o'qib bo'lmadi: %v", err)
	}
	return out
}

// Uch daraja ham ishlashi kerak: hujjatlar → moddalar → to'liq matn.
func TestLawsBrowseDrillsDown(t *testing.T) {
	h := lawsServer(t)

	docs := getLaws(t, h, "")
	if docs.Level != "docs" || len(docs.Docs) == 0 {
		t.Fatalf("hujjatlar bo'sh (daraja=%q)", docs.Level)
	}
	// Eng yirik hujjat birinchi bo'lishi kerak — deklarantga avvalo
	// kodekslar kerak, alifbo tartibi emas.
	first := docs.Docs[0]
	if first.Chunks < docs.Docs[len(docs.Docs)-1].Chunks {
		t.Error("hujjatlar parcha soni bo'yicha kamayish tartibida emas")
	}
	if first.Name == "" || first.Chunks == 0 {
		t.Errorf("birinchi hujjat to'liq emas: %+v", first)
	}

	arts := getLaws(t, h, "?doc="+itoa(first.ID))
	if arts.Level != "articles" || len(arts.Articles) != first.Chunks {
		t.Fatalf("moddalar soni %d; %d kutilgan", len(arts.Articles), first.Chunks)
	}
	// Ro'yxatda TO'LIQ matn kelmasligi kerak — faqat boshi.
	for _, a := range arts.Articles {
		if len([]rune(a.Preview)) > 200 {
			t.Errorf("modda %d: ro'yxatda matn juda uzun (%d belgi) — to'liq matn yuborilyaptimi?",
				a.Index, len([]rune(a.Preview)))
			break
		}
	}

	one := getLaws(t, h, "?doc="+itoa(first.ID)+"&i=0")
	if one.Level != "article" || one.Article == nil {
		t.Fatal("to'liq modda kelmadi")
	}
	if one.Article.Text == "" {
		t.Error("modda matni bo'sh")
	}
}

// Mavjud bo'lmagan hujjat va modda 404 qaytarishi kerak.
func TestLawsBrowseNotFound(t *testing.T) {
	h := lawsServer(t)
	for _, q := range []string{"?doc=999999999", "?doc=999999999&i=0"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/laws/browse"+q, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: status %d; 404 kutilgan", q, w.Code)
		}
	}
}

// Korpus yuklanmagan bo'lsa — aniq xato, bo'sh ro'yxat emas.
//
// Bo'sh ro'yxat "qonun yo'q" degan taassurot berardi; 503 esa
// "xizmat sozlanmagan" deb aniq aytadi.
func TestLawsBrowseWithoutCorpus(t *testing.T) {
	h := bareServer(t).Routes() // laws = nil
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/laws/browse", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status %d; 503 kutilgan", w.Code)
	}
	if !strings.Contains(w.Body.String(), "korpus") {
		t.Errorf("xato matni tushunarsiz: %s", w.Body.String())
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
