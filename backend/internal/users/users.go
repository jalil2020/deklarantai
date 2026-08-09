// Package users foydalanuvchilar, parollar va rollar bilan ishlaydi.
//
// SAQLASH JOYI: JSON fayl, xotirada indeks bilan.
//
// NEGA BAZA EMAS: loyihada birorta tashqi bog'liqlik yo'q va butun
// ma'lumot (13 142 kod, 1 405 qonun parchasi, 15 112 hujjat qoidasi)
// allaqachon JSON fayllarda turadi. PostgreSQL qo'shish serverni
// o'rnatish, ulanish, migratsiya va zaxira nusxa masalalarini olib
// keladi — bir necha yuz foydalanuvchi uchun bularning hech biri
// oqlanmaydi.
//
// Chegarasi ochiq: bitta server nusxasi uchun. Ko'p nusxaga o'tilganda
// `Store` interfeysi o'zgarmaydi, faqat amalga oshirilishi almashadi —
// shuning uchun handler'lar qayta yozilmaydi.
package users

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Role — foydalanuvchi turi.
//
// Rol UCHTA narsani belgilaydi: chat javobining sukut uslubi, kunlik
// so'rov kvotasi va admin panelga kirish. Boshqa hech narsa —
// funksiyalar ro'yxati hozircha hamma uchun bir xil, va buni yashirmaslik
// kerak: ishlamaydigan "ruxsatlar" ro'yxati xavfsizlik tuyg'usini
// beradi-yu, hech narsani himoya qilmaydi.
type Role string

const (
	// Declarant — kasbiy foydalanuvchi: TIF TN, GTD grafalari, rejimlar.
	Declarant Role = "DECLARANT"
	// Business — tadbirkor: atamalarni bilmaydi, sodda javob kerak.
	Business Role = "BUSINESS"
	// Inspector — bojxona xodimi. Bugun imkoniyatlari deklarantniki
	// bilan bir xil, farqi faqat kvotada; alohida rol sifatida
	// yozilishi keyingi bo'limlar (tekshiruv jurnali) uchun kerak.
	Inspector Role = "INSPECTOR"
	// Admin — statistika va sozlamalar paneli.
	Admin Role = "ADMIN"
)

// Roles — barcha rollar (tekshirish va ro'yxat uchun).
var Roles = []Role{Declarant, Business, Inspector, Admin}

// Valid — rol ro'yxatdami.
func (r Role) Valid() bool {
	for _, x := range Roles {
		if x == r {
			return true
		}
	}
	return false
}

// ChatMode — shu rol uchun chat javobining sukut uslubi.
func (r Role) ChatMode() string {
	if r == Business {
		return "tadbirkor"
	}
	return "deklarant"
}

// DailyQuota — kuniga chat so'rovi.
//
// Har so'rov ~0,1–0,3 dollar turadi, shuning uchun kvota rolga qarab
// ajratilgan: tadbirkor kuniga bir necha savol beradi, deklarant esa
// kun bo'yi ishlaydi.
func (r Role) DailyQuota() int {
	switch r {
	case Business:
		return 30
	case Declarant:
		return 200
	case Inspector:
		return 300
	case Admin:
		return 1000
	}
	return 0
}

// Xatolar.
var (
	ErrNotFound  = errors.New("foydalanuvchi topilmadi")
	ErrExists    = errors.New("bu login band")
	ErrBadLogin  = errors.New("login yoki parol noto'g'ri")
	ErrDisabled  = errors.New("foydalanuvchi o'chirilgan")
	ErrWeakPass  = errors.New("parol kamida 8 belgi bo'lishi kerak")
	ErrBadRole   = errors.New("noma'lum rol")
	ErrBadFormat = errors.New("login kamida 4 belgi bo'lishi kerak")
)

const (
	minPassword = 8
	minLogin    = 4
	// pbkdf2 iteratsiyalari. Yuqori son parolni topishni qiyinlashtiradi;
	// 210 000 — OWASP ning PBKDF2-SHA256 uchun tavsiyasi.
	iterations = 210_000
	keyLen     = 32
	saltLen    = 16
)

