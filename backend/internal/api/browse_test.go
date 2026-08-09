package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deklarant-ai/backend/internal/chat"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/llm"
)

// browseServer — taksonomiya ulangan server.
func browseServer(t *testing.T) http.Handler {
	t.Helper()
	srv := bareServer(t)
	tax, err := hscode.LoadTaxonomy("../../data/taxonomy.json")
	if err != nil {
		t.Fatalf("taksonomiya yuklanmadi: %v", err)
	}
	srv.SetTaxonomy(tax)
	return srv.Routes()
}

// bareServer — faqat kodlar bazasi bilan server (taksonomiyasiz).
//
// Brauzerga qonun korpusi va boshqa bazalar kerak emas, shuning uchun
// ular yuklanmaydi — test sezilarli tezroq ishlaydi.
func bareServer(t *testing.T) *Server {
	t.Helper()
	t.Setenv("ANTHROPIC_API_KEY", "")
	codes, err := hscode.Load("../../data/hscodes.json")
	if err != nil {
		t.Fatal(err)
	}
	client := llm.New()
	return New(codes, nil, chat.New(client, codes, nil, nil, nil), client, nil, nil, nil)
}

func getBrowse(t *testing.T, h http.Handler, query string) browseResponse {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/hscode/browse"+query, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("browse%s: status %d, %s", query, w.Code, w.Body.String())
	}
	var out browseResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("javobni o'qib bo'lmadi: %v", err)
	}
	return out
}

// To'rt daraja ham ishlashi va bir-biriga bog'lanishi kerak.
func TestBrowseDrillsDown(t *testing.T) {
	h := browseServer(t)

	// 1. Bo'limlar — 21 ta, hammasi sarlavhali va sanoqli.
	sections := getBrowse(t, h, "")
	if sections.Level != "sections" || len(sections.Items) != 21 {
		t.Fatalf("bo'limlar: daraja=%q soni=%d; 21 kutilgan", sections.Level, len(sections.Items))
	}
	for _, s := range sections.Items {
		if s.Title == "" {
			t.Errorf("bo'lim %s sarlavhasiz — taksonomiya ulanmaganmi?", s.ID)
		}
		if s.Count == 0 {
			t.Errorf("bo'lim %s da kod yo'q", s.ID)
		}
	}

	// 2. Guruhlar.
	groups := getBrowse(t, h, "?section=XVI")
	if groups.Level != "groups" || len(groups.Items) == 0 {
		t.Fatalf("XVI guruhlari bo'sh")
	}

	// 3. Tovar pozitsiyalari.
	headings := getBrowse(t, h, "?group=84")
	if headings.Level != "headings" || len(headings.Items) == 0 {
		t.Fatal("84-guruh pozitsiyalari bo'sh")
	}

	// 4. Kodlar — barg bo'lishi va stavkasi bo'lishi kerak.
	codes := getBrowse(t, h, "?heading=8418")
	if codes.Level != "codes" || len(codes.Items) == 0 {
		t.Fatal("8418 kodlari bo'sh")
	}
	for _, c := range codes.Items {
		if !c.Leaf {
			t.Errorf("%s barg deb belgilanmagan", c.ID)
		}
		if !strings.HasPrefix(c.ID, "8418") {
			t.Errorf("%s 8418 ga tegishli emas", c.ID)
		}
	}
}

// Yo'l zanjiri yuqoriga qaytish uchun to'liq bo'lishi kerak.
func TestBrowseBreadcrumbs(t *testing.T) {
	h := browseServer(t)

	got := getBrowse(t, h, "?heading=8418")
	if len(got.Path) != 3 {
		t.Fatalf("zanjir uzunligi %d; 3 kutilgan (bo'lim › guruh › pozitsiya)", len(got.Path))
	}
	want := []struct{ level, id string }{
		{"section", "XVI"}, {"group", "84"}, {"heading", "8418"},
	}
	for i, w := range want {
		if got.Path[i].Level != w.level || got.Path[i].ID != w.id {
			t.Errorf("zanjir[%d] = %s/%s; %s/%s kutilgan",
				i, got.Path[i].Level, got.Path[i].ID, w.level, w.id)
		}
		if got.Path[i].Title == "" {
			t.Errorf("zanjir[%d] (%s) sarlavhasiz", i, got.Path[i].ID)
		}
	}
}

