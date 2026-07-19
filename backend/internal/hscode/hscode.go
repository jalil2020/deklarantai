// Package hscode TIF TN (TN VED) kodlar bazasini yuklaydi va qidiruvni ta'minlaydi.
//
// Baza tools/extract-hscodes.mjs orqali manba manbasidan generatsiya qilinadi.
// Fayl formati: {"meta": {...}, "codes": [...]}
package hscode

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
)

// Meta — bazaning kelib chiqishi haqidagi ma'lumot.
// Foydalanuvchiga "bazada nima bor va u qachongi holat" deb ko'rsatish uchun.
type Meta struct {
	Nomenclature       string `json:"nomenclature"`
	LegalBasis         string `json:"legal_basis"`
	TransitionList     string `json:"transition_list,omitempty"`
	InternationalBasis string `json:"international_basis,omitempty"`
	RatesAsOf          string `json:"rates_as_of"`
	Source             string `json:"source,omitempty"`
	SourceDBVersion    int    `json:"source_db_version,omitempty"`
	ExtractedAt        string `json:"extracted_at"`
	TotalCodes         int    `json:"total_codes"`
	Note               string `json:"note,omitempty"`
	UnitNote           string `json:"unit_note,omitempty"`

	// ExciseNote — aksiz bu bazada YO'Qligini tushuntiradi.
	// ExciseKnownCodes — stavkasi ma'lum bo'lgan kodlar soni (hozircha 0).
	ExciseNote       string `json:"excise_note,omitempty"`
	ExciseKnownCodes int    `json:"excise_known_codes"`
}

// Code — bitta tovar nomenklatura yozuvi.
type Code struct {
	Code    string `json:"code"`    // 10 xonali, probelsiz
	NameRU  string `json:"name_ru"` // yakuniy tugun nomi
	NameUZ  string `json:"name_uz"` //
	PathRU  string `json:"path_ru"` // to'liq ierarxik tavsif
	PathUZ  string `json:"path_uz"` //
	Section string `json:"section"` // bo'lim (rim raqami)
	Group   string `json:"group"`   // guruh (kodning birinchi 2 raqami)

	// Qo'shimcha o'lchov birligi. Bo'sh bo'lsa — faqat kg (asosiy birlik).
	UnitCode string `json:"unit_code"`
	Unit     string `json:"unit"`
	UnitRU   string `json:"unit_ru"`

	// Stavkalar, foizda (meta.rates_as_of sanasiga).
	ImportDuty float64 `json:"import_duty"`
	ExportDuty float64 `json:"export_duty"`
	VAT        float64 `json:"vat"`
	DutyLaw    string  `json:"duty_law,omitempty"`

	// Excise — aksiz stavkasi, foizda.
	//
	// nil = NOMA'LUM (bu bazada ma'lumot yo'q), 0% DEGANI EMAS.
	// Manba bazaning "an" maydoni bo'sh, shuning uchun hozircha hamma kodda nil.
	// Haqiqiy stavkalar Soliq kodeksining 289¹–289³ moddalarida, tovar nomi
	// bo'yicha — ular qonun korpusida (laws.json) mavjud.
	Excise *float64 `json:"excise,omitempty"`

	// search — qidiruv uchun oldindan tayyorlangan kichik harfli matn.
	// JSON'ga chiqmaydi, yuklashda to'ldiriladi.
	search string
}

// Match — qidiruv natijasi, moslik bali bilan.
type Match struct {
	Code  Code    `json:"code"`
	Score float64 `json:"score"`
}

// Store — xotiradagi kodlar bazasi.
type Store struct {
	meta  Meta
	codes []Code
}

type storeFile struct {
	Meta  Meta   `json:"meta"`
	Codes []Code `json:"codes"`
}

// Load — JSON fayldan kodlar bazasini yuklaydi.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f storeFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	// Qidiruv matnini oldindan tayyorlaymiz — har so'rovda qayta hisoblamaslik uchun.
	for i := range f.Codes {
		c := &f.Codes[i]
		c.search = normalize(c.Code + " " + c.PathUZ + " " + c.PathRU)
	}
	return &Store{meta: f.Meta, codes: f.Codes}, nil
}

// Meta — baza haqidagi ma'lumot.
func (s *Store) Meta() Meta { return s.meta }

// All — barcha kodlarni qaytaradi.
func (s *Store) All() []Code { return s.codes }

// ByCode — aniq kod bo'yicha yozuvni topadi. Probel va tirelar e'tiborsiz.
func (s *Store) ByCode(code string) (Code, bool) {
	want := normalizeCode(code)
	for _, c := range s.codes {
		if c.Code == want {
			return c, true
		}
	}
	return Code{}, false
}

