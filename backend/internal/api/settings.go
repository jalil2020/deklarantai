package api

// Sozlamalar ro'yxati — server AYNAN qanday konfiguratsiya bilan
// ishlayotganini ko'rsatadi.
//
// NEGA KERAK: sozlamalar muhit o'zgaruvchilarida tarqoq va ularning bir
// qismi sukut qiymat bilan ishlaydi. "Server hozir qaysi modelda?",
// "kesh yoqilganmi?", "limit qancha?" degan savolga javob berish uchun
// serverga kirib `env` ni ko'rish kerak bo'lardi. Bu sahifa shuni bir
// joyga yig'adi.
//
// XAVFSIZLIK: MAXFIY qiymat HECH QACHON qaytarilmaydi — na to'liq, na
// qisman. Faqat "sozlangan / sozlanmagan" holati beriladi. Kalitning
// oxirgi belgilarini ko'rsatish odatiy amaliyot bo'lsa-da, bu panel
// parol bilan himoyalangan bo'lsa ham, sirni tarqatish uchun yana bir
// kanal ochib berardi.

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

// setting — bitta sozlama haqidagi ma'lumot.
type setting struct {
	Name    string `json:"name"`
	Group   string `json:"group"`
	Value   string `json:"value"`   // amaldagi qiymat (maxfiy bo'lsa bo'sh)
	Default string `json:"default"` // sukut qiymat
	Desc    string `json:"desc"`
	Source  string `json:"source"`  // "muhit" yoki "sukut"
	Secret  bool   `json:"secret"`  // qiymati yashiriladi
	Set     bool   `json:"set"`     // muhitda o'rnatilganmi
	Restart bool   `json:"restart"` // o'zgarishi uchun qayta ishga tushirish kerakmi
}

// settingDef — registrdagi ta'rif. Qiymat o'qish vaqtida olinadi.
type settingDef struct {
	name, group, def, desc string
	secret, restart        bool
}

