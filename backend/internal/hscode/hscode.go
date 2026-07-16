// Package hscode TIF TN (HS) kodlar bazasini yuklaydi va qidiruvni ta'minlaydi.
package hscode

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
)

// Code — bitta tovar nomenklatura yozuvi.
type Code struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Unit        string   `json:"unit"`
	ImportDuty  float64  `json:"import_duty"` // import boj stavkasi, %
	Excise      float64  `json:"excise"`      // aksiz stavkasi, %
	VAT         float64  `json:"vat"`         // QQS stavkasi, %
}

// Match — qidiruv natijasi, moslik bali bilan.
type Match struct {
	Code  Code    `json:"code"`
	Score float64 `json:"score"`
}

// Store — xotiradagi kodlar bazasi.
type Store struct {
	codes []Code
}

// Load — JSON fayldan kodlar bazasini yuklaydi.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var codes []Code
	if err := json.Unmarshal(b, &codes); err != nil {
		return nil, err
	}
	return &Store{codes: codes}, nil
}

// All — barcha kodlarni qaytaradi.
func (s *Store) All() []Code {
	return s.codes
}

// ByCode — aniq kod bo'yicha yozuvni topadi.
func (s *Store) ByCode(code string) (Code, bool) {
	code = strings.TrimSpace(code)
	for _, c := range s.codes {
		if c.Code == code {
			return c, true
		}
	}
	return Code{}, false
}

// Search — matnli so'rov bo'yicha eng mos kodlarni topadi (oddiy kalit so'z skoringi).
// LLM mavjud bo'lmaganda yoki tez natija kerak bo'lganda ishlatiladi.
func (s *Store) Search(query string, limit int) []Match {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	terms := strings.Fields(q)

	var matches []Match
	for _, c := range s.codes {
		score := scoreCode(c, q, terms)
		if score > 0 {
			matches = append(matches, Match{Code: c, Score: score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func scoreCode(c Code, q string, terms []string) float64 {
	hay := strings.ToLower(c.Name + " " + c.Description + " " + strings.Join(c.Keywords, " "))
	var score float64

	// To'liq so'rov mos kelsa — yuqori ball.
	if strings.Contains(hay, q) {
		score += 5
	}
	// Har bir so'z uchun ball.
	for _, t := range terms {
		if len(t) < 2 {
			continue
		}
		for _, kw := range c.Keywords {
			if strings.Contains(strings.ToLower(kw), t) {
				score += 3
			}
		}
		if strings.Contains(strings.ToLower(c.Name), t) {
			score += 2
		}
		if strings.Contains(strings.ToLower(c.Description), t) {
			score += 1
		}
	}
	return score
}
