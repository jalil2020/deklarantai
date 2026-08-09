package users

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) *FileStore {
	t.Helper()
	s, err := Load(filepath.Join(t.TempDir(), "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCreateAndAuthenticate(t *testing.T) {
	s := newStore(t)

	u, err := s.Create("+998901234567", "kuchli-parol", "Jalil", Declarant)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == "" || u.Role != Declarant {
		t.Fatalf("noto'g'ri yozuv: %+v", u)
	}

	got, err := s.Authenticate("+998901234567", "kuchli-parol")
	if err != nil {
		t.Fatalf("kirish ishlamadi: %v", err)
	}
	if got.ID != u.ID {
		t.Error("boshqa foydalanuvchi qaytdi")
	}

	if _, err := s.Authenticate("+998901234567", "boshqa-parol"); err != ErrBadLogin {
		t.Errorf("noto'g'ri parol → %v; ErrBadLogin kutilgan", err)
	}
	// Yo'q foydalanuvchi ham AYNAN shu xatoni berishi kerak: aks holda
	// qaysi loginlar ro'yxatda borligini bilib olish mumkin bo'lardi.
	if _, err := s.Authenticate("+998900000000", "kuchli-parol"); err != ErrBadLogin {
		t.Errorf("yo'q foydalanuvchi → %v; ErrBadLogin kutilgan", err)
	}
}

// Parol ochiq holda saqlanmasligi kerak — bu eng muhim tekshiruv.
func TestPasswordNeverStoredPlain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	const pass = "juda-maxfiy-parol"
	if _, err := s.Create("+998901234567", pass, "", Declarant); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), pass) {
		t.Fatal("parol faylda OCHIQ saqlangan")
	}
	// Tuz ham bo'lishi shart: tuzsiz bir xil parollar bir xil xesh
	// beradi va tayyor jadval bilan ochib olinadi.
	if !strings.Contains(string(raw), `"salt"`) {
		t.Error("tuz saqlanmagan")
	}
}

// Bir xil parolli ikki foydalanuvchining xeshi HAR XIL bo'lishi kerak.
func TestSaltsDiffer(t *testing.T) {
	s := newStore(t)
	a, _ := s.Create("+998901111111", "bir-xil-parol", "", Declarant)
	b, _ := s.Create("+998902222222", "bir-xil-parol", "", Business)
	if a.Hash == b.Hash {
		t.Error("bir xil parol bir xil xesh berdi — tuz ishlamayapti")
	}
}

func TestLoginNormalization(t *testing.T) {
	s := newStore(t)
	if _, err := s.Create("+998 90 123-45-67", "kuchli-parol", "", Declarant); err != nil {
		t.Fatal(err)
	}
	// Bir xil raqam boshqacha yozilsa ham o'sha akkaunt bo'lishi kerak.
	for _, form := range []string{"+998901234567", "+998 901234567", "+998(90)123-45-67"} {
		if _, err := s.Authenticate(form, "kuchli-parol"); err != nil {
			t.Errorf("%q bilan kirib bo'lmadi: %v", form, err)
		}
	}
	if _, err := s.Create("+998901234567", "kuchli-parol", "", Declarant); err != ErrExists {
		t.Errorf("takroriy raqam → %v; ErrExists kutilgan", err)
	}
}

func TestValidation(t *testing.T) {
	s := newStore(t)
	cases := []struct {
		login, pass string
		role        Role
		want        error
	}{
		{"+99890", "qisqa", Declarant, ErrWeakPass},
		{"abc", "kuchli-parol", Declarant, ErrBadFormat},
		{"+998901234567", "kuchli-parol", Role("BOSS"), ErrBadRole},
	}
	for _, c := range cases {
		if _, err := s.Create(c.login, c.pass, "", c.role); err != c.want {
			t.Errorf("Create(%q, %q, %q) → %v; %v kutilgan", c.login, c.pass, c.role, err, c.want)
		}
	}
}

