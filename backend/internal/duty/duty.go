// Package duty O'zbekiston bojxona to'lovlarini hisoblaydi.
//
// Huquqiy asos (hujjatlar manba bazasidan o'qib tekshirilgan):
//
//	Bojxona yig'imi   — ПКМ № 55 (31.01.2025), 1-ilova, 1-band «а».
//	                    ПКМ № 700 (06.11.2025) faqat «г» va «д» kichik bandlar
//	                    matnini o'zgartirgan — BRV shkalasi o'zgarmagan.
//	Ko'rik            — o'sha ilova, 3-band «б».
//	QQS bazasi        — Soliq kodeksi 254-modda: bojxona qiymati + aksiz + boj.
//	                    Bojxona yig'imi QQS bazasiga KIRMAYDI.
//	QQS stavkasi      — Soliq kodeksi 258-modda (12%).
//	Aksiz bazasi      — Soliq kodeksi 285-modda: advalor stavkada bojxona qiymati
//	                    (bojsiz), qat'iy stavkada — natural miqdor.
//
// DIQQAT: aksizning qat'iy va kombinatsiyalangan stavkalari, hamda
// utilizatsiya yig'imi (79) hozircha qo'llab-quvvatlanmaydi — ular alohida
// qoidalar bo'yicha hisoblanadi.
package duty

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// brvHistory — bazaviy hisoblash miqdori (BRV) tarixi, yangisidan eskisiga.
// Manba: manba bazasidagi mzp jadvali.
var brvHistory = []struct {
	From  time.Time
	Value float64
}{
	{time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), 412_000},
	{time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC), 375_000},
	{time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC), 340_000},
	{time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC), 330_000},
	{time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC), 300_000},
	{time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC), 270_000},
}

// BRV — berilgan sanaga amal qilgan bazaviy hisoblash miqdorini qaytaradi.
func BRV(on time.Time) float64 {
	for _, b := range brvHistory {
		if !on.Before(b.From) {
			return b.Value
		}
	}
	return brvHistory[len(brvHistory)-1].Value
}

// feeScale — bojxona yig'imi shkalasi: bojxona qiymati AQSh dollarida,
// yig'im esa BRV ning karrasi sifatida. ПКМ 55, 1-ilova, 1-band «а».
//
// Chegara qiymati (masalan aniq 10 000 USD) quyi pog'onaga kiritiladi:
// qonunda "10 000 gacha" va "10 000 dan 20 000 gacha" deyilgan, ya'ni
// chegara ikki bandda ham uchraydi — biz quyisini olamiz.
var feeScale = []struct {
	UpToUSD    float64 // shu qiymatgacha (shu qiymat bilan birga)
	Multiplier float64 // BRV karrasi
}{
	{10_000, 1},
	{20_000, 1.5},
	{40_000, 2.5},
	{60_000, 4},
	{100_000, 7},
	{200_000, 10},
	{500_000, 15},
	{1_000_000, 20},
	{math.Inf(1), 25},
}

// FeeScaleText — yig'im shkalasini chat kontekstiga qo'yish uchun matn
// ko'rinishida qaytaradi.
//
// NEGA GENERATSIYA: shkalani tizim ko'rsatmasiga qo'lda ko'chirib yozsak,
// ikkita manba paydo bo'lardi — stavka o'zgarganda biri yangilanib,
// ikkinchisi eskirib qolishi mumkin. Shu sababli matn aynan feeScale va
// brvHistory dan chiqariladi: raqam bir joyda o'zgarsa, prompt ham o'zgaradi.
func FeeScaleText(on time.Time) string {
	brv := BRV(on)
	var b strings.Builder
	fmt.Fprintf(&b, "BRV: %s so'm\n", thousands(brv))
	prev := float64(0)
	for _, s := range feeScale {
		var band string
		switch {
		case math.IsInf(s.UpToUSD, 1):
			band = thousands(prev) + " USD dan yuqori"
		case prev == 0:
			band = thousands(s.UpToUSD) + " USD gacha"
		default:
			band = thousands(prev) + "–" + thousands(s.UpToUSD) + " USD"
		}
		fmt.Fprintf(&b, "  %s → %s×BRV = %s so'm\n",
			band, trimNum(s.Multiplier), thousands(s.Multiplier*brv))
		prev = s.UpToUSD
	}
	return b.String()
}

