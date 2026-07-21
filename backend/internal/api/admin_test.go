package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// adminServer — admin yoqilgan server (parol o'rnatilgan).
func adminServer(t *testing.T, password string) http.Handler {
	t.Helper()
	t.Setenv("ADMIN_PASSWORD", password)
	return newServer(t, "", "")
}

// login — parol bilan kiradi, sessiya cookie'sini qaytaradi.
func login(t *testing.T, h http.Handler, password string) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/admin/api/login",
		strings.NewReader(`{"password":"`+password+`"}`))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("login xato: status %d, %s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == adminCookie {
			return c
		}
	}
	t.Fatal("login sessiya cookie'sini qaytarmadi")
	return nil
}

// FAIL CLOSED: parol o'rnatilmagan bo'lsa, /admin yo'llari UMUMAN
// mavjud emas (404). Bu tasodifan himoyasiz panel ochib qo'yishning
// oldini oladi — eng muhim xavfsizlik xususiyati.
func TestAdminDisabledByDefault(t *testing.T) {
	h := newServer(t, "", "") // ADMIN_PASSWORD yo'q

	for _, path := range []string{"/admin", "/admin/api/stats", "/admin/api/data"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: parolsiz status %d; 404 kutilgan (panel mavjud bo'lmasligi kerak)", path, w.Code)
		}
	}
}

// Noto'g'ri parol — 401, sessiya berilmaydi.
func TestAdminWrongPassword(t *testing.T) {
	h := adminServer(t, "maxfiy")
	w, out := do(t, h, http.MethodPost, "/admin/api/login", `{"password":"noto'g'ri"}`)
	wantStatus(t, w, http.StatusUnauthorized, "noto'g'ri parol")
	if out["error"] == nil {
		t.Error("xato matni yo'q")
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == adminCookie && c.Value != "" {
			t.Error("noto'g'ri parolda ham sessiya berildi")
		}
	}
}

// To'g'ri parol — sessiya cookie'si xavfsiz bayroqlar bilan.
func TestAdminLoginSetsSecureCookie(t *testing.T) {
	h := adminServer(t, "maxfiy")
	c := login(t, h, "maxfiy")

	if !c.HttpOnly {
		t.Error("cookie HttpOnly emas — JS o'qiy oladi")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Error("cookie SameSite=Strict emas — CSRF xavfi")
	}
	if c.Path != "/admin" {
		t.Errorf("cookie Path = %q; /admin kutilgan", c.Path)
	}
	if len(c.Value) < 32 {
		t.Error("sessiya tokeni juda qisqa — taxmin qilinishi mumkin")
	}
}

// Himoyalangan endpoint — sessiyasiz 401.
func TestAdminProtectedNeedsSession(t *testing.T) {
	h := adminServer(t, "maxfiy")
	for _, path := range []string{"/admin/api/stats", "/admin/api/data"} {
		w, _ := do(t, h, http.MethodGet, path, "")
		wantStatus(t, w, http.StatusUnauthorized, path+" sessiyasiz")
	}
}

// Sessiya bilan — 200 va haqiqiy ma'lumot.
func TestAdminStatsWithSession(t *testing.T) {
	h := adminServer(t, "maxfiy")
	c := login(t, h, "maxfiy")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/admin/api/stats", nil)
	r.AddCookie(c)
	h.ServeHTTP(w, r)
	wantStatus(t, w, http.StatusOK, "stats sessiya bilan")

	if !strings.Contains(w.Body.String(), "uptime_seconds") {
		t.Error("statistika javobida uptime yo'q")
	}
}

// Data endpoint bazalar holatini qaytaradi.
func TestAdminData(t *testing.T) {
	h := adminServer(t, "maxfiy")
	c := login(t, h, "maxfiy")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/admin/api/data", nil)
	r.AddCookie(c)
	h.ServeHTTP(w, r)
	wantStatus(t, w, http.StatusOK, "data")

	body := w.Body.String()
	for _, want := range []string{"codes", "laws", "laws_link_coverage", "docs", "countries"} {
		if !strings.Contains(body, want) {
			t.Errorf("data javobida %q yo'q", want)
		}
	}
}

