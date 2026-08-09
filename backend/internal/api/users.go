package api

// Foydalanuvchi seansi: ro'yxatdan o'tish, kirish, chiqish.
//
// TOKEN SERVERDA SAQLANMAYDI — u imzolangan. Shu tufayli server qayta
// ishga tushganda seanslar yo'qolmaydi va bir necha nusxa ishlaganda
// umumiy sessiya ombori kerak bo'lmaydi (sir bir xil bo'lsa).
//
// "Chiqish" esa shundan qiyinlashadi: imzolangan tokenni serverdan
// o'chirib bo'lmaydi. Yechim — foydalanuvchidagi VERSIYA raqami: u
// token ichida ham bor va chiqishda oshiriladi, ya'ni eski tokenlar
// darrov kuchini yo'qotadi.

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"deklarant-ai/backend/internal/chat"
	"deklarant-ai/backend/internal/users"
)

const (
	userTokenVersion = "u1"
	// defaultUserTTL — seans muddati. Anonim tokendan uzunroq: bu
	// odamning ish seansi, har kuni qayta kirishga majburlash
	// noqulaylikdan boshqa narsa bermaydi.
	defaultUserTTL = 30 * 24 * time.Hour
)

// SetUsers — foydalanuvchilar omborini ulaydi.
//
// New() ga parametr qilib qo'shilmadi: ombor ixtiyoriy va uni
// bermaydigan chaqiruv joylari (testlar) o'zgarishsiz qolishi kerak.
func (s *Server) SetUsers(store users.Store) { s.users = store }

// issueUser — foydalanuvchi uchun imzolangan token.
//
// Ichida: versiya, ID, token versiyasi, tugash vaqti.
func (s *Server) issueUser(u *users.User) (string, time.Time) {
	exp := s.client.now().Add(defaultUserTTL)
	payload := strings.Join([]string{
		userTokenVersion, u.ID, strconv.Itoa(u.TokenVer), strconv.FormatInt(exp.Unix(), 10),
	}, ".")
	return payload + "." + s.client.sign(payload), exp
}

// userToken — tokenni tekshirib, foydalanuvchini qaytaradi.
func (s *Server) userToken(tok string) (*users.User, bool) {
	i := strings.LastIndexByte(tok, '.')
	if i <= 0 {
		return nil, false
	}
	payload, sig := tok[:i], tok[i+1:]
	if !strings.HasPrefix(payload, userTokenVersion+".") {
		return nil, false
	}
	// Imzo BIRINCHI: tekshirilmagan tokenning ichidagi ma'lumotga
	// ishonib bo'lmaydi.
	if s.client.sign(payload) != sig {
		return nil, false
	}
	parts := strings.Split(payload, ".")
	if len(parts) != 4 {
		return nil, false
	}
	exp, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || !s.client.now().Before(time.Unix(exp, 0)) {
		return nil, false
	}
	u, err := s.users.ByID(parts[1])
	if err != nil {
		return nil, false // o'chirilgan yoki yo'q
	}
	// Chiqishdan keyin eski token o'tmasligi kerak.
	if ver, err := strconv.Atoi(parts[2]); err != nil || ver != u.TokenVer {
		return nil, false
	}
	return u, true
}

// ---- HTTP ----

func (s *Server) userRoutes(mux *http.ServeMux) {
	if s.users == nil {
		return
	}
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleMe)
	mux.HandleFunc("GET /api/auth/roles", s.handleRoles)
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
	Role     string `json:"role,omitempty"`
}

