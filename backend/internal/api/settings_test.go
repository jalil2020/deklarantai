package api

import (
	"os"
	"strings"
	"testing"
)

// .env.example registrga mos bo'lishi kerak.
//
// NEGA TEST: bu fayl jimgina eskiradi. Yangi o'zgaruvchi qo'shilganda
// registrga yoziladi, namunaga esa yozilmaydi — va serverni ko'taradigan
// odam sozlama BORLIGINI bilmay qoladi. Xato chiqmaydi, shunchaki
// sukut qiymat ishlaydi.
//
// Aynan shu holat yuz bergan edi: namunada 4 ta o'zgaruvchi qolgan,
// registrda 26 ta bo'lgan.
func TestEnvExampleCoversRegistry(t *testing.T) {
	raw, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)

	inExample := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, val, ok := strings.Cut(line, "=")
		if !ok {
			t.Errorf("qator \"=\" siz: %q", line)
			continue
		}
		inExample[strings.TrimSpace(name)] = strings.TrimSpace(val)
	}

	for _, d := range settingsRegistry {
		val, ok := inExample[d.name]
		if !ok {
			t.Errorf(".env.example da %s yo'q (registrda bor)", d.name)
			continue
		}
		delete(inExample, d.name)

		// Maxfiy qiymat namunada BO'SH turishi shart — aks holda kimdir
		// haqiqiy kalitni shu yerga yozib, git ga yuborib yuboradi.
		if d.secret && val != "" {
			t.Errorf("%s MAXFIY, lekin .env.example da qiymati bor: %q", d.name, val)
		}
		// Maxfiy bo'lmagan sozlamada sukut qiymat ko'rinib tursin.
		if !d.secret && d.def != "" && val != d.def {
			t.Errorf("%s: namunada %q, registrda sukut %q", d.name, val, d.def)
		}
	}

	for name := range inExample {
		t.Errorf(".env.example da %s bor, lekin registrda yo'q", name)
	}
}

// Fayl uni O'QILMASLIGI haqida ogohlantirishi kerak.
//
// NEGA TEST: `.env.example` nomi odatda "nusxa ol, .env qil, ishlaydi"
// degan ma'noni beradi. Bu loyihada bermaydi — hech qanday yuklovchi
// yo'q. Ogohlantirish olib tashlansa, odam bir soat nega kalit
// ko'rinmayotganini qidiradi.
func TestEnvExampleWarnsItIsNotLoaded(t *testing.T) {
	raw, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "AVTOMATIK O'QILMAYDI") {
		t.Error(".env.example da 'avtomatik o'qilmaydi' ogohlantirishi yo'q")
	}
}
