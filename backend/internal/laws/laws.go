// Package laws bojxona qonunchiligi korpusini yuklaydi va qidiruvni ta'minlaydi.
//
// Korpus tools/extract-laws.mjs orqali generatsiya qilinadi: hujjatlar
// moddalarga bo'linadi va faqat bojxonaga oid parchalar saqlanadi.
// Fayl formati: {"meta": {...}, "chunks": [...]}
package laws

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"strings"
)

// Meta — korpusning kelib chiqishi haqidagi ma'lumot.
type Meta struct {
	Source      string `json:"source"`
	Script      string `json:"script"`
	Selection   string `json:"selection"`
	Chunking    string `json:"chunking"`
	Docs        int    `json:"docs"`
	Chunks      int    `json:"chunks"`
	ExtractedAt string `json:"extracted_at"`
	Note        string `json:"note,omitempty"`

	// ExpiredExcluded — korpusdan chiqarilgan bekor qilingan hujjatlar soni.
	// Bu muhim: eskirgan qonun korpusda qolsa, RAG eskirgan stavkani
	// qaytarishi va AI uni ishonch bilan aytishi mumkin.
	ExpiredExcluded int `json:"expired_excluded,omitempty"`
}

// Chunk — qonunning bitta parchasi (odatda bitta modda).
type Chunk struct {
	Doc   int    `json:"doc"`   // laws.id — manba hujjat
	Name  string `json:"name"`  // hujjat nomi (ruscha — bazada shunday saqlanadi)
	Date  string `json:"date"`  // hujjat sanasi
	Since string `json:"since"` // amal qilish boshlangan sana

	// Lex — hujjatning lex.uz dagi rasmiy havolasi (topilmasa bo'sh).
	// Moslik tools/lex-links.mjs da qo'lda yuritiladi — manba bazasida
	// lex.uz identifikatorlari yo'q.
	Lex   string `json:"lex,omitempty"`
	Title string `json:"title"` // modda sarlavhasi
	Text  string `json:"text"`  // parcha matni
	Lang  string `json:"lang"`  // "uz" (lotinlashtirilgan) yoki "ru"

	search string // kichik harfli qidiruv matni; JSON'ga chiqmaydi
}

// Match — qidiruv natijasi.
type Match struct {
	Chunk Chunk   `json:"chunk"`
	Score float64 `json:"score"`
}

// Store — xotiradagi korpus.
type Store struct {
	meta   Meta
	chunks []Chunk
	avgLen float64 // parchalarning o'rtacha uzunligi — uzunlik normalizatsiyasi uchun
}

type storeFile struct {
	Meta   Meta    `json:"meta"`
	Chunks []Chunk `json:"chunks"`
}

// Load — JSON fayldan korpusni yuklaydi.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f storeFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	var total float64
	for i := range f.Chunks {
		c := &f.Chunks[i]
		c.search = normalize(c.Title + "\n" + c.Text)
		total += float64(len(c.search))
	}
	avg := 1.0
	if len(f.Chunks) > 0 {
		avg = total / float64(len(f.Chunks))
	}
	return &Store{meta: f.Meta, chunks: f.Chunks, avgLen: avg}, nil
}

// Meta — korpus haqidagi ma'lumot.
func (s *Store) Meta() Meta { return s.meta }

// Len — parchalar soni.
func (s *Store) Len() int { return len(s.chunks) }

// hit — bitta atamaning bitta parchadagi uchrashi.
type hit struct {
	n    int
	conf float64
}

// normalize — matnni qidiruv uchun bir ko'rinishga keltiradi.
//
// O'zbek lotin yozuvida "oʻ"/"gʻ" uchun U+02BB, tutuq uchun U+02BC
// ishlatiladi; foydalanuvchi esa klaviaturadagi oddiy ' ni bosadi.
// Normallashtirmasak, "toʻlov" matnini "to'lov" so'rovi topmaydi.
func normalize(s string) string {
	return apostrophes.Replace(strings.ToLower(s))
}

var apostrophes = strings.NewReplacer(
	"ʻ", "'", "ʼ", "'", "‘", "'", "’", "'", "´", "'", "`", "'",
)

