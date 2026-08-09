package api

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"deklarant-ai/backend/internal/users"
)

//go:embed admin.html
var adminHTML []byte

// adminPage — panel sahifasi. Autentifikatsiya JS ichida: sahifa har
// doim beriladi, lekin ma'lumot API si sessiyasiz 401 qaytaradi.
func (s *Server) adminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Panel boshqa saytga <iframe> ichida joylashtirilmasin (clickjacking).
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
	_, _ = w.Write(adminHTML)
}

// Admin paneli — parol bilan himoyalangan, statistika va ma'lumot ko'rish.
//
// XAVFSIZLIK PRINSIPLARI:
//
//  1. FAIL CLOSED: ADMIN_PASSWORD o'rnatilmagan bo'lsa, panel UMUMAN
//     mavjud emas — barcha /admin yo'llari 404 qaytaradi. Bu tasodifan
//     himoyasiz panel ochib qo'yishning oldini oladi.
//  2. Parol solishtirish — DOIMIY VAQTDA (crypto/subtle), timing hujumiga
//     qarshi.
//  3. Sessiya tokeni — crypto/rand dan, taxmin qilib bo'lmaydi.
//  4. Cookie — HttpOnly (JS o'qiy olmaydi), SameSite=Strict (CSRF ga
//     qarshi), Path=/admin.
//  5. Login urinishlari cheklangan — parolni brute-force qilishni
//     sekinlashtiradi.

const (
	adminCookie   = "admin_session"
	sessionTTL    = 8 * time.Hour
	loginMaxTries = 5 // bitta IP dan daqiqasiga
	loginWindow   = time.Minute
)

// adminAuth — sessiyalar va login cheklovi.
type adminAuth struct {
	password string // bo'sh bo'lsa panel o'chirilgan
	now      func() time.Time

	mu       sync.Mutex
	sessions map[string]time.Time // token → tugash vaqti
	tries    map[string]loginTry  // IP → urinishlar
}

type loginTry struct {
	count  int
	window time.Time
}

func newAdminAuth() *adminAuth {
	return &adminAuth{
		password: os.Getenv("ADMIN_PASSWORD"),
		now:      time.Now,
		sessions: map[string]time.Time{},
		tries:    map[string]loginTry{},
	}
}

// enabled — panel yoqilganmi (parol o'rnatilganmi).
func (a *adminAuth) enabled() bool { return a.password != "" }

// checkPassword — parolni doimiy vaqtda solishtiradi.
func (a *adminAuth) checkPassword(got string) bool {
	// subtle.ConstantTimeCompare uzunliklar teng bo'lmasa 0 qaytaradi,
	// lekin bu ham vaqt jihatidan xavfsiz (uzunlik sirni oshkor qilmaydi).
	return subtle.ConstantTimeCompare([]byte(got), []byte(a.password)) == 1
}

// allowLogin — bu IP dan login urinishiga ruxsat bormi.
func (a *adminAuth) allowLogin(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	t := a.tries[ip]
	if now.Sub(t.window) > loginWindow {
		t = loginTry{window: now}
	}
	if t.count >= loginMaxTries {
		return false
	}
	t.count++
	a.tries[ip] = t
	return true
}

// newSession — yangi sessiya yaratadi va tokenni qaytaradi.
func (a *adminAuth) newSession() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	a.mu.Lock()
	a.sessions[token] = a.now().Add(sessionTTL)
	// Eskirgan sessiyalarni tozalaymiz — xotira cheksiz o'smasin.
	for tok, exp := range a.sessions {
		if a.now().After(exp) {
			delete(a.sessions, tok)
		}
	}
	a.mu.Unlock()
	return token, nil
}

// valid — token amaldagi sessiyaga tegishlimi.
func (a *adminAuth) valid(token string) bool {
	if token == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[token]
	if !ok || a.now().After(exp) {
		return false
	}
	return true
}

func (a *adminAuth) logout(token string) {
	a.mu.Lock()
	delete(a.sessions, token)
	a.mu.Unlock()
}

// ---------------------------------------------------------------- handlerlar

// adminRoutes — /admin yo'llarini ro'yxatga oladi.
//
// Panel o'chirilgan bo'lsa (parol yo'q), umuman ro'yxatga olinmaydi —
// yo'llar mavjud emas.
func (s *Server) adminRoutes(mux *http.ServeMux) {
	if !s.admin.enabled() {
		return
	}
	mux.HandleFunc("GET /admin", s.adminPage)
	mux.HandleFunc("POST /admin/api/login", s.adminLogin)
	mux.HandleFunc("POST /admin/api/logout", s.adminLogout)
	// Himoyalangan API — har biri protectAdmin bilan o'raladi.
	mux.Handle("GET /admin/api/stats", s.protectAdmin(s.adminStats))
	mux.Handle("GET /admin/api/data", s.protectAdmin(s.adminData))
	mux.Handle("GET /admin/api/settings", s.protectAdmin(s.adminSettings))
}

// protectAdmin — sessiyani tekshiradi; bo'lmasa 401.
//
// Ikki yo'l: panel paroli (cookie) yoki ADMIN rolidagi foydalanuvchi
// tokeni. Ikkinchisi kerak, chunki parol HAMMA admin uchun bitta —
// kim nima qilgani bilinmaydi va odam ishdan ketganda parolni
// almashtirishga to'g'ri keladi.
func (s *Server) protectAdmin(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(adminCookie); err == nil && s.admin.valid(c.Value) {
			h(w, r)
			return
		}
		if c, ok := s.identify(r); ok && c.user != nil && c.user.Role == users.Admin {
			h(w, r)
			return
		}
		writeErr(w, http.StatusUnauthorized, "avtorizatsiya kerak")
	})
}

func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r, os.Getenv("TRUST_PROXY") == "1")
	if !s.admin.allowLogin(ip) {
		writeErr(w, http.StatusTooManyRequests, "juda ko'p urinish, biroz kuting")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "so'rovni o'qib bo'lmadi")
		return
	}
	if !s.admin.checkPassword(req.Password) {
		writeErr(w, http.StatusUnauthorized, "parol noto'g'ri")
		return
	}

	token, err := s.admin.newSession()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "sessiya yaratib bo'lmadi")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookie,
		Value:    token,
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isHTTPS(r),
		MaxAge:   int(sessionTTL.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) adminLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(adminCookie); err == nil {
		s.admin.logout(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: adminCookie, Value: "", Path: "/admin", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// adminStats — foydalanish statistikasi.
func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.stats.snapshot(time.Now()))
}

// adminData — bazalar holati (ma'lumot ko'rish).
func (s *Server) adminData(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"codes": s.codes.Meta()}
	if s.laws != nil {
		lm := s.laws.Meta()
		out["laws"] = lm
		out["laws_link_coverage"] = s.lawsLinkCoverage()
	}
	if s.docs != nil {
		out["docs"] = s.docs.Meta()
		out["exemption_programs"] = len(s.docs.Programs())
	}
	if s.countries != nil {
		out["countries"] = s.countries.Meta()
	}
	writeJSON(w, http.StatusOK, out)
}

// isHTTPS — so'rov xavfsiz ulanish orqali kelganmi.
// Secure cookie faqat HTTPS da qo'yiladi; localhost da HTTP ham ruxsat.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	// Proksi orqasida (TRUST_PROXY) X-Forwarded-Proto tekshiriladi.
	if os.Getenv("TRUST_PROXY") == "1" &&
		strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	return false
}
