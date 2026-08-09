package duty

import (
	"math"
	"strings"
	"testing"
	"time"
)

var testDate = time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC) // BRV = 412 000

func find(r Result, code string) LineItem {
	for _, it := range r.Items {
		if it.Code == code {
			return it
		}
	}
	return LineItem{}
}

func eq(t *testing.T, got, want float64, what string) {
	t.Helper()
	if math.Abs(got-want) > 0.5 {
		t.Errorf("%s = %.2f; kutilgan %.2f", what, got, want)
	}
}

// BRV sanaga qarab to'g'ri tanlanishi (mzp jadvali bo'yicha).
func TestBRV(t *testing.T) {
	cases := []struct {
		on   time.Time
		want float64
	}{
		{time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC), 412_000},
		{time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), 412_000}, // amal qilish kuni
		{time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC), 375_000},
		{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 340_000},
		{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 270_000}, // eng eskisidan oldin
	}
	for _, c := range cases {
		if got := BRV(c.on); got != c.want {
			t.Errorf("BRV(%s) = %.0f; kutilgan %.0f", c.on.Format("2006-01-02"), got, c.want)
		}
	}
}

// Yig'im shkalasi — ПКМ 55, 1-ilova, 1-band «а».
func TestFeeScale(t *testing.T) {
	cases := []struct{ usd, mult float64 }{
		{5_000, 1}, {10_000, 1}, // "10 000 gacha"
		{15_000, 1.5}, {20_000, 1.5},
		{30_000, 2.5}, {50_000, 4}, {80_000, 7},
		{150_000, 10}, {300_000, 15}, {700_000, 20},
		{1_000_000, 20}, {2_000_000, 25}, // "1 mln va undan yuqori"
	}
	for _, c := range cases {
		if got := feeMultiplier(c.usd); got != c.mult {
			t.Errorf("feeMultiplier(%.0f USD) = %v; kutilgan %v", c.usd, got, c.mult)
		}
	}
}

// Asosiy hisob: 10 000 USD traktor, boj 5% + qo'shimcha 5%, QQS 12%.
func TestCalculateTractor(t *testing.T) {
	const usd = 12_093.35
	r := Calculate(Request{
		Date:         testDate,
		Invoice:      10_000,
		CurrencyRate: usd,
		USDRate:      usd,
		ImportDuty:   5,
		ExtraDuty:    5,
		VAT:          12,
	})

	cv := 10_000 * usd // 120 933 500
	eq(t, r.CustomsValue, cv, "bojxona qiymati")
	eq(t, r.BRV, 412_000, "BRV")

	// 10 000 USD → 1 × BRV
	eq(t, find(r, "10").Amount, 412_000, "yig'im")

	duty := cv * 0.05
	eq(t, find(r, "20").Amount, duty, "boj")
	eq(t, find(r, "21").Amount, duty, "qo'shimcha boj")

	// SK 254-modda: QQS bazasiga yig'im KIRMAYDI.
	vatBase := cv + duty + duty
	eq(t, find(r, "29").Base, vatBase, "QQS bazasi")
	eq(t, find(r, "29").Amount, vatBase*0.12, "QQS")

	eq(t, r.Total, 412_000+duty+duty+vatBase*0.12, "jami")
}

// SK 285-modda: advalor aksiz bazasi — bojxona qiymati (bojsiz).
func TestExciseBaseIsCustomsValueOnly(t *testing.T) {
	r := Calculate(Request{
		Date: testDate, CustomsValue: 100_000_000, USDRate: 12_000,
		ImportDuty: 10, Excise: 20, VAT: 12,
	})
	eq(t, find(r, "27").Base, 100_000_000, "aksiz bazasi")
	eq(t, find(r, "27").Amount, 20_000_000, "aksiz")

	// QQS bazasi = qiymat + boj + aksiz
	eq(t, find(r, "29").Base, 100_000_000+10_000_000+20_000_000, "QQS bazasi")
}

