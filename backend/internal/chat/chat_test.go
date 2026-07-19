package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deklarant-ai/backend/internal/docs"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
	"deklarant-ai/backend/internal/llm"
)

// Haqiqiy bazalar bilan sinaymiz — retrieval sifati shu bilan o'lchanadi.
func newService(t *testing.T) *Service {
	t.Helper()
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
	return New(llm.New(), codes, lawStore, docStore)
}

func userMsg(text string) []llm.Message {
	return []llm.Message{{Role: "user", Content: text}}
}

// -------------------------------------------------------------- retrieval

// Oxirgi savolga baza bloklari qo'shilishi kerak.
func TestWithRetrievalAppendsBlocks(t *testing.T) {
	got := newService(t).withRetrieval(userMsg("traktor import qilsam qancha boj"))
	if len(got) != 1 {
		t.Fatalf("xabarlar soni %d; 1 kutilgan", len(got))
	}
	body := got[0].Content
	for _, want := range []string{"<TIF_TN_BAZASIDAN>", "8701"} {
		if !strings.Contains(body, want) {
			t.Errorf("kontekstda %q yo'q", want)
		}
	}
}

// Asl tarix O'ZGARMASLIGI kerak — chaqiruvchi uni keyin ham ishlatadi.
// Bu jim buziladigan xato: nusxa olinmasa, har so'rovda kontekst
// oldingisining ustiga yopishib, tarix shishib ketardi.
func TestWithRetrievalDoesNotMutateHistory(t *testing.T) {
	s := newService(t)
	orig := userMsg("traktor")
	before := orig[0].Content

	s.withRetrieval(orig)

	if orig[0].Content != before {
		// Kontekst bloklari juda uzun — xato xabarida faqat boshini
		// ko'rsatamiz, aks holda test chiqishi o'qib bo'lmas holga keladi.
		t.Errorf("asl tarix o'zgardi:\n  edi   : %q\n  bo'ldi: %q…(%d belgi)",
			before, clip(orig[0].Content, 80), len(orig[0].Content))
	}
}

// clip — uzun matnni xato xabari uchun qisqartiradi.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Rasm-only xabar yoki bo'sh matn — qidiradigan narsa yo'q.
func TestWithRetrievalSkipsWhenNothingToSearch(t *testing.T) {
	s := newService(t)
	cases := map[string][]llm.Message{
		"bo'sh matn":         {{Role: "user", Content: "   "}},
		"oxirgisi assistant": {{Role: "user", Content: "traktor"}, {Role: "assistant", Content: "javob"}},
		"bo'sh tarix":        {},
	}
	for name, h := range cases {
		got := s.withRetrieval(h)
		if len(got) != len(h) {
			t.Errorf("%s: xabarlar soni o'zgardi (%d -> %d)", name, len(h), len(got))
			continue
		}
		for i := range h {
			if got[i].Content != h[i].Content {
				t.Errorf("%s: xabar o'zgardi", name)
			}
		}
	}
}

// Hech narsa topilmasa, tarix o'zgarmasdan qaytishi kerak.
func TestWithRetrievalNoMatches(t *testing.T) {
	h := userMsg("zzzqwertyuiop")
	got := newService(t).withRetrieval(h)
	if got[0].Content != h[0].Content {
		t.Errorf("moslik yo'q, lekin kontekst qo'shilgan: %q", got[0].Content)
	}
}

// -------------------------------------------------------------- system prompt

// Yig'im shkalasi promptga duty paketidan qo'shiladi. Busiz model
// yig'imni hisoblab bermay, "alohida belgilanadi" deb qoldirardi.
func TestSystemPromptHasFeeScale(t *testing.T) {
	got := newService(t).systemPrompt()
	for _, want := range []string{
		"BOJXONA YIG'IMI SHKALASI",
		"BRV: 412 000 so'm",
		"25×BRV",
		"289¹", // aksiz moddalariga yo'naltirish
		"IMTIYOZ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tizim ko'rsatmasida %q yo'q", want)
		}
	}
}

// Baza mavjud bo'lmasa ham prompt ishlashi kerak.
func TestSystemPromptWithoutStores(t *testing.T) {
	s := New(llm.New(), nil, nil, nil)
	if got := s.systemPrompt(); !strings.Contains(got, "Deklarant AI") {
		t.Error("bazasiz prompt asosiy ko'rsatmani yo'qotdi")
	}
}

// -------------------------------------------------------------- bloklar

// Aksiz nil bo'lsa — "yo'q" emas, "NOMA'LUM" deb yozilishi kerak.
// Bu eng xavfli jim xato edi: 0% deb ko'rsatilsa, model aroq yoki
// sigaret uchun ham "aksiz to'lanmaydi" deb javob berardi.
func TestFormatMatchesExciseUnknown(t *testing.T) {
	s := newService(t)
	m := s.codes.Search("traktor", 2)
	if len(m) == 0 {
		t.Fatal("kod topilmadi")
	}
	got := formatMatches(s.codes.Meta(), m, nil)

	if !strings.Contains(got, "aksiz: bu bazada yo'q") {
		t.Error("aksiz noma'lumligi ko'rsatilmagan")
	}
	if !strings.Contains(got, "HECH BIRIDA aksiz ma'lumoti yo'q") {
		t.Error("umumiy aksiz ogohlantirishi yo'q")
	}
}

