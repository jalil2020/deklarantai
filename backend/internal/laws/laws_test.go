package laws

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Haqiqiy korpus bilan sinaymiz — qidiruv sifati shu bilan o'lchanadi.
func load(t *testing.T) *Store {
	t.Helper()
	const p = "../../data/laws.json"
	if _, err := os.Stat(p); err != nil {
		t.Skip("laws.json yo'q — tools/extract-laws.mjs ishga tushiring")
	}
	s, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return s
}

func TestLoad(t *testing.T) {
	s := load(t)
	if s.Len() < 100 {
		t.Errorf("parchalar soni juda kam: %d", s.Len())
	}
	if s.Meta().Docs == 0 {
		t.Error("meta.docs bo'sh")
	}
	// Har bir parchada sarlavha va matn bo'lishi shart.
	for i, c := range s.chunks {
		if strings.TrimSpace(c.Title) == "" {
			t.Fatalf("chunk %d: sarlavha bo'sh", i)
		}
		if len(c.Text) < 50 {
			t.Fatalf("chunk %d: matn juda qisqa (%d)", i, len(c.Text))
		}
	}
}

// Qidiruv haqiqiy savollarga tegishli moddalarni topishi kerak.
func TestSearchRelevance(t *testing.T) {
	s := load(t)
	cases := []struct {
		query string
		want  string // natijalarning birortasida uchrashi kerak
	}{
		{"bojxona toʻlovlari kafillik bilan taʼminlash", "kafillik"},
		{"kontrabanda uchun javobgarlik", "kontrabanda"},
		{"bojxona qiymatini aniqlash usullari", "qiymat"},
		// Diqqat: hujjatda "vaqtincha", so'rovda "vaqtinchalik" —
		// o'zak bo'yicha moslik ishlashi kerak (countTerm).
		{"vaqtinchalik olib kirish rejimi", "vaqtincha"},
		{"bojxona deklaratsiyasi berish muddati", "deklarats"},
	}
	for _, c := range cases {
		got := s.Search(c.query, 5)
		if len(got) == 0 {
			t.Errorf("%q → hech narsa topilmadi", c.query)
			continue
		}
		found := false
		for _, m := range got {
			if strings.Contains(strings.ToLower(m.Chunk.Title+m.Chunk.Text), c.want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q → top-5 da %q yo'q. Birinchisi: %s",
				c.query, c.want, trunc(got[0].Chunk.Title, 70))
		}
	}
}

// Bo'sh yoki ma'nosiz so'rov natija bermasligi kerak.
func TestSearchEmpty(t *testing.T) {
	s := load(t)
	for _, q := range []string{"", "   ", "va", "bu"} {
		if got := s.Search(q, 5); len(got) != 0 {
			t.Errorf("%q → %d natija, 0 kutilgan", q, len(got))
		}
	}
}

// Limit hurmat qilinishi kerak.
func TestSearchLimit(t *testing.T) {
	s := load(t)
	if got := s.Search("bojxona", 3); len(got) > 3 {
		t.Errorf("limit 3, olindi %d", len(got))
	}
}

func trunc(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// O'zak bo'yicha moslik: so'rovdagi so'z hujjatdagidan uzunroq bo'lsa ham topilsin.
func TestStemMatch(t *testing.T) {
	cases := [][2]string{
		{"vaqtinchalik", "vaqtincha"}, // qo'shimcha tushadi
		{"bojxonaning", "bojxona"},
		{"deklaratsiyalar", "deklaratsiya"},
	}
	for _, c := range cases {
		if n, conf := countTerm("... "+c[1]+" ...", c[0]); n == 0 {
			t.Errorf("countTerm(%q, %q) = 0; topilishi kerak edi", c[1], c[0])
		} else if conf >= 1 {
			t.Errorf("countTerm(%q, %q): o'zak mosligi ishonchi %v; 1 dan kam bo'lishi kerak", c[1], c[0], conf)
		}
	}
	// To'liq moslik — to'liq ishonch.
	if _, conf := countTerm("bojxona kodeksi", "bojxona"); conf != 1 {
		t.Errorf("to'liq moslik ishonchi = %v; 1 kutilgan", conf)
	}
}

// Ustki indeksli moddalar to'g'ri saqlanishi.
//
// Manba HTML da ular <sup> bilan beriladi: 289<sup>1</sup>-модда.
// Teglar shunchaki olib tashlansa, raqamlar BIRIKIB ketadi va korpusda
// mavjud bo'lmagan "2891-modda" paydo bo'ladi — AI esa o'sha sarlavhani
// iqtibos qilib, YO'Q moddaga havola beradi. Aksiz stavkalari aynan
// 289¹–289³ moddalarida, ya'ni xato eng muhim joyga tushgan edi.
func TestSuperscriptArticles(t *testing.T) {
	s, err := Load("../../data/laws.json")
	if err != nil {
		t.Fatal(err)
	}

	// Aksiz moddalari — chat ko'rsatmasi aynan shu raqamlarga yo'naltiradi.
	for _, want := range []string{"289¹-modda", "289²-modda", "289³-modda"} {
		found := false
		for _, c := range s.chunks {
			if strings.HasPrefix(strings.TrimSpace(c.Title), want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q sarlavhali parcha topilmadi", want)
		}
	}

	// Birikib ketgan raqam qolmasligi kerak: O'zbekiston kodekslarida
	// to'rt xonali modda raqami yo'q, demak "2891-modda" — xato belgisi.
	bad := regexp.MustCompile(`^\d{4,}-modda`)
	for _, c := range s.chunks {
		if bad.MatchString(strings.TrimSpace(c.Title)) {
			t.Errorf("birikib ketgan modda raqami: %q (%s)", c.Title, c.Name)
		}
	}
}

// Mundarija qatorlari korpusga tushmasligi, lekin haqiqiy qisqa moddalar
// saqlanishi kerak. Ilgari "200 belgidan qisqa bo'lsa tashla" qoidasi
// Bojxona kodeksidan 7, 103, 110, 257-moddani ham yo'q qilgan edi.
func TestShortArticlesKept(t *testing.T) {
	s, err := Load("../../data/laws.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"7-modda", "103-modda", "110-modda", "257-modda"} {
		found := false
		for _, c := range s.chunks {
			if strings.Contains(c.Name, "Таможенный кодекс") &&
				strings.HasPrefix(strings.TrimSpace(c.Title), want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Bojxona kodeksida %q topilmadi", want)
		}
	}
}
