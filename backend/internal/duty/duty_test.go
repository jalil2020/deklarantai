package duty

import (
	"math"
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
