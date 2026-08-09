// Package countries davlatlar ma'lumotnomasini va ularning boj rejimini beradi.
//
// HUQUQIY ASOS — Bojxona kodeksi 300-modda:
//
//	erkin savdo davlatlari            → bojxona bojlari qo'llanilmaydi
//	eng qulaylik rejimi (EQR)         → boj tarifidagi odatdagi stavka
//	rejim yo'q yoki kelib chiqishi
//	aniqlanmagan                      → stavkalar IKKI BARAVAR oshiriladi
//
// Ya'ni boj faqat tovar kodiga emas, KELIB CHIQISH DAVLATIGA ham bog'liq.
package countries

import (
	"encoding/json"
	"os"
	"strings"
)

// Meta — ma'lumotnoma haqida.
type Meta struct {
	Source      string `json:"source"`
	LegalBasis  string `json:"legal_basis"`
	LegalLex    string `json:"legal_basis_lex"`
	Note        string `json:"note"`
	Warning     string `json:"warning"`
	ExtractedAt string `json:"extracted_at"`
	Total       int    `json:"total"`
}

// Country — bitta davlat.
type Country struct {
	Code    string   `json:"code"`    // raqamli kod (GTD da shu ishlatiladi)
	NameRU  string   `json:"name_ru"` //
	NameUZ  string   `json:"name_uz"` //
	Aliases []string `json:"aliases,omitempty"`
	ISO     string   `json:"iso,omitempty"`
	Regime  string   `json:"regime"`

	// DutyMultiplier — boj stavkasi shu songa ko'paytiriladi (0, 1 yoki 2).
	DutyMultiplier float64 `json:"duty_multiplier"`

	// Offshore — offshor zona. Boj rejimiga ta'sir qilmaydi, lekin
	// qo'shimcha nazorat va hujjat talablari bo'lishi mumkin.
	Offshore bool `json:"offshore,omitempty"`
}

// Store — xotiradagi ma'lumotnoma.
type Store struct {
	meta Meta
	list []Country
	// byKey — kod, ISO, nom va sinonimlar bo'yicha indeks.
	byKey map[string]*Country
}

type storeFile struct {
	Meta      Meta      `json:"meta"`
	Countries []Country `json:"countries"`
}

// Load — JSON fayldan ma'lumotnomani yuklaydi.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f storeFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	s := &Store{meta: f.Meta, list: f.Countries, byKey: map[string]*Country{}}
	for i := range s.list {
		c := &s.list[i]
		s.index(c.Code, c)
		s.index(c.ISO, c)
		s.index(c.NameUZ, c)
		s.index(c.NameRU, c)
		for _, a := range c.Aliases {
			s.index(a, c)
		}
	}
	return s, nil
}

func (s *Store) index(key string, c *Country) {
	if k := normalize(key); k != "" {
		// Birinchi yozuv ustun: ro'yxat kod bo'yicha tartiblangan va
		// takrorlanuvchi nom bo'lsa, kichik kodli davlat qoladi.
		if _, exists := s.byKey[k]; !exists {
			s.byKey[k] = c
		}
	}
}

// normalize — qidiruv kaliti. Apostrof shakllari va registr farqi yo'qoladi.
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return apostrophes.Replace(s)
}

var apostrophes = strings.NewReplacer(
	"ʻ", "'", "ʼ", "'", "‘", "'", "’", "'", "´", "'", "`", "'",
)

// Meta — ma'lumotnoma haqidagi ma'lumot.
func (s *Store) Meta() Meta { return s.meta }

// Len — davlatlar soni.
func (s *Store) Len() int { return len(s.list) }

// Find — kod, ISO yoki nom bo'yicha davlatni topadi.
//
// "643", "RU", "Rossiya", "РОССИЯ", "Rossiya Federatsiyasi" — hammasi ishlaydi.
func (s *Store) Find(q string) (Country, bool) {
	if c, ok := s.byKey[normalize(q)]; ok {
		return *c, true
	}
	return Country{}, false
}

// FreeTrade — erkin savdo davlatlari (boj qo'llanilmaydi).
func (s *Store) FreeTrade() []Country {
	var out []Country
	for _, c := range s.list {
		if c.DutyMultiplier == 0 {
			out = append(out, c)
		}
	}
	return out
}

// UnknownOriginMultiplier — kelib chiqishi aniqlanmagan tovar uchun
// koeffitsient. BK 300-modda: "ishlab chiqarilgan mamlakati aniqlanmagan
// tovarlarga nisbatan bojxona bojlarining stavkalari ikki baravar oshiriladi".
const UnknownOriginMultiplier = 2.0

// List — barcha davlatlar, nomi bo'yicha emas, RO'YXATDAGI tartibda.
//
// Tartib manba bazadan keladi va u yerda ko'p ishlatiladiganlar
// boshida turadi — tanlagich uchun shu qulay.
func (s *Store) List() []Country {
	out := make([]Country, len(s.list))
	copy(out, s.list)
	return out
}