// normalize — matnni qidiruv uchun bir ko'rinishga keltiradi.
//
// O'zbek lotin yozuvida "oʻ" va "gʻ" uchun maxsus belgi (U+02BB) ishlatiladi,
// tutuq uchun U+02BC. Foydalanuvchi esa klaviaturadagi oddiy ' ni bosadi.
// Normallashtirmasak, "qog'oz" so'rovi "qogʻoz" matnini topmaydi — bu
// o'zbekchadagi juda ko'p so'zga ta'sir qiladi.
//
// Shuningdek: "muzlatkich" ↔ "muzlatgich" kabi -kich/-gich almashinuvi
// rasmiy matn bilan kundalik yozuv orasidagi keng tarqalgan farq.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = apostrophes.Replace(s)
	return strings.ReplaceAll(s, "kich", "gich")
}

var apostrophes = strings.NewReplacer(
	"ʻ", "'", // ʻ  oʻ, gʻ
	"ʼ", "'", // ʼ  tutuq belgisi
	"‘", "'", // '
	"’", "'", // '
	"´", "'", // ´
	"`", "'",
)

func normalizeCode(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Search — matnli so'rov bo'yicha eng mos kodlarni topadi.
//
// Kod bo'yicha qidiruv (raqamlar kiritilsa) va matn bo'yicha qidiruv
// (o'zbekcha yoki ruscha) qo'llab-quvvatlanadi.
func (s *Store) Search(query string, limit int) []Match {
	q := normalize(strings.TrimSpace(query))
	if q == "" {
		return nil
	}

	// So'rov asosan raqamlardan iborat bo'lsa — kod bo'yicha qidiramiz.
	if digits := normalizeCode(q); len(digits) >= 4 && float64(len(digits)) > float64(len(q))*0.6 {
		return s.searchByCode(digits, limit)
	}

	terms := strings.Fields(q)
	var matches []Match
	for i := range s.codes {
		if sc := score(&s.codes[i], q, terms); sc > 0 {
			matches = append(matches, Match{Code: s.codes[i], Score: sc})
		}
	}
	sortMatches(matches)
	return trim(matches, limit)
}

func (s *Store) searchByCode(digits string, limit int) []Match {
	var matches []Match
	for i := range s.codes {
		c := &s.codes[i]
		switch {
		case c.Code == digits:
			matches = append(matches, Match{Code: *c, Score: 100})
		case strings.HasPrefix(c.Code, digits):
			matches = append(matches, Match{Code: *c, Score: 50})
		}
	}
	sortMatches(matches)
	return trim(matches, limit)
}

func score(c *Code, q string, terms []string) float64 {
	var sc float64
	if strings.Contains(c.search, q) {
		sc += 10
	}
	for _, t := range terms {
		if len([]rune(t)) < 3 {
			continue // juda qisqa so'zlar shovqin beradi
		}
		if strings.Contains(c.search, t) {
			sc += 3
		}
	}
	if sc == 0 {
		return 0
	}

	// Ierarxiyaning BOSH bo'g'ini — tovar pozitsiyasi sarlavhasi, ya'ni
	// "bu narsa NIMA". Izohlardan ancha kuchli signal.
	//
	// Misol: "traktor" so'rovida
	//   8701…  "Traktorlar (…); …; boshqalar"        ← bosh bo'g'inda
	//   8407…  "Ichki yonuv dvigatellari; traktorlar uchun"  ← izohda
	// Ikkalasi ham mos keladi, lekin foydalanuvchi traktorni so'ragan.
	head := normalize(headSegment(c.PathUZ) + " " + headSegment(c.PathRU))
	leaf := normalize(c.NameUZ + " " + c.NameRU)
	for _, t := range terms {
		if len([]rune(t)) < 3 {
			continue
		}
		if strings.Contains(head, t) {
			sc += 8
		}
		// Yakuniy tugun nomida uchrasa — aniqroq moslik.
		if strings.Contains(leaf, t) {
			sc += 4
		}
	}
	return sc
}

// headSegment — ierarxik tavsifning birinchi bo'g'ini ("A; B; C" → "A").
func headSegment(path string) string {
	if i := strings.Index(path, ";"); i >= 0 {
		return path[:i]
	}
	return path
}

func sortMatches(m []Match) {
	sort.SliceStable(m, func(i, j int) bool {
		if m[i].Score != m[j].Score {
			return m[i].Score > m[j].Score
		}
		return m[i].Code.Code < m[j].Code.Code
	})
}

func trim(m []Match, limit int) []Match {
	if limit > 0 && len(m) > limit {
		return m[:limit]
	}
	return m
}
