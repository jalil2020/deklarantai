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
