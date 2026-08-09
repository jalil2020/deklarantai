package chat

import (
	"deklarant-ai/backend/internal/hscode"
	"strings"
	"testing"

	"deklarant-ai/backend/internal/llm"
)

// img — berilgan hajmdagi soxta surat.
func img(bytes int) llm.Image {
	return llm.Image{MediaType: "image/jpeg", Data: strings.Repeat("A", bytes)}
}

func user(text string, images ...llm.Image) llm.Message {
	return llm.Message{Role: "user", Content: text, Images: images}
}

func bot(text string) llm.Message {
	return llm.Message{Role: "assistant", Content: text}
}

func totalSize(h []llm.Message) int {
	n := 0
	for _, m := range h {
		n += size(m)
	}
	return n
}

func TestTrimKeepsShortConversation(t *testing.T) {
	h := []llm.Message{user("salom"), bot("assalom"), user("noutbuk kodi?")}
	got := trimHistory(h)
	if len(got) != len(h) {
		t.Errorf("qisqa suhbat kesildi: %d → %d", len(h), len(got))
	}
}

func TestTrimEmpty(t *testing.T) {
	if got := trimHistory(nil); len(got) != 0 {
		t.Errorf("bo'sh tarixdan %d xabar chiqdi", len(got))
	}
}

// Eng katta tejash shu yerda: surat suhbat oxirigacha qayta-qayta
// yuborilardi.
func TestTrimDropsOldImages(t *testing.T) {
	const imgBytes = 300_000 // ~1600px JPEG base64 ga yaqin

	h := []llm.Message{
		user("bu nima?", img(imgBytes)),
		bot("bu noutbuk"),
		user("yana buni ko'r", img(imgBytes)),
		bot("bu telefon"),
		user("va buni", img(imgBytes)),
		bot("bu monitor"),
		user("endi bojini hisobla"),
	}

	before := totalSize(h)
	got := trimHistory(h)
	after := totalSize(got)

	// Oxirgi ikkita suratli xabar qoladi, birinchisi ketadi.
	withImages := 0
	for _, m := range got {
		if len(m.Images) > 0 {
			withImages++
		}
	}
	if withImages != imageMessages {
		t.Errorf("suratli xabarlar soni %d; %d kutilgan", withImages, imageMessages)
	}
	if got[0].Images != nil {
		t.Error("eng eski surat olib tashlanmadi")
	}
	// Model suratning BOR EDI ekanini bilishi kerak.
	if !strings.Contains(got[0].Content, "olib tashlandi") {
		t.Errorf("surat jimgina o'chirildi, eslatma yo'q: %q", got[0].Content)
	}
	if after >= before {
		t.Errorf("hajm kamaymadi: %d → %d", before, after)
	}
	t.Logf("hajm: %d → %d belgi (%.0f%% kamaydi)",
		before, after, 100*(1-float64(after)/float64(before)))

	// Asl ro'yxat o'zgarmasligi kerak — chaqiruvchi uni boshqa joyda
	// ishlatayotgan bo'lishi mumkin.
	if h[0].Images == nil {
		t.Error("asl tarix o'zgartirildi")
	}
}

func TestTrimKeepsRecentImages(t *testing.T) {
	h := []llm.Message{user("birinchi"), bot("javob"), user("bu nima?", img(1000))}
	got := trimHistory(h)
	if len(got[len(got)-1].Images) != 1 {
		t.Error("joriy savoldagi surat olib tashlandi")
	}
}

func TestTrimLimitsMessageCount(t *testing.T) {
	var h []llm.Message
	for i := 0; i < 40; i++ {
		h = append(h, user("savol"), bot("javob"))
	}
	h = append(h, user("oxirgi savol"))

	got := trimHistory(h)
	if len(got) > maxHistoryMessages+1 {
		t.Errorf("xabarlar soni %d; ko'pi bilan %d kutilgan", len(got), maxHistoryMessages+1)
	}
	if got[len(got)-1].Content != "oxirgi savol" {
		t.Error("joriy savol yo'qoldi")
	}
}

func TestTrimLimitsChars(t *testing.T) {
	long := strings.Repeat("x", 10_000)
	var h []llm.Message
	for i := 0; i < 20; i++ {
		h = append(h, user(long), bot(long))
	}
	h = append(h, user("oxirgi savol"))

	got := trimHistory(h)
	if n := totalSize(got); n > maxHistoryChars+len(long) {
		t.Errorf("hajm %d belgi; ko'pi bilan ~%d kutilgan", n, maxHistoryChars)
	}
}