// Taksonomiyasiz ham ishlashi kerak — u IXTIYORIY.
//
// Sarlavhalar bo'lmaydi, lekin ro'yxat va sanoq qoladi. Aks holda
// taxonomy.json yo'qolsa butun brauzer o'chib qolardi.
func TestBrowseWorksWithoutTaxonomy(t *testing.T) {
	h := bareServer(t).Routes() // SetTaxonomy chaqirilmagan

	got := getBrowse(t, h, "")
	if len(got.Items) != 21 {
		t.Fatalf("taksonomiyasiz bo'limlar soni %d; 21 kutilgan", len(got.Items))
	}
	if got.Items[0].Count == 0 {
		t.Error("taksonomiyasiz sanoq ham yo'qoldi")
	}
}

// ---- Hujjat talablari (risk tekshiruvi) ----

func TestRequirements(t *testing.T) {
	h := newServer(t, "", "")

	// Yengil avtomobil — litsenziya/sertifikat talablari bo'lishi kutiladi.
	w, out := do(t, h, http.MethodGet, "/api/requirements?code=8703231940", "")
	wantStatus(t, w, http.StatusOK, "talablar")

	reqs, _ := out["requirements"].([]any)
	if len(reqs) == 0 {
		t.Fatal("8703 23 19 40 uchun birorta talab topilmadi")
	}
	counts, _ := out["counts"].(map[string]any)
	if len(counts) == 0 {
		t.Error("kategoriyalar bo'yicha sanoq bo'sh")
	}
	t.Logf("talablar: %d, kategoriyalar: %v", len(reqs), counts)

	// Ro'yxat to'liq emasligi javobda AYTILISHI shart — aks holda bo'sh
	// ro'yxat "hech narsa kerak emas" deb o'qilardi.
	if note, _ := out["note"].(string); !strings.Contains(note, "TO'LIQ EMAS") {
		t.Errorf("to'liqsizlik ogohlantirishi yo'q: %q", note)
	}
}

func TestRequirementsBadInput(t *testing.T) {
	h := newServer(t, "", "")
	w, _ := do(t, h, http.MethodGet, "/api/requirements", "")
	wantStatus(t, w, http.StatusBadRequest, "kodsiz")
}

// Eksport rejimi import bilan bir xil bo'lib qolmasligi kerak.
func TestRequirementsRegime(t *testing.T) {
	h := newServer(t, "", "")
	_, imp := do(t, h, http.MethodGet, "/api/requirements?code=8703231940", "")
	_, exp := do(t, h, http.MethodGet, "/api/requirements?code=8703231940&regime=export", "")

	if imp["regime"] != "im" || exp["regime"] != "ex" {
		t.Errorf("rejim noto'g'ri: import=%v eksport=%v", imp["regime"], exp["regime"])
	}
}

// ---- Davlatlar ----

func TestCountries(t *testing.T) {
	h := newServer(t, "", "")
	w, out := do(t, h, http.MethodGet, "/api/countries", "")
	wantStatus(t, w, http.StatusOK, "davlatlar")

	list, _ := out["countries"].([]any)
	if len(list) < 200 {
		t.Fatalf("davlatlar soni %d; 200+ kutilgan", len(list))
	}
	// Xitoy (156) tanlagichda ishlatiladigan hamma maydon bilan kelsin.
	var china map[string]any
	for _, c := range list {
		m := c.(map[string]any)
		if m["code"] == "156" {
			china = m
			break
		}
	}
	if china == nil {
		t.Fatal("156 (Xitoy) ro'yxatda yo'q")
	}
	if china["name_uz"] != "Xitoy" || china["duty_multiplier"] != float64(1) {
		t.Errorf("Xitoy yozuvi: %+v", china)
	}
	// Erkin savdo davlati 0 koeffitsient bilan kelishi kerak — kalkulyator
	// bojni shu orqali nolga tushiradi.
	free := 0
	for _, c := range list {
		if c.(map[string]any)["duty_multiplier"] == float64(0) {
			free++
		}
	}
	if free == 0 {
		t.Error("birorta erkin savdo davlati yo'q")
	}
}