// User — foydalanuvchi yozuvi.
//
// Parol maydonlari JSON'ga chiqadi (faylga yozish uchun), lekin
// HTTP javobiga hech qachon bu tur berilmaydi — buning uchun Public()
// bor.
type User struct {
	ID      string    `json:"id"`
	Login   string    `json:"login"` // telefon yoki e-pochta, kichik harfda
	Name    string    `json:"name"`
	Role    Role      `json:"role"`
	Created time.Time `json:"created"`

	Hash string `json:"hash"`
	Salt string `json:"salt"`

	// TokenVer — chiqishda oshiriladi. Token ichida ham shu raqam bor,
	// mos kelmasa token qabul qilinmaydi.
	//
	// NEGA KERAK: token IMZOLANGAN va serverda saqlanmaydi, ya'ni uni
	// "o'chirish" mumkin emas. Versiya bo'lmasa "chiqish" tugmasi
	// faqat brauzerdagi nusxani o'chirardi — o'g'irlangan token esa
	// muddati tugagunicha ishlayverardi.
	TokenVer int `json:"token_ver"`

	Disabled bool `json:"disabled,omitempty"`
}

// PublicUser — tashqariga beriladigan ko'rinish. Parol yo'q.
type PublicUser struct {
	ID      string    `json:"id"`
	Login   string    `json:"login"`
	Name    string    `json:"name"`
	Role    Role      `json:"role"`
	Created time.Time `json:"created"`
	Quota   int       `json:"daily_quota"`
	Mode    string    `json:"chat_mode"`
}

func (u *User) Public() PublicUser {
	return PublicUser{
		ID: u.ID, Login: u.Login, Name: u.Name, Role: u.Role,
		Created: u.Created, Quota: u.Role.DailyQuota(), Mode: u.Role.ChatMode(),
	}
}

// Store — foydalanuvchilar ombori.
//
// Interfeys ATAYLAB: bazaga o'tilganda handler'lar o'zgarmasin.
type Store interface {
	Create(login, password, name string, role Role) (*User, error)
	Authenticate(login, password string) (*User, error)
	ByID(id string) (*User, error)
	Count() int
	List() []PublicUser
	SetRole(id string, role Role) error
	SetDisabled(id string, disabled bool) error
	BumpToken(id string) error
}

// FileStore — JSON faylga yozadigan ombor.
type FileStore struct {
	path string

	mu      sync.RWMutex
	byID    map[string]*User
	byLogin map[string]*User
	now     func() time.Time
}