// Ko'rik to'liq soatga yuqoriga yaxlitlanadi (***** izoh).
func TestInspectionRounding(t *testing.T) {
	r := Calculate(Request{
		Date: testDate, CustomsValue: 1_000_000, USDRate: 12_000,
		InspectDay: 0.2, InspectNight: 1.1,
	})
	// 0,2 soat → 1 to'liq soat × 25% BRV
	eq(t, find(r, "12").Amount, 412_000*0.25, "kunduzgi ko'rik")

	var night float64
	for _, it := range r.Items {
		if it.Code == "12" && it.Name == "Ko'rik (ish vaqtidan tashqari)" {
			night = it.Amount
		}
	}
	// 1,1 soat → 2 to'liq soat × 2 BRV
	eq(t, night, 412_000*2*2, "tungi ko'rik")
}

// Oldindan deklaratsiya — yig'imga 20% chegirma.
func TestPreliminaryDiscount(t *testing.T) {
	base := Request{Date: testDate, CustomsValue: 60_000_000, USDRate: 12_000, VAT: 12}
	full := Calculate(base)

	base.Preliminary = true
	disc := Calculate(base)

	eq(t, disc.Items[0].Amount, full.Items[0].Amount*0.8, "chegirmali yig'im")
}

// Yig'im olinmaydigan rejim.
func TestFeeExempt(t *testing.T) {
	r := Calculate(Request{
		Date: testDate, CustomsValue: 60_000_000, USDRate: 12_000, VAT: 12, FeeExempt: true,
	})
	eq(t, find(r, "10").Amount, 0, "yig'im (ozod)")
}

// FeeScaleText tizim ko'rsatmasiga qo'yiladi — u yerda raqamlar qo'lda
// yozilmasligi kerak. Bu test matn haqiqatan feeScale/brvHistory dan
// chiqayotganini va chegaralar tushib qolmaganini qo'riqlaydi.
func TestFeeScaleText(t *testing.T) {
	got := FeeScaleText(testDate)
	for _, want := range []string{
		"BRV: 412 000 so'm",
		"10 000 USD gacha → 1×BRV = 412 000 so'm",
		"10 000–20 000 USD → 1,5×BRV = 618 000 so'm",
		"1 000 000 USD dan yuqori → 25×BRV = 10 300 000 so'm",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("shkala matnida yo'q:\n  kutilgan: %q\n  olingan:\n%s", want, got)
		}
	}
}

// Rasmiy manba kalkulyatori natijasi bilan solishtirish.
//
// Manba: dasturning ekrani, kod 3001209000, hisob sanasi 19.07.2026.
// Kiritilgan: faktura 1 230 000 USD, transport 25 000 USD, kurs 12 093,35,
// miqdor 5 622 kg, boj/aksiz "не установлено", QQS 12%.
//
// Dastur chiqargan natija:
//
//	Итого там. стоимость  15 177 154 250,00
//	10. Тамож. сбор           10 300 000,00   (25 БРВ)
//	29. НДС                1 821 258 510,00
//	Итого                  1 831 558 510,00
//
// Bu test IKKI narsani qo'riqlaydi:
//  1. Yig'im shkalasining eng yuqori pog'onasi (1 mln USD dan yuqori -> 25×BRV).
//  2. QQS bazasiga bojxona yig'imi KIRMASLIGI (SK 254-modda). Agar yig'im
//     bazaga qo'shilsa, QQS 1 822 494 510 chiqardi — ya'ni 1 236 000 so'm
//     ortiq. Rasmiy dastur ham yig'imni bazaga qo'shmaydi.
func TestReferenceReferenceCase(t *testing.T) {
	const rate = 12_093.35
	r := Calculate(Request{
		Date:         testDate,
		Invoice:      1_230_000,
		Transport:    25_000,
		CurrencyRate: rate,
		USDRate:      rate,
		ImportDuty:   0,
		VAT:          12,
		Quantity:     5_622,
	})

	eq(t, r.CustomsValue, 15_177_154_250, "bojxona qiymati")
	eq(t, find(r, "10").Amount, 10_300_000, "yig'im (25×BRV)")
	eq(t, find(r, "29").Amount, 1_821_258_510, "QQS")
	eq(t, r.Total, 1_831_558_510, "jami")

	// Boj va aksiz belgilanmagan — to'lov chiqmasligi kerak.
	eq(t, find(r, "20").Amount, 0, "boj")
	eq(t, find(r, "27").Amount, 0, "aksiz")
}

