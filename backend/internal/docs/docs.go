// Package docs TIF TN kodiga bog'liq hujjat talablarini beradi:
// litsenziya, sertifikat, imtiyoz va boshqa shartlar.
//
// Korpus tools/extract-docs.mjs orqali generatsiya qilinadi. Talablar
// KOD ORALIG'I bo'yicha berilgan (masalan 3001000000–3001999999), shuning
// uchun qidiruv — oraliqqa tegishlilikni tekshirish.
package docs

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
)

// Rejimlar.
const (
	Import  = "im"
	Export  = "ex"
	Transit = "tr"
)

// Meta — korpus haqidagi ma'lumot.
type Meta struct {
	Source     string `json:"source"`
	Script     string `json:"script"`
	Note       string `json:"note"`
	Warning    string `json:"warning"`
	RulesAsOf  string `json:"rules_as_of"`
	Extracted  string `json:"extracted_at"`
	Types      int    `json:"types"`
	Rules      int    `json:"rules"`
	ExpiredOut int    `json:"expired_excluded,omitempty"`
}

// TypeInfo — talab turining tavsifi.
type TypeInfo struct {
	Category string   `json:"category"`
	Text     string   `json:"text"`
	Im       bool     `json:"im"`
	Ex       bool     `json:"ex"`
	Tr       bool     `json:"tr"`
	Free     []string `json:"free,omitempty"` // qaysi to'lovdan ozod: boj, aksiz, qqs, yigim
}

// Rule — bitta kod oralig'iga qo'yilgan talab.
type Rule struct {
	Type     string `json:"type"`
	Category string `json:"category"`
	Min      string `json:"min"`
	Max      string `json:"max"`
	Law      string `json:"law,omitempty"`
	Text     string `json:"text,omitempty"` // o'z matni; bo'sh bo'lsa turnikini olamiz
	Im       bool   `json:"im,omitempty"`
	Ex       bool   `json:"ex,omitempty"`
	Tr       bool   `json:"tr,omitempty"`
}

// Requirement — kod uchun topilgan talab (tur tavsifi bilan birga).
type Requirement struct {
	Category string   `json:"category"`
	Text     string   `json:"text"`
	Law      string   `json:"law,omitempty"`
	Free     []string `json:"free,omitempty"`
}

// Store — xotiradagi talablar bazasi.
type Store struct {
	meta  Meta
	types map[string]TypeInfo
	rules []Rule
}

type storeFile struct {
	Meta  Meta                `json:"meta"`
	Types map[string]TypeInfo `json:"types"`
	Rules []Rule              `json:"rules"`
}

// Load — JSON fayldan talablarni yuklaydi.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f storeFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return &Store{meta: f.Meta, types: f.Types, rules: f.Rules}, nil
}

// Meta — korpus haqidagi ma'lumot.
func (s *Store) Meta() Meta { return s.meta }

