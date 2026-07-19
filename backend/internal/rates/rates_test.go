package rates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// cbu.uz javobining haqiqiy shakli.
const sample = `[
 {"id":68,"Code":"840","Ccy":"USD","CcyNm_UZ":"AQSH dollari","Nominal":"1","Rate":"12093.35","Date":"17.07.2026"},
 {"id":69,"Code":"978","Ccy":"EUR","CcyNm_UZ":"Yevro","Nominal":"1","Rate":"13867.44","Date":"17.07.2026"},
 {"id":70,"Code":"643","Ccy":"RUB","CcyNm_UZ":"Rossiya rubli","Nominal":"1","Rate":"155.22","Date":"17.07.2026"},
 {"id":71,"Code":"704","Ccy":"VND","CcyNm_UZ":"Vetnam dongi","Nominal":"10","Rate":"4.61","Date":"17.07.2026"}
]`

func server(t *testing.T, body string, status int, hits *int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			atomic.AddInt64(hits, 1)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestGetRate(t *testing.T) {
	srv := server(t, sample, http.StatusOK, nil)
	defer srv.Close()

	got, err := New(srv.URL).Get(context.Background(), "USD", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != 12093.35 {
		t.Errorf("kurs = %v; 12093.35 kutilgan", got.Value)
	}
	if got.Code != "840" || got.Ccy != "USD" {
		t.Errorf("kod/valyuta = %s/%s", got.Code, got.Ccy)
	}
	// Sana ISO ko'rinishga o'girilishi kerak.
	if got.Date != "2026-07-17" {
		t.Errorf("sana = %q; \"2026-07-17\" kutilgan", got.Date)
	}
}

// Nominal — eng xavfli joy: VND 10 birlik uchun kotirovka qilinadi.
// Bo'lmasak, kurs o'n barobar xato chiqadi.
func TestNominalIsApplied(t *testing.T) {
	srv := server(t, sample, http.StatusOK, nil)
	defer srv.Close()

	got, err := New(srv.URL).Get(context.Background(), "VND", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	// 4.61 / 10 = 0.461
	if got.Value != 0.461 {
		t.Errorf("VND kursi = %v; 0.461 kutilgan (4.61/10)", got.Value)
	}
}

// Valyutani harf kodi bilan ham, raqamli kod bilan ham topish mumkin —
// GTD da raqamli kod ishlatiladi.
func TestFindByNumericCode(t *testing.T) {
	srv := server(t, sample, http.StatusOK, nil)
	defer srv.Close()
	c := New(srv.URL)

	byCcy, _ := c.Get(context.Background(), "usd", time.Time{})
	byCode, err := c.Get(context.Background(), "840", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if byCcy.Value != byCode.Value {
		t.Errorf("harf va raqam kodi turlicha: %v va %v", byCcy.Value, byCode.Value)
	}
}

// Kesh: bir xil sana uchun ikkinchi marta so'rov ketmasligi kerak.
func TestCache(t *testing.T) {
	var hits int64
	srv := server(t, sample, http.StatusOK, &hits)
	defer srv.Close()
	c := New(srv.URL)

	for i := 0; i < 3; i++ {
		if _, err := c.Get(context.Background(), "USD", time.Time{}); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 1 {
		t.Errorf("tashqi so'rovlar soni = %d; 1 kutilgan (kesh ishlamayapti)", hits)
	}
}

// Turli sanalar alohida keshlanishi kerak.
func TestCachePerDate(t *testing.T) {
	var hits int64
	srv := server(t, sample, http.StatusOK, &hits)
	defer srv.Close()
	c := New(srv.URL)

	day1 := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	c.Get(context.Background(), "USD", day1)
	c.Get(context.Background(), "USD", day1)
	c.Get(context.Background(), "USD", day2)

	if hits != 2 {
		t.Errorf("so'rovlar = %d; 2 kutilgan (har sana alohida)", hits)
	}
}

// Sana berilganda URL da /all/<sana>/ bo'lishi kerak.
func TestDateURL(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(sample))
	}))
	defer srv.Close()

	New(srv.URL).Get(context.Background(), "USD", time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if !strings.Contains(gotPath, "/all/2026-07-10/") {
		t.Errorf("so'rov yo'li = %q; /all/2026-07-10/ kutilgan", gotPath)
	}
}

// Xizmat ishlamasa — XATO qaytishi kerak, taxminiy kurs EMAS.
// Bu eng muhim shart: noto'g'ri kurs butun hisobni buzadi.
func TestServiceFailureReturnsError(t *testing.T) {
	srv := server(t, `{"error":"xizmat ishlamayapti"}`, http.StatusInternalServerError, nil)
	defer srv.Close()

	if _, err := New(srv.URL).Get(context.Background(), "USD", time.Time{}); err == nil {
		t.Fatal("xato kutilgan edi; taxminiy kurs qaytarilmasligi kerak")
	}
}

func TestBadJSON(t *testing.T) {
	srv := server(t, `<html>502</html>`, http.StatusOK, nil)
	defer srv.Close()

	if _, err := New(srv.URL).Get(context.Background(), "USD", time.Time{}); err == nil {
		t.Error("buzuq javobda xato kutilgan edi")
	}
}

func TestUnknownCurrency(t *testing.T) {
	srv := server(t, sample, http.StatusOK, nil)
	defer srv.Close()

	if _, err := New(srv.URL).Get(context.Background(), "XYZ", time.Time{}); err == nil {
		t.Error("noma'lum valyutada xato kutilgan edi")
	}
}

// Bitta valyuta buzuq bo'lsa, qolganlari ishlashi kerak.
func TestPartiallyBrokenResponse(t *testing.T) {
	broken := `[
	 {"Code":"840","Ccy":"USD","Nominal":"1","Rate":"12093.35","Date":"17.07.2026"},
	 {"Code":"999","Ccy":"BAD","Nominal":"1","Rate":"salom","Date":"17.07.2026"}
	]`
	srv := server(t, broken, http.StatusOK, nil)
	defer srv.Close()
	c := New(srv.URL)

	if _, err := c.Get(context.Background(), "USD", time.Time{}); err != nil {
		t.Errorf("buzuq yozuv tufayli USD ham yo'qoldi: %v", err)
	}
	if _, err := c.Get(context.Background(), "BAD", time.Time{}); err == nil {
		t.Error("buzuq yozuv qabul qilindi")
	}
}

// Kontekst bekor qilinsa, so'rov to'xtashi kerak.
func TestContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := New(srv.URL).Get(ctx, "USD", time.Time{}); err == nil {
		t.Error("bekor qilingan kontekstda xato kutilgan edi")
	}
}
