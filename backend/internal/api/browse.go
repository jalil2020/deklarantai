package api

// TIF TN ierarxiyasi bo'yicha ko'rish — qidiruvsiz kirish nuqtasi.
//
// NEGA KERAK: qidiruv foydalanuvchidan tovarni NOMENKLATURA TILIDA
// atashni talab qiladi. Bilmasa — hech narsa topilmaydi ("musor tashuvchi
// mashina" nomenklaturada "maxsus avtotransport"). Ierarxiya esa hech
// qanday atama bilishni talab qilmaydi: bo'limdan pastga tushiladi.
//
// Bitta endpoint, daraja parametr bilan aniqlanadi. Alohida yo'llar
// qilinmadi: UI uchun ular bir xil shakldagi ro'yxat, farqi faqat
// qayerdan olinishida.
//
//	GET /api/hscode/browse                 → bo'limlar
//	GET /api/hscode/browse?section=XVI     → guruhlar
//	GET /api/hscode/browse?group=84        → tovar pozitsiyalari (4 xonali)
//	GET /api/hscode/browse?heading=8418    → kodlar (barglar)

import (
	"net/http"
	"strconv"
	"strings"

	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
)

type browseResponse struct {
	Level  string        `json:"level"` // sections | groups | headings | codes
	Parent string        `json:"parent,omitempty"`
	Items  []hscode.Node `json:"items"`
	// Path — yuqoriga qaytish zanjiri. Server beradi, chunki sarlavhalar
	// (bo'lim/guruh nomi) faqat shu yerda ma'lum.
	Path []crumb `json:"path,omitempty"`
}

type crumb struct {
	Level string `json:"level"` // qaysi parametr bilan qayta so'raladi
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	section := strings.TrimSpace(q.Get("section"))
	group := strings.TrimSpace(q.Get("group"))
	heading := strings.TrimSpace(q.Get("heading"))

	switch {
	case heading != "":
		writeJSON(w, http.StatusOK, browseResponse{
			Level: "codes", Parent: heading,
			Items: s.codes.InHeading(heading),
			Path:  s.crumbs(heading[:min(2, len(heading))], heading),
		})
	case group != "":
		writeJSON(w, http.StatusOK, browseResponse{
			Level: "headings", Parent: group,
			Items: s.codes.Headings(group),
			Path:  s.crumbs(group, ""),
		})
	case section != "":
		writeJSON(w, http.StatusOK, browseResponse{
			Level: "groups", Parent: section,
			Items: s.codes.Groups(s.taxonomy, section),
			Path:  []crumb{s.sectionCrumb(section)},
		})
	default:
		writeJSON(w, http.StatusOK, browseResponse{
			Level: "sections",
			Items: s.codes.Sections(s.taxonomy),
		})
	}
}

// crumbs — guruh (va ixtiyoriy pozitsiya) uchun to'liq zanjir.
//
// Guruhdan bo'limni topish uchun taksonomiya kerak emas: kodlar bazasida
// har bir kodda ikkalasi ham bor, shuning uchun guruhning bo'limi
// birinchi mos kelgan koddan olinadi.
func (s *Server) crumbs(group, heading string) []crumb {
	var out []crumb
	if sec := s.sectionOfGroup(group); sec != "" {
		out = append(out, s.sectionCrumb(sec))
	}
	if group != "" {
		out = append(out, crumb{Level: "group", ID: group, Title: s.groupTitle(group)})
	}
	if heading != "" {
		out = append(out, crumb{Level: "heading", ID: heading, Title: s.headingTitle(heading)})
	}
	return out
}

func (s *Server) sectionOfGroup(group string) string {
	if group == "" {
		return ""
	}
	for _, c := range s.codes.All() {
		if c.Group == group {
			return c.Section
		}
	}
	return ""
}

func (s *Server) sectionCrumb(section string) crumb {
	title := ""
	if s.taxonomy != nil {
		for _, n := range s.codes.Sections(s.taxonomy) {
			if n.ID == section {
				title = n.Title
				break
			}
		}
	}
	return crumb{Level: "section", ID: section, Title: title}
}

func (s *Server) groupTitle(group string) string {
	if s.taxonomy == nil {
		return ""
	}
	if sec := s.sectionOfGroup(group); sec != "" {
		for _, n := range s.codes.Groups(s.taxonomy, sec) {
			if n.ID == group {
				return n.Title
			}
		}
	}
	return ""
}

func (s *Server) headingTitle(heading string) string {
	if len(heading) < 2 {
		return ""
	}
	for _, n := range s.codes.Headings(heading[:2]) {
		if n.ID == heading {
			return n.Title
		}
	}
	return ""
}

// ---------------------------------------------------------------- qonunlar

// Qonun korpusi bo'yicha ko'rish.
//
//	GET /api/laws/browse            → hujjatlar
//	GET /api/laws/browse?doc=12     → moddalar
//	GET /api/laws/browse?doc=12&i=3 → bitta moddaning to'liq matni
type lawsBrowseResponse struct {
	Level string `json:"level"` // docs | articles | article

	Docs     []laws.Doc     `json:"docs,omitempty"`
	Articles []laws.Article `json:"articles,omitempty"`

	// Article — to'liq matn (level=article bo'lganda).
	Article *laws.Chunk `json:"article,omitempty"`

	// DocName — moddalar ro'yxatida qaysi hujjat ekanini ko'rsatish uchun.
	DocName string `json:"doc_name,omitempty"`
	DocLex  string `json:"doc_lex,omitempty"`
}

func (s *Server) handleLawsBrowse(w http.ResponseWriter, r *http.Request) {
	if s.laws == nil {
		writeErr(w, http.StatusServiceUnavailable, "qonun korpusi yuklanmagan")
		return
	}
	q := r.URL.Query()
	docRaw := strings.TrimSpace(q.Get("doc"))

	if docRaw == "" {
		writeJSON(w, http.StatusOK, lawsBrowseResponse{Level: "docs", Docs: s.laws.Docs()})
		return
	}
	doc, err := strconv.Atoi(docRaw)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "hujjat raqami noto'g'ri")
		return
	}

	// Bitta modda so'ralgan bo'lsa — to'liq matn.
	if iRaw := strings.TrimSpace(q.Get("i")); iRaw != "" {
		idx, err := strconv.Atoi(iRaw)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "modda raqami noto'g'ri")
			return
		}
		c, ok := s.laws.Article(doc, idx)
		if !ok {
			writeErr(w, http.StatusNotFound, "modda topilmadi")
			return
		}
		writeJSON(w, http.StatusOK, lawsBrowseResponse{
			Level: "article", Article: &c, DocName: c.Name, DocLex: c.Lex,
		})
		return
	}

	arts := s.laws.Articles(doc)
	if len(arts) == 0 {
		writeErr(w, http.StatusNotFound, "hujjat topilmadi")
		return
	}
	// Hujjat nomini birinchi moddadan olamiz — hammasida bir xil.
	name, lex := "", ""
	if c, ok := s.laws.Article(doc, 0); ok {
		name, lex = c.Name, c.Lex
	}
	writeJSON(w, http.StatusOK, lawsBrowseResponse{
		Level: "articles", Articles: arts, DocName: name, DocLex: lex,
	})
}
