package duty

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Utilizatsiya yig'imi (GTD to'lov kodi 79).
//
// HUQUQIY ASOS: ПКМ № 347 (02.06.2020), 1-ilova.
// O'zgartirishlar: ПКМ 665 (29.10.2020), ПП 422 (29.12.2023),
// ПКМ 727 (01.11.2024), ПКМ 52 (31.01.2025 — 1-ilova joriy tahriri,
// 2025 yil 1 maydan kuchga kirgan).
//
// Stavkalar BRV da berilgan: "BHM — toʻlov kuni uchun Oʻzbekiston
// Respublikasida tasdiqlangan bazaviy hisoblash miqdori". Ya'ni bizda
// allaqachon bor bo'lgan BRV jadvali ishlatiladi.
//
// Ikkita yosh toifasi: ishlab chiqarilganiga 3 yildan ORTIQ EMAS va
// 3 yildan ORTIQ. Ko'p toifada yangi texnika uchun stavka belgilanmagan
// (jadvalda "—") — bunday holda yig'im olinmaydi.

// utilMeasure — toifa qaysi o'lchov bo'yicha bo'linadi.
type utilMeasure string

const (
	measureNone utilMeasure = ""    // bo'linmaydi, bitta stavka
	measureCM3  utilMeasure = "cm3" // dvigatel ish hajmi, sm³
	measureKW   utilMeasure = "kw"  // quvvat, kVt
	measureHP   utilMeasure = "hp"  // quvvat, ot kuchi
	measureTon  utilMeasure = "ton" // to'la vazn, tonna
)

var measureName = map[utilMeasure]string{
	measureCM3: "dvigatel hajmi (sm³)",
	measureKW:  "quvvat (kVt)",
	measureHP:  "quvvat (ot kuchi)",
	measureTon: "to'la vazn (tonna)",
}

// utilRow — bitta pog'ona. New/Old — BRV karrasi.
// -1 qiymati "stavka belgilanmagan" (jadvaldagi "—") degani: yig'im olinmaydi.
type utilRow struct {
	UpTo float64 // shu qiymatgacha (shu qiymat bilan birga)
	New  float64 // ishlab chiqarilganiga 3 yildan ortiq emas
	Old  float64 // 3 yildan ortiq
}

type utilCategory struct {
	Name    string
	Codes   []string // TIF TN prefikslari (probelsiz)
	Except  []string // shu prefikslar bu toifaga KIRMAYDI
	Measure utilMeasure
	Rows    []utilRow
}

const notSet = -1.0