// Anthropic birinchi xabar "user" bo'lishini talab qiladi. Kesish
// tasodifan "assistant" dan boshlanib qolsa, API so'rovni rad etardi.
func TestTrimStartsWithUser(t *testing.T) {
	long := strings.Repeat("x", 30_000)
	// Ataylab shunday joylashtiramizki, byudjet "assistant" ustida tugasin.
	h := []llm.Message{
		user(long), bot(long), user(long), bot(long), user("oxirgi"),
	}
	got := trimHistory(h)
	if got[0].Role != "user" {
		t.Errorf("tarix %q rolidan boshlandi; \"user\" bo'lishi shart", got[0].Role)
	}
}

// Joriy savol byudjetdan katta bo'lsa ham qolishi kerak — aks holda
// so'rov umuman ma'nosiz bo'lardi.
func TestTrimAlwaysKeepsLast(t *testing.T) {
	h := []llm.Message{
		user("eski"), bot("javob"),
		user(strings.Repeat("x", maxHistoryChars*2)),
	}
	got := trimHistory(h)
	if len(got) == 0 || len(got[len(got)-1].Content) != maxHistoryChars*2 {
		t.Fatalf("juda uzun joriy savol yo'qoldi (%d xabar qoldi)", len(got))
	}
}

// Tizim ko'rsatmasi keshlanadigan hajmda qolishi kerak.
//
// NEGA TEST: ko'rsatma qisqartirilsa, kesh XATO BERMASDAN o'chadi —
// so'rovlar shunchaki qimmatlashadi va buni faqat hisobda sezish
// mumkin. Shu chegara ostiga tushib ketmasin.
func TestSystemPromptStaysCacheable(t *testing.T) {
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Skip("baza yo'q:", err)
	}
	s := New(nil, codes, nil, nil, nil)

	for _, mode := range []Mode{ModeDeclarant, ModeBusiness} {
		p := s.systemPrompt(mode)
		if !llm.Cacheable(p) {
			t.Errorf("rejim %q: ko'rsatma %d belgi — keshlash chegarasidan past", mode, len(p))
		}
		t.Logf("rejim %q: %d belgi", mode, len(p))
	}
}

// Kombinatsiyalangan stavka kontekstga TUSHISHI shart.
//
// NEGA TEST: jonli sinovda model 9405 42 003 9 uchun faqat 10% ni
// hisoblab, bojni 2,56 mln so'm dedi. Aslida 1 000 kg × $0,5 qat'iy
// qismi kattaroq — 5,96 mln. Sabab: formatMatches faqat foizni
// yozardi. Bu 1 555 kodga tegishli.
func TestContextIncludesSpecificDuty(t *testing.T) {
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Skip("baza yo'q:", err)
	}
	// Kombinatsiyalangan stavkali kodni bazadan topamiz.
	var combo *hscode.Code
	for _, c := range codes.All() {
		if c.ImportDutySpecific != nil {
			combo = &c
			break
		}
	}
	if combo == nil {
		t.Fatal("bazada kombinatsiyalangan stavkali kod topilmadi")
	}

	got := formatMatches(codes.Meta(), []hscode.Match{{Code: *combo}}, nil)
	if !strings.Contains(got, "QAT'IY QISM") {
		t.Errorf("kontekstda qat'iy qism yo'q:\n%s", got)
	}
	if !strings.Contains(got, combo.ImportDutySpecificUnit) {
		t.Errorf("qat'iy qism birligi (%q) yo'q", combo.ImportDutySpecificUnit)
	}

	// Qat'iy qismi yo'q kodda ortiqcha yozuv chiqmasin.
	plain := hscode.Code{Code: "1234567890", ImportDuty: 5, VAT: 12}
	if strings.Contains(formatMatches(codes.Meta(), []hscode.Match{{Code: plain}}, nil), "QAT'IY QISM") {
		t.Error("oddiy kodga ham qat'iy qism yozildi")
	}
}

// Tizim ko'rsatmasi kattasini olish qoidasini o'z ichiga olishi kerak.
func TestSystemPromptExplainsCombinedRate(t *testing.T) {
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Skip("baza yo'q:", err)
	}
	p := New(nil, codes, nil, nil, nil).systemPrompt(ModeDeclarant)
	for _, want := range []string{"KOMBINATSIYALANGAN STAVKA", "max(foizli, qat'iy)"} {
		if !strings.Contains(p, want) {
			t.Errorf("ko'rsatmada %q yo'q", want)
		}
	}
}