// Exemptions — kod uchun QAYSI to'lovlardan imtiyoz bo'lishi MUMKINligini
// qaytaradi ("boj", "aksiz", "qqs", "yigim"). Bo'sh bo'lsa — imtiyoz qoidasi
// topilmadi.
//
// DIQQAT: bu "ozod qilinadi" degani EMAS, "ozod qilinishi mumkin" degani.
// Imtiyozlar shartli: "yuridik shaxslar tomonidan olib kirilganda",
// "ro'yxatga kiritilgan bo'lsa", "ishlab chiqarish uchun" va h.k. Shartni
// tovar va importchi holatiga qarab odam tekshiradi. Shuning uchun stavkani
// bu yerda 0 qilib qo'ymaymiz — faqat imkoniyatni bildiramiz.
func (s *Store) Exemptions(code, regime string) []string {
	code = normalizeCode(code)
	if code == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range s.rules {
		if r.Min > code || r.Max < code || !applies(r, regime) {
			continue
		}
		for _, f := range s.types[r.Type].Free {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	sort.Strings(out)
	return out
}

// Len — qoidalar soni.
func (s *Store) Len() int { return len(s.rules) }

// applies — qoida shu rejimga tegishlimi.
//
// Bayroqlarning HECH BIRI qo'yilmagan bo'lsa, cheklov yo'q deb qaraymiz:
// manbada ko'p qoidalarda rejim ko'rsatilmagan va ularni "hech qaysi
// rejimga tegishli emas" deb tashlab yuborsak, talab yo'qolib qolardi.
func applies(r Rule, regime string) bool {
	if !r.Im && !r.Ex && !r.Tr {
		return true
	}
	switch regime {
	case Import:
		return r.Im
	case Export:
		return r.Ex
	case Transit:
		return r.Tr
	}
	return true
}

// For — kod uchun amal qiluvchi talablarni qaytaradi.
//
// code — 10 xonali kod (bo'shliqsiz). Qisqaroq berilsa, o'ngdan nol bilan
// to'ldiriladi: "3001" → "3001000000", chunki oraliqlar 10 xonali.
func (s *Store) For(code, regime string) []Requirement {
	code = normalizeCode(code)
	if code == "" {
		return nil
	}

	var out []Requirement
	seen := map[string]bool{}
	for _, r := range s.rules {
		if r.Min > code || r.Max < code || !applies(r, regime) {
			continue
		}
		t := s.types[r.Type]
		text := r.Text
		if text == "" {
			text = t.Text
		}
		if text == "" && r.Law == "" {
			continue
		}
		// Bir xil talab turli oraliqlar orqali ikki marta tushishi mumkin.
		key := r.Category + "|" + r.Law + "|" + text
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Requirement{
			Category: r.Category,
			Text:     text,
			Law:      r.Law,
			Free:     t.Free,
		})
	}

	// Muhimlik tartibi: litsenziya → sertifikat → imtiyoz → boshqa → tavsif.
	sort.SliceStable(out, func(i, j int) bool {
		return categoryRank(out[i].Category) < categoryRank(out[j].Category)
	})
	return out
}

var categoryOrder = map[string]int{
	"litsenziya": 0,
	"sertifikat": 1,
	"imtiyoz":    2,
	"boshqa":     3,
	"tavsif":     4,
}

func categoryRank(c string) int {
	if n, ok := categoryOrder[c]; ok {
		return n
	}
	return 9
}

// normalizeCode — bo'shliq va nuqtalarni olib tashlab, 10 xonaga to'ldiradi.
func normalizeCode(code string) string {
	var b strings.Builder
	for _, r := range code {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	c := b.String()
	if c == "" || len(c) > 10 {
		if len(c) > 10 {
			return c[:10]
		}
		return ""
	}
	return c + strings.Repeat("0", 10-len(c))
}

// Program — bitta imtiyoz dasturi (masalan ПКМ 352 — texnologik uskunalar).
type Program struct {
	Type   string   `json:"type"`
	Text   string   `json:"text"`
	Free   []string `json:"free"`            // qaysi to'lovdan: boj, aksiz, qqs, yigim
	Laws   []string `json:"laws,omitempty"`  // asos hujjatlari
	Ranges int      `json:"ranges"`          // nechta kod oralig'ini qamraydi
	Codes  int      `json:"codes,omitempty"` // taxminiy kod soni (oraliqlar kengligi bo'yicha emas)
}

// Exemptions — barcha imtiyoz dasturlari ro'yxati.
//
// NEGA KERAK: imtiyoz ma'lumoti faqat aniq kod so'ralganda ko'rinardi.
// "Qanday imtiyozlar bor?" degan savolga javob yo'q edi, holbuki
// tadbirkor uchun bu eng qimmatli savollardan biri — imtiyoz boj va
// QQS ni butunlay olib tashlashi mumkin.
func (s *Store) Programs() []Program {
	byType := map[string]*Program{}
	for _, r := range s.rules {
		t := s.types[r.Type]
		if len(t.Free) == 0 {
			continue
		}
		p := byType[r.Type]
		if p == nil {
			p = &Program{Type: r.Type, Text: t.Text, Free: t.Free}
			byType[r.Type] = p
		}
		p.Ranges++
		if r.Law != "" && !contains(p.Laws, r.Law) {
			p.Laws = append(p.Laws, r.Law)
		}
		// Turdagi matn bo'sh bo'lsa, oraliqning o'z matnini olamiz.
		if p.Text == "" && r.Text != "" {
			p.Text = r.Text
		}
	}

	out := make([]Program, 0, len(byType))
	for _, p := range byType {
		sort.Strings(p.Laws)
		out = append(out, *p)
	}
	// Ko'p tovarni qamragani oldinda — foydalanuvchi uchun ehtimoli yuqori.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Ranges > out[j].Ranges })
	return out
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