// utilTable — ПКМ 347, 1-ilova. Tartib muhim emas: kod bo'yicha eng
// ANIQ (uzun) mos keluvchi toifa tanlanadi.
var utilTable = []utilCategory{
	// I. Traktorlar — quvvat bo'yicha. Yangi texnikaga yig'im yo'q.
	{
		Name:    "Traktorlar",
		Codes:   []string{"8701100000", "8701201090", "8701209090", "870191", "870192", "870193", "870194", "870195"},
		Measure: measureKW,
		Rows: []utilRow{
			{18, notSet, 120},
			{37, notSet, 240},
			{75, notSet, 360},
			{130, notSet, 480},
			{math.Inf(1), notSet, 600},
		},
	},
	// Egarli tyagachlar — 8701 20 (yuqoridagi ikkita aniq koddan tashqari).
	{
		Name:    "Egarli tyagachlar",
		Codes:   []string{"870120"},
		Measure: measureNone,
		Rows:    []utilRow{{math.Inf(1), 670, 1360}},
	},

	// II. M2, M3 — mikroavtobus va avtobuslar.
	{
		Name:    "Avtobuslar (M2, M3)",
		Codes:   []string{"8702"},
		Except:  []string{"870240000"},
		Measure: measureCM3,
		Rows: []utilRow{
			{2500, 120, 150},
			{5000, 185, 360},
			{10000, 320, 540},
			{math.Inf(1), 540, 1080},
		},
	},
	{
		Name:    "Avtobuslar, elektrodvigatelli (gibrid bundan mustasno)",
		Codes:   []string{"870240000"},
		Measure: measureNone,
		Rows:    []utilRow{{math.Inf(1), 70, 150}},
	},

	// III. M1 — yengil avtomobillar. Eng ko'p uchraydigan toifa.
	{
		Name:    "Yengil avtomobillar (M1)",
		Codes:   []string{"8703"},
		Except:  []string{"870310", "870380000"},
		Measure: measureCM3,
		Rows: []utilRow{
			{1000, 30, 90},
			{2000, 120, 210},
			{3000, 180, 330},
			{3500, 180, 390},
			{math.Inf(1), 300, 480},
		},
	},
	{
		Name:    "Qorda yuruvchi va golf avtomobillari",
		Codes:   []string{"870310"},
		Measure: measureNone,
		Rows:    []utilRow{{math.Inf(1), 30, 90}},
	},
	{
		Name:    "Yengil avtomobil, elektrodvigatelli (gibrid bundan mustasno)",
		Codes:   []string{"870380000"},
		Measure: measureNone,
		Rows:    []utilRow{{math.Inf(1), 120, 210}},
	},

	// IV. N1, N2, N3 — yuk avtomobillari, to'la vazn bo'yicha.
	{
		Name:    "Yuk avtomobillari (N1, N2, N3)",
		Codes:   []string{"8704"},
		Except:  []string{"870490000"},
		Measure: measureTon,
		Rows: []utilRow{
			{2.5, 100, 150},
			{3.5, 210, 300},
			{5, 210, 300},
			{8, 210, 300},
			{12, 300, 810},
			{20, 330, 1200},
			{50, 690, 1410},
			// Qonunda 50 tonnadan yuqorisi ko'rsatilmagan.
			{math.Inf(1), notSet, notSet},
		},
	},
	{
		Name:    "Yuk avtomobili, elektrodvigatelli (gibrid bundan mustasno)",
		Codes:   []string{"870490000"},
		Measure: measureNone,
		Rows:    []utilRow{{math.Inf(1), 120, 150}},
	},

	// V. Maxsus transport vositalari.
	{
		Name:    "Maxsus transport vositalari",
		Codes:   []string{"8705"},
		Except:  []string{"8705400000"},
		Measure: measureNone,
		Rows:    []utilRow{{math.Inf(1), 195, 550}},
	},
	{
		Name:    "Avtobetonaralashtirgichlar",
		Codes:   []string{"8705400000"},
		Measure: measureNone,
		Rows:    []utilRow{{math.Inf(1), 600, 1500}},
	},

	// VI. Tirkamalar.
	{
		Name:    "Tirkama va yarim tirkamalar",
		Codes:   []string{"8716"},
		Except:  []string{"871680", "871690"},
		Measure: measureNone,
		Rows:    []utilRow{{math.Inf(1), 60, 660}},
	},

	// VII. Boshqa o'ziyurar mashinalar — ot kuchi bo'yicha.
	// Bularning hammasida yangi texnikaga yig'im belgilanmagan.
	{
		Name:    "Avtogreyderlar",
		Codes:   []string{"8429200010", "8429200091", "8429200099"},
		Measure: measureHP,
		Rows: []utilRow{
			{100, notSet, 240}, {140, notSet, 360}, {200, notSet, 480},
			{math.Inf(1), notSet, 600},
		},
	},
	{
		Name:    "Buldozerlar",
		Codes:   []string{"8429110010", "8429110020", "8429110090", "8429190001", "8429190009"},
		Measure: measureHP,
		Rows: []utilRow{
			{100, notSet, 240}, {200, notSet, 360}, {300, notSet, 480},
			{math.Inf(1), notSet, 600},
		},
	},
	{
		Name:    "Ekskavatorlar",
		Codes:   []string{"842952", "8429590000"},
		Measure: measureHP,
		Rows: []utilRow{
			{170, notSet, 240}, {250, notSet, 360}, {math.Inf(1), notSet, 480},
		},
	},
	{
		Name:    "Yo'l roliklari",
		Codes:   []string{"8429401000", "8429403000"},
		Measure: measureHP,
		Rows: []utilRow{
			{40, notSet, 240}, {80, notSet, 360}, {math.Inf(1), notSet, 480},
		},
	},
	{
		Name:    "O'ziyurar kranlar",
		Codes:   []string{"842641000"},
		Measure: measureHP,
		Rows: []utilRow{
			{170, notSet, 240}, {250, notSet, 360}, {math.Inf(1), notSet, 480},
		},
	},
	{
		Name:    "Quvur yotqizuvchi va gusenitsali kranlar",
		Codes:   []string{"8426490010", "8426490090"},
		Measure: measureHP,
		Rows: []utilRow{
			{130, notSet, 240}, {300, notSet, 360}, {math.Inf(1), notSet, 480},
		},
	},
	{
		Name:    "Old yuk ko'targichlar",
		Codes:   []string{"842710", "842720", "842951"},
		Measure: measureHP,
		Rows: []utilRow{
			{50, notSet, 240}, {100, notSet, 360}, {300, notSet, 480},
			{math.Inf(1), notSet, 600},
		},
	},
	{
		Name:    "O'rmon xo'jaligi mashinalari",
		Codes:   []string{"843680100"},
		Measure: measureHP,
		Rows: []utilRow{
			{100, notSet, 240}, {300, notSet, 360}, {math.Inf(1), notSet, 480},
		},
	},
	{
		Name:    "G'alla o'ruvchi kombaynlar",
		Codes:   []string{"8433510000"},
		Measure: measureHP,
		Rows: []utilRow{
			{160, notSet, 240}, {220, notSet, 360}, {320, notSet, 480},
			{math.Inf(1), notSet, 600},
		},
	},
	{
		Name:    "O'ziyurar yig'im-terim kombaynlari",
		Codes:   []string{"843359110"},
		Measure: measureHP,
		Rows: []utilRow{
			{250, notSet, 240}, {400, notSet, 360}, {math.Inf(1), notSet, 480},
		},
	},
	{
		Name:    "O'simliklarni himoya qilish uchun o'ziyurar purkagichlar",
		Codes:   []string{"842482", "8433201000"},
		Measure: measureHP,
		Rows: []utilRow{
			{120, notSet, 240}, {300, notSet, 360}, {math.Inf(1), notSet, 480},
		},
	},
}