// Imtiyoz belgisi aynan STAVKA yonida chiqishi kerak. Alohida blokda
// qolsa, model "QQS 12%" ni ko'rib hisoblab yuborardi.
func TestFormatMatchesExemptionWarning(t *testing.T) {
	s := newService(t)
	m := s.codes.Search("traktor", 1)
	if len(m) == 0 {
		t.Fatal("kod topilmadi")
	}
	code := m[0].Code.Code

	withOut := formatMatches(s.codes.Meta(), m, nil)
	withIn := formatMatches(s.codes.Meta(), m, map[string][]string{code: {"boj", "qqs"}})

	if strings.Contains(withOut, "IMTIYOZ") {
		t.Error("imtiyozsiz holatda ogohlantirish chiqdi")
	}
	if !strings.Contains(withIn, "IMTIYOZ") || !strings.Contains(withIn, "boj, qqs") {
		t.Errorf("imtiyoz ogohlantirishi yo'q:\n%s", withIn)
	}
}

// Qonun parchasida rasmiy havola ko'rsatilishi kerak — foydalanuvchi
// o'zi tekshira olsin, model esa havolani to'qib chiqarmasin.
func TestFormatLawsShowsLexLink(t *testing.T) {
	s := newService(t)
	m := s.laws.Search("kontrabanda", 3)
	if len(m) == 0 {
		t.Fatal("qonun parchasi topilmadi")
	}
	got := formatLaws(m)

	if !strings.Contains(got, "<QONUNCHILIKDAN>") {
		t.Error("blok sarlavhasi yo'q")
	}
	if !strings.Contains(got, "Rasmiy manba: https://lex.uz/") {
		t.Errorf("lex.uz havolasi yo'q:\n%s", got[:min(400, len(got))])
	}
}

// Hujjat talablari bo'limlarga ajratilishi va bo'shliq ochiq aytilishi kerak.
func TestFormatDocsSectionsAndCaveat(t *testing.T) {
	s := newService(t)
	m := s.codes.Search("dori vositalari", topCodes)
	if len(m) == 0 {
		t.Fatal("kod topilmadi")
	}
	got := formatDocs(s.docs, m)
	if got == "" {
		t.Fatal("hujjat talablari topilmadi")
	}

	if !strings.Contains(got, "<HUJJAT_TALABLARI>") {
		t.Error("blok sarlavhasi yo'q")
	}
	// Bo'sh bo'limni model "talab yo'q" deb o'qimasligi kerak.
	if !strings.Contains(got, "DEGANI EMAS") {
		t.Error("bo'sh bo'lim haqidagi ogohlantirish yo'q")
	}
}

// Talab topilmasa bo'sh satr qaytishi kerak — bo'sh blok qo'shilmasin.
func TestFormatDocsEmpty(t *testing.T) {
	if got := formatDocs(newService(t).docs, nil); got != "" {
		t.Errorf("moslik yo'q, lekin blok yasaldi: %q", got)
	}
}

// -------------------------------------------------------------- Reply

// To'liq yo'l: soxta API serveri bilan.
func TestReplySendsPromptAndReturnsText(t *testing.T) {
	var gotSystem string
	var gotMessages []struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			t.Error("x-api-key sarlavhasi yuborilmadi")
		}
		var req struct {
			System   string `json:"system"`
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("so'rovni o'qib bo'lmadi: %v", err)
		}
		gotSystem = req.System
		gotMessages = req.Messages
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"javob matni"}]}`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-kalit")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)
	s := newService(t)

	got, err := s.Reply(context.Background(), userMsg("traktor import qilsam qancha boj"))
	if err != nil {
		t.Fatalf("Reply xatosi: %v", err)
	}
	if got != "javob matni" {
		t.Errorf("javob = %q; \"javob matni\" kutilgan", got)
	}
	if !strings.Contains(gotSystem, "Deklarant AI") {
		t.Error("tizim ko'rsatmasi yuborilmadi")
	}
	// Retrieval bloklari aynan so'rov ichida ketishi kerak.
	if len(gotMessages) != 1 {
		t.Fatalf("xabarlar soni %d; 1 kutilgan", len(gotMessages))
	}
	if body, ok := gotMessages[0].Content.(string); !ok || !strings.Contains(body, "<TIF_TN_BAZASIDAN>") {
		t.Error("baza konteksti so'rovga qo'shilmagan")
	}
}

// API xatosi foydalanuvchiga yetib borishi kerak, jim yutilmasligi.
func TestReplyPropagatesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"kalit yaroqsiz"}}`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-kalit")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)

	_, err := newService(t).Reply(context.Background(), userMsg("traktor"))
	if err == nil {
		t.Fatal("xato kutilgan edi, nil qaytdi")
	}
	if !strings.Contains(err.Error(), "kalit yaroqsiz") {
		t.Errorf("xato matni yo'qoldi: %v", err)
	}
}

// Kalit yo'q bo'lsa — aniq xato, panika emas.
func TestReplyWithoutAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	if _, err := newService(t).Reply(context.Background(), userMsg("traktor")); err == nil {
		t.Error("kalitsiz xato kutilgan edi")
	}
}