// Kelib chiqish davlati bojga ta'sir qiladi — Bojxona kodeksi 300-modda:
//
//	"Boj tarifi bilan belgilangan stavkalar miqdoridagi bojxona bojlari
//	 … eng koʻp qulaylik berish rejimini qoʻllayotgan mamlakatlarda ishlab
//	 chiqarilgan tovarlarga nisbatan … qoʻllaniladi."
//	"Savdo-iqtisodiy munosabatlarda eng koʻp qulaylik berish rejimi nazarda
//	 tutilmagan mamlakatlarda ishlab chiqarilgan yoxud ishlab chiqarilgan
//	 mamlakati aniqlanmagan tovarlarga nisbatan bojxona bojlarining
//	 stavkalari IKKI BARAVAR oshiriladi."
//	"…erkin savdo rejimini belgilagan davlatlarda ishlab chiqarilgan …
//	 tovarlarga … bojxona bojlari qoʻllanilmaydi."
func TestOriginAffectsDuty(t *testing.T) {
	mult := func(v float64) *float64 { return &v }
	base := Request{
		Date: testDate, CustomsValue: 100_000_000, USDRate: 12_000,
		ImportDuty: 10, VAT: 12,
	}

	cases := []struct {
		name       string
		multiplier *float64
		wantRate   float64
		wantDuty   float64
	}{
		{"erkin savdo (Rossiya)", mult(0), 0, 0},
		{"eng qulaylik (Xitoy)", mult(1), 10, 10_000_000},
		{"rejim yo'q", mult(2), 20, 20_000_000},
		{"ko'rsatilmagan", nil, 10, 10_000_000},
	}
	for _, c := range cases {
		r := base
		r.OriginMultiplier = c.multiplier
		got := Calculate(r)
		item := find(got, "20")

		eq(t, item.Rate, c.wantRate, c.name+": stavka")
		eq(t, item.Amount, c.wantDuty, c.name+": boj")
		if item.Note == "" {
			t.Errorf("%s: izoh yo'q — foydalanuvchi nega bunday ekanini bilishi kerak", c.name)
		}
	}
}

// Kelib chiqish ko'rsatilmasa, buni YASHIRMASLIK kerak: javob ikkala
// tomonga ham xato bo'lishi mumkin (erkin savdoda boj yo'q, rejimsiz
// davlatda ikki barobar).
func TestUnspecifiedOriginIsFlagged(t *testing.T) {
	r := Calculate(Request{
		Date: testDate, CustomsValue: 100_000_000, USDRate: 12_000,
		ImportDuty: 10, VAT: 12,
	})
	note := find(r, "20").Note
	if !strings.Contains(note, "ko'rsatilmagan") {
		t.Errorf("kelib chiqish ko'rsatilmagani aytilmagan: %q", note)
	}
	if !strings.Contains(note, "300-modda") {
		t.Errorf("qonuniy asos ko'rsatilmagan: %q", note)
	}
}

// Erkin savdo imtiyozi shartli — sertifikat kerakligi aytilishi shart.
func TestFreeTradeMentionsCertificate(t *testing.T) {
	m := 0.0
	r := Calculate(Request{
		Date: testDate, CustomsValue: 100_000_000, USDRate: 12_000,
		ImportDuty: 10, VAT: 12, OriginMultiplier: &m, OriginCountry: "Rossiya",
	})
	note := find(r, "20").Note
	if !strings.Contains(note, "ST-1") {
		t.Errorf("kelib chiqish sertifikati eslatilmagan: %q", note)
	}
	if !strings.Contains(note, "Rossiya") {
		t.Errorf("davlat nomi izohda yo'q: %q", note)
	}
}

// Boj o'zgarsa, QQS bazasi ham o'zgarishi kerak — ular bog'liq.
func TestOriginChangesVATBase(t *testing.T) {
	mult := func(v float64) *float64 { return &v }
	base := Request{
		Date: testDate, CustomsValue: 100_000_000, USDRate: 12_000,
		ImportDuty: 10, VAT: 12,
	}

	free, mfn := base, base
	free.OriginMultiplier = mult(0)
	mfn.OriginMultiplier = mult(1)

	// Erkin savdo: QQS bazasi = faqat bojxona qiymati.
	eq(t, find(Calculate(free), "29").Base, 100_000_000, "QQS bazasi (erkin savdo)")
	// EQR: baza = qiymat + boj.
	eq(t, find(Calculate(mfn), "29").Base, 110_000_000, "QQS bazasi (EQR)")
}