// Load — faylni o'qiydi. Fayl bo'lmasa bo'sh ombor qaytariladi:
// birinchi ro'yxatdan o'tish uni o'zi yaratadi.
func Load(path string) (*FileStore, error) {
	s := &FileStore{
		path:    path,
		byID:    map[string]*User{},
		byLogin: map[string]*User{},
		now:     time.Now,
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var list []*User
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, u := range list {
		s.byID[u.ID] = u
		s.byLogin[u.Login] = u
	}
	return s, nil
}

// saveLocked — faylga yozadi. mu ushlab turilgan holda chaqiriladi.
//
// Avval vaqtinchalik faylga yozib, keyin almashtiriladi: yozish
// o'rtasida uzilib qolsa, eski fayl butun holicha qoladi.
func (s *FileStore) saveLocked() error {
	list := make([]*User, 0, len(s.byID))
	for _, u := range s.byID {
		list = append(list, u)
	}
	raw, err := json.MarshalIndent(list, "", " ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := s.path + ".tmp"
	// 0600 — faylda parol xeshlari bor.
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// NormalizeLogin — login bir xil ko'rinishga keltiriladi.
//
// Telefon raqamidagi bo'shliq, qavs va chiziqchalar olib tashlanadi:
// "+998 90 123-45-67" va "+998901234567" BIR XIL foydalanuvchi bo'lishi
// kerak, aks holda odam o'z akkauntiga kira olmay qolardi.
func NormalizeLogin(login string) string {
	login = strings.ToLower(strings.TrimSpace(login))
	if strings.ContainsAny(login, "@") {
		return login // e-pochta — tegmaymiz
	}
	var b strings.Builder
	for _, r := range login {
		if r >= '0' && r <= '9' || r == '+' {
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		return b.String()
	}
	return login
}

func (s *FileStore) Create(login, password, name string, role Role) (*User, error) {
	login = NormalizeLogin(login)
	if len([]rune(login)) < minLogin {
		return nil, ErrBadFormat
	}
	if len([]rune(password)) < minPassword {
		return nil, ErrWeakPass
	}
	if !role.Valid() {
		return nil, ErrBadRole
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byLogin[login]; ok {
		return nil, ErrExists
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	hash, err := derive(password, salt)
	if err != nil {
		return nil, err
	}
	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}

	u := &User{
		ID:      hex.EncodeToString(id),
		Login:   login,
		Name:    strings.TrimSpace(name),
		Role:    role,
		Created: s.now(),
		Hash:    hex.EncodeToString(hash),
		Salt:    hex.EncodeToString(salt),
	}
	s.byID[u.ID] = u
	s.byLogin[u.Login] = u
	if err := s.saveLocked(); err != nil {
		// Saqlanmagan foydalanuvchi xotirada qolmasin: server qayta
		// ishga tushganda u yo'qolardi va odam "akkauntim yo'qoldi"
		// degan holatga tushardi.
		delete(s.byID, u.ID)
		delete(s.byLogin, u.Login)
		return nil, err
	}
	return u, nil
}

func (s *FileStore) Authenticate(login, password string) (*User, error) {
	login = NormalizeLogin(login)

	s.mu.RLock()
	u := s.byLogin[login]
	s.mu.RUnlock()

	if u == nil {
		// Foydalanuvchi yo'q bo'lsa ham parol hisoblanadi: aks holda
		// javob vaqti bo'yicha qaysi loginlar mavjudligini aniqlash
		// mumkin bo'lardi.
		_, _ = derive(password, make([]byte, saltLen))
		return nil, ErrBadLogin
	}
	salt, err := hex.DecodeString(u.Salt)
	if err != nil {
		return nil, ErrBadLogin
	}
	got, err := derive(password, salt)
	if err != nil {
		return nil, err
	}
	want, err := hex.DecodeString(u.Hash)
	if err != nil {
		return nil, ErrBadLogin
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return nil, ErrBadLogin
	}
	if u.Disabled {
		return nil, ErrDisabled
	}
	return u, nil
}

func (s *FileStore) ByID(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u := s.byID[id]
	if u == nil {
		return nil, ErrNotFound
	}
	if u.Disabled {
		return nil, ErrDisabled
	}
	return u, nil
}

func (s *FileStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byID)
}

func (s *FileStore) List() []PublicUser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PublicUser, 0, len(s.byID))
	for _, u := range s.byID {
		out = append(out, u.Public())
	}
	return out
}

func (s *FileStore) SetRole(id string, role Role) error {
	if !role.Valid() {
		return ErrBadRole
	}
	return s.update(id, func(u *User) {
		u.Role = role
	})
}

func (s *FileStore) SetDisabled(id string, disabled bool) error {
	return s.update(id, func(u *User) {
		u.Disabled = disabled
		// O'chirilgan foydalanuvchining amaldagi tokenlari ham
		// kuchini yo'qotishi kerak.
		if disabled {
			u.TokenVer++
		}
	})
}

// BumpToken — chiqish: amaldagi barcha tokenlarni bekor qiladi.
func (s *FileStore) BumpToken(id string) error {
	return s.update(id, func(u *User) { u.TokenVer++ })
}

func (s *FileStore) update(id string, fn func(*User)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.byID[id]
	if u == nil {
		return ErrNotFound
	}
	fn(u)
	return s.saveLocked()
}

func derive(password string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, iterations, keyLen)
}