// settingsRegistry — BARCHA sozlamalar. Yangi muhit o'zgaruvchisi
// qo'shilsa, shu yerga ham yoziladi — aks holda panel to'liq bo'lmaydi.
//
// restart=true — qiymat ishga tushishda bir marta o'qiladi (masalan
// llm.New() klientni o'shanda yasaydi), shuning uchun o'zgartirish uchun
// serverni qayta ishga tushirish kerak.
// restart=false — qiymat har so'rovda o'qiladi, ya'ni darrov ta'sir qiladi.
var settingsRegistry = []settingDef{
	// ---- AI provayderlari
	{"ANTHROPIC_API_KEY", "AI — Claude", "", "Claude kaliti. Bo'lmasa chat butunlay o'chiq.", true, true},
	{"ANTHROPIC_MODEL", "AI — Claude", "claude-opus-4-8", "Asosiy model: rasm va hisob-kitob.", false, true},
	{"ANTHROPIC_MID_MODEL", "AI — Claude", "", "⚠️ SUKUT BO'YICHA O'CHIQ. Yoqilsa, faqat sof ma'lumot o'qish savollari («bu kod nimani anglatadi») shu modelga ketadi. Tasnif, imtiyoz, kelib chiqish va hisob-kitob baribir asosiy modelda qoladi.", false, true},
	{"ANTHROPIC_FAST_MODEL", "AI — Claude", "", "⚠️ SUKUT BO'YICHA O'CHIQ. Yoqilsa, faqat bo'sh gap («salom», «rahmat») shu modelga ketadi.", false, true},
	{"ANTHROPIC_API_URL", "AI — Claude", "https://api.anthropic.com/v1/messages", "Korporativ shlyuz yoki testdagi soxta server.", false, true},
	{"ANTHROPIC_MAX_TOKENS", "AI — Claude", "2048", "Javob uzunligi chegarasi (ikkala provayder uchun).", false, true},
	{"ANTHROPIC_CACHE_TTL", "AI — Claude", "", "Kesh muddati. Bo'sh = 5 daqiqa. \"1h\" — siyrak trafikda arzonroq (kesh so'rovlar orasida tirik qoladi).", false, true},

	{"GLM_ENABLED", "AI — GLM (Z.ai)", "", "⚠️ O'CHIQ. \"1\" qilinsagina GLM ishlatiladi; aks holda HAMMA so'rov Claude da.", false, true},
	{"GLM_API_KEY", "AI — GLM (Z.ai)", "", "Arzon provayder kaliti (GLM_ENABLED=1 bilan birga).", true, true},
	{"GLM_MODEL", "AI — GLM (Z.ai)", "glm-5.2", "Z.ai modeli. ⚠️ GLM-5.2 rasmni qo'llab-quvvatlamaydi.", false, true},
	{"GLM_API_URL", "AI — GLM (Z.ai)", "https://api.z.ai/api/paas/v4/chat/completions", "OpenAI-mos endpoint.", false, true},
	{"GLM_TIMEOUT_SECONDS", "AI — GLM (Z.ai)", "180", "1M kontekstda birinchi belgi kechikishi uzun bo'lishi mumkin.", false, true},

	// ---- Cheklovlar
	{"RATE_PER_MIN", "Cheklovlar", "10", "Bitta IP dan daqiqasiga chat so'rovi (0 = o'chiq).", false, true},
	{"DAILY_QUOTA", "Cheklovlar", "100", "Bitta IP dan kuniga chat so'rovi (0 = o'chiq).", false, true},
	{"MAX_BODY_BYTES", "Cheklovlar", "8388608", "So'rov hajmi chegarasi (rasm base64 bilan keladi).", false, false},
	{"TRUST_PROXY", "Cheklovlar", "", "1 bo'lsa X-Forwarded-For ga ishoniladi. ⚠️ Faqat ishonchli proksi orqasida.", false, false},

	// ---- Mijozni tanish
	{"API_KEYS", "Mijoz", "", "\"nom:sir\" ro'yxati (vergul bilan). Mobil ilova va hamkorlar uchun muddatsiz kalitlar.", true, true},
	{"CLIENT_TOKEN_SECRET", "Mijoz", "", "Anonim token imzosi. Berilmasa har ishga tushishda tasodifiy — ya'ni tokenlar kuchini yo'qotadi. Bir nechta nusxada BIR XIL bo'lishi shart.", true, true},
	{"CLIENT_TOKEN_TTL", "Mijoz", "24h", "Anonim token muddati (masalan 24h, 12h).", false, true},
	{"ALLOWED_ORIGINS", "Mijoz", "", "CORS uchun ruxsat etilgan manbalar. Bo'sh bo'lsa — hammasi (*).", false, false},

	{"USERS_DATA", "Mijoz", "data/users.json", "Foydalanuvchilar ombori. Parol xeshlari bor — git ga tushmasin.", false, true},

	// ---- Server
	{"PORT", "Server", "8080", "Server porti.", false, true},
	{"ADMIN_PASSWORD", "Server", "", "Admin paneli paroli. Bo'lmasa /admin yo'llari umuman mavjud emas.", true, true},

	// ---- Ma'lumot manbalari
	{"HSCODE_DATA", "Ma'lumot", "data/hscodes.json", "TIF TN kodlar bazasi.", false, true},
	{"LAWS_DATA", "Ma'lumot", "data/laws.json", "Qonun korpusi.", false, true},
	{"DOCS_DATA", "Ma'lumot", "data/docs.json", "Hujjat talablari.", false, true},
	{"COUNTRIES_DATA", "Ma'lumot", "data/countries.json", "Davlatlar ma'lumotnomasi (kelib chiqish).", false, true},
	{"TAXONOMY_DATA", "Ma'lumot", "data/taxonomy.json", "Bo'lim/guruh sarlavhalari (sidebar brauzeri). Ixtiyoriy.", false, true},

	// ---- Tashqi xizmatlar
	{"CBU_API_URL", "Tashqi xizmatlar", "https://cbu.uz/uz/arkhiv-kursov-valyut/json", "Markaziy bank valyuta kurslari.", false, true},
	{"CONTACT_TELEGRAM", "Tashqi xizmatlar", "https://t.me/declarant_pro", "Tadbirkor rejimida beriladigan murojaat kanali.", false, false},
}

// settings — registrni amaldagi qiymatlar bilan to'ldiradi.
func settings() []setting {
	out := make([]setting, 0, len(settingsRegistry))
	for _, d := range settingsRegistry {
		raw := os.Getenv(d.name)
		s := setting{
			Name: d.name, Group: d.group, Default: d.def, Desc: d.desc,
			Secret: d.secret, Set: raw != "", Restart: d.restart,
			Source: "sukut",
		}
		if raw != "" {
			s.Source = "muhit"
		}
		// Maxfiy qiymat hech qachon chiqmaydi.
		if !d.secret {
			s.Value = raw
			if raw == "" {
				s.Value = d.def
			}
		}
		out = append(out, s)
	}
	return out
}

// settingsGroups — guruhlangan ko'rinish (panel uchun qulay tartib).
type settingsGroup struct {
	Group string    `json:"group"`
	Items []setting `json:"items"`
}

