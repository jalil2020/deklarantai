package hscode

import (
	"os"
	"strings"
	"testing"
)

func load(t *testing.T) *Store {
	t.Helper()
	const p = "../../data/hscodes.json"
	if _, err := os.Stat(p); err != nil {
		t.Skip("hscodes.json yo'q — tools/extract-hscodes.mjs ishga tushiring")
	}
	s, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return s
}

func TestLoad(t *testing.T) {
	s := load(t)
	if len(s.All()) < 10_000 {
		t.Errorf("kodlar soni juda kam: %d", len(s.All()))
	}
	m := s.Meta()
	if m.Nomenclature == "" || m.LegalBasis == "" || m.RatesAsOf == "" {
		t.Errorf("meta to'liq emas: %+v", m)
	}
	// Har bir kodda asosiy maydonlar bo'lishi shart.
	for i, c := range s.All() {
		if len(c.Code) != 10 {
			t.Fatalf("kod %d: uzunligi 10 emas: %q", i, c.Code)
		}
		if strings.TrimSpace(c.PathUZ) == "" {
			t.Fatalf("kod %s: o'zbekcha tavsif bo'sh", c.Code)
		}
	}
}

// Aksiz NIL bo'lishi kerak, 0 emas.
//
// Manba bazada aksiz ma'lumoti yo'q. Uni 0 deb yozish "aksiz to'lanmaydi"
// degan yolg'on da'vo bo'lardi — aroq, sigaret, benzin aksizli.
// Bu test kimdir uni "tuzatib" 0 ga qaytarib qo'yishidan himoya qiladi.
func TestExciseIsUnknownNotZero(t *testing.T) {
	s := load(t)
	m := s.Meta()

	if m.ExciseNote == "" {
		t.Error("meta.excise_note bo'sh — aksiz yo'qligi tushuntirilishi kerak")
	}
	if m.ExciseKnownCodes != 0 {
		t.Logf("diqqat: %d kodda aksiz paydo bo'libdi — testni yangilash kerak", m.ExciseKnownCodes)
	}

	var zero, known int
	for _, c := range s.All() {
		if c.Excise == nil {
			continue
		}
		known++
		if *c.Excise == 0 {
			zero++
		}
	}
	if known != m.ExciseKnownCodes {
		t.Errorf("aksizi ma'lum kodlar: %d, meta da: %d", known, m.ExciseKnownCodes)
	}
	// Agar manba bo'sh bo'lsa, hech bir kodda "0%" turmasligi kerak.
	if m.ExciseKnownCodes == 0 && zero > 0 {
		t.Errorf("%d kodda aksiz 0%% deb yozilgan — nil bo'lishi kerak edi", zero)
	}
}

func TestSearchByCode(t *testing.T) {
	s := load(t)
	// To'liq kod
	got := s.Search("8701211019", 5)
	if len(got) == 0 || got[0].Code.Code != "8701211019" {
		t.Fatalf("to'liq kod bo'yicha qidiruv ishlamadi: %+v", got)
	}
	// Probel bilan ham topilishi kerak
	if _, ok := s.ByCode("8701 21 101 9"); !ok {
		t.Error("probelli kod topilmadi")
	}
	// Prefiks
	pre := s.Search("870121", 5)
	if len(pre) == 0 {
		t.Fatal("prefiks bo'yicha topilmadi")
	}
	for _, m := range pre {
		if !strings.HasPrefix(m.Code.Code, "870121") {
			t.Errorf("prefiksga mos kelmaydigan natija: %s", m.Code.Code)
		}
	}
}

