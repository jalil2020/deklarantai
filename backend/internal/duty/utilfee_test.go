package duty

import (
	"strings"
	"testing"
)

// ПКМ 347, 1-ilova bo'yicha asosiy toifalar.
//
// Stavkalar BRV karrasida; testDate da BRV = 412 000.
func TestUtilFeeRates(t *testing.T) {
	const brv = 412_000
	cases := []struct {
		name     string
		code     string
		measure  float64
		age      float64
		wantBRV  float64 // BRV karrasi
		wantCat  string
	}{
		// III. Yengil avtomobillar — eng ko'p uchraydigan toifa.
		{"avtomobil 900 sm³, yangi", "8703 21 100 0", 900, 1, 30, "Yengil"},
		{"avtomobil 900 sm³, eski", "8703 21 100 0", 900, 5, 90, "Yengil"},
		{"avtomobil 2000 sm³, yangi", "8703 23 194 0", 2000, 1, 120, "Yengil"},
		{"avtomobil 2000 sm³, eski", "8703 23 194 0", 2000, 5, 210, "Yengil"},
		{"avtomobil 3200 sm³, eski", "8703 24 100 0", 3200, 5, 390, "Yengil"},
		{"avtomobil 4000 sm³, eski", "8703 24 100 0", 4000, 5, 480, "Yengil"},

		// II. Avtobuslar.
		{"avtobus 2000 sm³, yangi", "8702 10 199 0", 2000, 1, 120, "Avtobus"},
		{"avtobus 12000 sm³, eski", "8702 10 199 0", 12000, 5, 1080, "Avtobus"},

		// IV. Yuk avtomobillari — vazn bo'yicha.
		{"yuk 2 t, yangi", "8704 21 310 0", 2, 1, 100, "Yuk"},
		{"yuk 15 t, eski", "8704 23 910 0", 15, 5, 1200, "Yuk"},

		// V, VI.
		{"betonaralashtirgich", "8705 40 000 0", 0, 5, 1500, "Avtobeton"},
		{"tirkama", "8716 39 800 0", 0, 5, 660, "Tirkama"},

		// VII. Ot kuchi bo'yicha.
		{"ekskavator 200 o.k.", "8429 52 100 0", 200, 5, 360, "Ekskavator"},
	}

	for _, c := range cases {
		got, err := UtilizationFee(UtilFeeRequest{
			Date: testDate, Code: c.code, Measure: c.measure, AgeYears: c.age,
		})
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !strings.Contains(got.Category, c.wantCat) {
			t.Errorf("%s: toifa = %q; %q kutilgan", c.name, got.Category, c.wantCat)
		}
		eq(t, got.Amount, c.wantBRV*brv, c.name)
	}
}