// ---- Kombinatsiyalangan stavka (10%, lekin kg uchun $0,5 dan kam emas) ----

// combinedReq — 9405 42 003 9 kodiga o'xshash holat.
func combinedReq(value, qty float64) Request {
	return Request{
		Date:               time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		CustomsValue:       value,
		USDRate:            12_093.35,
		ImportDuty:         10,
		ImportDutySpecific: 0.5,
		SpecificQuantity:   qty,
		VAT:                12,
		FeeExempt:          true,
	}
}

func dutyLine(t *testing.T, res Result) LineItem {
	t.Helper()
	for _, it := range res.Items {
		if it.Code == "20" {
			return it
		}
	}
	t.Fatal("20-kod (bojxona boji) qatori yo'q")
	return LineItem{}
}

// Foizli qism kattaroq bo'lsa — o'sha qo'llanadi.
func TestCombinedRatePercentWins(t *testing.T) {
	// 100 mln so'mning 10% i = 10 mln. Qat'iy: 10 kg × $0,5 × 12093 ≈ 60 tys.
	res := Calculate(combinedReq(100_000_000, 10))
	got := dutyLine(t, res)

	if want := 10_000_000.0; got.Amount != want {
		t.Errorf("boj %.0f; %.0f kutilgan", got.Amount, want)
	}
	if got.Rate != 10 {
		t.Errorf("stavka %g; 10 kutilgan", got.Rate)
	}
	// Foydalanuvchi qat'iy qism ham hisoblanganini KO'RISHI kerak.
	if !strings.Contains(got.Note, "qat'iy qism") {
		t.Errorf("izohda qat'iy qism eslatilmagan: %q", got.Note)
	}
}

// Qat'iy qism kattaroq bo'lsa — u qo'llanadi. AYNAN SHU holat ilgari
// yo'qolardi: boj kam hisoblanardi.
func TestCombinedRateSpecificWins(t *testing.T) {
	// 1 mln so'mning 10% i = 100 000. Qat'iy: 1000 kg × $0,5 × 12093,35 = 6 046 675.
	res := Calculate(combinedReq(1_000_000, 1000))
	got := dutyLine(t, res)

	want := 1000 * 0.5 * 12_093.35
	if math.Abs(got.Amount-want) > 1 {
		t.Errorf("boj %.0f; %.0f kutilgan (qat'iy qism)", got.Amount, want)
	}
	if !strings.Contains(got.Note, "qat'iy qism qo'llanildi") {
		t.Errorf("qaysi qism qo'llangani aytilmagan: %q", got.Note)
	}
	// QQS bazasiga ham KATTA boj kirishi kerak.
	for _, it := range res.Items {
		if it.Code == "29" && it.Base < want {
			t.Errorf("QQS bazasi %.0f; kamida %.0f kutilgan", it.Base, want)
		}
	}
}

// Miqdor yoki kurs berilmasa — JIM QOLMASLIK kerak.
func TestCombinedRateMissingInputWarns(t *testing.T) {
	cases := map[string]Request{
		"miqdorsiz": func() Request { r := combinedReq(1_000_000, 0); return r }(),
		"kurssiz":   func() Request { r := combinedReq(1_000_000, 1000); r.USDRate = 0; return r }(),
	}
	for name, req := range cases {
		got := dutyLine(t, Calculate(req))
		if !strings.Contains(got.Note, "⚠️") {
			t.Errorf("%s: ogohlantirish yo'q, izoh: %q", name, got.Note)
		}
		// Foizli qism baribir hisoblanishi kerak.
		if got.Amount != 100_000 {
			t.Errorf("%s: boj %.0f; 100000 kutilgan", name, got.Amount)
		}
	}
}

// Erkin savdo (×0) da qat'iy qism ham qo'llanmasligi kerak.
func TestCombinedRateFreeTradeZero(t *testing.T) {
	req := combinedReq(1_000_000, 1000)
	zero := 0.0
	req.OriginMultiplier = &zero

	if got := dutyLine(t, Calculate(req)); got.Amount != 0 {
		t.Errorf("erkin savdoda boj %.0f; 0 kutilgan", got.Amount)
	}
}
