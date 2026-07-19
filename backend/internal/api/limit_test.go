package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Vaqtni boshqarib sinaymiz — haqiqiy soatga bog'lanib qolmaslik uchun.
func fixedLimiter(rate, daily int, now *time.Time) *limiter {
	return &limiter{
		ratePerMin: rate,
		dailyQuota: daily,
		seen:       map[string]*counter{},
		now:        func() time.Time { return *now },
	}
}

// Daqiqalik chegara: ruxsat berilgani o'tadi, ortiqchasi rad etiladi,
// keyingi daqiqada esa yana ochiladi.
func TestRatePerMinute(t *testing.T) {
	now := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)
	l := fixedLimiter(3, 0, &now)

	for i := 1; i <= 3; i++ {
		if ok, _, _ := l.allow("1.1.1.1"); !ok {
			t.Fatalf("%d-so'rov rad etildi; 3 tagacha ruxsat", i)
		}
	}
	ok, reason, wait := l.allow("1.1.1.1")
	if ok {
		t.Error("4-so'rov o'tib ketdi")
	}
	if !strings.Contains(reason, "tez-tez") {
		t.Errorf("sabab = %q", reason)
	}
	if wait <= 0 || wait > 61 {
		t.Errorf("kutish = %d soniya; 1..61 kutilgan", wait)
	}

	now = now.Add(time.Minute)
	if ok, _, _ := l.allow("1.1.1.1"); !ok {
		t.Error("yangi daqiqada ham rad etildi")
	}
}

// Kunlik kvota daqiqalik chegaradan qat'i nazar ishlashi kerak.
func TestDailyQuota(t *testing.T) {
	now := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)
	l := fixedLimiter(0, 2, &now)

	l.allow("2.2.2.2")
	l.allow("2.2.2.2")
	ok, reason, _ := l.allow("2.2.2.2")
	if ok {
		t.Error("kunlik kvotadan oshib ketdi")
	}
	if !strings.Contains(reason, "kunlik") {
		t.Errorf("sabab = %q", reason)
	}

	now = now.Add(24 * time.Hour)
	if ok, _, _ := l.allow("2.2.2.2"); !ok {
		t.Error("yangi kunda ham rad etildi")
	}
}

// Bir IP ning chegarasi boshqasiga ta'sir qilmasligi kerak.
func TestLimitPerIP(t *testing.T) {
	now := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)
	l := fixedLimiter(1, 0, &now)

	l.allow("1.1.1.1")
	if ok, _, _ := l.allow("1.1.1.1"); ok {
		t.Error("birinchi IP chegaradan o'tdi")
	}
	if ok, _, _ := l.allow("9.9.9.9"); !ok {
		t.Error("boshqa IP ham cheklandi")
	}
}

// Nol qiymat cheklovni o'chirishi kerak.
func TestLimitsDisabled(t *testing.T) {
	now := time.Now()
	l := fixedLimiter(0, 0, &now)
	for i := 0; i < 50; i++ {
		if ok, _, _ := l.allow("1.1.1.1"); !ok {
			t.Fatal("cheklov o'chirilgan, lekin rad etildi")
		}
	}
}

// Eski yozuvlar tozalanishi — busiz xotira cheksiz o'sardi.
func TestCleanup(t *testing.T) {
	now := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)
	l := fixedLimiter(10, 10, &now)
	l.allow("eski")

	now = now.Add(48 * time.Hour)
	l.allow("yangi")
	l.cleanup()

	if _, ok := l.seen["eski"]; ok {
		t.Error("eski yozuv tozalanmadi")
	}
	if _, ok := l.seen["yangi"]; !ok {
		t.Error("yangi yozuv o'chirib yuborildi")
	}
}

// Proksi sarlavhasi FAQAT TRUST_PROXY yoqilganda ishonilishi kerak —
// aks holda mijoz uni qalbakilashtirib chegarani aylanib o'tardi.
func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.5:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")

	if got := clientIP(r, false); got != "10.0.0.5" {
		t.Errorf("ishonchsiz proksi: %q; \"10.0.0.5\" kutilgan", got)
	}
	if got := clientIP(r, true); got != "203.0.113.7" {
		t.Errorf("ishonchli proksi: %q; \"203.0.113.7\" kutilgan", got)
	}
}

// Cheklov HTTP darajasida qo'llanishi va CORS sarlavhasi saqlanishi.
func TestChatRateLimitedOverHTTP(t *testing.T) {
	t.Setenv("RATE_PER_MIN", "1")
	h := newServer(t, "", "")

	body := `{"messages":[{"role":"user","content":"salom"}]}`
	// Birinchi so'rov cheklovdan o'tadi (keyin kalitsizlik uchun 503).
	do(t, h, http.MethodPost, "/api/chat", body)

	w, out := do(t, h, http.MethodPost, "/api/chat", body)
	wantStatus(t, w, http.StatusTooManyRequests, "ikkinchi so'rov")

	if w.Header().Get("Retry-After") == "" {
		t.Error("Retry-After sarlavhasi yo'q")
	}
	// Brauzer xato matnini o'qiy olishi uchun CORS saqlanishi shart.
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("cheklangan javobda CORS sarlavhasi yo'q")
	}
	if out["error"] == nil {
		t.Error("xato matni yo'q")
	}
}

// Kalkulyator va health cheklanmasligi kerak — ular arzon.
func TestCheapPathsNotLimited(t *testing.T) {
	t.Setenv("RATE_PER_MIN", "1")
	h := newServer(t, "", "")
	for i := 0; i < 5; i++ {
		w, _ := do(t, h, http.MethodGet, "/api/health", "")
		wantStatus(t, w, http.StatusOK, "health")
	}
}

// So'rov hajmi chegarasi.
func TestBodySizeLimit(t *testing.T) {
	t.Setenv("MAX_BODY_BYTES", "100")
	h := newServer(t, "", "")

	big := `{"query":"` + strings.Repeat("a", 500) + `"}`
	w, _ := do(t, h, http.MethodPost, "/api/hscode/search", big)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("katta so'rov status %d bilan o'tdi", w.Code)
	}
}