// Yaroqsiz sessiya tokeni — 401.
func TestAdminInvalidToken(t *testing.T) {
	h := adminServer(t, "maxfiy")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/admin/api/stats", nil)
	r.AddCookie(&http.Cookie{Name: adminCookie, Value: "soxta-token"})
	h.ServeHTTP(w, r)
	wantStatus(t, w, http.StatusUnauthorized, "soxta token")
}

// Chiqish sessiyani bekor qiladi.
func TestAdminLogout(t *testing.T) {
	h := adminServer(t, "maxfiy")
	c := login(t, h, "maxfiy")

	// Chiqamiz.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/admin/api/logout", nil)
	r.AddCookie(c)
	h.ServeHTTP(w, r)
	wantStatus(t, w, http.StatusOK, "chiqish")

	// Endi eski cookie ishlamasligi kerak.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/admin/api/stats", nil)
	r2.AddCookie(c)
	h.ServeHTTP(w2, r2)
	wantStatus(t, w2, http.StatusUnauthorized, "chiqishdan keyin")
}

// ---------------------------------------------------------------- birlik testlari

// Parol solishtirish to'g'ri ishlashi (doimiy vaqt logikasi buzilmasin).
func TestCheckPassword(t *testing.T) {
	a := &adminAuth{password: "to'g'ri-parol"}
	if !a.checkPassword("to'g'ri-parol") {
		t.Error("to'g'ri parol rad etildi")
	}
	if a.checkPassword("to'g'ri-paro") { // bir belgi kam
		t.Error("noto'g'ri parol qabul qilindi")
	}
	if a.checkPassword("") {
		t.Error("bo'sh parol qabul qilindi")
	}
}

// Sessiya muddati o'tsa — yaroqsiz.
func TestSessionExpiry(t *testing.T) {
	now := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)
	a := &adminAuth{
		password: "x", now: func() time.Time { return now },
		sessions: map[string]time.Time{}, tries: map[string]loginTry{},
	}
	token, err := a.newSession()
	if err != nil {
		t.Fatal(err)
	}
	if !a.valid(token) {
		t.Fatal("yangi sessiya yaroqsiz")
	}

	now = now.Add(sessionTTL + time.Minute)
	if a.valid(token) {
		t.Error("muddati o'tgan sessiya hali yaroqli")
	}
}

// Login urinishlari cheklangan — brute-force ga qarshi.
func TestLoginRateLimit(t *testing.T) {
	now := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)
	a := &adminAuth{
		password: "x", now: func() time.Time { return now },
		sessions: map[string]time.Time{}, tries: map[string]loginTry{},
	}
	for i := 0; i < loginMaxTries; i++ {
		if !a.allowLogin("1.1.1.1") {
			t.Fatalf("%d-urinish rad etildi; %d tagacha ruxsat", i+1, loginMaxTries)
		}
	}
	if a.allowLogin("1.1.1.1") {
		t.Error("chegaradan oshgan urinish o'tdi")
	}
	// Boshqa IP ta'sirlanmasligi kerak.
	if !a.allowLogin("2.2.2.2") {
		t.Error("boshqa IP ham cheklandi")
	}
	// Vaqt oynasi o'tgach yana ochiladi.
	now = now.Add(loginWindow + time.Second)
	if !a.allowLogin("1.1.1.1") {
		t.Error("yangi oynada ham rad etildi")
	}
}

// Sessiya tokenlari noyob bo'lishi kerak.
func TestSessionTokensUnique(t *testing.T) {
	a := &adminAuth{
		password: "x", now: time.Now,
		sessions: map[string]time.Time{}, tries: map[string]loginTry{},
	}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tok, err := a.newSession()
		if err != nil {
			t.Fatal(err)
		}
		if seen[tok] {
			t.Fatal("takrorlangan sessiya tokeni")
		}
		seen[tok] = true
	}
}
