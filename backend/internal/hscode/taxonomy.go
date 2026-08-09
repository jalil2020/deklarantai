package hscode

// TIF TN ierarxiyasi bo'yicha KO'RISH (qidiruvsiz).
//
// NEGA KERAK: qidiruv foydalanuvchi tovarni QANDAY ATASHNI bilishini
// talab qiladi. Ko'p hollarda bilmaydi — "musor tashuvchi mashina" ni
// nomenklatura "maxsus avtotransport" deydi. Ierarxiya bo'ylab yurish
// esa hech qanday atama bilishni talab qilmaydi: bo'limdan boshlab
// pastga tushiladi.
//
// To'rt daraja, hammasi mavjud ma'lumotdan:
//
//	Bo'lim   XVI    taxonomy.json          "Mashinalar va uskunalar…"
//	Guruh    84     taxonomy.json          "Yadroviy reaktorlar; qozonlar…"
//	Pozitsiya 8418  hscodes.json path bosh bo'g'ini
//	Kod      8418108002                    barg
//
// Faqat bo'lim va guruh NOMLARI yangi (taxonomy.json), qolgani allaqachon
// bazada edi.

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
)

// TaxonomyMeta — taksonomiya manbasi haqida.
type TaxonomyMeta struct {
	Source          string `json:"source"`
	SourceDBVersion int    `json:"source_db_version,omitempty"`
	Nomenclature    string `json:"nomenclature"`
	ExtractedAt     string `json:"extracted_at"`
	Script          string `json:"script,omitempty"`
	Note            string `json:"note,omitempty"`
	Sections        int    `json:"sections"`
	Groups          int    `json:"groups"`
}

type taxSection struct {
	Section string     `json:"section"` // rim raqami: "XVI"
	Title   string     `json:"title"`
	Groups  []taxGroup `json:"groups"`
}

type taxGroup struct {
	Group string `json:"group"` // ikki raqam: "84"
	Title string `json:"title"`
}

// Taxonomy — bo'lim va guruh sarlavhalari.
type Taxonomy struct {
	meta     TaxonomyMeta
	sections []taxSection
	// Tez qidirish uchun.
	groupTitle   map[string]string
	sectionTitle map[string]string
}

type taxonomyFile struct {
	Meta     TaxonomyMeta `json:"meta"`
	Sections []taxSection `json:"sections"`
}

// LoadTaxonomy — taksonomiyani JSON dan yuklaydi.
func LoadTaxonomy(path string) (*Taxonomy, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f taxonomyFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	t := &Taxonomy{
		meta:         f.Meta,
		sections:     f.Sections,
		groupTitle:   map[string]string{},
		sectionTitle: map[string]string{},
	}
	for _, s := range f.Sections {
		t.sectionTitle[s.Section] = s.Title
		for _, g := range s.Groups {
			t.groupTitle[g.Group] = g.Title
		}
	}
	return t, nil
}

func (t *Taxonomy) Meta() TaxonomyMeta { return t.meta }

// ---------------------------------------------------------------- ko'rish

// Node — brauzerning bitta qatori.
//
// Bitta tur ishlatiladi, chunki daraja o'zgarsa ham UI bir xil ishlaydi:
// har doim "id + sarlavha + ichidagi kodlar soni".
type Node struct {
	ID    string `json:"id"`             // "XVI" | "84" | "8418" | "8418108002"
	Title string `json:"title"`          //
	Count int    `json:"count"`          // ichidagi kod soni (barg uchun 0)
	Leaf  bool   `json:"leaf,omitempty"` // barg bo'lsa — kod kartochkasi

	// Barg uchun stavkalar (ro'yxatda darrov ko'rinsin).
	ImportDuty float64 `json:"import_duty,omitempty"`
	VAT        float64 `json:"vat,omitempty"`
	Unit       string  `json:"unit,omitempty"`
}

// Sections — barcha bo'limlar, ichidagi kodlar soni bilan.
func (s *Store) Sections(t *Taxonomy) []Node {
	count := map[string]int{}
	for i := range s.codes {
		count[s.codes[i].Section]++
	}
	var out []Node
	if t != nil {
		for _, sec := range t.sections {
			out = append(out, Node{ID: sec.Section, Title: sec.Title, Count: count[sec.Section]})
		}
		return out
	}
	// Taksonomiya yuklanmagan bo'lsa — hech bo'lmasa raqamlar chiqsin.
	for id, n := range count {
		out = append(out, Node{ID: id, Count: n})
	}
	sortNodes(out)
	return out
}

// Groups — bo'lim ichidagi guruhlar.
func (s *Store) Groups(t *Taxonomy, section string) []Node {
	count := map[string]int{}
	for i := range s.codes {
		if s.codes[i].Section == section {
			count[s.codes[i].Group]++
		}
	}
	var out []Node
	if t != nil {
		for _, sec := range t.sections {
			if sec.Section != section {
				continue
			}
			for _, g := range sec.Groups {
				out = append(out, Node{ID: g.Group, Title: g.Title, Count: count[g.Group]})
			}
			return out
		}
	}
	for id, n := range count {
		out = append(out, Node{ID: id, Count: n})
	}
	sortNodes(out)
	return out
}

// Headings — guruh ichidagi 4 xonali tovar pozitsiyalari.
//
// Sarlavha path ning BOSH bo'g'inidan olinadi — u aynan 4 xonali
// pozitsiyaning nomi (extract-hscodes.mjs shunday yig'adi).
func (s *Store) Headings(group string) []Node {
	count := map[string]int{}
	title := map[string]string{}
	for i := range s.codes {
		c := &s.codes[i]
		if c.Group != group || len(c.Code) < 4 {
			continue
		}
		h := c.Code[:4]
		count[h]++
		if _, ok := title[h]; !ok {
			title[h] = strings.TrimSpace(headSegment(c.PathUZ))
		}
	}
	out := make([]Node, 0, len(count))
	for id, n := range count {
		out = append(out, Node{ID: id, Title: title[id], Count: n})
	}
	sortNodes(out)
	return out
}

// InHeading — 4 xonali pozitsiya ichidagi kodlar (barglar).
func (s *Store) InHeading(heading string) []Node {
	var out []Node
	for i := range s.codes {
		c := &s.codes[i]
		if !strings.HasPrefix(c.Code, heading) {
			continue
		}
		// Nomi bo'sh barglar uchrab turadi ("прочие" ham bo'lmasa) —
		// bunday holda pozitsiya nomiga qaytamiz, aks holda ro'yxatda
		// nomsiz qator paydo bo'lardi.
		name := strings.TrimSpace(c.NameUZ)
		if name == "" {
			name = strings.TrimSpace(c.NameRU)
		}
		if name == "" {
			name = strings.TrimSpace(headSegment(c.PathUZ))
		}
		out = append(out, Node{
			ID: c.Code, Title: name, Leaf: true,
			ImportDuty: c.ImportDuty, VAT: c.VAT, Unit: c.Unit,
		})
	}
	sortNodes(out)
	return out
}

func sortNodes(n []Node) {
	sort.Slice(n, func(i, j int) bool { return n[i].ID < n[j].ID })
}
