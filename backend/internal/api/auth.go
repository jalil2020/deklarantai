package api

// Mijozni tanish — qimmat (AI) endpointlarga kirish uchun.
//
// NEGA KERAK: chat so'rovi pul turadi. Ilgari `/api/chat` ni internetdagi
// istalgan skript to'g'ridan-to'g'ri chaqira olardi va yagona to'siq IP
// bo'yicha limit edi — u IP almashtirilsa yoki mobil tarmoqdan kelinsa
// oson aylanib o'tiladi.
//
// BU AUTENTIFIKATSIYA EMAS. Anonim tokenni istalgan odam bir chaqiruv
// bilan olishi mumkin — ya'ni bu qat'iy to'siq emas, TEZLIK CHEKLOVI:
//
//   - boshqa saytlar sizning API ingizni o'z sahifasiga ulay olmaydi;
//     avval qo'l siltash (handshake) qilish kerak
//   - har bir mijozga BARQAROR belgi beriladi, shuning uchun limit endi
//     IP ga emas, mijozga bog'lanadi (bitta ofisdagi 20 kishi bir-birining
//     kvotasini yeb qo'ymaydi)
//   - akkauntlar kelganda o'sha joyga foydalanuvchi seansi qo'yiladi —
//     endpointlar va mijozlar o'zgarmaydi
//
// Ikki turdagi belgi bir xil sarlavhada (`X-API-Key`) yuboriladi:
//
//	API kaliti    — API_KEYS da e'lon qilingan, muddatsiz. Server-server,
//	                mobil ilova va hamkorlar uchun.
//	Anonim token  — POST /api/session dan olinadi, HMAC bilan imzolangan,
//	                muddati bor. Brauzer uchun.
//
// Anonim token ATAYLAB xotirada saqlanmaydi (imzo yetarli): aks holda
// har bir tashrif buyuruvchi uchun yozuv paydo bo'lardi va uni cheksiz
// yaratish serverning xotirasini to'ldirish yo'liga aylanardi.

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"deklarant-ai/backend/internal/users"
)

const (
	// clientHeader — kalit ham, token ham shu sarlavhada keladi.
	clientHeader = "X-API-Key"
	// tokenVersion — format o'zgarsa, eski tokenlarni ajratish uchun.
	tokenVersion = "v1"
	// defaultTokenTTL — anonim token muddati. Bir kunlik ish seansiga
	// yetadi; uzunroq qilishning ma'nosi yo'q, mijoz uni bemalol
	// yangilay oladi.
	defaultTokenTTL = 24 * time.Hour
)

// anonName — anonim mijozning statistikadagi nomi.
const anonName = "anonim"

// apiKey — e'lon qilingan kalit.
type apiKey struct {
	name   string
	secret string
}

// clientAuth — kalitlar ro'yxati va token imzolagich.
type clientAuth struct {
	keys   []apiKey
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func newClientAuth() *clientAuth {
	a := &clientAuth{
		keys: parseAPIKeys(os.Getenv("API_KEYS")),
		ttl:  defaultTokenTTL,
		now:  time.Now,
	}
	if d, err := time.ParseDuration(os.Getenv("CLIENT_TOKEN_TTL")); err == nil && d > 0 {
		a.ttl = d
	}
	// Sir berilmagan bo'lsa tasodifiy yasaymiz. Bu ishlaydi, lekin server
	// qayta ishga tushganda barcha tokenlar kuchini yo'qotadi — mijozlar
	// yangisini oladi. Bir nechta nusxa ishlaganda sir BIR XIL bo'lishi
	// shart, aks holda token boshqa nusxada tan olinmaydi.
	if s := os.Getenv("CLIENT_TOKEN_SECRET"); s != "" {
		a.secret = []byte(s)
	} else {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("token siri uchun tasodifiy son olinmadi: " + err.Error())
		}
		a.secret = b
	}
	return a
}

// parseAPIKeys — "nom:sir,nom2:sir2" ko'rinishidagi ro'yxat.
//
// Noto'g'ri yozilgan bo'lak jimgina tashlab yuboriladi: bitta xato
// butun ro'yxatni ishdan chiqarmasin.
func parseAPIKeys(raw string) []apiKey {
	var out []apiKey
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, secret, ok := strings.Cut(part, ":")
		name, secret = strings.TrimSpace(name), strings.TrimSpace(secret)
		if !ok || name == "" || secret == "" {
			continue
		}
		out = append(out, apiKey{name: name, secret: secret})
	}
	return out
}

// issue — yangi anonim token.
func (a *clientAuth) issue() (token string, expires time.Time) {
	expires = a.now().Add(a.ttl)
	payload := tokenVersion + "." + strconv.FormatInt(expires.Unix(), 10)
	return payload + "." + a.sign(payload), expires
}