// utilTyres — 2-ilova: shinalar. Yig'im HAR KILOGRAMM uchun, BRV ning
// foizida. ПП 422 (29.12.2023) bilan kiritilgan, 2024 yil 1 iyuldan.
//
// DIQQAT: qonun matnida ishlatilgan/qayta tiklangan shinalar qatorining
// foizi BO'SH qolgan (manba bazadagi jadvalda qiymat yo'q). Shuning uchun
// ular uchun stavka NOMA'LUM deb qaytariladi — 0 deb hisoblash yolg'on
// javob bo'lardi.
var utilTyres = []struct {
	Name   string
	Codes  []string
	PctBRV float64 // BRV ning foizi, 1 kg uchun; 0 = noma'lum
}{
	{"Yangi pnevmatik rezina shinalar", []string{"401110000", "401120", "4011900000"}, 0.3},
	{"Qayta tiklangan yoki ishlatilgan shinalar",
		[]string{"4012110000", "4012120000", "4012200009", "4012902000"}, 0},
}

// UtilFeeRequest — utilizatsiya yig'imini hisoblash uchun.
type UtilFeeRequest struct {
	Date time.Time `json:"date,omitempty"` // BRV shu sanaga olinadi

	Code string `json:"code"` // TIF TN kodi

	// Measure — toifaga mos o'lchov: dvigatel hajmi (sm³), quvvat (kVt
	// yoki ot kuchi) yoki to'la vazn (tonna). Qaysi biri kerakligi
	// toifaga qarab aniqlanadi; UtilFeeMeasure() bilan bilib olish mumkin.
	Measure float64 `json:"measure,omitempty"`

	// AgeYears — ishlab chiqarilganidan beri o'tgan yillar.
	// 3 dan katta bo'lsa yuqori stavka qo'llanadi.
	AgeYears float64 `json:"age_years,omitempty"`

	// WeightKg — shinalar uchun (2-ilova).
	WeightKg float64 `json:"weight_kg,omitempty"`
}

// UtilFeeResult — hisob natijasi.
type UtilFeeResult struct {
	Category string  `json:"category"`
	BRV      float64 `json:"brv"`
	Rate     float64 `json:"rate,omitempty"` // BRV karrasi yoki foizi
	Amount   float64 `json:"amount"`
	Note     string  `json:"note,omitempty"`
	// NeedsMeasure — qaysi o'lchov kerakligi (berilmagan bo'lsa).
	NeedsMeasure string `json:"needs_measure,omitempty"`
}