// Yangi texnikaga stavka belgilanmagan toifalar (jadvalda "—").
// Bu "yig'im yo'q" degani, lekin buni IZOH bilan aytish kerak —
// foydalanuvchi nega nol ekanini bilsin.
func TestUtilFeeNotSetForNew(t *testing.T) {
	got, err := UtilizationFee(UtilFeeRequest{
		Date: testDate, Code: "8701 94 100 1", Measure: 100, AgeYears: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount != 0 {
		t.Errorf("yangi traktorga yig'im = %v; 0 kutilgan", got.Amount)
	}
	if !strings.Contains(got.Note, "belgilanmagan") {
		t.Errorf("nega nol ekani izohlanmagan: %q", got.Note)
	}
	// Eskisi uchun esa yig'im bor.
	old, _ := UtilizationFee(UtilFeeRequest{
		Date: testDate, Code: "8701 94 100 1", Measure: 100, AgeYears: 5,
	})
	eq(t, old.Amount, 480*412_000, "eski traktor")
}

// Kod jadvalga tushmasa — XATO, 0 emas. Aksiz bilan bir xil mantiq:
// "yig'im yo'q" deb javob berish, aslida bor bo'lsa, xavfli.
func TestUtilFeeUnknownCode(t *testing.T) {
	_, err := UtilizationFee(UtilFeeRequest{Date: testDate, Code: "6109 10 000 0"})
	if err == nil {
		t.Fatal("ro'yxatda yo'q kod uchun xato kutilgan edi")
	}
	if !strings.Contains(err.Error(), "ПКМ 347") {
		t.Errorf("xato huquqiy asosni ko'rsatmaydi: %v", err)
	}
}

// O'lchov kerak bo'lsa, uni SO'RASH kerak — taxmin qilmaslik.
func TestUtilFeeAsksForMeasure(t *testing.T) {
	cases := map[string]string{
		"8703 23 194 0": "sm³",    // dvigatel hajmi
		"8704 21 310 0": "tonna",  // to'la vazn
		"8701 94 100 1": "kVt",    // quvvat
		"8429 52 100 0": "ot kuchi",
	}
	for code, want := range cases {
		got, err := UtilizationFee(UtilFeeRequest{Date: testDate, Code: code, AgeYears: 5})
		if err != nil {
			t.Errorf("%s: %v", code, err)
			continue
		}
		if got.NeedsMeasure == "" {
			t.Errorf("%s: o'lchov so'ralmadi (summa %v)", code, got.Amount)
			continue
		}
		if !strings.Contains(got.NeedsMeasure, want) {
			t.Errorf("%s: so'ralgan o'lchov %q; %q kutilgan", code, got.NeedsMeasure, want)
		}
	}
}

// Aniqroq kod umumiy koddan ustun bo'lishi kerak.
//
// "8701 20 109 0" — traktor (quvvat bo'yicha), "8701 20" esa egarli
// tyagach. Uzunroq prefiks aniqroq.
func TestUtilFeeMostSpecificCode(t *testing.T) {
	tractor, err := UtilizationFee(UtilFeeRequest{
		Date: testDate, Code: "8701 20 109 0", Measure: 100, AgeYears: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tractor.Category, "Traktor") {
		t.Errorf("8701 20 109 0 -> %q; traktor kutilgan", tractor.Category)
	}

	tyagach, err := UtilizationFee(UtilFeeRequest{
		Date: testDate, Code: "8701 20 900 0", AgeYears: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tyagach.Category, "tyagach") {
		t.Errorf("8701 20 900 0 -> %q; egarli tyagach kutilgan", tyagach.Category)
	}
	eq(t, tyagach.Amount, 1360*412_000, "egarli tyagach")
}

// Elektromobil alohida toifa — umumiy 8703 dan chiqarilgan.
func TestUtilFeeElectricVehicles(t *testing.T) {
	got, err := UtilizationFee(UtilFeeRequest{
		Date: testDate, Code: "8703 80 000 0", AgeYears: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Category, "elektro") {
		t.Errorf("toifa = %q; elektrodvigatelli kutilgan", got.Category)
	}
	eq(t, got.Amount, 120*412_000, "elektromobil")
}

// Shinalar — 2-ilova, har kilogramm uchun BRV ning foizida.
func TestUtilFeeTyres(t *testing.T) {
	// Vaznsiz — so'rashi kerak.
	need, err := UtilizationFee(UtilFeeRequest{Date: testDate, Code: "4011 10 000 0"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(need.NeedsMeasure, "vazn") {
		t.Errorf("vazn so'ralmadi: %+v", need)
	}

	// 100 kg × 0,3% × 412 000 = 123 600
	got, err := UtilizationFee(UtilFeeRequest{
		Date: testDate, Code: "4011 10 000 0", WeightKg: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, got.Amount, 123_600, "yangi shina, 100 kg")
}

// Ishlatilgan shinalar stavkasi manba hujjatda ko'rsatilmagan —
// 0 deb hisoblash yolg'on javob bo'lardi.
func TestUtilFeeUsedTyresUnknown(t *testing.T) {
	_, err := UtilizationFee(UtilFeeRequest{
		Date: testDate, Code: "4012 11 000 0", WeightKg: 100,
	})
	if err == nil {
		t.Fatal("noma'lum stavkada xato kutilgan edi")
	}
	if !strings.Contains(err.Error(), "2-ilova") {
		t.Errorf("xato manbani ko'rsatmaydi: %v", err)
	}
}

// BRV sanaga bog'liq — yig'im ham.
func TestUtilFeeUsesBRVOfDate(t *testing.T) {
	older := testDate.AddDate(-3, 0, 0) // 2023-07-19, BRV = 330 000
	got, err := UtilizationFee(UtilFeeRequest{
		Date: older, Code: "8703 23 194 0", Measure: 2000, AgeYears: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, got.BRV, 330_000, "BRV")
	eq(t, got.Amount, 120*330_000, "yig'im (2023)")
}

func TestUtilFeeMeasureHint(t *testing.T) {
	if got := UtilFeeMeasure("8703 23 194 0"); !strings.Contains(got, "sm³") {
		t.Errorf("o'lchov = %q", got)
	}
	if got := UtilFeeMeasure("6109 10 000 0"); got != "" {
		t.Errorf("ro'yxatda yo'q kod uchun o'lchov = %q; bo'sh kutilgan", got)
	}
}