// thousands — 412000 → "412 000".
func thousands(v float64) string {
	s := itoa(int64(math.Round(v)))
	var b []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ' ')
		}
		b = append(b, c)
	}
	return string(b)
}

// feeMultiplier — bojxona qiymatiga (USD da) mos BRV karrasini qaytaradi.
func feeMultiplier(valueUSD float64) float64 {
	for _, s := range feeScale {
		if valueUSD <= s.UpToUSD {
			return s.Multiplier
		}
	}
	return 25
}

const (
	inspectDayBRV   = 0.25 // ish vaqtida: 25% BRV / soat
	inspectNightBRV = 2.0  // ish vaqtidan tashqari, dam olish va bayram: 2× BRV / soat
	preliminaryOff  = 0.20 // oldindan deklaratsiya qilinganda yig'imga 20% chegirma
)

// Request — hisoblash uchun kirish ma'lumotlari.
type Request struct {
	// Date — hisob-kitob sanasi. BRV shu sanaga olinadi.
	// Bo'sh bo'lsa joriy sana ishlatiladi.
	Date time.Time `json:"date,omitempty"`

	// Bojxona qiymatini ikki usulda berish mumkin.
	// 1) To'g'ridan-to'g'ri so'mda:
	CustomsValue float64 `json:"customs_value,omitempty"`
	// 2) Yoki komponentlardan (faktura + transport) valyuta kursi bilan:
	Invoice      float64 `json:"invoice,omitempty"`       // faktura qiymati, valyutada
	Transport    float64 `json:"transport,omitempty"`     // transport xarajati, valyutada
	CurrencyRate float64 `json:"currency_rate,omitempty"` // 1 birlik = N so'm

	// USDRate — 1 USD necha so'm. Yig'im shkalasi dollarda bo'lgani uchun zarur.
	USDRate float64 `json:"usd_rate,omitempty"`

	// Stavkalar, foizda.
	ImportDuty float64 `json:"import_duty"`          // 20. Bojxona boji
	ExtraDuty  float64 `json:"extra_duty,omitempty"` // 21. Qo'shimcha boj
	Excise     float64 `json:"excise"`               // 27. Aksiz (advalor)
	VAT        float64 `json:"vat"`                  // 29. QQS

	// Yig'im shartlari.
	FeeExempt   bool `json:"fee_exempt,omitempty"`  // yig'im olinmaydigan rejim
	Preliminary bool `json:"preliminary,omitempty"` // oldindan deklaratsiya (20% chegirma)

	// Ko'rik soatlari (to'liq soatga yuqoriga yaxlitlanadi).
	InspectDay   float64 `json:"inspect_day,omitempty"`   // ish vaqtida
	InspectNight float64 `json:"inspect_night,omitempty"` // ish vaqtidan tashqari

	Quantity float64 `json:"quantity,omitempty"` // miqdor (ma'lumot uchun)
}

// LineItem — hisob-kitobning bitta qatori.
type LineItem struct {
	Code   string  `json:"code"`           // GTD to'lov kodi: 10, 12, 20, 21, 27, 29
	Name   string  `json:"name"`           //
	Rate   float64 `json:"rate,omitempty"` // stavka, % (bo'lsa)
	Base   float64 `json:"base,omitempty"` // hisoblash bazasi
	Amount float64 `json:"amount"`         // to'lov summasi
	Note   string  `json:"note,omitempty"` //
}

// Result — to'liq hisob-kitob natijasi.
type Result struct {
	CustomsValue float64    `json:"customs_value"` // bojxona qiymati, so'm
	BRV          float64    `json:"brv"`           // qo'llanilgan BRV
	Items        []LineItem `json:"items"`
	Total        float64    `json:"total"`
}