// normCode — "8703 23 194 0" → "8703231940".
func normCode(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// UtilizationFee — utilizatsiya yig'imini hisoblaydi (GTD kodi 79).
//
// Kod jadvalga tushmasa XATO qaytariladi — 0 emas. Sababi aksiz bilan
// bir xil: "yig'im yo'q" deb javob berish, aslida bor bo'lsa, jim va
// xavfli xato bo'lardi.
func UtilizationFee(r UtilFeeRequest) (UtilFeeResult, error) {
	code := normCode(r.Code)
	if code == "" {
		return UtilFeeResult{}, fmt.Errorf("TIF TN kodi ko'rsatilmagan")
	}
	on := r.Date
	if on.IsZero() {
		on = time.Now()
	}
	brv := BRV(on)

	// Shinalar (2-ilova) — vazn bo'yicha.
	for _, t := range utilTyres {
		if !matchAny(code, t.Codes) {
			continue
		}
		if t.PctBRV == 0 {
			return UtilFeeResult{Category: t.Name, BRV: brv}, fmt.Errorf(
				"%s uchun stavka manba hujjatda ko'rsatilmagan — ПКМ 347, 2-ilovani tekshiring", t.Name)
		}
		if r.WeightKg <= 0 {
			return UtilFeeResult{
				Category: t.Name, BRV: brv, Rate: t.PctBRV,
				NeedsMeasure: "vazn (kg)",
				Note:         "yig'im har kilogramm uchun hisoblanadi — vaznni ko'rsating",
			}, nil
		}
		amount := round(r.WeightKg * brv * t.PctBRV / 100)
		return UtilFeeResult{
			Category: t.Name, BRV: brv, Rate: t.PctBRV, Amount: amount,
			Note: fmt.Sprintf("%s kg × %g%% × BRV (ПКМ 347, 2-ilova)", trimNum(r.WeightKg), t.PctBRV),
		}, nil
	}

	// Transport vositalari (1-ilova) — eng aniq mos keluvchi toifa.
	cat := bestUtilCategory(code)
	if cat == nil {
		return UtilFeeResult{}, fmt.Errorf(
			"bu kod utilizatsiya yig'imi ro'yxatida topilmadi (ПКМ 347). "+
				"Ro'yxatda yo'q tovar uchun yig'im olinmaydi, lekin buni ilovadan tasdiqlang: kod %s", r.Code)
	}

	if cat.Measure != measureNone && r.Measure <= 0 {
		return UtilFeeResult{
			Category: cat.Name, BRV: brv,
			NeedsMeasure: measureName[cat.Measure],
			Note:         "stavka " + measureName[cat.Measure] + " ga bog'liq — uni ko'rsating",
		}, nil
	}

	row := cat.Rows[len(cat.Rows)-1]
	for _, x := range cat.Rows {
		if r.Measure <= x.UpTo {
			row = x
			break
		}
	}

	rate, age := row.New, "3 yildan ortiq emas"
	if r.AgeYears > 3 {
		rate, age = row.Old, "3 yildan ortiq"
	}
	if rate == notSet {
		return UtilFeeResult{
			Category: cat.Name, BRV: brv, Amount: 0,
			Note: fmt.Sprintf("%s: ishlab chiqarilganiga %s bo'lgan texnika uchun "+
				"ПКМ 347 ning 1-ilovasida stavka belgilanmagan — yig'im olinmaydi", cat.Name, age),
		}, nil
	}

	return UtilFeeResult{
		Category: cat.Name, BRV: brv, Rate: rate, Amount: round(rate * brv),
		Note: fmt.Sprintf("%s, %s → %s × BRV (ПКМ 347, 1-ilova)", cat.Name, age, trimNum(rate)),
	}, nil
}

// bestUtilCategory — kodga eng ANIQ mos keluvchi toifa.
//
// "8701 20 109 0" ham "8701201090" (traktor), ham "870120" (egarli
// tyagach) prefikslariga mos keladi. Uzunroq prefiks aniqroq, shuning
// uchun u yutadi.
func bestUtilCategory(code string) *utilCategory {
	var best *utilCategory
	bestLen := -1
	for i := range utilTable {
		c := &utilTable[i]
		if matchAny(code, c.Except) {
			continue
		}
		n := longestMatch(code, c.Codes)
		if n > bestLen {
			best, bestLen = c, n
		}
	}
	if bestLen < 0 {
		return nil
	}
	return best
}

func matchAny(code string, prefixes []string) bool {
	return longestMatch(code, prefixes) >= 0
}

// longestMatch — mos kelgan eng uzun prefiks uzunligi; mos kelmasa -1.
func longestMatch(code string, prefixes []string) int {
	best := -1
	for _, p := range prefixes {
		if strings.HasPrefix(code, p) && len(p) > best {
			best = len(p)
		}
	}
	return best
}

// UtilFeeMeasure — kod uchun qanday o'lchov kerakligini aytadi
// (chat foydalanuvchidan nimani so'rashini bilishi uchun).
func UtilFeeMeasure(code string) string {
	c := bestUtilCategory(normCode(code))
	if c == nil {
		return ""
	}
	return measureName[c.Measure]
}

// HasUtilFee — kodga utilizatsiya yig'imi qo'llanadimi.
// O'lchov kerak bo'lmagan toifalar uchun ham true qaytaradi
// (UtilFeeMeasure bunday holda bo'sh satr beradi).
func HasUtilFee(code string) bool {
	c := normCode(code)
	if c == "" {
		return false
	}
	for _, t := range utilTyres {
		if matchAny(c, t.Codes) {
			return true
		}
	}
	return bestUtilCategory(c) != nil
}
