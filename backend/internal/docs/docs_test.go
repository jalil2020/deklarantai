package docs

import (
	"strings"
	"testing"
)

func load(t *testing.T) *Store {
	t.Helper()
	s, err := Load("../../data/docs.json")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// rasmiy kalkulyator ekranidagi "Необходимые документы" bo'limini takrorlash.
//
// Kod 3001209000 uchun dastur ko'rsatgan:
//
//	Лицензии    — "Лицензий не требуется"
//	Сертификаты — sertifikat yoki Uzstandart xati (ПКМ 43 от 30.01.2021)
//	Льготы      — "Льготы не предусмотрены"
//	Иные        — Dori vositalari sifatini nazorat qilish bosh boshqarmasida
//	              ro'yxatdan o'tish (МЮ 2809 от 12.07.2016)
func TestReferenceCode(t *testing.T) {
	got := load(t).For("3001209000", Import)
	if len(got) == 0 {
		t.Fatal("talablar topilmadi")
	}

	byCat := map[string][]Requirement{}
	for _, r := range got {
		byCat[r.Category] = append(byCat[r.Category], r)
	}

	if len(byCat["litsenziya"]) != 0 {
		t.Errorf("litsenziya talab qilinmasligi kerak, %d ta topildi", len(byCat["litsenziya"]))
	}
	if len(byCat["imtiyoz"]) != 0 {
		t.Errorf("imtiyoz ko'zda tutilmagan, %d ta topildi", len(byCat["imtiyoz"]))
	}

	mustHave(t, byCat["sertifikat"], "ПКМ 43", "sertifikat")
	mustHave(t, byCat["boshqa"], "МЮ 2809", "boshqa talab")
}

func mustHave(t *testing.T, rs []Requirement, law, what string) {
	t.Helper()
	for _, r := range rs {
		if strings.Contains(r.Law, law) {
			return
		}
	}
	t.Errorf("%s bo'limida %q qonuni topilmadi (%d ta yozuv)", what, law, len(rs))
}

// Amal muddati o'tgan talab ko'rsatilmasligi kerak.
//
// Manbada sertifikat talabi IKKI marta yozilgan: 2021-11-17..2024-11-24 va
// 2024-11-25 dan boshlab. Ekstraktor sana bo'yicha filtrlaydi, shuning uchun
// bitta kodga bitta sertifikat talabi tushishi kerak — ikkitasi emas.
func TestNoDuplicateCertificate(t *testing.T) {
	got := load(t).For("3001209000", Import)
	n := 0
	for _, r := range got {
		if r.Category == "sertifikat" && strings.Contains(r.Law, "ПКМ 43") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("ПКМ 43 sertifikat talabi %d marta chiqdi; 1 marta kutilgan", n)
	}
}

// Kod turli ko'rinishda berilishi mumkin.
func TestCodeNormalization(t *testing.T) {
	s := load(t)
	full := s.For("3001209000", Import)
	for _, variant := range []string{"3001 20 900 0", "3001209000"} {
		if got := s.For(variant, Import); len(got) != len(full) {
			t.Errorf("For(%q) = %d ta; %d kutilgan", variant, len(got), len(full))
		}
	}
	// Qisqa kod guruh boshiga to'ldiriladi.
	if got := s.For("3001", Import); len(got) == 0 {
		t.Error("qisqa kod bo'yicha hech narsa topilmadi")
	}
}

// "Specs_*" yozuvlari hujjat emas — tovar tavsifiga oid ma'lumot.
// Ular "boshqa talab" ga aralashib ketmasligi kerak, aks holda
// foydalanuvchi mavjud bo'lmagan hujjatni izlab yurardi.
func TestSpecsAreSeparate(t *testing.T) {
	for _, r := range load(t).For("3001209000", Import) {
		if strings.Contains(r.Text, "непатентованное название") && r.Category != "tavsif" {
			t.Errorf("tavsif ma'lumoti %q bo'limiga tushib qolgan", r.Category)
		}
	}
}

// Eksport rejimida import uchungina belgilangan talab chiqmasligi kerak.
func TestRegimeFilter(t *testing.T) {
	s := load(t)
	imp := s.For("3001209000", Import)
	exp := s.For("3001209000", Export)
	if len(exp) >= len(imp) {
		t.Errorf("eksportda %d ta, importda %d ta talab; eksportda kamroq kutilgan", len(exp), len(imp))
	}
}

// Imtiyoz qoidasi bor kodlarni aniqlash.
//
// NEGA MUHIM: hscodes.json da QQS hamma kodda 12%, boj esa tarif bo'yicha.
// Lekin 3 856 kod (29%) imtiyoz qoidasiga tushadi — ulardan 1 287 tasi
// QQS dan, 2 520 tasi bojdan ozod bo'lishi mumkin. Ikkala baza bir-biriga
// zid, shuning uchun chat stavka yonida imtiyoz borligini aytishi shart:
// aks holda kalkulyator ortiqcha hisoblab beradi.
//
// PKM 352 — o'xshashi ishlab chiqarilmaydigan texnologik uskunalar,
// bojdan ham QQS dan ham ozod.
func TestExemptionsFound(t *testing.T) {
	s := load(t)

	// Imtiyoz qoidasi umuman ishlashini tekshiramiz: bazada QQS dan ozod
	// qiluvchi qoidalar bor, demak hech bo'lmasa bitta kod topilishi kerak.
	found := false
	for _, r := range s.rules {
		if len(s.types[r.Type].Free) == 0 {
			continue
		}
		if e := s.Exemptions(r.Min, Import); len(e) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("imtiyoz qoidalari bor, lekin Exemptions hech narsa qaytarmadi")
	}
}

// Imtiyozi yo'q kodga bo'sh natija qaytishi kerak — "ozod" deb
// noto'g'ri belgilab qo'ymasligimiz uchun.
func TestNoExemptionIsEmpty(t *testing.T) {
	s := load(t)
	n := 0
	for _, code := range []string{"3001209000", "8701100000", "0101210000"} {
		if len(s.Exemptions(code, Import)) > 0 {
			n++
		}
	}
	// Hammasi imtiyozli bo'lib chiqsa, qoida juda keng ishlayapti degani.
	if n == 3 {
		t.Error("uchala kod ham imtiyozli chiqdi — oraliq mosligi juda keng bo'lishi mumkin")
	}
}

// Imtiyoz dasturlari ro'yxati.
//
// NEGA KERAK: imtiyoz ma'lumoti faqat aniq kod so'ralganda ko'rinardi.
// "Qanday imtiyozlar bor?" degan savolga javob yo'q edi.
func TestPrograms(t *testing.T) {
	got := load(t).Programs()
	if len(got) < 20 {
		t.Fatalf("dasturlar soni %d; 20 dan ko'p kutilgan", len(got))
	}

	// Eng yirigi oldinda bo'lishi kerak — foydalanuvchi uchun ehtimoli yuqori.
	if got[0].Ranges < got[len(got)-1].Ranges {
		t.Error("dasturlar qamrov bo'yicha tartiblanmagan")
	}

	// ПКМ 352 — o'xshashi ishlab chiqarilmaydigan texnologik uskunalar,
	// eng ko'p qo'llaniladigan imtiyoz.
	var pkm352 *Program
	for i := range got {
		if got[i].Type == "PKM352" {
			pkm352 = &got[i]
		}
	}
	if pkm352 == nil {
		t.Fatal("ПКМ 352 dasturi topilmadi")
	}
	if len(pkm352.Free) != 2 {
		t.Errorf("ПКМ 352 ozod qiladigan to'lovlar: %v; boj va qqs kutilgan", pkm352.Free)
	}
	if len(pkm352.Laws) == 0 {
		t.Error("ПКМ 352 uchun huquqiy asos ko'rsatilmagan")
	}
	if pkm352.Text == "" {
		t.Error("ПКМ 352 tavsifi bo'sh")
	}
}

// Har bir dasturda kamida bitta to'lov turi va tavsif yoki asos bo'lishi kerak —
// aks holda foydalanuvchiga hech narsa demaydi.
func TestProgramsAreUsable(t *testing.T) {
	for _, p := range load(t).Programs() {
		if len(p.Free) == 0 {
			t.Errorf("%s: qaysi to'lovdan ozod ekani ko'rsatilmagan", p.Type)
		}
		if p.Text == "" && len(p.Laws) == 0 {
			t.Errorf("%s: na tavsif, na huquqiy asos bor", p.Type)
		}
	}
}
