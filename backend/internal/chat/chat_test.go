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
	"deklarant-ai/backend/internal/rates"
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
	return New(llm.New(), codes, lawStore, docStore, nil)
}

func userMsg(text string) []llm.Message {
	return []llm.Message{{Role: "user", Content: text}}
}

// -------------------------------------------------------------- retrieval

// Oxirgi savolga baza bloklari qo'shilishi kerak.
func TestWithRetrievalAppendsBlocks(t *testing.T) {
	got, retrieved := newService(t).withRetrieval(context.Background(), userMsg("traktor import qilsam qancha boj"))
	if !retrieved {
		t.Error("baza konteksti topilmadi deb belgilandi")
	}
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

	s.withRetrieval(context.Background(), orig)

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
		got, _ := s.withRetrieval(context.Background(), h)
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
	got, retrieved := newService(t).withRetrieval(context.Background(), h)
	if retrieved {
		t.Error("moslik yo'q, lekin topildi deb belgilandi")
	}
	if got[0].Content != h[0].Content {
		t.Errorf("moslik yo'q, lekin kontekst qo'shilgan: %q", got[0].Content)
	}
}

// -------------------------------------------------------------- system prompt

// Yig'im shkalasi promptga duty paketidan qo'shiladi. Busiz model
// yig'imni hisoblab bermay, "alohida belgilanadi" deb qoldirardi.
func TestSystemPromptHasFeeScale(t *testing.T) {
	got := newService(t).systemPrompt(ModeDeclarant)
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
	s := New(llm.New(), nil, nil, nil, nil)
	if got := s.systemPrompt(ModeDeclarant); !strings.Contains(got, "Deklarant AI") {
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
	withIn := formatMatches(s.codes.Meta(), m, map[string][]docs.Requirement{code: {{Category: "imtiyoz", Free: []string{"boj", "qqs"}, Text: "Yevro-5 talablariga mos avtotransport", Law: "ПП 251"}}})

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
	got := formatLaws(m, maxContext)

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
	var gotCache bool
	var gotMessages []struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			t.Error("x-api-key sarlavhasi yuborilmadi")
		}
		var req struct {
			System []struct {
				Text         string `json:"text"`
				CacheControl *struct {
					Type string `json:"type"`
				} `json:"cache_control"`
			} `json:"system"`
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("so'rovni o'qib bo'lmadi: %v", err)
		}
		if len(req.System) > 0 {
			gotSystem = req.System[0].Text
			gotCache = req.System[0].CacheControl != nil
		}
		gotMessages = req.Messages
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"javob matni"}]}`))
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_API_KEY", "test-kalit")
	t.Setenv("ANTHROPIC_API_URL", srv.URL)
	s := newService(t)

	got, err := s.Reply(context.Background(), ModeDeclarant, userMsg("traktor import qilsam qancha boj"))
	if err != nil {
		t.Fatalf("Reply xatosi: %v", err)
	}
	if got != "javob matni" {
		t.Errorf("javob = %q; \"javob matni\" kutilgan", got)
	}
	if !strings.Contains(gotSystem, "Deklarant AI") {
		t.Error("tizim ko'rsatmasi yuborilmadi")
	}
	// Ko'rsatma ~6 000 belgi va har so'rovda BIR XIL — keshsiz u har safar
	// qaytadan to'lanardi. Kesh o'qish narxi asl narxning ~10%i.
	if !gotCache {
		t.Error("tizim ko'rsatmasiga cache_control qo'yilmadi — kesh ishlamaydi")
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

	_, err := newService(t).Reply(context.Background(), ModeDeclarant, userMsg("traktor"))
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
	if _, err := newService(t).Reply(context.Background(), ModeDeclarant, userMsg("traktor")); err == nil {
		t.Error("kalitsiz xato kutilgan edi")
	}
}

// MA'NOLI savolning HAMMASI asosiy modelga (Claude Opus) ketishi kerak.
//
// Arzon daraja faqat bazaga aloqasi yo'q qisqa gap uchun qoladi —
// salomlashish, minnatdorchilik. Ular bojxonaga oid javob emas.
//
// Xavf aniq: noto'g'ri hisoblangan boj deklarantga real pul va jarima
// turadi, noto'g'ri hujjat ro'yxati esa yukni chegarada to'xtatadi.
// Ikkalasi ham arzonlashtiriladigan joy emas.
func TestModelRouting(t *testing.T) {
	cases := []struct {
		name      string
		text      string
		retrieved bool
		want      llm.Model
	}{
		// Hisob-kitob
		{"boj so'raldi", "traktor import qilsam qancha boj", true, llm.Full},
		{"qqs", "bu kodga QQS qancha", true, llm.Full},
		{"aksiz", "aksiz stavkasi qanday", true, llm.Full},
		{"utilizatsiya", "utilizatsiya yig'imi qancha", true, llm.Full},
		{"hisobla", "shuni hisoblab ber", false, llm.Full},

		// ---- O'LCHANGAN REGRESSIYA ----
		// Bular Mid ga tushib ketgan va foydalanuvchi sifat pasayganini
		// sezgan. Hammasi HUKM talab qiladi, kontekstdan o'qish emas.
		{"tasnif", "elektr skuter qaysi TIF TN kodiga kiradi", true, llm.Full},
		{"tasnif-2", "quyosh panellari qaysi guruhga kiritiladi", true, llm.Full},
		{"imtiyoz", "texnologik uskunalarga imtiyoz bormi", true, llm.Full},
		{"imtiyoz-2", "qanday tovarlar bojdan ozod qilingan", true, llm.Full},
		{"kelib chiqish", "Qozog'istondan kelgan tovarga ST-1 kerakmi", true, llm.Full},
		{"kelib chiqish-2", "Rossiyadan kelsa erkin savdo rejimi qo'llanadimi", true, llm.Full},
		{"bilvosita hisob", "muzlatgich olib kirsam nima bo'ladi", true, llm.Full},
		{"bilvosita hisob-2", "yuk mashinasini olib kirsam nimalar to'lanadi", true, llm.Full},
		{"hujjat", "dori vositalari import qilishda litsenziya kerakmi", true, llm.Full},
		{"hujjat-2", "sut mahsulotlariga sertifikat talab qilinadimi", true, llm.Full},

		// ---- O'RTA daraja: FAQAT sof ma'lumot o'qish ----
		{"kod izohi", "8705 kodi nimani anglatadi", true, llm.Mid},
		{"modda", "bu qaysi modda", true, llm.Mid},

		// Kontekst topilmagan bo'lsa o'rta daraja ham berilmaydi.
		{"kontekstsiz izoh", "nimani anglatadi", false, llm.Full},

		// Faqat bo'sh gap arzon darajada.
		{"salomlashish", "salom", false, llm.Cheap},
		{"minnatdorchilik", "rahmat", false, llm.Cheap},
		{"assalomu alaykum", "Assalomu alaykum!", false, llm.Cheap},

		// Bo'sh gap + ish aralashsa — ARZON EMAS.
		{"salom + ish", "salom, muzlatgich kodi qaysi", true, llm.Full},
	}
	for _, c := range cases {
		if got := modelFor(c.text, c.retrieved); got != c.want {
			t.Errorf("%s (%q): model = %v; %v kutilgan", c.name, c.text, got, c.want)
		}
	}
}

// Hisob-kitob so'zi tutuq belgisining har xil ko'rinishida ham tanilishi
// kerak — foydalanuvchi klaviaturadagi oddiy ' ni bosadi, rasmiy matnda
// esa U+02BB turadi.
func TestCalcIntentApostrophes(t *testing.T) {
	for _, s := range []string{"to'lov qancha", "toʻlov qancha", "yig'im", "yigim"} {
		if !hasCalcIntent(s) {
			t.Errorf("%q: hisob-kitob niyati tanilmadi", s)
		}
	}
}

// Kontekst byudjeti eng yomon holatni chegaralashi kerak.
//
// O'lchov: "salom" so'zi bir vaqtlar 57 KB kontekst yasagan (qonun
// parchalari 94 KB gacha bo'lgani uchun). Parchalash tuzatilgach 13,5 KB
// ga tushdi, byudjet esa yuqori chegarani kafolatlaydi.
func TestContextBudget(t *testing.T) {
	s := newService(t)
	for _, q := range []string{"salom", "bojxona", "traktor import qilsam qancha boj"} {
		msgs, _ := s.withRetrieval(context.Background(), userMsg(q))
		got := len(msgs[len(msgs)-1].Content)
		// Byudjet qonun blokiga qo'llanadi; kod va hujjat bloklari
		// ustiga qo'shiladi, shuning uchun biroz zaxira qoldiramiz.
		if got > maxContext+16_000 {
			t.Errorf("%q: kontekst %d belgi; byudjet %d", q, got, maxContext)
		}
	}
}

// Byudjetga sig'magan parchalar borligi YASHIRILMASLIGI kerak —
// model "hammasi shu" deb o'ylamasligi kerak.
func TestBudgetTruncationIsVisible(t *testing.T) {
	s := newService(t)
	m := s.laws.Search("bojxona", topLaws)
	if len(m) < 2 {
		t.Skip("kamida 2 ta parcha kerak")
	}
	// Faqat bittasi sig'adigan byudjet.
	got := formatLaws(m, 100)
	if !strings.Contains(got, "hajm chegarasi tufayli kiritilmadi") {
		t.Error("tushib qolgan parchalar haqida ogohlantirish yo'q")
	}
}

// -------------------------------------------------------------- valyuta kursi

func cbuServer(t *testing.T, ok bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ok {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`[
		 {"Code":"840","Ccy":"USD","CcyNm_UZ":"AQSH dollari","Nominal":"1","Rate":"12093.35","Date":"17.07.2026"},
		 {"Code":"643","Ccy":"RUB","CcyNm_UZ":"Rossiya rubli","Nominal":"1","Rate":"155.22","Date":"17.07.2026"}
		]`))
	}))
}

func serviceWithRates(t *testing.T, cbuURL string) *Service {
	t.Helper()
	s := newService(t)
	s.rates = rates.New(cbuURL)
	return s
}

// Kurs kontekstga qo'shilishi kerak — busiz model uni TAXMIN qilardi.
func TestRatesBlock(t *testing.T) {
	srv := cbuServer(t, true)
	defer srv.Close()

	got := serviceWithRates(t, srv.URL).ratesBlock(context.Background())
	for _, want := range []string{
		"<VALYUTA_KURSI>",
		"1 USD = 12093.35 so'm",
		"2026-07-17",                // kurs sanasi
		"RO'YXATGA OLINGAN kundagi", // qaysi sana kursi kerakligi
	} {
		if !strings.Contains(got, want) {
			t.Errorf("kurs blokida %q yo'q:\n%s", want, got)
		}
	}
}

// Xizmat ishlamasa — blok QO'SHILMAYDI va model ko'rsatmaga ko'ra
// foydalanuvchidan kursni so'raydi. Taxminiy kurs bermaslik shart.
func TestRatesBlockWhenServiceDown(t *testing.T) {
	srv := cbuServer(t, false)
	defer srv.Close()

	if got := serviceWithRates(t, srv.URL).ratesBlock(context.Background()); got != "" {
		t.Errorf("xizmat ishlamayapti, lekin blok yasaldi: %q", got)
	}
}

// Kurs xizmati ulanmagan bo'lsa ham chat ishlashi kerak.
func TestRatesBlockWithoutClient(t *testing.T) {
	if got := newService(t).ratesBlock(context.Background()); got != "" {
		t.Errorf("klient yo'q, lekin blok yasaldi: %q", got)
	}
}

// Kurs bloki retrieval kontekstiga tushishi kerak.
func TestRetrievalIncludesRates(t *testing.T) {
	srv := cbuServer(t, true)
	defer srv.Close()

	msgs, _ := serviceWithRates(t, srv.URL).withRetrieval(
		context.Background(), userMsg("traktor import qilsam qancha boj"))
	if !strings.Contains(msgs[len(msgs)-1].Content, "<VALYUTA_KURSI>") {
		t.Error("kurs bloki kontekstga qo'shilmadi")
	}
}

// -------------------------------------------------------------- rejimlar

// ENG MUHIM TEST: soddalashtirilgan rejimda ogohlantirish TUSHIB
// QOLMASLIGI kerak.
//
// Bu loyihada tuzatilgan xatolarning aksariyati bir turdan edi — xavfli
// narsani jim tushirib qoldirish (aksiz "0%", imtiyoz e'tiborsiz, kelib
// chiqish hisobga olinmagan, kurs taxmin qilingan). "Tadbirkor rejimi"
// aynan o'sha xatoning qaytish yo'li: soddalashtirganda birinchi navbatda
// ogohlantirishlar qisqaradi.
//
// Shuning uchun ikkala promptda ham bir xil bo'lishi SHART.
func TestBothModesKeepWarnings(t *testing.T) {
	s := newService(t)
	mustHave := []string{
		// Aksiz — "yo'q" deb aytish taqiqlangan
		"289¹", "aksiz yo'q",
		// Kelib chiqish davlati
		"300-modda", "ST-1",
		// Imtiyozlar shartli
		"IMTIYOZ", "SHARTLI",
		// Kursni taxmin qilmaslik
		"TAXMIN QILMA",
		// Hujjat bo'limi bo'sh bo'lsa "kerak emas" degani emas
		"DEGANI EMAS",
		// Utilizatsiya yig'imi
		"UTILIZATSIYA", "ПКМ 347",
		// Manbani to'qib chiqarmaslik
		"to'qib",
	}
	for _, mode := range []Mode{ModeDeclarant, ModeBusiness} {
		got := s.systemPrompt(mode)
		for _, want := range mustHave {
			if !strings.Contains(got, want) {
				t.Errorf("%s rejimida %q ogohlantirishi yo'q", mode, want)
			}
		}
	}
}

// Rejimlar uslubi haqiqatan farq qilishi kerak.
func TestModesDifferInStyle(t *testing.T) {
	s := newService(t)
	dec := s.systemPrompt(ModeDeclarant)
	biz := s.systemPrompt(ModeBusiness)

	if dec == biz {
		t.Fatal("ikkala rejim bir xil prompt berdi")
	}
	// Deklarant: GTD kodlari bilan ishlaydi.
	if !strings.Contains(dec, "TAJRIBALI DEKLARANT") {
		t.Error("deklarant rejimida auditoriya ko'rsatilmagan")
	}
	// Tadbirkor: atama tushuntiriladi, jami summa oldinda.
	for _, want := range []string{"TADBIRKOR", "JAMI SUMMA", "Keyingi qadamlar"} {
		if !strings.Contains(biz, want) {
			t.Errorf("tadbirkor rejimida %q yo'q", want)
		}
	}
	// Tadbirkorga GTD kodlarini ko'rsatmaslik aytilgan bo'lishi kerak.
	if !strings.Contains(biz, "ko'rsatma") {
		t.Error("tadbirkor rejimida GTD kodlari haqidagi ko'rsatma yo'q")
	}
}

// Noma'lum yoki bo'sh rejim — deklarant (asosiy auditoriya).
func TestParseMode(t *testing.T) {
	cases := map[string]Mode{
		"deklarant": ModeDeclarant,
		"tadbirkor": ModeBusiness,
		"TADBIRKOR": ModeBusiness,
		" tadbirkor ": ModeBusiness,
		"":           ModeDeclarant,
		"broker":     ModeDeclarant,
	}
	for in, want := range cases {
		if got := ParseMode(in); got != want {
			t.Errorf("ParseMode(%q) = %q; %q kutilgan", in, got, want)
		}
	}
}

// Faktlar bloki ikkala rejimda AYNAN bir xil bo'lishi kerak —
// nusxa emas, bitta manba.
func TestSharedRulesIdentical(t *testing.T) {
	s := newService(t)
	dec := s.systemPrompt(ModeDeclarant)
	biz := s.systemPrompt(ModeBusiness)

	// sharedRules "BOJXONA TO'LOVLARI" dan boshlanadi.
	const marker = "BOJXONA TO'LOVLARI"
	di, bi := strings.Index(dec, marker), strings.Index(biz, marker)
	if di < 0 || bi < 0 {
		t.Fatal("umumiy blok topilmadi")
	}
	if dec[di:] != biz[bi:] {
		t.Error("umumiy faktlar bloki rejimlar orasida FARQ QILADI — " +
			"ogohlantirishlar ajralib ketishi mumkin")
	}
}

// Imtiyoz ro'yxati FAQAT so'rov shu haqda bo'lganda qo'shilishi kerak —
// u uzun va har savolga qo'shilsa kontekstni behuda to'ldirardi.
func TestProgramsBlockOnlyWhenAsked(t *testing.T) {
	s := newService(t)

	for _, q := range []string{
		"qanday imtiyozlar bor",
		"bojdan ozod tovarlar",
		"какие льготы есть",
		"nol stavka qo'llaniladigan tovarlar",
	} {
		if got := s.programsBlock(q); got == "" {
			t.Errorf("%q: imtiyoz ro'yxati qo'shilmadi", q)
		}
	}
	for _, q := range []string{
		"traktor boji qancha",
		"salom",
	} {
		if got := s.programsBlock(q); got != "" {
			t.Errorf("%q: imtiyoz haqida emas, lekin ro'yxat qo'shildi", q)
		}
	}
}

// Ro'yxat imtiyozni VA'DA QILMASLIGI kerak — hammasi shartli.
func TestProgramsBlockWarnsConditional(t *testing.T) {
	got := newService(t).programsBlock("qanday imtiyozlar bor")
	if got == "" {
		t.Fatal("ro'yxat bo'sh")
	}
	for _, want := range []string{"SHARTLI", "VA'DA QILMA", "asos:"} {
		if !strings.Contains(got, want) {
			t.Errorf("ro'yxatda %q yo'q", want)
		}
	}
}

// Aniq kodga bog'langan imtiyoz topilganda, UMUMIY ro'yxat
// qo'shilmasligi kerak.
//
// Sinovda "Volvo FH egarli tyagach, imtiyoz bormi?" so'roviga model
// ПКМ 352 (texnologik uskunalar) ni keltirib "tushmaydi" dedi —
// holbuki shu kod uchun aniq imtiyoz bor edi (ПП 251, yuk tashish
// texnikasi). Sabab: umumiy ro'yxat qamrov bo'yicha tartiblangan,
// ПКМ 352 birinchi turadi va aniq javobni bosib qo'yadi.
func TestSpecificExemptionBeatsGeneralList(t *testing.T) {
	s := newService(t)

	// Aniq kodi imtiyozli tovar — umumiy ro'yxat BO'LMASLIGI kerak.
	msgs, _ := s.withRetrieval(context.Background(),
		userMsg("Volvo FH egarli tyagach import qilaman, imtiyoz bormi?"))
	body := msgs[len(msgs)-1].Content

	if strings.Contains(body, "<IMTIYOZ_DASTURLARI>") {
		t.Error("aniq kod imtiyozi topilgan, lekin umumiy ro'yxat ham qo'shildi — " +
			"u aniq javobni bosib qo'yadi")
	}
	// Aniq imtiyoz esa kontekstda bo'lishi shart.
	if !strings.Contains(body, "251") {
		t.Error("kodga tegishli imtiyoz (ПП 251) kontekstda yo'q")
	}

	// Umumiy savolda esa ro'yxat kerak.
	msgs2, _ := s.withRetrieval(context.Background(),
		userMsg("umuman qanday imtiyozlar mavjud"))
	if !strings.Contains(msgs2[len(msgs2)-1].Content, "<IMTIYOZ_DASTURLARI>") {
		t.Error("umumiy savolga imtiyoz ro'yxati qo'shilmadi")
	}
}

// Murojaat kanali FAQAT tadbirkor rejimida bo'lishi kerak —
// deklarant professional, unga reklama kerak emas.
func TestContactOnlyInBusinessMode(t *testing.T) {
	s := newService(t)

	biz := s.systemPrompt(ModeBusiness)
	if !strings.Contains(biz, defaultContact) {
		t.Errorf("tadbirkor rejimida murojaat kanali yo'q")
	}
	if dec := s.systemPrompt(ModeDeclarant); strings.Contains(dec, defaultContact) {
		t.Error("deklarant rejimiga ham kanal qo'shildi")
	}
	// Shablon o'rniga haqiqiy havola qo'yilishi kerak.
	if strings.Contains(biz, "{{CONTACT}}") {
		t.Error("shablon almashtirilmadi")
	}
}

// Kanal xavfsizlik maslahatini ALMASHTIRMASLIGI kerak.
//
// Havola — oddiy murojaat yo'li, rasmiy manba emas. customs.uz va
// broker tasdig'i haqidagi maslahat baribir qolishi shart.
func TestContactDoesNotReplaceSafetyAdvice(t *testing.T) {
	got := newService(t).systemPrompt(ModeBusiness)
	for _, want := range []string{
		"ALMASHTIRMAYDI",
		"rasmiy davlat manbasi sifatida EMAS",
		"customs.uz",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ko'rsatmada %q yo'q", want)
		}
	}
}

// Kanal sozlanadigan bo'lishi kerak — o'zgarsa promptga tegmasdan.
func TestContactConfigurable(t *testing.T) {
	t.Setenv("CONTACT_TELEGRAM", "https://t.me/boshqa_kanal")
	got := newService(t).systemPrompt(ModeBusiness)

	if !strings.Contains(got, "https://t.me/boshqa_kanal") {
		t.Error("muhit o'zgaruvchisidagi kanal ishlatilmadi")
	}
	if strings.Contains(got, defaultContact) {
		t.Error("sukutdagi kanal ham qoldi")
	}
}

// Valyuta kursi bloki YOLG'IZ O'ZI "bazadan topildi" hisoblanmasligi kerak.
//
// NEGA TEST: kurs bloki deyarli har so'rovga qo'shiladi. U ham sanalsa,
// `retrieved` hech qachon false bo'lmaydi va arzon daraja O'LIK KOD ga
// aylanadi. Aynan shu holat ishlab chiqarishda bor edi — o'lchandi:
// "salom" ham Opus ga ketardi.
func TestRatesBlockDoesNotCountAsRetrieval(t *testing.T) {
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Skip("baza yo'q:", err)
	}
	// Kurs manbai BOR — ishlab chiqarishdagi holat.
	s := New(nil, codes, nil, nil, rates.New(""))

	for _, q := range []string{"salom", "rahmat", "yaxshimisiz"} {
		msgs, retrieved := s.withRetrieval(t.Context(), []llm.Message{user(q)})
		if retrieved {
			t.Errorf("%q: retrieved=true — kurs bloki noto'g'ri sanaldi", q)
		}
		if got := modelFor(q, retrieved); got != llm.Cheap {
			t.Errorf("%q: model = %v; Cheap kutilgan", q, got)
		}
		// Kurs baribir kontekstda qolishi kerak — u foydali.
		if len(msgs) > 0 && !strings.Contains(msgs[0].Content, "KURS") &&
			!strings.Contains(msgs[0].Content, "kurs") {
			t.Logf("%q: kurs bloki qo'shilmadi (manba oflayn bo'lishi mumkin)", q)
		}
	}
}

// Suhbatda bir marta hisob-kitob boshlansa, u OXIRIGACHA Full da qoladi.
//
// NEGA TEST: davomi qisqa savol bilan keladi — "unda 500 kg bo'lsa-chi?".
// Unda na hisob so'zi, na kod bor. Faqat oxirgi xabarga qaralsa, javob
// jimgina arzon modeldan kelardi va foydalanuvchi buni bilmasdi.
func TestRoutingRemembersCalcIntent(t *testing.T) {
	s := New(nil, nil, nil, nil, nil)

	history := []llm.Message{
		user("9405 42 003 9, Xitoydan 1000 kg, faktura $2000, bojni hisobla"),
		bot("boj 6 046 675 so'm"),
		user("unda 500 kg bo'lsa-chi?"),
	}
	// Davomi O'ZI qaralsa arzon darajaga tushishini tasdiqlaymiz —
	// test haqiqiy xavfni qo'riqlayotganiga ishonch hosil qilamiz.
	if got := modelFor("unda 500 kg bo'lsa-chi?", false); got == llm.Full {
		t.Log("diqqat: davomi o'zi ham Full berdi — test kuchsizlandi")
	}
	if got := s.modelOf(history, false); got != llm.Full {
		t.Errorf("hisob-kitob davomi: model = %v; Full kutilgan", got)
	}
}

// Rasm — har doim Full.
func TestRoutingImageAlwaysFull(t *testing.T) {
	s := New(nil, nil, nil, nil, nil)
	h := []llm.Message{user("bu nima?", img(100))}
	if got := s.modelOf(h, false); got != llm.Full {
		t.Errorf("suratli savol: model = %v; Full kutilgan", got)
	}
}

// Sof ma'lumot o'qish savoli o'rta darajaga tushadi.
//
// ⚠️ Ro'yxat ATAYLAB tor. Ilgari "bazadan biror narsa topildimi —
// demak Mid" degan keng qoida bor edi va u haqiqiy savollarning 76% ini
// Opus dan Sonnet ga tushirib yuborgan; foydalanuvchi javob sifati
// pasayganini sezgan. Shuning uchun bu test IKKI tomonni ham tekshiradi:
// nima Mid ga tushadi va nima TUSHMASLIGI kerak.
func TestRoutingLookupOnlyGoesMid(t *testing.T) {
	s := New(nil, nil, nil, nil, nil)

	lookup := []llm.Message{
		user("8705 kodi nimani anglatadi"),
	}
	if got := s.modelOf(lookup, true); got != llm.Mid {
		t.Errorf("kod izohi: model = %v; Mid kutilgan", got)
	}

	// Bular HUKM talab qiladi — Full da qolishi shart.
	for _, q := range []string{
		"qanday hujjat kerak",
		"elektr skuter qaysi kodga kiradi",
		"texnologik uskunalarga imtiyoz bormi",
		"Qozog'istondan kelsa ST-1 kerakmi",
		"muzlatgich olib kirsam nima bo'ladi",
	} {
		if got := s.modelOf([]llm.Message{user(q)}, true); got != llm.Full {
			t.Errorf("%q: model = %v; Full kutilgan", q, got)
		}
	}
}

// Bo'sh gapga kontekst QO'SHILMASLIGI va arzon darajaga tushishi kerak.
//
// NEGA TEST: qidiruv "salom" ga ham qonun parchasi topadi (so'z ichidan:
// "salom" ⊂ "salomatlik"). O'lchangan: salomlashishga 13 878 belgi
// kontekst ≈ $0,039 — haqiqiy huquqiy savoldan ham qimmat. Bir marta bu
// ball chegarasi bilan hal qilishga urinilgan va u "jarima" kabi
// haqiqiy savollarni ham kesib yuborgan.
func TestSmallTalkGetsNoContext(t *testing.T) {
	s := newService(t)

	for _, q := range []string{"salom", "rahmat", "Assalomu alaykum", "xayr"} {
		got, retrieved := s.withRetrieval(context.Background(), userMsg(q))
		if retrieved {
			t.Errorf("%q: kontekst qo'shildi", q)
		}
		if got[0].Content != q {
			t.Errorf("%q: xabar o'zgardi (%d belgi)", q, len(got[0].Content))
		}
		if m := s.modelOf(userMsg(q), retrieved); m != llm.Cheap {
			t.Errorf("%q: model = %v; Cheap kutilgan", q, m)
		}
	}

	// Ish aralashgan xabar bo'sh gap EMAS.
	for _, q := range []string{
		"salom, muzlatgich bojini hisobla",
		"rahmat, endi QQS ni ayting",
	} {
		_, retrieved := s.withRetrieval(context.Background(), userMsg(q))
		if !retrieved {
			t.Errorf("%q: kontekst qo'shilmadi", q)
		}
		if m := s.modelOf(userMsg(q), retrieved); m == llm.Cheap {
			t.Errorf("%q: arzon darajaga tushdi", q)
		}
	}
}