func (a *clientAuth) sign(payload string) string {
	m := hmac.New(sha256.New, a.secret)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// caller — so'rov egasi.
//
// Uch xil bo'lishi mumkin va farqi MUHIM: foydalanuvchi kvota va rolga
// ega, API kaliti — yo'q, anonim esa faqat chatdan tashqari yo'llarga
// kiradi.
type caller struct {
	kind string      // "user" | "key" | "anon"
	name string      // statistika va limit uchun
	user *users.User // faqat kind == "user"
}

// identify — so'rov egasini aniqlaydi.
func (s *Server) identify(r *http.Request) (caller, bool) {
	got := strings.TrimSpace(r.Header.Get(clientHeader))
	if got == "" {
		return caller{}, false
	}
	// Foydalanuvchi tokeni birinchi tekshiriladi: u eng aniq belgi.
	if s.users != nil {
		if u, ok := s.userToken(got); ok {
			return caller{kind: "user", name: "u:" + u.ID, user: u}, true
		}
	}
	if name, ok := s.client.identify(got); ok {
		kind := "key"
		if name == anonName {
			kind = "anon"
		}
		return caller{kind: kind, name: name}, true
	}
	return caller{}, false
}

// identify — kalit yoki anonim token bo'yicha nom.
//
// Nom statistika va limit uchun ishlatiladi; "" bo'lsa mijoz tanilmadi.
func (a *clientAuth) identify(got string) (string, bool) {
	if got == "" {
		return "", false
	}
	// Kalitlar doimiy vaqtda solishtiriladi va sikl ERTA TUGATILMAYDI:
	// mos kelgan joyda to'xtash javob vaqti orqali kalitni taxmin
	// qilishga yo'l ochardi.
	name := ""
	for _, k := range a.keys {
		if subtle.ConstantTimeCompare([]byte(k.secret), []byte(got)) == 1 {
			name = k.name
		}
	}
	if name != "" {
		return name, true
	}
	if a.validToken(got) {
		return anonName, true
	}
	return "", false
}

// validToken — imzo va muddatni tekshiradi.
func (a *clientAuth) validToken(tok string) bool {
	i := strings.LastIndexByte(tok, '.')
	if i <= 0 {
		return false
	}
	payload, sig := tok[:i], tok[i+1:]
	if subtle.ConstantTimeCompare([]byte(a.sign(payload)), []byte(sig)) != 1 {
		return false
	}
	// Imzo to'g'ri — endi muddat. Tartib muhim: imzo tekshirilmasdan
	// muddatni o'qish qalbaki tokenning ichiga ishonish bo'lardi.
	ver, unix, ok := strings.Cut(payload, ".")
	if !ok || ver != tokenVersion {
		return false
	}
	exp, err := strconv.ParseInt(unix, 10, 64)
	if err != nil {
		return false
	}
	return a.now().Before(time.Unix(exp, 0))
}

// ---- HTTP ----

// handleSession — anonim token beradi.
//
// Ochiq endpoint: himoyasi — IP bo'yicha tezlik chegarasi (withLimits).
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	token, exp := s.client.issue()
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_at": exp.UTC().Format(time.RFC3339),
		"header":     clientHeader,
	})
}

// requireClient — tanilmagan mijozni qaytaradi.
func (s *Server) requireClient(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.identify(r); !ok {
			// Xato matni yo'lni AYTADI: mijoz nima qilishini bilsin,
			// hujjatni qidirib yurmasin.
			writeErr(w, http.StatusUnauthorized,
				"mijoz tanilmadi: POST /api/session dan token oling va uni "+
					clientHeader+" sarlavhasida yuboring")
			return
		}
		h(w, r)
	})
}

// requireUser — chat uchun: ANONIM token yetarli emas.
//
// NEGA QAT'IYROQ: anonim tokenni istalgan skript bir chaqiruv bilan
// oladi, ya'ni u xarajatni hech kimga bog'lamaydi. Foydalanuvchi
// bo'lsa — kvota, rol va statistika unga tegishli bo'ladi.
//
// API kaliti ham o'tadi: mobil ilova va hamkorlar server-server
// ulanishida foydalanuvchi seansiga ega emas.
func (s *Server) requireUser(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Foydalanuvchilar ombori ulanmagan bo'lsa (testlar) — eski
		// qoida: mijoz belgisi yetarli.
		if s.users == nil {
			s.requireClient(h).ServeHTTP(w, r)
			return
		}
		c, ok := s.identify(r)
		if !ok || c.kind == "anon" {
			writeErr(w, http.StatusUnauthorized,
				"kirish talab qilinadi: POST /api/auth/login orqali kiring")
			return
		}
		h(w, r)
	})
}

// clientOK — mijoz tanilganmi (rad etmasdan tekshirish uchun).
func (s *Server) clientOK(r *http.Request) bool {
	_, ok := s.identify(r)
	return ok
}