// Foydalanuvchi ANIQ kod yozsa, u kontekst boshida turishi shart.
//
// NEGA TEST: jonli sinovda "9405 42 003 9 kodi bo'yicha…" deb so'ralganda
// qidiruv aynan shu kodni top-8 ga ham chiqarmadi (jumladagi so'zlar
// boshqa kodlarni yuqoriga ko'tardi) va model yaqin kod bo'yicha javob
// berdi.
func TestExplicitCodeGoesFirst(t *testing.T) {
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Skip("baza yo'q:", err)
	}
	const want = "9405420039"
	text := "9405 42 003 9 kodi bo'yicha Xitoydan 1000 kg svetodiod chiroq " +
		"olib kiraman. Faktura 2000 dollar, transport 150 dollar."

	// Oddiy qidiruv bu kodni topa olmasligini tasdiqlaymiz — test
	// haqiqiy muammoni qo'riqlayotganiga ishonch hosil qilamiz.
	plain := codes.Search(text, topCodes)
	found := false
	for _, m := range plain {
		if m.Code.Code == want {
			found = true
		}
	}
	if found {
		t.Log("diqqat: qidiruvning o'zi ham topdi — test kuchsizlandi")
	}

	got := promoteExplicit(codes, text, plain)
	if len(got) == 0 || got[0].Code.Code != want {
		t.Fatalf("birinchi kod %q; %q kutilgan", firstCode(got), want)
	}
	if len(got) > topCodes {
		t.Errorf("ro'yxat %d ta; ko'pi bilan %d", len(got), topCodes)
	}
	// Takrorlanmasin.
	seen := 0
	for _, m := range got {
		if m.Code.Code == want {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("kod %d marta uchradi; 1 kutilgan", seen)
	}
}

// Raqamlar kod deb qabul qilinmasligi kerak — bazada yo'q bo'lsa e'tiborsiz.
func TestExplicitCodeIgnoresPlainNumbers(t *testing.T) {
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Skip("baza yo'q:", err)
	}
	base := codes.Search("noutbuk", topCodes)
	if len(base) == 0 {
		t.Fatal("boshlang'ich natija bo'sh")
	}
	for _, text := range []string{
		"faktura 2000 dollar, transport 150, kurs 12093.35",
		"1000 kg keldi, 500 dona quti",
		"telefon raqamim +998 90 123 45 67",
	} {
		got := promoteExplicit(codes, text, base)
		if got[0].Code.Code != base[0].Code.Code {
			t.Errorf("%q: tartib o'zgardi (%q birinchi bo'lib qoldi)", text, got[0].Code.Code)
		}
	}
}

func firstCode(m []hscode.Match) string {
	if len(m) == 0 {
		return "(bo'sh)"
	}
	return m[0].Code.Code
}

// Surat bo'lsa, kontekst ogohlantirish bilan boshlanishi kerak.
//
// NEGA TEST: qidiruv MATNDAN ishlaydi va suratni ko'rmaydi. Invoys
// surati yuborilganda savol matnida tovar nomi bo'lmaydi — jonli
// sinovda konditsioner invoysiga bazadan yog'-moy guruhi keldi.
// Ogohlantirish olib tashlansa, model begona stavkani ishonchli deb
// olishi mumkin va bu jimgina noto'g'ri boj beradi.
func TestRetrievalWarnsAboutImages(t *testing.T) {
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Skip("baza yo'q:", err)
	}
	s := New(nil, codes, nil, nil, nil)

	const q = "Bu invoysdagi tovarni o'qi va bojni hisobla"

	withImg, ok := s.withRetrieval(t.Context(), []llm.Message{user(q, img(100))})
	if !ok {
		t.Fatal("kontekst umuman qo'shilmadi")
	}
	if !strings.Contains(withImg[0].Content, "QIDIRUV SURATNI KO'RMAYDI") {
		t.Errorf("suratli savolda ogohlantirish yo'q:\n%.400s", withImg[0].Content)
	}

	// Suratsiz savolda ortiqcha ogohlantirish chiqmasin.
	noImg, _ := s.withRetrieval(t.Context(), []llm.Message{user(q)})
	if len(noImg) > 0 && strings.Contains(noImg[0].Content, "QIDIRUV SURATNI KO'RMAYDI") {
		t.Error("suratsiz savolga ham ogohlantirish qo'shildi")
	}
}

// Model o'ylab topilgan raqam bilan hisoblamasligi kerak.
//
// NEGA TEST: model "Hozircha misol bilan tushuntiraman (aniq raqamlar
// bergach, qayta hisoblab beraman)" deb butun boshli hisobni SOXTA
// summalar bilan chiqarardi. Javobda u haqiqiy hisobdan farq qilmaydi
// — foydalanuvchi summani ko'chirib olib deklaratsiyada ishlatishi
// mumkin va uning "misol" ekanini keyin eslamaydi.
//
// Qoida ikkala rejimda ham bo'lishi shart (sharedRules).
func TestSystemPromptForbidsInventedNumbers(t *testing.T) {
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Skip("baza yo'q:", err)
	}
	s := New(nil, codes, nil, nil, nil)
	for _, mode := range []Mode{ModeDeclarant, ModeBusiness} {
		p := s.systemPrompt(mode)
		for _, want := range []string{
			"O'YLAB TOPILGAN RAQAM BILAN HISOBLAMA",
			"Hozircha misol bilan tushuntiraman",
		} {
			if !strings.Contains(p, want) {
				t.Errorf("rejim %q: ko'rsatmada %q yo'q", mode, want)
			}
		}
	}
}