func TestSearchByText(t *testing.T) {
	s := load(t)
	for _, c := range []struct{ query, want string }{
		{"egarli tyagach", "8701"},
		{"traktor", "8701"},
	} {
		got := s.Search(c.query, 5)
		if len(got) == 0 {
			t.Errorf("%q → hech narsa topilmadi", c.query)
			continue
		}
		found := false
		for _, m := range got {
			if strings.HasPrefix(m.Code.Code, c.want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q → top-5 da %s* yo'q (birinchisi %s)", c.query, c.want, got[0].Code.Code)
		}
	}
}

func TestSearchEmpty(t *testing.T) {
	s := load(t)
	for _, q := range []string{"", "   "} {
		if got := s.Search(q, 5); len(got) != 0 {
			t.Errorf("%q → %d natija, 0 kutilgan", q, len(got))
		}
	}
}

// Apostrof normalizatsiyasi: foydalanuvchi klaviaturadagi ' ni bosadi,
// matnda esa maxsus belgi (ʻ / ʼ) turadi. Bularsiz o'zbekchadagi juda ko'p
// so'z topilmay qolardi (qog'oz, bo'yoq, o'g'it, yog'och...).
func TestApostropheAndSuffixVariants(t *testing.T) {
	s := load(t)
	for _, q := range []string{"qog'oz", "bo'yoq", "o'g'it", "yog'och", "muzlatgich", "sovutgich"} {
		if got := s.Search(q, 1); len(got) == 0 {
			t.Errorf("%q → topilmadi", q)
		}
	}
	// Maxsus belgili yozuv ham ishlashi kerak.
	if got := s.Search("qogʻoz", 1); len(got) == 0 {
		t.Error("maxsus apostrofli yozuv topilmadi")
	}
}

// Bosh bo'g'in ustuvorligi: "traktor" so'rovi traktorlarni (8701) qaytarishi
// kerak, traktor uchun dvigatelni (8407) emas.
func TestHeadSegmentPriority(t *testing.T) {
	s := load(t)
	got := s.Search("traktor", 3)
	if len(got) == 0 {
		t.Fatal("topilmadi")
	}
	if !strings.HasPrefix(got[0].Code.Code, "8701") {
		t.Errorf("birinchi natija %s; 8701* kutilgan (%s)", got[0].Code.Code,
			trunc(got[0].Code.PathUZ, 60))
	}
}

func trunc(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// Butun jumla bilan qidirilganda ham TOVAR topilishi kerak.
//
// Foydalanuvchi qisqa so'z emas, gap yozadi. Undagi "to'lov", "jadval",
// "hujjat" kabi so'zlar JARAYON haqida, tovar haqida emas — lekin ular
// tovar tavsiflarida ham uchraydi va qidiruvni buzadi:
//
//	"…traktor import qilsam qancha to'lov chiqadi?" → 9704 "to'lov markalari"
//	"…hujjatlar kerak? Jadval bilan ko'rsat."       → 9504 "stol o'yinlari"
//
// Ikkalasi ham haqiqiy xato edi; contentTerms shuni to'g'irlaydi.
func TestSentenceQueryFindsGoods(t *testing.T) {
	s := load(t)
	cases := []struct{ query, wantPrefix string }{
		{"10 000 dollarlik traktor import qilsam qancha to'lov chiqadi? Kurs 12 800. Javobni jadval bilan ber.", "8701"},
		{"dori vositalari (bez ekstraktlari) import qilmoqchiman. Qanday hujjatlar kerak? Jadval bilan ko'rsat.", "30"},
		{"muzlatgich import qilsam qancha to'lov chiqadi", "8418"},
	}
	for _, c := range cases {
		got := s.Search(c.query, 5)
		if len(got) == 0 {
			t.Errorf("%q: hech narsa topilmadi", c.query)
			continue
		}
		found := false
		for _, m := range got {
			if strings.HasPrefix(m.Code.Code, c.wantPrefix) {
				found = true
				break
			}
		}
		if !found {
			var codes []string
			for _, m := range got {
				codes = append(codes, m.Code.Code)
			}
			t.Errorf("%q:\n  kutilgan %s… bilan boshlanuvchi kod\n  olingan: %v",
				c.query, c.wantPrefix, codes)
		}
	}
}

// Faqat jarayon so'zlaridan iborat so'rov bo'sh qolmasligi kerak.
func TestAllProcessWordsFallback(t *testing.T) {
	if got := load(t).Search("bojxona to'lov", 3); len(got) == 0 {
		t.Error("hamma atama filtrlanib ketdi; filtrsiz qidirish kerak edi")
	}
}

// Kundalik so'z nomenklatura atamasiga bog'lanishi kerak.
//
// TIF TN rasmiy til bilan yozilgan: unda "noutbuk" ham, "Hyundai" ham
// yo'q. Noutbuk u yerda "hisoblash mashinalari", avtomobil esa markasi
// bilan emas, turi bilan ataladi.
func TestSynonymExpansion(t *testing.T) {
	s := load(t)
	cases := []struct{ query, wantPrefix string }{
		{"noutbuk", "8471"},
		{"Chevrolet Cobalt import qilsam qancha", "8703"},
		{"Koreyadan 2019-yilgi Hyundai Sonata, 2.0 litr import qilaman", "8703"},
		{"kompyuter", "8471"},
	}
	for _, c := range cases {
		got := s.Search(c.query, 5)
		found := false
		for _, m := range got {
			if strings.HasPrefix(m.Code.Code, c.wantPrefix) {
				found = true
				break
			}
		}
		if !found {
			var codes []string
			for _, m := range got {
				codes = append(codes, m.Code.Code)
			}
			t.Errorf("%q:\n  kutilgan %s…\n  olingan: %v", c.query, c.wantPrefix, codes)
		}
	}
}

// O'lchov birligi tovar nomini bosib ketmasligi kerak.
//
// "litr" so'zi yolg'iz o'zi 7309 (rezervuarlar) ni beradi. Avtomobil
// so'rovida u yagona mos so'z bo'lib qolib, natijani buzardi.
func TestUnitWordsDoNotDominate(t *testing.T) {
	s := load(t)
	// Birlik so'zi bo'lsa ham tovar topilishi kerak.
	got := s.Search("2.0 litr benzinli avtomobil", 5)
	found := false
	for _, m := range got {
		if strings.HasPrefix(m.Code.Code, "8703") {
			found = true
		}
	}
	if !found {
		t.Error("birlik so'zi tovar nomini bosib ketdi")
	}
	// Lekin birlik tovarning O'ZI bo'lgan holat buzilmasligi kerak.
	if got := s.Search("plastik idish", 3); len(got) == 0 {
		t.Error("oddiy so'rov buzildi")
	}
}
