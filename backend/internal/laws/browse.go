package laws

// Korpus bo'ylab KO'RISH (qidiruvsiz).
//
// NEGA KERAK: qidiruv "nima izlayotganingizni bilasiz" deb faraz qiladi.
// Deklarant ko'pincha aksincha ish tutadi — "Bojxona kodeksida bu haqda
// nima deyilgan?" deb hujjatni ochib, moddalarni ko'zdan kechiradi.
// TIF TN brauzeri bilan bir xil mantiq: ikki daraja, sanoq bilan.
//
//	Hujjat (89)  →  Modda/parcha (1405)

import "sort"

// Doc — korpusdagi bitta hujjat.
type Doc struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Date   string `json:"date,omitempty"`
	Since  string `json:"since,omitempty"`
	Lex    string `json:"lex,omitempty"` // lex.uz rasmiy havolasi
	Chunks int    `json:"chunks"`        // ichidagi parchalar soni
}

// Article — hujjat ichidagi bitta parcha (odatda modda).
type Article struct {
	Doc     int    `json:"doc"`
	Index   int    `json:"index"` // hujjat ichidagi tartib raqami
	Title   string `json:"title"`
	Preview string `json:"preview"` // matnning boshi
	Lex     string `json:"lex,omitempty"`
}

// previewLen — ro'yxatda ko'rsatiladigan matn uzunligi.
//
// To'liq matn YUBORILMAYDI: bitta hujjatda 100 dan ortiq modda bo'lishi
// mumkin va ularning to'liq matni yuzlab kilobayt bo'lardi. Panelga
// baribir sig'maydi — kerak bo'lsa foydalanuvchi chatda so'raydi yoki
// lex.uz ga o'tadi.
const previewLen = 160

// Docs — korpusdagi hujjatlar, parchalar soni bilan.
//
// Tartib: parchalar soni bo'yicha kamayish tartibida. Alifbo tartibi
// emas, chunki foydalanuvchiga avvalo YIRIK hujjatlar kerak — Bojxona
// kodeksi va Soliq kodeksi ro'yxat boshida turishi mantiqiy.
func (s *Store) Docs() []Doc {
	byID := map[int]*Doc{}
	for i := range s.chunks {
		c := &s.chunks[i]
		d, ok := byID[c.Doc]
		if !ok {
			d = &Doc{ID: c.Doc, Name: c.Name, Date: c.Date, Since: c.Since, Lex: c.Lex}
			byID[c.Doc] = d
		}
		// Havola ba'zi parchalarda bo'lib, ba'zisida bo'lmasligi mumkin —
		// birinchi topilganini olamiz.
		if d.Lex == "" && c.Lex != "" {
			d.Lex = c.Lex
		}
		d.Chunks++
	}

	out := make([]Doc, 0, len(byID))
	for _, d := range byID {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Chunks != out[j].Chunks {
			return out[i].Chunks > out[j].Chunks
		}
		return out[i].Name < out[j].Name // barqaror tartib
	})
	return out
}

// Articles — bitta hujjatning parchalari, korpusdagi tartibda.
func (s *Store) Articles(doc int) []Article {
	var out []Article
	for i := range s.chunks {
		c := &s.chunks[i]
		if c.Doc != doc {
			continue
		}
		out = append(out, Article{
			Doc:     c.Doc,
			Index:   len(out),
			Title:   c.Title,
			Preview: preview(c.Text, previewLen),
			Lex:     c.Lex,
		})
	}
	return out
}

// Article — bitta parchaning TO'LIQ matni.
func (s *Store) Article(doc, index int) (Chunk, bool) {
	n := 0
	for i := range s.chunks {
		if s.chunks[i].Doc != doc {
			continue
		}
		if n == index {
			return s.chunks[i], true
		}
		n++
	}
	return Chunk{}, false
}

// preview — matnning boshini so'z chegarasida kesadi.
func preview(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := string(r[:n])
	// So'z o'rtasida kesilmasin.
	for i := len(cut) - 1; i > n/2; i-- {
		if cut[i] == ' ' {
			return cut[:i] + "…"
		}
	}
	return cut + "…"
}