// Calculate — bojxona to'lovlarini hisoblaydi.
//
//	Bojxona qiymati (TQ) = (faktura + transport) × kurs
//	20. Boj          = TQ × boj%
//	21. Qo'shimcha   = TQ × qo'shimcha%
//	27. Aksiz        = TQ × aksiz%            (SK 285-modda: baza — bojxona qiymati)
//	29. QQS          = (TQ + boj + qo'sh + aksiz) × QQS%   (SK 254-modda)
//	10. Yig'im       = BRV × shkala(TQ dollarda)           (ПКМ 55)
//	12. Ko'rik       = BRV × (0,25×kunduzgi + 2×tungi) soat
func Calculate(r Request) Result {
	on := r.Date
	if on.IsZero() {
		on = time.Now()
	}
	brv := BRV(on)

	// Bojxona qiymati.
	cv := r.CustomsValue
	if cv == 0 && r.CurrencyRate > 0 {
		cv = (r.Invoice + r.Transport) * r.CurrencyRate
	}
	if cv < 0 {
		cv = 0
	}

	var items []LineItem
	var total float64
	add := func(it LineItem) {
		it.Base = round(it.Base)
		it.Amount = round(it.Amount)
		items = append(items, it)
		total += it.Amount
	}

	// 10. Bojxona yig'imi — bojxona qiymatining dollardagi ekvivalenti bo'yicha.
	fee, feeNote := 0.0, ""
	switch {
	case r.FeeExempt:
		feeNote = "bu rejimda yig'im olinmaydi"
	case r.USDRate <= 0:
		feeNote = "USD kursi berilmagan — yig'im hisoblanmadi"
	default:
		mult := feeMultiplier(cv / r.USDRate)
		fee = brv * mult
		feeNote = formatMult(mult)
		if r.Preliminary {
			fee *= 1 - preliminaryOff
			feeNote += ", oldindan deklaratsiya uchun -20%"
		}
	}
	add(LineItem{Code: "10", Name: "Bojxona yig'imi", Amount: fee, Note: feeNote})

	// 12. Ko'rik — to'liq soatga yuqoriga yaxlitlanadi (ПКМ 55, ***** izoh).
	if h := math.Ceil(r.InspectDay); h > 0 {
		add(LineItem{Code: "12", Name: "Ko'rik (ish vaqtida)",
			Amount: brv * inspectDayBRV * h, Note: hoursNote(h, "25% BRV/soat")})
	}
	if h := math.Ceil(r.InspectNight); h > 0 {
		add(LineItem{Code: "12", Name: "Ko'rik (ish vaqtidan tashqari)",
			Amount: brv * inspectNightBRV * h, Note: hoursNote(h, "2× BRV/soat")})
	}

	// 20. Import boji.
	importDuty := cv * r.ImportDuty / 100
	add(LineItem{Code: "20", Name: "Bojxona boji", Rate: r.ImportDuty, Base: cv, Amount: importDuty})

	// 21. Qo'shimcha boj.
	extraDuty := cv * r.ExtraDuty / 100
	if r.ExtraDuty > 0 {
		add(LineItem{Code: "21", Name: "Qo'shimcha bojxona boji", Rate: r.ExtraDuty, Base: cv, Amount: extraDuty})
	}

	// 27. Aksiz — SK 285-modda: advalor stavkada baza bojxona qiymati (bojsiz).
	excise := cv * r.Excise / 100
	if r.Excise > 0 {
		add(LineItem{Code: "27", Name: "Aksiz solig'i", Rate: r.Excise, Base: cv, Amount: excise})
	}

	// 29. QQS — SK 254-modda: bojxona qiymati + boj + aksiz (yig'imsiz).
	vatBase := cv + importDuty + extraDuty + excise
	add(LineItem{Code: "29", Name: "QQS", Rate: r.VAT, Base: vatBase, Amount: vatBase * r.VAT / 100})

	return Result{CustomsValue: round(cv), BRV: brv, Items: items, Total: round(total)}
}

func hoursNote(h float64, tariff string) string {
	return trimNum(h) + " soat × " + tariff
}

func formatMult(m float64) string {
	return trimNum(m) + " × BRV"
}

// trimNum — 1.5 → "1,5", 4 → "4" (ortiqcha nolsiz).
func trimNum(v float64) string {
	s := ""
	if v == math.Trunc(v) {
		s = itoa(int64(v))
	} else {
		s = itoa(int64(v)) + "," + itoa(int64(math.Round((v-math.Trunc(v))*10)))
	}
	return s
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func round(v float64) float64 {
	return math.Round(v*100) / 100
}