// authError — foydalanuvchi xatosini HTTP statusiga o'giradi.
//
// DIQQAT: "login topilmadi" va "parol noto'g'ri" BIR XIL javob beradi
// (ErrBadLogin) — aks holda qaysi loginlar ro'yxatda borligini bilib
// olish mumkin bo'lardi.
func authError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, users.ErrExists):
		writeErr(w, http.StatusConflict, "bu login allaqachon ro'yxatdan o'tgan")
	case errors.Is(err, users.ErrBadLogin):
		writeErr(w, http.StatusUnauthorized, "login yoki parol noto'g'ri")
	case errors.Is(err, users.ErrDisabled):
		writeErr(w, http.StatusForbidden, "foydalanuvchi o'chirilgan")
	case errors.Is(err, users.ErrWeakPass):
		writeErr(w, http.StatusBadRequest, "parol kamida 8 belgi bo'lishi kerak")
	case errors.Is(err, users.ErrBadFormat):
		writeErr(w, http.StatusBadRequest, "login kamida 4 belgi bo'lishi kerak")
	case errors.Is(err, users.ErrBadRole):
		writeErr(w, http.StatusBadRequest, "noma'lum rol")
	default:
		writeErr(w, http.StatusInternalServerError, "saqlab bo'lmadi: "+err.Error())
	}
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "so'rovni o'qib bo'lmadi")
		return
	}

	role := users.Role(strings.ToUpper(strings.TrimSpace(req.Role)))
	if role == "" {
		role = users.Declarant
	}
	// ADMIN o'zini o'zi tayinlay olmaydi — aks holda ochiq ro'yxatdan
	// o'tish orqali kim xohlasa admin bo'lib olardi.
	//
	// Istisno: BIRINCHI foydalanuvchi. Aks holda yangi o'rnatmada
	// birorta admin bo'lmasdi va rol tayinlash imkoni yo'q edi.
	if role == users.Admin && s.users.Count() > 0 {
		writeErr(w, http.StatusForbidden,
			"ADMIN roli ro'yxatdan o'tishda berilmaydi — mavjud admin tayinlaydi")
		return
	}

	u, err := s.users.Create(req.Login, req.Password, req.Name, role)
	if err != nil {
		authError(w, err)
		return
	}
	s.writeSession(w, u, http.StatusCreated)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "so'rovni o'qib bo'lmadi")
		return
	}
	u, err := s.users.Authenticate(req.Login, req.Password)
	if err != nil {
		authError(w, err)
		return
	}
	s.writeSession(w, u, http.StatusOK)
}

// handleLogout — barcha qurilmalardagi tokenlarni bekor qiladi.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	c, ok := s.identify(r)
	if !ok || c.user == nil {
		writeErr(w, http.StatusUnauthorized, "kirilmagan")
		return
	}
	if err := s.users.BumpToken(c.user.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	c, ok := s.identify(r)
	if !ok || c.user == nil {
		writeErr(w, http.StatusUnauthorized, "kirilmagan")
		return
	}
	writeJSON(w, http.StatusOK, c.user.Public())
}

// handleRoles — mijoz rollar ro'yxatini qattiq yozib qo'ymasin.
func (s *Server) handleRoles(w http.ResponseWriter, _ *http.Request) {
	type roleInfo struct {
		Role  users.Role `json:"role"`
		Mode  string     `json:"chat_mode"`
		Quota int        `json:"daily_quota"`
		/* Ro'yxatdan o'tishda tanlash mumkinmi. */
		SelfSignup bool `json:"self_signup"`
	}
	out := make([]roleInfo, 0, len(users.Roles))
	for _, r := range users.Roles {
		out = append(out, roleInfo{
			Role: r, Mode: r.ChatMode(), Quota: r.DailyQuota(),
			SelfSignup: r != users.Admin,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": out})
}

// modeFor — chat javobining uslubi.
//
// So'rovda ochiq ko'rsatilgan rejim USTUN: foydalanuvchi bir savolga
// sodda javob olishni xohlashi mumkin. Ko'rsatilmagan bo'lsa — rolining
// sukut uslubi (BUSINESS → tadbirkor, qolgani → deklarant).
//
// DIQQAT: rejim faqat USLUBNI o'zgartiradi. Stavkalar, hisob-kitob va
// ogohlantirishlar ikkala uslubda bir xil — buni backend testi
// (TestBothModesKeepWarnings) qo'riqlaydi.
func (s *Server) modeFor(r *http.Request, requested string) chat.Mode {
	if strings.TrimSpace(requested) != "" {
		return chat.ParseMode(requested)
	}
	if c, ok := s.identify(r); ok && c.user != nil {
		return chat.ParseMode(c.user.Role.ChatMode())
	}
	return chat.ParseMode("")
}

// writeSession — token va foydalanuvchi ma'lumotini qaytaradi.
func (s *Server) writeSession(w http.ResponseWriter, u *users.User, status int) {
	token, exp := s.issueUser(u)
	writeJSON(w, status, map[string]any{
		"token":      token,
		"expires_at": exp.UTC().Format(time.RFC3339),
		"header":     clientHeader,
		"user":       u.Public(),
	})
}