// Search — so'rovga eng mos parchalarni topadi.
func (s *Store) Search(query string, limit int) []Match {
	q := normalize(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	terms := uniqueTerms(q)
	if len(terms) == 0 {
		return nil
	}

	// Bir o'tishda har bir parcha uchun atama uchrashlarini yig'amiz va
	// ayni paytda har bir atama nechta parchada borligini (df) sanaymiz.
	hits := make([][]hit, len(s.chunks))
	df := make([]int, len(terms))
	for i := range s.chunks {
		row := make([]hit, len(terms))
		for j, t := range terms {
			n, conf := countTerm(s.chunks[i].search, t)
			row[j] = hit{n, conf}
			if n > 0 {
				df[j]++
			}
		}
		hits[i] = row
	}

	// IDF: ko'p parchada uchraydigan atama kam ma'lumot beradi.
	// Masalan "jazo" Jinoyat kodeksida hamma joyda bor, "kontrabanda" esa kam —
	// demak "kontrabanda" mosligi ancha qimmatli.
	idf := make([]float64, len(terms))
	for j := range terms {
		idf[j] = math.Log(float64(len(s.chunks)+1) / float64(df[j]+1))
	}

	var out []Match
	for i := range s.chunks {
		if sc := score(&s.chunks[i], terms, hits[i], idf, s.avgLen); sc > 0 {
			out = append(out, Match{Chunk: s.chunks[i], Score: sc})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Chunk.Doc < out[j].Chunk.Doc
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// uniqueTerms — so'rovdan mazmunli so'zlarni ajratadi (qisqa so'zlar tashlanadi).
func uniqueTerms(q string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range strings.FieldsFunc(q, func(r rune) bool {
		return !isWordRune(r)
	}) {
		if len([]rune(t)) < 4 || seen[t] {
			continue // qisqa so'zlar ("va", "uchun") shovqin beradi
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func isWordRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r >= 'а' && r <= 'я', r >= 'А' && r <= 'Я':
		return true
	case r == 'ʻ' || r == 'ʼ' || r == '\'':
		return true
	}
	return false
}

// countTerm — atamani matnda qidiradi.
//
// O'zbek tili qo'shimchali (agglutinativ), shuning uchun so'rovdagi so'z
// hujjatdagidan uzunroq bo'lishi mumkin: "vaqtinchalik" ↔ "vaqtincha".
// To'liq moslik bo'lmasa, oxiridan bir-ikki bo'g'in qisqartirib ko'ramiz.
// Teskari holat (so'rov qisqa, hujjatda qo'shimchali) substring bilan
// o'zi hal bo'ladi: "bojxona" ⊂ "bojxonaning".
//
// Qaytaradi: uchrash soni va ishonch koeffitsienti.
func countTerm(hay, term string) (int, float64) {
	if n := strings.Count(hay, term); n > 0 {
		return n, 1
	}
	// O'zbek qo'shimchalari uzun bo'lishi mumkin (-ning, -lardan, -dagi),
	// shuning uchun 5 belgigacha qisqartiramiz. Qanchalik chuqur kessak —
	// moslik shunchalik shubhali, shuning uchun ishonch pasayadi.
	r := []rune(term)
	for cut := 2; cut <= 5; cut++ {
		if len(r)-cut < 5 {
			break // o'zak juda qisqarib ketmasin
		}
		if n := strings.Count(hay, string(r[:len(r)-cut])); n > 0 {
			return n, 1 - float64(cut)*0.15
		}
	}
	return 0, 0
}

// score — parchaning so'rovga mosligini baholaydi.
//
// hits[j] — j-atamaning shu parchadagi uchrashi, idf[j] — atamaning
// noyobligi. Ball = Σ (uchrash × ishonch × noyoblik), ustiga sarlavhadagi
// moslik va turli atamalar qamrovi uchun bonus.
// BM25 parametrlari: k1 — takrorlanishning to'yinish tezligi,
// b — uzunlik normalizatsiyasining kuchi (0 = yo'q, 1 = to'liq).
const (
	bm25K1 = 1.5
	bm25B  = 0.75
)

func score(c *Chunk, terms []string, hits []hit, idf []float64, avgLen float64) float64 {
	var sc, matchedIdf, totalIdf float64
	title := normalize(c.Title)
	for j := range terms {
		totalIdf += idf[j]
	}
	// Uzunlik normalizatsiyasi: qisqa parchada bir marta uchragan atama
	// uzun parchadagidan qimmatliroq. Busiz "aroq" so'zi 1 800 belgilik
	// alkogol moddasida bir marta uchrasa, uzunroq parchalar hajmi bilan
	// yutib ketardi.
	norm := 1 - bm25B + bm25B*float64(len(c.search))/avgLen
	for j, h := range hits {
		if h.n == 0 {
			continue
		}
		matchedIdf += idf[j]
		// BM25 tf: takrorlanish foyda beradi, lekin to'yinadi.
		tf := float64(h.n)
		sc += (tf * (bm25K1 + 1) / (tf + bm25K1*norm)) * h.conf * idf[j]
		// Sarlavhada uchrasa — kuchli signal.
		if tn, tconf := countTerm(title, terms[j]); tn > 0 {
			sc += 6 * tconf * idf[j]
		}
	}
	if matchedIdf == 0 {
		return 0
	}
	// Qamrov bonusi. DIQQAT: atamalarni oddiy sanash noto'g'ri ishlaydi —
	// "kontrabanda uchun jazo" so'rovida aynan "Kontrabanda" moddasi bitta
	// atamaga mos keladi, umumiy moddalar esa "uchun"+"jazo" ga mos kelib
	// yutib ketardi. Shuning uchun qamrov ham atamaning noyobligi bo'yicha
	// o'lchanadi: bo'sh so'zlarni qoplash deyarli hech narsa bermaydi.
	return sc * (1 + matchedIdf/totalIdf)
}

// LinkCoverage — lex.uz havolasi bor parchalar soni va ulushi.
func (s *Store) LinkCoverage() (withLink, total int) {
	total = len(s.chunks)
	for i := range s.chunks {
		if s.chunks[i].Lex != "" {
			withLink++
		}
	}
	return withLink, total
}
