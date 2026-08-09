// Package hscode TIF TN (TN VED) kodlar bazasini yuklaydi va qidiruvni ta'minlaydi.
//
// Baza tools/extract-hscodes.mjs orqali manba manbasidan generatsiya qilinadi.
// Fayl formati: {"meta": {...}, "codes": [...]}
package hscode

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"strings"
)

// Meta — bazaning kelib chiqishi haqidagi ma'lumot.
// Foydalanuvchiga "bazada nima bor va u qachongi holat" deb ko'rsatish uchun.
type Meta struct {
	Nomenclature       string `json:"nomenclature"`
	LegalBasis         string `json:"legal_basis"`
	TransitionList     string `json:"transition_list,omitempty"`
	InternationalBasis string `json:"international_basis,omitempty"`
	RatesAsOf          string `json:"rates_as_of"`
	Source             string `json:"source,omitempty"`
	SourceDBVersion    int    `json:"source_db_version,omitempty"`
	ExtractedAt        string `json:"extracted_at"`
	TotalCodes         int    `json:"total_codes"`
	Note               string `json:"note,omitempty"`
	UnitNote           string `json:"unit_note,omitempty"`

	// ExciseNote — aksiz bu bazada YO'Qligini tushuntiradi.
	// ExciseKnownCodes — stavkasi ma'lum bo'lgan kodlar soni (hozircha 0).
	ExciseNote string `json:"excise_note,omitempty"`

	// VATNote — QQS kodga xos emas, umumiy stavka ekanini tushuntiradi.
	// Manba bazada "nds" maydoni hamma yozuvda bir xil, ya'ni imtiyozlar
	// bu yerda aks etmagan.
	VATNote string `json:"vat_note,omitempty"`

	ExciseKnownCodes int `json:"excise_known_codes"`
}

// Code — bitta tovar nomenklatura yozuvi.
type Code struct {
	Code    string `json:"code"`    // 10 xonali, probelsiz
	NameRU  string `json:"name_ru"` // yakuniy tugun nomi
	NameUZ  string `json:"name_uz"` //
	PathRU  string `json:"path_ru"` // to'liq ierarxik tavsif
	PathUZ  string `json:"path_uz"` //
	Section string `json:"section"` // bo'lim (rim raqami)
	Group   string `json:"group"`   // guruh (kodning birinchi 2 raqami)

	// Qo'shimcha o'lchov birligi. Bo'sh bo'lsa — faqat kg (asosiy birlik).
	UnitCode string `json:"unit_code"`
	Unit     string `json:"unit"`
	UnitRU   string `json:"unit_ru"`

	// Stavkalar, foizda (meta.rates_as_of sanasiga).
	ImportDuty float64 `json:"import_duty"`
	ExportDuty float64 `json:"export_duty"`
	VAT        float64 `json:"vat"`
	DutyLaw    string  `json:"duty_law,omitempty"`

	// KOMBINATSIYALANGAN STAVKA — 1 555 kodda (13 142 dan).
	//
	// "10%, lekin 1 kg uchun 0,5 dollardan kam bo'lmagan" ko'rinishidagi
	// stavkaning qat'iy qismi: dollarda, bitta birlik uchun.
	// Birlik — ImportDutySpecificUnit (kg, dona, l, juft, m²).
	//
	// nil = qat'iy qism YO'Q. 0 deb yozib qo'yish uni "0 dollar" bilan
	// aralashtirib yuborardi.
	ImportDutySpecific     *float64 `json:"import_duty_specific,omitempty"`
	ImportDutySpecificUnit string   `json:"import_duty_specific_unit,omitempty"`

	// Excise — aksiz stavkasi, foizda.
	//
	// nil = NOMA'LUM (bu bazada ma'lumot yo'q), 0% DEGANI EMAS.
	// Manba bazaning "an" maydoni bo'sh, shuning uchun hozircha hamma kodda nil.
	// Haqiqiy stavkalar Soliq kodeksining 289¹–289³ moddalarida, tovar nomi
	// bo'yicha — ular qonun korpusida (laws.json) mavjud.
	Excise *float64 `json:"excise,omitempty"`

	// search — qidiruv uchun oldindan tayyorlangan kichik harfli matn.
	// JSON'ga chiqmaydi, yuklashda to'ldiriladi.
	search string
}

// Match — qidiruv natijasi, moslik bali bilan.
type Match struct {
	Code  Code    `json:"code"`
	Score float64 `json:"score"`
}

// Store — xotiradagi kodlar bazasi.
type Store struct {
	meta  Meta
	codes []Code
}

type storeFile struct {
	Meta  Meta   `json:"meta"`
	Codes []Code `json:"codes"`
}

