package countries

import (
	"strings"
	"testing"
)

func load(t *testing.T) *Store {
	t.Helper()
	s, err := Load("../../data/countries.json")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Bojxona kodeksi 300-moddasi bo'yicha uch daraja.
func TestDutyRegimes(t *testing.T) {
	s := load(t)
	cases := []struct {
		q    string
		want float64
	}{
		{"Rossiya", 0}, // erkin savdo
		{"643", 0},     // kod bo'yicha ham
		{"Qozogʻiston", 0},
		{"Xitoy", 1}, // eng qulaylik rejimi
		{"AQSh", 1},
		{"Turkiya", 1},
	}
	for _, c := range cases {
		got, ok := s.Find(c.q)
		if !ok {
			t.Errorf("%q topilmadi", c.q)
			continue
		}
		if got.DutyMultiplier != c.want {
			t.Errorf("%q: koeffitsient %v; %v kutilgan (%s)", c.q, got.DutyMultiplier, c.want, got.NameUZ)
		}
	}
}

// Foydalanuvchi turli ko'rinishda yozadi — hammasi topilishi kerak.
func TestFindVariants(t *testing.T) {
	s := load(t)
	// Bazadagi nom ruscha, transliteratsiya "KITAY" beradi — foydalanuvchi
	// esa "Xitoy" deb yozadi. Shuning uchun qo'lda nom va sinonim qo'shilgan.
	for _, q := range []string{
		"Xitoy", "XITOY", "xitoy", "XXR", "Xitoy Xalq Respublikasi", "156", "КИТАЙ",
	} {
		got, ok := s.Find(q)
		if !ok {
			t.Errorf("%q topilmadi", q)
			continue
		}
		if got.Code != "156" {
			t.Errorf("%q -> %s (%s); 156 kutilgan", q, got.Code, got.NameUZ)
		}
	}
	// Apostrof shakllari farq qilmasligi kerak.
	a, okA := s.Find("Qozogʻiston")
	b, okB := s.Find("Qozog'iston")
	if !okA || !okB || a.Code != b.Code {
		t.Error("apostrof shakllari turlicha topildi")
	}
}

func TestNotFound(t *testing.T) {
	if _, ok := load(t).Find("Elfiya podsholigi"); ok {
		t.Error("mavjud bo'lmagan davlat topildi")
	}
}

// Erkin savdo ro'yxati — MDH bitimi ishtirokchilari.
func TestFreeTradeList(t *testing.T) {
	got := load(t).FreeTrade()
	if len(got) < 8 || len(got) > 15 {
		t.Errorf("erkin savdo davlatlari %d ta; 8..15 kutilgan", len(got))
	}
	var names []string
	for _, c := range got {
		names = append(names, c.NameUZ)
	}
	all := strings.Join(names, ", ")
	for _, want := range []string{"Rossiya", "Belarus", "Tojikiston"} {
		if !strings.Contains(all, want) {
			t.Errorf("erkin savdo ro'yxatida %q yo'q: %s", want, all)
		}
	}
}

// Huquqiy asos meta'da ko'rsatilishi kerak — foydalanuvchi tekshira olsin.
func TestMetaHasLegalBasis(t *testing.T) {
	m := load(t).Meta()
	if !strings.Contains(m.LegalBasis, "300-modda") {
		t.Errorf("huquqiy asos = %q", m.LegalBasis)
	}
	if !strings.Contains(m.Warning, "ST-1") {
		t.Errorf("sertifikat sharti ogohlantirishda yo'q: %q", m.Warning)
	}
}
