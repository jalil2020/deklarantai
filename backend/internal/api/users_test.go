package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"deklarant-ai/backend/internal/users"
)

// userServer — foydalanuvchilar ombori ULANGAN server.
func userServer(t *testing.T) (http.Handler, users.Store) {
	t.Helper()
	store, err := users.Load(filepath.Join(t.TempDir(), "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	// buildServer Routes() ni qaytaradi, bizga esa Server kerak —
	// shuning uchun qayta yig'amiz.
	srv := newServerObj(t, "sk-test", "")
	srv.SetUsers(store)
	return srv.Routes(), store
}

// post — JSON so'rov, belgisi bilan.
func post(t *testing.T, h http.Handler, path, body, key string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if key != "" {
		r.Header.Set(clientHeader, key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var out map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &out)
	}
	return w, out
}

func get(t *testing.T, h http.Handler, path, key string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if key != "" {
		r.Header.Set(clientHeader, key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var out map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &out)
	}
	return w, out
}

const regBody = `{"login":"+998901234567","password":"kuchli-parol","name":"Jalil","role":"DECLARANT"}`

func TestRegisterAndLogin(t *testing.T) {
	h, _ := userServer(t)

	w, out := post(t, h, "/api/auth/register", regBody, "")
	wantStatus(t, w, http.StatusCreated, "ro'yxatdan o'tish")
	token, _ := out["token"].(string)
	if token == "" {
		t.Fatal("token berilmadi")
	}
	user, _ := out["user"].(map[string]any)
	if user["role"] != "DECLARANT" || user["daily_quota"] != float64(200) {
		t.Errorf("foydalanuvchi ma'lumoti: %+v", user)
	}
	// Parol javobda BO'LMASLIGI shart.
	if strings.Contains(w.Body.String(), "kuchli-parol") || strings.Contains(w.Body.String(), "hash") {
		t.Error("javobda parol yoki xesh bor")
	}

	// Takroriy login — 409.
	w, _ = post(t, h, "/api/auth/register", regBody, "")
	wantStatus(t, w, http.StatusConflict, "takroriy login")

	// Kirish.
	w, out = post(t, h, "/api/auth/login", `{"login":"+998901234567","password":"kuchli-parol"}`, "")
	wantStatus(t, w, http.StatusOK, "kirish")
	if out["token"] == "" {
		t.Error("kirishda token berilmadi")
	}

	// Noto'g'ri parol.
	w, _ = post(t, h, "/api/auth/login", `{"login":"+998901234567","password":"boshqa-parol"}`, "")
	wantStatus(t, w, http.StatusUnauthorized, "noto'g'ri parol")
}

func TestMeAndLogout(t *testing.T) {
	h, _ := userServer(t)
	_, out := post(t, h, "/api/auth/register", regBody, "")
	token := out["token"].(string)

	w, me := get(t, h, "/api/auth/me", token)
	wantStatus(t, w, http.StatusOK, "me")
	if me["login"] != "+998901234567" {
		t.Errorf("me: %+v", me)
	}

	// Chiqishdan keyin AYNAN SHU token ishlamasligi kerak — token
	// serverda saqlanmagani uchun buni versiya raqami ta'minlaydi.
	w, _ = post(t, h, "/api/auth/logout", "", token)
	wantStatus(t, w, http.StatusOK, "chiqish")

	w, _ = get(t, h, "/api/auth/me", token)
	wantStatus(t, w, http.StatusUnauthorized, "chiqishdan keyingi token")
}

// Eng muhim tekshiruv: chat kirishsiz ishlamasligi kerak.
func TestChatRequiresLogin(t *testing.T) {
	h, _ := userServer(t)

	// Anonim token — chatga YETARLI EMAS.
	w, sess := post(t, h, "/api/session", "", "")
	wantStatus(t, w, http.StatusOK, "anonim token")
	anon := sess["token"].(string)

	for _, path := range []string{"/api/chat", "/api/chat/stream"} {
		w, _ := post(t, h, path, chatBody, anon)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s anonim token bilan: status %d; 401 kutilgan", path, w.Code)
		}
		w, _ = post(t, h, path, chatBody, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s belgisiz: status %d; 401 kutilgan", path, w.Code)
		}
	}

	// Kirgan foydalanuvchi o'tishi kerak (AI kaliti soxta bo'lgani uchun
	// 401 dan boshqa xato chiqadi — bu boshqa bosqich).
	_, out := post(t, h, "/api/auth/register", regBody, "")
	token := out["token"].(string)
	w, _ = post(t, h, "/api/chat", chatBody, token)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("kirgan foydalanuvchi ham rad etildi: %s", w.Body.String())
	}
}