// Load — JSON fayldan kodlar bazasini yuklaydi.
func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f storeFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	// Qidiruv matnini oldindan tayyorlaymiz — har so'rovda qayta hisoblamaslik uchun.
	for i := range f.Codes {
		c := &f.Codes[i]
		c.search = normalize(c.Code + " " + c.PathUZ + " " + c.PathRU)
	}
	return &Store{meta: f.Meta, codes: f.Codes}, nil
}

// Meta — baza haqidagi ma'lumot.
func (s *Store) Meta() Meta { return s.meta }

// All — barcha kodlarni qaytaradi.
func (s *Store) All() []Code { return s.codes }

// ByCode — aniq kod bo'yicha yozuvni topadi. Probel va tirelar e'tiborsiz.
func (s *Store) ByCode(code string) (Code, bool) {
	want := normalizeCode(code)
	for _, c := range s.codes {
		if c.Code == want {
			return c, true
		}
	}
	return Code{}, false
}

// normalize — matnni qidiruv uchun bir ko'rinishga keltiradi.
//
// O'zbek lotin yozuvida "oʻ" va "gʻ" uchun maxsus belgi (U+02BB) ishlatiladi,
// tutuq uchun U+02BC. Foydalanuvchi esa klaviaturadagi oddiy ' ni bosadi.
// Normallashtirmasak, "qog'oz" so'rovi "qogʻoz" matnini topmaydi — bu
// o'zbekchadagi juda ko'p so'zga ta'sir qiladi.
//
// Shuningdek: "muzlatkich" ↔ "muzlatgich" kabi -kich/-gich almashinuvi
// rasmiy matn bilan kundalik yozuv orasidagi keng tarqalgan farq.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = apostrophes.Replace(s)
	return strings.ReplaceAll(s, "kich", "gich")
}

var apostrophes = strings.NewReplacer(
	"ʻ", "'", // ʻ  oʻ, gʻ
	"ʼ", "'", // ʼ  tutuq belgisi
	"‘", "'", // '
	"’", "'", // '
	"´", "'", // ´
	"`", "'",
)