func TestRoles(t *testing.T) {
	// Rol UCH narsani belgilaydi — uchalasi ham tekshiriladi.
	cases := []struct {
		role  Role
		mode  string
		quota int
	}{
		{Declarant, "deklarant", 200},
		{Business, "tadbirkor", 30},
		{Inspector, "deklarant", 300},
		{Admin, "deklarant", 1000},
	}
	for _, c := range cases {
		if !c.role.Valid() {
			t.Errorf("%s yaroqsiz deb topildi", c.role)
		}
		if got := c.role.ChatMode(); got != c.mode {
			t.Errorf("%s uslubi %q; %q kutilgan", c.role, got, c.mode)
		}
		if got := c.role.DailyQuota(); got != c.quota {
			t.Errorf("%s kvotasi %d; %d kutilgan", c.role, got, c.quota)
		}
	}
	if Role("BOSS").Valid() {
		t.Error("noma'lum rol yaroqli deb topildi")
	}
}

// Ombor faylga yozilib, qayta o'qilganda saqlanishi kerak.
func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	s1, _ := Load(path)
	u, err := s1.Create("+998901234567", "kuchli-parol", "Jalil", Inspector)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Count() != 1 {
		t.Fatalf("qayta o'qilgandan keyin %d foydalanuvchi", s2.Count())
	}
	got, err := s2.Authenticate("+998901234567", "kuchli-parol")
	if err != nil {
		t.Fatalf("qayta o'qilgandan keyin kirib bo'lmadi: %v", err)
	}
	if got.ID != u.ID || got.Role != Inspector {
		t.Errorf("yozuv o'zgargan: %+v", got)
	}
}

func TestDisableAndTokenVersion(t *testing.T) {
	s := newStore(t)
	u, _ := s.Create("+998901234567", "kuchli-parol", "", Declarant)

	if err := s.BumpToken(u.ID); err != nil {
		t.Fatal(err)
	}
	if u.TokenVer != 1 {
		t.Errorf("token versiyasi %d; 1 kutilgan", u.TokenVer)
	}

	if err := s.SetDisabled(u.ID, true); err != nil {
		t.Fatal(err)
	}
	// O'chirilgan foydalanuvchi kira olmasligi va tokeni ham
	// kuchini yo'qotishi kerak.
	if _, err := s.Authenticate("+998901234567", "kuchli-parol"); err != ErrDisabled {
		t.Errorf("o'chirilgan foydalanuvchi kirdi: %v", err)
	}
	if _, err := s.ByID(u.ID); err != ErrDisabled {
		t.Errorf("ByID o'chirilganni qaytardi: %v", err)
	}
	if u.TokenVer != 2 {
		t.Errorf("o'chirishda token versiyasi oshmadi: %d", u.TokenVer)
	}
}

func TestSetRole(t *testing.T) {
	s := newStore(t)
	u, _ := s.Create("+998901234567", "kuchli-parol", "", Declarant)
	if err := s.SetRole(u.ID, Admin); err != nil {
		t.Fatal(err)
	}
	if u.Role != Admin {
		t.Errorf("rol %s; ADMIN kutilgan", u.Role)
	}
	if err := s.SetRole(u.ID, Role("BOSS")); err != ErrBadRole {
		t.Errorf("noma'lum rol qabul qilindi: %v", err)
	}
}

// Public() parol maydonlarini chiqarmasligi kerak — u HTTP javobiga
// beriladi.
func TestPublicHidesSecrets(t *testing.T) {
	s := newStore(t)
	u, _ := s.Create("+998901234567", "kuchli-parol", "Jalil", Business)
	p := u.Public()
	if p.Login != u.Login || p.Role != Business {
		t.Errorf("ma'lumot yo'qoldi: %+v", p)
	}
	if p.Quota != Business.DailyQuota() || p.Mode != "tadbirkor" {
		t.Errorf("rol ma'lumoti noto'g'ri: %+v", p)
	}
}