func groupedSettings() []settingsGroup {
	order := []string{}
	byGroup := map[string][]setting{}
	for _, s := range settings() {
		if _, ok := byGroup[s.Group]; !ok {
			order = append(order, s.Group)
		}
		byGroup[s.Group] = append(byGroup[s.Group], s)
	}
	out := make([]settingsGroup, 0, len(order))
	for _, g := range order {
		out = append(out, settingsGroup{Group: g, Items: byGroup[g]})
	}
	return out
}

// health — sozlamalardan kelib chiqadigan ogohlantirishlar.
//
// Sozlama ro'yxatining o'zi "hammasi joyidami" degan savolga javob
// bermaydi — ADMIN_PASSWORD borligini ko'rish bilan uning kuchliligini
// bilib bo'lmaydi. Shuning uchun xavfli kombinatsiyalar alohida
// ta'kidlanadi.
func settingsWarnings() []string {
	var w []string

	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		w = append(w, "ANTHROPIC_API_KEY sozlanmagan — chat butunlay ishlamaydi.")
	}
	// GLM yoqilgan bo'lsa — buni AYTAMIZ. Javob boshqa provayderdan
	// kelayotgani ko'rinib turishi kerak: bu shunchaki xarajat masalasi
	// emas, javob sifati va yuridik mas'uliyat masalasi.
	if os.Getenv("GLM_ENABLED") == "1" && os.Getenv("GLM_API_KEY") != "" {
		w = append(w, "GLM yoqilgan — ma'lumot savollariga javobni Claude emas, Z.ai modeli beradi.")
	}
	if os.Getenv("GLM_ENABLED") == "1" && os.Getenv("GLM_API_KEY") == "" {
		w = append(w, "GLM_ENABLED=1, lekin GLM_API_KEY yo'q — provayder baribir ishlamaydi.")
	}
	if os.Getenv("TRUST_PROXY") == "1" {
		w = append(w, "TRUST_PROXY yoqilgan — X-Forwarded-For ga ishoniladi. Ishonchli proksi orqasida emassangiz, mijoz limitni aylanib o'tadi.")
	}
	if p := os.Getenv("ADMIN_PASSWORD"); p != "" && len(p) < 12 {
		w = append(w, fmt.Sprintf("ADMIN_PASSWORD qisqa (%d belgi) — kamida 12 belgi tavsiya etiladi.", len(p)))
	}
	if envInt("RATE_PER_MIN", 10) == 0 && envInt("DAILY_QUOTA", 100) == 0 {
		w = append(w, "Ikkala so'rov chegarasi ham o'chirilgan — bitta IP cheksiz so'rov yubora oladi.")
	}
	if u := os.Getenv("ANTHROPIC_API_URL"); u != "" && !strings.HasPrefix(u, "https://") {
		w = append(w, "ANTHROPIC_API_URL HTTPS emas — kalit ochiq kanal orqali ketadi.")
	}
	if u := os.Getenv("GLM_API_URL"); u != "" && !strings.HasPrefix(u, "https://") {
		w = append(w, "GLM_API_URL HTTPS emas — kalit ochiq kanal orqali ketadi.")
	}
	if os.Getenv("ALLOWED_ORIGINS") == "" {
		w = append(w, "ALLOWED_ORIGINS bo'sh — API ni istalgan sayt o'z sahifasidan chaqira oladi.")
	}
	// Sir tasodifiy bo'lsa, qayta ishga tushirish barcha mijozlarni
	// tokensiz qoldiradi. Bitta nusxada bu shunchaki noqulaylik, bir
	// nechta nusxada esa tokenlar umuman ishlamay qoladi.
	if os.Getenv("CLIENT_TOKEN_SECRET") == "" {
		w = append(w, "CLIENT_TOKEN_SECRET sozlanmagan — anonim tokenlar server qayta ishga tushganda kuchini yo'qotadi.")
	}

	sort.Strings(w)
	return w
}

// adminSettings — sozlamalar endpointi (faqat o'qish).
func (s *Server) adminSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"groups":   groupedSettings(),
		"warnings": settingsWarnings(),
		// Panel maxfiy qiymatlarni KO'RSATMAYDI — buni foydalanuvchiga
		// aytib qo'yamiz, aks holda "nega bo'sh?" degan savol tug'iladi.
		"note": "Maxfiy qiymatlar (kalitlar, parol) ataylab ko'rsatilmaydi — faqat sozlangan/sozlanmagan holati.",
	})
}