func normalizeCode(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Search — matnli so'rov bo'yicha eng mos kodlarni topadi.
//
// Kod bo'yicha qidiruv (raqamlar kiritilsa) va matn bo'yicha qidiruv
// (o'zbekcha yoki ruscha) qo'llab-quvvatlanadi.
func (s *Store) Search(query string, limit int) []Match {
	q := normalize(strings.TrimSpace(query))
	if q == "" {
		return nil
	}

	// So'rov asosan raqamlardan iborat bo'lsa — kod bo'yicha qidiramiz.
	if digits := normalizeCode(q); len(digits) >= 4 && float64(len(digits)) > float64(len(q))*0.6 {
		return s.searchByCode(digits, limit)
	}

	terms := contentTerms(strings.Fields(q))
	terms = appendPhraseTerms(terms, q)
	if len(terms) == 0 {
		return nil
	}
	idf := idfOf(s.codes, terms)

	var matches []Match
	for i := range s.codes {
		if sc := score(&s.codes[i], q, terms, idf); sc > 0 {
			matches = append(matches, Match{Code: s.codes[i], Score: sc})
		}
	}
	sortMatches(matches)
	return trim(matches, limit)
}

// ⚠️ AHAMIYATLILIK CHEGARASI BU YERDA MUMKIN EMAS — sinab ko'rilgan va
// O'LCHOV BILAN RAD ETILGAN. Qayta urinilmasin.
//
// Fikr shunday edi: xom ball hadlar yig'indisi bo'lgani uchun so'rovlar
// orasida taqqoslanmaydi (ma'nosiz uzun jumla 109, aniq "noutbuk" 73),
// demak uni so'rovning nazariy maksimal balliga bo'lib normallashtiraylik
// va past nisbatlilarni tashlaylik.
//
// Qisqa so'rovlarda bu chiroyli ishladi (shovqin 0,15–0,27, haqiqiy
// 0,49–1,11). Lekin BUTUN EVAL to'plamida o'lchanganda ikki to'plam
// USTMA-UST tushdi:
//
//	0,066  HAQIQIY  "Koreyadan 2019-yilgi Hyundai Sonata, 2.0 litr…"
//	0,124  HAQIQIY  "Dongfeng musor tashuvchi mashina, 3 dona…"
//	0,153  shovqin  "Bu invoysdagi tovar, miqdor va narxni o'qi…"
//	0,272  shovqin  "kim yaratgan seni"
//	0,333  HAQIQIY  "naushnik"
//
// Sabab: "Hyundai", "Sonata", "Dongfeng", "Koreyadan" — noyob so'zlar,
// ular MAXRAJNI shishiradi, lekin nomenklatura matnida hech qachon
// uchramaydi. Ya'ni maxraj real erishish mumkin bo'lgan tavanni emas,
// xayoliy tavanni o'lchaydi. Aynan brend nomi yozadigan foydalanuvchi
// eng ko'p jazolanardi.
//
// 0,35 chegarasi bilan eval top-5 22/22 dan 18/22 ga tushdi.
// Chinakam yechim — embedding qidiruvi (README, "Ma'lum kamchiliklar").

// processWords — TOVAR emas, JARAYON haqidagi so'zlar.
//
// Foydalanuvchi butun jumla yozadi ("10 000 dollarlik traktor import qilsam
// qancha to'lov chiqadi?"), biz esa undan TOVARNI topishimiz kerak. Bu
// so'zlar tovar tavsiflarida ham uchraydi va qidiruvni buzadi:
//
//	"to'lov"  → 9704 "to'lov markalari" (pochta markalari)
//	"jadval"  → 9504 "stol o'yinlari"
//
// Haqiqiy misolda "…traktor import qilsam qancha to'lov chiqadi?" so'rovi
// 8701 (traktorlar) o'rniga 9704 (pochta markalari) ni birinchi qilgan edi:
// "to'lov" atamasi "traktor" dan kamroq uchraganligi uchun og'irroq bo'lgan.
//
// DIQQAT: ro'yxat faqat MA'NOLI atama qolganda qo'llanadi. "bojxona to'lovi"
// deb qidirilsa, hamma so'z chiqib ketardi — bunda filtrsiz qidiramiz.
var processWords = map[string]bool{
	"import": true, "eksport": true, "qancha": true, "qanday": true,
	"kerak": true, "javob": true, "javobni": true, "ber": true, "bering": true,
	"ko'rsat": true, "jadval": true, "bilan": true, "uchun": true,
	"to'lov": true, "to'lovlar": true, "tolov": true, "kurs": true,
	"narx": true, "narxi": true, "summa": true, "hujjat": true, "hujjatlar": true,
	"qilsam": true, "qilmoqchiman": true, "chiqadi": true, "boj": true,
	"bojxona": true, "soliq": true, "aksiz": true, "dollarlik": true,
	"dollar": true, "so'm": true, "som": true, "olib": true, "kirish": true,
	"hisobla": true, "hisoblab": true, "mening": true,

	// O'LCHOV BIRLIKLARI. Ular tovar tavsifida ham uchraydi
	// ("100 litrli rezervuar"), shuning uchun qidiruvni buzadi:
	// "Hyundai Sonata 2.0 litr" so'rovi 7309 (rezervuarlar) ni
	// birinchi qilgan edi — chunki mos kelgan yagona so'z "litr".
	"litr": true, "litrli": true, "tonna": true, "tonnalik": true,
	"kilogramm": true, "yilgi": true, "yillik": true, "metr": true,
}

// synonyms — kundalik so'zni nomenklatura so'ziga bog'laydi.
//
// NEGA KERAK: TIF TN rasmiy til bilan yozilgan va unda "noutbuk" ham,
// "Hyundai" ham YO'Q. Noutbuk u yerda "hisoblash mashinalari", avtomobil
// esa markasi bilan emas, turi bilan ataladi. Kalit so'z qidiruvi bu
// bo'shliqni o'zi to'ldira olmaydi.
//
// Bu to'liq yechim EMAS — chinakam yechim embedding qidiruvi. Lekin
// eng ko'p uchraydigan holatlarni arzon narxda qoplaydi.
//
// Markalar manbasi: manba "avto" jadvali (hammasida CODE = 8703).
var synonyms = map[string]string{
	// Avtomobil markalari
	"chevrolet": "avtomobil", "daewoo": "avtomobil", "hyundai": "avtomobil",
	"kia": "avtomobil", "toyota": "avtomobil", "lexus": "avtomobil",
	"honda": "avtomobil", "nissan": "avtomobil", "mazda": "avtomobil",
	"mitsubishi": "avtomobil", "subaru": "avtomobil", "suzuki": "avtomobil",
	"infiniti": "avtomobil", "daihatsu": "avtomobil", "bmw": "avtomobil",
	"mercedes": "avtomobil", "audi": "avtomobil", "volkswagen": "avtomobil",
	"cobalt": "avtomobil", "malibu": "avtomobil", "captiva": "avtomobil",
	"nexia": "avtomobil", "matiz": "avtomobil", "spark": "avtomobil",

	// MAXSUS AVTOTRANSPORT (8705) — nomenklatura bo'shlig'i.
	//
	// 8705 tavsifida misollar sanab o'tilgan ("аварийные, автокраны,
	// пожарные, автобетономешалки, для уборки дорог, поливомоечные,
	// автомастерские, рентгеновские"), lekin MUSOROVOZ yo'q — bazada
	// "musor"/"мусор" so'zi 8705 da umuman uchramaydi.
	//
	// Shuning uchun "musor tashuvchi mashina" so'rovi 8433 (qishloq
	// xo'jaligi mashinalari) ni birinchi qilgan edi — bu indeks xatosi
	// emas, LUG'AT bo'shlig'i: hech qanday kalit so'z qidiruvi mavjud
	// bo'lmagan so'zni topa olmaydi.
	//
	// Nishon "avtotransport" — u faqat 8705 ning bosh bo'g'inida uchraydi
	// ("Maxsus avtotransport vositalari"), shuning uchun boshqa guruhni
	// tortib kelmaydi. "maxsus" nishon sifatida yaramaydi: u seyf, dastgoh
	// qismlari va videoo'yin konsollarida ham bor.
	//
	// DIQQAT: bu yerga faqat AYNAN TRANSPORT bildiruvchi so'z qo'shiladi.
	// "musor"/"chiqindi" yakka o'zi qo'shilmadi — "plastik chiqindi
	// import qilsam" so'rovini 3825 (chiqindilar) dan 8705 ga burib
	// yuborardi. Tovar emas, transport nomlari kiritilgan.
	"musorovoz": "avtotransport", "musorvoz": "avtotransport",
	"axlatvoz": "avtotransport", "assenizator": "avtotransport",
	"evakuator": "avtotransport", "avtokran": "avtotransport",
	"betonaralashtirgich": "avtotransport", "avtobeton": "avtotransport",
	"suvsepar": "avtotransport", "poliv": "avtotransport",
	"avtomastеrskaya": "avtotransport", "avtoustaxona": "avtotransport",

	// Texnika — nomenklaturadagi atama bilan
	"noutbuk": "hisoblash", "laptop": "hisoblash", "kompyuter": "hisoblash",
	"planshet": "hisoblash",
	"smartfon": "telefon", "telefon": "telefon",
	"televizor": "televizor", "konditsioner": "konditsioner",
	"muzlatkich": "muzlatgich",
}

// phraseSynonyms — KO'P SO'ZLI ibora → nomenklatura atamasi.
//
// NEGA ALOHIDA: synonyms yakka so'zga ishlaydi, lekin foydalanuvchi
// texnikani ibora bilan ataydi — "musor tashuvchi mashina", "axlat
// presslovchi avtomobil". Bu yerdagi so'zlarning HECH BIRI yakka o'zi
// transportni bildirmaydi:
//
//	"musor"   yakka o'zi → chiqindi (3825), transport emas
//	"mashina" yakka o'zi → har qanday mashina, juda umumiy
//
// Faqat BIRGALIKDA transport ma'nosini beradi. Shuning uchun ibora butun
// so'rov matnida qidiriladi, tokenlarda emas.
var phraseSynonyms = map[string]string{
	"musor tashuvchi":      "avtotransport",
	"musor tashish":        "avtotransport",
	"axlat tashuvchi":      "avtotransport",
	"axlat tashish":        "avtotransport",
	"chiqindi tashuvchi":   "avtotransport",
	"musor mashina":        "avtotransport",
	"axlat mashina":        "avtotransport",
	"axlat presslovchi":    "avtotransport",
	"musor presslovchi":    "avtotransport",
	"yo'l tozalash":        "avtotransport",
	"yol tozalash":         "avtotransport",
	"o't o'chirish":        "avtotransport",
	"ot ochirish":          "avtotransport",
	"beton aralashtir":     "avtotransport",
	"suv sepuvchi":         "avtotransport",
	"maxsus avtotransport": "avtotransport",
}

// appendPhraseTerms — so'rovda ibora uchrasa, nomenklatura atamasini qo'shadi.
// Asl atamalar o'z joyida qoladi — ular boshqa guruhda mos kelishi mumkin.
func appendPhraseTerms(terms []term, q string) []term {
	seen := make(map[string]bool, len(terms))
	for _, t := range terms {
		seen[t.text] = true
	}
	for phrase, target := range phraseSynonyms {
		if !strings.Contains(q, normalize(phrase)) {
			continue
		}
		// Ibora tan olindi — uning SO'ZLARI endi alohida ma'no bermaydi.
		//
		// "musor tashuvchi" birgalikda transportni anglatadi. Yakka
		// "musor" esa chiqindini anglatadi va 8417 (musor YOQUVCHI pech)
		// ni tortib keladi — o'lchovda u 8705 ni birinchi o'rindan
		// siqib chiqargan edi. Ibora aniqlangach, uning tarkibiy
		// so'zlari tuzilmaviy ball bermaydi: ma'noni endi nishon
		// atama ("avtotransport") ko'taradi.
		for _, w := range strings.Fields(normalize(phrase)) {
			for i := range terms {
				if terms[i].text == w {
					terms[i].weak = true
				}
			}
		}
		if !seen[target] {
			seen[target] = true
			terms = append(terms, newTerm(target))
		}
	}
	return terms
}

// ---------------------------------------------------- atama variantlari

// term — qidiruv atamasi va uning yozuv/qo'shimcha variantlari.
//
// NEGA VARIANT KERAK: indeks ikki tilda — path_uz LOTIN o'zbekcha,
// path_ru KIRILL ruscha. Foydalanuvchi esa uchinchi ko'rinishda yozishi
// mumkin: kirill o'zbekcha ("наушниклар") yoki ruscha so'zning lotin
// translitiratsiyasi ("naushnik"). Ikkalasi ham indeksdagi hech bir
// matnga to'g'ridan-to'g'ri mos kelmaydi.
//
// Haqiqiy misol (deklarant.uz dagi so'rov):
//
//	"наушниклар" → indeksda "наушники" bor, lekin Contains ishlamaydi
//	               (so'rov UZUNROQ), natija — hech narsa
//	"naushnik"   → indeksda umuman lotin ruscha yo'q, natija — hech narsa
//
// DIQQAT: variantlar ATAMA darajasida, ball darajasida emas. Har bir
// atama baribir BIR MARTA ball beradi — aks holda ko'p variantli so'z
// sun'iy og'irlashib, reytingni buzardi.
type term struct {
	text     string   // asl atama — IDF va nosozlikni tekshirish uchun
	variants []string // asl + qo'shimchasiz + o'girilgan ko'rinishlar

	// weak — bu atamaning nomenklaturada O'Z ekvivalenti bor, ya'ni
	// sinonim orqali almashtirildi ("kompyuter" → "hisoblash").
	//
	// Sinonim mavjudligining o'zi shuni bildiradi: BAZA BU SO'ZNI
	// ISHLATMAYDI. Shuning uchun so'z baribir matnda uchrasa, u odatda
	// tovarning O'ZI emas, unga HAVOLA bo'ladi.
	//
	// Haqiqiy misol: "kompyuter" so'rovi monitorni (8528) kompyuterdan
	// (8471) oldinga chiqarib yubordi. Sabab — 8471 tavsifida "kompyuter"
	// so'zi UMUMAN yo'q ("hisoblash mashinalari" deb yozilgan), monitor
	// tavsifida esa bor: "kompyuterlarga to'g'ridan-to'g'ri ulangan".
	//
	// Shuning uchun weak atama faqat umumiy matnda ball beradi, bosh/ona/
	// barg bo'g'inlarida bermaydi — tuzilmaviy signalni sinonim ko'taradi.
	weak bool
}

// in — atamaning biror varianti matnda uchraydimi.
func (t term) in(hay string) bool {
	for _, v := range t.variants {
		if strings.Contains(hay, v) {
			return true
		}
	}
	return false
}

// short — juda qisqa atama (shovqin beradi, e'tiborsiz qoldiriladi).
func (t term) short() bool { return len([]rune(t.text)) < 3 }

// newTerm — atamadan barcha qidiriladigan ko'rinishlarni yasaydi.
func newTerm(s string) term {
	t := term{text: s}
	seen := map[string]bool{}
	add := func(v string) {
		if v == "" || len([]rune(v)) < 3 || seen[v] {
			return
		}
		seen[v] = true
		t.variants = append(t.variants, v)
	}

	add(s)
	add(stripUzSuffix(s))
	// Boshqa yozuvga o'girib, u yerda ham qo'shimchani kesamiz.
	if other := translit(s); other != s {
		add(other)
		add(stripUzSuffix(other))
	}
	return t
}

// uzSuffixes — o'zbek tilining eng ko'p uchraydigan qo'shimchalari,
// ikkala yozuvda. Uzunroqlari OLDINDA turishi shart: "ларни" dan avval
// "лар" kesilsa, "ни" osilib qolardi.
//
// Ro'yxat ataylab qisqa: har bir qo'shimcha noto'g'ri kesish xavfini
// oshiradi. Masalan "-i" kiritilmadi — u "kali" → "kal" kabi begona
// mosliklarni keltirib chiqarardi.
var uzSuffixes = []string{
	// kirill
	"ларни", "лардан", "ларга", "ларда", "лари", "лар",
	"нинг", "дан", "га", "да", "ни", "си",
	// lotin
	"larni", "lardan", "larga", "larda", "lari", "lar",
	"ning", "dan", "ga", "da", "ni", "si",
}

// minStem — qo'shimcha kesilgandan keyin qolishi shart bo'lgan eng kam
// uzunlik. Bo'lmasa "gaz" → "z" ga aylanib, hamma narsaga mos kelardi.
const minStem = 4

// stripUzSuffix — so'z oxiridagi bitta qo'shimchani kesadi.
func stripUzSuffix(s string) string {
	for _, suf := range uzSuffixes {
		if !strings.HasSuffix(s, suf) {
			continue
		}
		stem := strings.TrimSuffix(s, suf)
		if len([]rune(stem)) >= minStem {
			return stem
		}
	}
	return stripIzafat(s)
}

// stripIzafat — UNDOSHDAN keyingi qaratqich "-i" sini kesadi.
//
// NEGA ALOHIDA: o'zbekchada tovar deyarli DOIM shu shaklda ataladi —
// "havo konditsioneri", "telefon apparati", "yuk mashinasi". Unlidan
// keyin qo'shimcha "-si" bo'ladi va u yuqoridagi ro'yxatda bor;
// undoshdan keyin esa yalang "-i" va u yo'q edi.
//
// Jonli oqibat: "havo konditsioneri" so'rovi 8415 ni UMUMAN qaytarmadi
// (bazada "konditsionerlar" deb yozilgan, "konditsioneri" esa unga
// substring bo'lib tushmaydi) — o'rniga yog'-moy guruhi chiqdi.
//
// NEGA UNDOSH SHARTI: "-si" bilan chegara aniq bo'lgani kabi, bu ham
// haqiqiy qoidaga tayanadi. Shartsiz kesilsa "dori" → "dor" kabi asl
// so'zning oxiri yeyilardi.
//
// Eski izoh "-i" ni "kali" → "kal" xavfi uchun rad etgan edi; aslida
// minStem uni allaqachon to'sadi ("kal" — 3 belgi, chegara 4).
func stripIzafat(s string) string {
	r := []rune(s)
	if len(r) < minStem+1 || (r[len(r)-1] != 'i' && r[len(r)-1] != 'и') {
		return s
	}
	if isVowel(r[len(r)-2]) {
		return s // unlidan keyin qo'shimcha "-si" bo'ladi, yalang "-i" emas
	}
	return string(r[:len(r)-1])
}

func isVowel(r rune) bool {
	return strings.ContainsRune("aeiouаеёиоуўэюя", r)
}

// cyrToLat / latToCyr — kirill ↔ lotin o'girish jadvallari.
//
// Ko'p harfli birikmalar (ch, sh, yo…) BIRINCHI bo'lishi kerak, aks
// holda "ch" → "цх" bo'lib ketardi. strings.Replacer buni o'zi
// ta'minlaydi: u eng uzun mos keluvchini oladi.
var cyrToLat = strings.NewReplacer(
	"ё", "yo", "ж", "j", "ц", "ts", "ч", "ch", "ш", "sh", "щ", "sh",
	"ю", "yu", "я", "ya", "ў", "o'", "қ", "q", "ғ", "g'", "ҳ", "h",
	"а", "a", "б", "b", "в", "v", "г", "g", "д", "d", "е", "e",
	"з", "z", "и", "i", "й", "y", "к", "k", "л", "l", "м", "m",
	"н", "n", "о", "o", "п", "p", "р", "r", "с", "s", "т", "t",
	"у", "u", "ф", "f", "х", "x", "ъ", "", "ы", "i", "ь", "", "э", "e",
)

var latToCyr = strings.NewReplacer(
	"yo", "ё", "yu", "ю", "ya", "я", "ch", "ч", "sh", "ш", "ts", "ц",
	"o'", "ў", "g'", "ғ",
	"a", "а", "b", "б", "v", "в", "g", "г", "d", "д", "e", "е",
	"j", "ж", "z", "з", "i", "и", "y", "й", "k", "к", "l", "л",
	"m", "м", "n", "н", "o", "о", "p", "п", "r", "р", "s", "с",
	"t", "т", "u", "у", "f", "ф", "x", "х", "h", "ҳ", "q", "қ",
)

// translit — so'zni qarama-qarshi yozuvga o'giradi.
func translit(s string) string {
	if hasCyrillic(s) {
		return cyrToLat.Replace(s)
	}
	return latToCyr.Replace(s)
}

func hasCyrillic(s string) bool {
	for _, r := range s {
		if r >= 0x0400 && r <= 0x04FF {
			return true
		}
	}
	return false
}

// procSuffixes — jarayon so'zini tanish uchun kesiladigan qo'shimchalar.
//
// Bu ro'yxat uzSuffixes dan KENGROQ va eng kam o'zak ham qisqaroq (3).
// Buni xavfsiz qiladigan narsa — natija baribir processWords da
// bo'lishi shart, ya'ni noto'g'ri kesishning zarari YOPIQ lug'at bilan
// chegaralangan: "mashinini" → "mash" hech qayerda yo'q, e'tiborsiz
// qoladi. Umumiy qidiruvda bunday agressiv kesish xavfli bo'lardi.
var procSuffixes = []string{
	"larini", "ining", "lari", "ini", "ning", "lar",
	"dan", "ga", "da", "ni", "si", "i",
}

// isProcessWord — atama TOVAR emas, JARAYON haqidami.
//
// NEGA QO'SHIMCHA BILAN: foydalanuvchi "boj" deb emas, "bojini",
// "to'lovini", "narxini" deb yozadi. Aniq moslik bilan ular filtrdan
// o'tib ketardi.
//
// Bu nosozlik ilgari ham bor edi, lekin ZARARSIZ edi: "bojini" indeksdagi
// hech narsaga mos kelmasdi. Yozuv o'girish qo'shilgach u "божини" ni ham
// sinaydigan bo'ldi va 4907 (marka) ga yuqori ball berib, "musor tashish
// mashinasi" so'rovini butunlay buzdi. Ya'ni yangi imkoniyat eski
// bo'shliqni ochib berdi.
func isProcessWord(t string) bool {
	if processWords[t] {
		return true
	}
	// Kirill o'zbekchada yozilgan bo'lsa ham tanilsin ("божини").
	cands := []string{t}
	if lat := translit(t); lat != t {
		cands = append(cands, lat)
	}
	for _, c := range cands {
		for _, suf := range procSuffixes {
			stem := strings.TrimSuffix(c, suf)
			if stem != c && len([]rune(stem)) >= 3 && processWords[stem] {
				return true
			}
		}
	}
	return false
}

// contentTerms — so'rovdan tovarga oid atamalarni ajratadi.
func contentTerms(fields []string) []term {
	var kept, all []string
	for _, t := range fields {
		t = strings.Trim(t, ".,!?()[]:;\"")
		if len([]rune(t)) < 3 {
			continue
		}
		all = append(all, t)
		if !isProcessWord(t) {
			kept = append(kept, t)
		}
	}
	// Hammasi jarayon so'zi bo'lsa (masalan "bojxona to'lovi") — filtrsiz.
	if len(kept) == 0 {
		kept = all
	}

	// Nomenklatura atamasini qo'shamiz: "Hyundai" → "avtomobil".
	// Asl so'z ham qoladi — u boshqa joyda mos kelishi mumkin.
	seen := make(map[string]bool, len(kept))
	for _, t := range kept {
		seen[t] = true
	}
	for _, t := range kept {
		if syn, ok := synonyms[t]; ok && !seen[syn] {
			seen[syn] = true
			kept = append(kept, syn)
		}
	}
	// Sinonim qidiruvi ASL so'z bo'yicha bo'ldi (yuqorida), endi har bir
	// atamaga yozuv va qo'shimcha variantlarini yasaymiz.
	out := make([]term, 0, len(kept))
	for _, t := range kept {
		nt := newTerm(t)
		// Nomenklatura ekvivalenti bor — asl so'z tuzilmaviy ball bermaydi.
		if _, replaced := synonyms[t]; replaced {
			nt.weak = true
		}
		out = append(out, nt)
	}
	return out
}

// idfOf — har bir atamaning noyobligini hisoblaydi.
//
// NEGA KERAK: atamalar teng vaznda bo'lsa, ko'p UMUMIY so'zga mos kelgan kod
// bitta MA'NOLI so'zga mos kelganini bosib ketadi. Haqiqiy misol:
//
//	"dori vositalari (bez ekstraktlari) import qilmoqchiman.
//	 Qanday hujjatlar kerak? Jadval bilan ko'rsat."
//
// Bu so'rov 9504 (bilyard, videoo'yin konsollari) ni birinchi o'ringa
// chiqargan edi — u "bilan", "kerak", "jadval", "ko'rsat" kabi bo'sh
// so'zlarga mos kelgani uchun. Holbuki "ekstrakt" atamasi 3001 ni aniq
// ko'rsatib turardi. IDF bilan noyob atama og'irroq bo'ladi.
func idfOf(codes []Code, terms []term) []float64 {
	idf := make([]float64, len(terms))
	n := float64(len(codes))
	for j, t := range terms {
		if t.short() {
			continue
		}
		// Hujjat chastotasi VARIANTLAR bo'yicha — atama qaysi yozuvda
		// yozilganidan qat'i nazar, uning noyobligi bir xil bo'lishi kerak.
		df := 0
		for i := range codes {
			if t.in(codes[i].search) {
				df++
			}
		}
		idf[j] = math.Log((n + 1) / (float64(df) + 1))
	}
	return idf
}

func (s *Store) searchByCode(digits string, limit int) []Match {
	var matches []Match
	for i := range s.codes {
		c := &s.codes[i]
		switch {
		case c.Code == digits:
			matches = append(matches, Match{Code: *c, Score: 100})
		case strings.HasPrefix(c.Code, digits):
			matches = append(matches, Match{Code: *c, Score: 50})
		}
	}
	sortMatches(matches)
	return trim(matches, limit)
}

func score(c *Code, q string, terms []term, idf []float64) float64 {
	var sc float64
	// To'liq so'rov aynan uchrasa — eng kuchli signal, IDF siz.
	if strings.Contains(c.search, q) {
		sc += 10
	}
	for j, t := range terms {
		if t.short() {
			continue // juda qisqa so'zlar shovqin beradi
		}
		if t.in(c.search) {
			sc += 3 * idf[j]
		}
	}
	if sc == 0 {
		return 0
	}

	// Ierarxiyaning BOSH bo'g'ini — tovar pozitsiyasi sarlavhasi, ya'ni
	// "bu narsa NIMA". Izohlardan ancha kuchli signal.
	//
	// Misol: "traktor" so'rovida
	//   8701…  "Traktorlar (…); …; boshqalar"        ← bosh bo'g'inda
	//   8407…  "Ichki yonuv dvigatellari; traktorlar uchun"  ← izohda
	// Ikkalasi ham mos keladi, lekin foydalanuvchi traktorni so'ragan.
	head := normalize(headSegment(c.PathUZ) + " " + headSegment(c.PathRU))
	leaf := normalize(c.NameUZ + " " + c.NameRU)
	// ONA BO'G'IN — bargdan bittagina yuqoridagi, ya'ni ENG ANIQ tasnif.
	//
	// NEGA KERAK: 4 xonali guruh sarlavhasi bir nechta tovar oilasini
	// sanaydi va u guruhdagi HAMMA kod uchun bir xil. Barg nomi esa
	// ko'pincha shunchaki "прочие". Ya'ni farqlovchi matn ikkalasida ham
	// emas — u o'rtada.
	//
	// Haqiqiy misol, 8518 guruhi (deklarant.uz da sinalgan "наушниклар"):
	//
	//	[0] Микрофоны и подставки для них          ← hamma 8518 da bir xil
	//	[5] наушники и телефоны головные…          ← FAQAT 8518 30 da
	//	[6] прочие                                  ← barg, foydasiz
	//
	// Ona bo'g'in baholanmagani uchun butun 8518 guruhi teng ball olgan
	// va tartib kod raqamiga qarab qolgan edi — natijada "наушники"
	// so'roviga mikrofon (8518 10) quloqchindan (8518 30) oldin chiqardi.
	parent := normalize(parentSegment(c.PathUZ) + " " + parentSegment(c.PathRU))
	for j, t := range terms {
		if t.short() || t.weak {
			continue
		}
		if t.in(head) {
			sc += 8 * idf[j]
		}
		if t.in(parent) {
			sc += parentWeight * idf[j]
		}
		// Yakuniy tugun nomida uchrasa — aniqroq moslik.
		if t.in(leaf) {
			sc += 4 * idf[j]
		}
	}
	return sc
}

// parentWeight — ona bo'g'in ballining og'irligi.
//
// Qiymat TAXMIN QILINMADI, 18 ta holatli eval to'plamida o'lchandi
// (TestParentWeightIsMeasured):
//
//	vazn  top-1   top-5
//	  0   11/18   15/18   ← naushnik holatlari yiqiladi (tuzatish maqsadi)
//	  2   13/18   18/18
//	  3   13/18   18/18   ← tanlangan
//	  4   12/18   17/18
//	  6   13/18   17/18   ← 8417 (musor yoquvchi pech) 8705 ni bosib ketadi
//
// Yuqori vazn HAVOLALI mosliklarni tovar pozitsiyasining o'zidan ustun
// qilib qo'yadi: 8417 tavsifida musor yoqish pechlari sanalgan va u
// "musor tashuvchi mashina" so'rovini o'ziga tortib oladi.
var parentWeight = 3.0

// parentSegment — ierarxiyaning oxirgidan oldingi bo'g'ini ("A; B; C" → "B").
//
// Oxirgi bo'g'in — bargning o'zi (NameRU/NameUZ bilan bir xil), shuning
// uchun u alohida baholanadi. Bo'g'in bitta bo'lsa, ona yo'q.
func parentSegment(path string) string {
	parts := strings.Split(path, ";")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-2])
}

// headSegment — ierarxik tavsifning birinchi bo'g'ini ("A; B; C" → "A").
func headSegment(path string) string {
	if i := strings.Index(path, ";"); i >= 0 {
		return path[:i]
	}
	return path
}

func sortMatches(m []Match) {
	sort.SliceStable(m, func(i, j int) bool {
		if m[i].Score != m[j].Score {
			return m[i].Score > m[j].Score
		}
		return m[i].Code.Code < m[j].Code.Code
	})
}

func trim(m []Match, limit int) []Match {
	if limit > 0 && len(m) > limit {
		return m[:limit]
	}
	return m
}