// Rol chat uslubini belgilaydi.
func TestRoleSetsChatMode(t *testing.T) {
	h, _ := userServer(t)

	cases := map[string]string{
		"DECLARANT": "deklarant",
		"BUSINESS":  "tadbirkor",
		"INSPECTOR": "deklarant",
	}
	i := 0
	for role, wantMode := range cases {
		i++
		login := "+99890000000" + string(rune('0'+i))
		body := `{"login":"` + login + `","password":"kuchli-parol","role":"` + role + `"}`
		_, out := post(t, h, "/api/auth/register", body, "")
		user, _ := out["user"].(map[string]any)
		if user["chat_mode"] != wantMode {
			t.Errorf("%s uslubi %v; %q kutilgan", role, user["chat_mode"], wantMode)
		}
	}
}

// ADMIN rolini ochiq ro'yxatdan o'tish orqali olib bo'lmasligi kerak —
// aks holda kim xohlasa admin bo'lardi.
func TestAdminRoleNotSelfAssignable(t *testing.T) {
	h, store := userServer(t)

	// BIRINCHI foydalanuvchi — istisno, aks holda birorta admin
	// bo'lmasdi va rol tayinlash imkoni yo'q edi.
	w, _ := post(t, h, "/api/auth/register",
		`{"login":"+998901111111","password":"kuchli-parol","role":"ADMIN"}`, "")
	wantStatus(t, w, http.StatusCreated, "birinchi admin")

	// Ikkinchisi — rad etiladi.
	w, _ = post(t, h, "/api/auth/register",
		`{"login":"+998902222222","password":"kuchli-parol","role":"ADMIN"}`, "")
	wantStatus(t, w, http.StatusForbidden, "ikkinchi admin")

	if store.Count() != 1 {
		t.Errorf("foydalanuvchilar soni %d; 1 kutilgan", store.Count())
	}
}

func TestRolesEndpoint(t *testing.T) {
	h, _ := userServer(t)
	w, out := get(t, h, "/api/auth/roles", "")
	wantStatus(t, w, http.StatusOK, "rollar")

	list, _ := out["roles"].([]any)
	if len(list) != len(users.Roles) {
		t.Fatalf("rollar soni %d; %d kutilgan", len(list), len(users.Roles))
	}
	for _, r := range list {
		m := r.(map[string]any)
		if m["role"] == "ADMIN" && m["self_signup"] != false {
			t.Error("ADMIN ro'yxatdan o'tishda tanlanadigan ko'rinmoqda")
		}
	}
}

// Buzilgan yoki soxta token o'tmasligi kerak.
func TestUserTokenRejectsTampering(t *testing.T) {
	h, _ := userServer(t)
	_, out := post(t, h, "/api/auth/register", regBody, "")
	token := out["token"].(string)

	i := strings.LastIndexByte(token, '.')
	payload, sig := token[:i], token[i+1:]

	bad := []string{
		payload + "." + sig + "x",        // imzo o'zgartirilgan
		payload,                          // imzosiz
		"u1.boshqa-id.0.99999999." + sig, // ID almashtirilgan
	}
	for _, tok := range bad {
		w, _ := get(t, h, "/api/auth/me", tok)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("buzilgan token qabul qilindi (%q): status %d", tok, w.Code)
		}
	}
}
