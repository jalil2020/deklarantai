# 🛃 Deklarant AI

O'zbekiston bojxona rasmiylashtiruvi uchun AI yordamchi. Yagona interfeys — **chat**.

1. **💬 Chat** — bojxona kod, boj va qonunchilik bo'yicha suhbat.
   **Rasm o'qiydi**: tovar surati yoki invoysni yuklang — AI (Claude vision) undagi tovar,
   miqdor va narxni o'qib, TIF TN kodini taklif qiladi va bojni hisoblab beradi.
   Rasm tanlash, nusxa-joylash (paste) yoki sudrab tashlash (drag & drop) mumkin.
Boj hisoblash va kod qidirish AI ning o'zi orqali, suhbat ichida bajariladi —
alohida kalkulyator yoki qidiruv oynasi yo'q. Ular backendda API sifatida
qoladi (`/api/duty/calculate`, `/api/hscode/search`), lekin frontendda faqat
chat bor.

> ⚠️ Stavkalar baza olingan sanaga tegishli va o'zgarib turadi. Muhim
> qarorlar uchun [customs.uz](https://customs.uz) yoki bojxona brokeridan
> tasdiqlash kerak.

## Texnologiyalar

| Qism      | Texnologiya                        |
|-----------|------------------------------------|
| Backend   | Go (standart `net/http`, tashqi kutubxonasiz) |
| Frontend  | React + TypeScript + Vite          |
| AI        | Anthropic Claude Messages API      |

## Loyiha tuzilishi

```
Deklarant AI/
├── backend/                  # Go REST API
│   ├── main.go
│   ├── data/hscodes.json     # TIF TN bazasi (13 142 kod)
│   ├── data/laws.json        # qonun korpusi (1 405 parcha)
│   ├── data/docs.json        # hujjat talablari (15 112 qoida)
│   ├── data/countries.json   # davlatlar va boj rejimi (254 ta)
│   └── internal/
│       ├── api/              # HTTP handlerlar + CORS
│       ├── hscode/           # kod qidiruv
│       ├── duty/             # boj hisoblash
│       ├── docs/             # hujjat talablari (kod oralig'i bo'yicha)
│       ├── countries/        # kelib chiqish davlati -> boj koeffitsienti
│       ├── rates/            # Markaziy bank valyuta kurslari
│       ├── laws/             # qonun korpusi qidiruvi
│       ├── chat/             # RAG + suhbat
│       └── llm/              # Claude API klienti
├── frontend/                 # React SPA
│   └── src/
│       ├── api.ts            # backend klienti
│       └── components/       # Chat, Sidebar, LawsPanel, Markdown
└── android/                  # Native ilova (Kotlin + Compose + Voyager)
    └── app/src/main/kotlin/uz/deklarant/ai/
        ├── domain/           # modellar + repozitoriy interfeyslari
        ├── data/             # DTO, Ktor klienti, rasm siqish
        └── ui/               # Compose ekranlari + ScreenModel
```

Uchala mijoz ham **bitta backend'ga** ulanadi. Android uchun manzil
qurish vaqtida beriladi (`-PapiBaseUrl=...`) — tafsilotlar
[android/README.md](android/README.md) da.

## Interfeys tuzilishi

Chapda doimiy menyu, o'ngda bo'lim. Ikki guruh:

| Bo'lim | Nima qiladi |
|---|---|
| **Chat** | Suhbat, rasm biriktirish, oqim bilan javob |
| **Qidiruv** | TIF TN kodini topish — erkin matn yoki ierarxiya |
| **Kalkulyator** | Boj va soliqlar; utilizatsiya yig'imi (79) |
| **Risk baholash** | Kod bo'yicha ruxsatnoma, shartli imtiyoz va ma'lum bo'shliqlar |
| **Qonunlar** | Hujjat → modda → to'liq matn, lex.uz havolasi bilan |
| — *Saqlangan* — | |
| **Tarixcha** | O'tgan suhbatlar — ochish, davom ettirish, o'chirish |
| **Sevimlilar** | Belgilangan kodlar va moddalar |

Menyu keng ekranda oqimdagi ustun (`position: sticky`), tor ekranda esa
chetdan chiqadigan panel. Ikki holat ATAYLAB boshqa mexanizmda: ilgari
ikkalasi ham `transform` ga tayanardi va keng ekrandagi bekor qilish
ishlamay, menyu ekrandan tashqarida qolib ketgan edi.

### Risk baholash

Kod bo'yicha rasmiylashtiruvni **nima to'xtatib qo'yishi** mumkinligini
ko'rsatadi. Uch manba: `/api/requirements` (litsenziya va sertifikat),
`/api/exemptions` (shartli imtiyoz), `/api/utilfee` (yig'im qo'llanadimi).
Ularga hisoblashdagi bilib turib qoldirilgan bo'shliqlar qo'shiladi
(aksiz, kelib chiqish koeffitsienti).

⚠️ **Raqamli "risk foizi" ATAYLAB YO'Q.** Har bir band bazadagi aniq
yozuvga tayanadi. Ball qo'yish ishonchli ko'rinardi-yu, ortida hech narsa
turmasdi — deklarant esa unga qarab qaror qabul qilardi.

`/api/requirements` javobida ro'yxat **to'liq emasligi** aytiladi va bu
interfeysda ham ko'rsatiladi: manba bazada kod bo'yicha aniq ko'rsatma
bermaydigan hujjatlar yo'q, ya'ni bo'sh ro'yxat "hech narsa kerak emas"
degani emas.

Sinaldi: `8703 23 194 0` (yengil avtomobil) → 1 sertifikat, 5 boshqa
talab (МЮ 2773-4, ПКМ 41…), utilizatsiya yig'imi qo'llanadi.

### Tarixcha va Sevimlilar

Yozuvlar **brauzerda** (`localStorage`) saqlanadi — server tarafda
saqlash akkaunt va bazani talab qiladi, ular keyingi bosqichda. Chegara
interfeysda ochiq aytiladi: yozuvlar shu qurilmada qoladi va brauzer
tozalansa yo'qoladi.

⚠️ **Suratlar saqlanmaydi.** Bitta 1600px surat base64 da ~300 KB,
brauzer xotirasi esa odatda 5 MB — ikki-uchta suratli suhbat butun
tarixchani to'ldirib, saqlashni umuman ishdan chiqarardi. Sinaldi:
50 KB lik surat bilan uchta suhbat → 457 bayt.

Dizayn maketidagi **Bosh sahifa** va **Profil** hozircha qo'shilmadi:
birinchisi foydalanuvchi ko'rsatkichlarini, ikkinchisi akkauntni talab
qiladi.

### Qidiruv natijalari

Har kartochkada: kod, nomi, ota-tugun, **boj/QQS stavkasi**, o'lchov
birligi va imtiyoz nishoni.

⚠️ **"Eng mos" nishoni faqat g'olib YOLG'IZ bo'lganda chiqadi.** O'lchov
shuni ko'rsatdi: "noutbuk" so'roviga to'rtta kod aynan bir xil ball
oladi (73,177). Bunday holatda birinchisiga "eng mos" deb yozish qidiruv
topmagan g'olibni o'ylab topish bo'lardi — deklarant esa shu nishonga
qarab kod tanlaydi. Teng ball bo'lsa, ro'yxat tepasida nechta kod teng
kelgani va tartib tasodifiy ekani **ochiq aytiladi**.

Ball yonidagi chiziq ham foiz emas — u eng yaxshi natijaga NISBATAN
ko'rsatkich (ball IDF asosidagi ochiq shkalada, ehtimollik emas).

### Ierarxiya brauzeri

TIF TN bo'ylab **qidiruvsiz** yurish (Qidiruv bo'limining "Ko'rish" tabi).

**Nega kerak.** Qidiruv foydalanuvchidan tovarni *nomenklatura tilida*
atashni talab qiladi. Bilmasa — hech narsa topilmaydi: "musor tashuvchi
mashina" nomenklaturada "maxsus avtotransport", "noutbuk" esa "hisoblash
mashinalari". Ierarxiya hech qanday atama bilishni talab qilmaydi.

To'rt daraja, har birida ichidagi kodlar soni ko'rsatiladi:

```
Bo'lim (21)  →  Guruh (96)  →  Tovar pozitsiyasi (4 xonali)  →  Kod
   XVI            84                    8418                  8418 10 200 1
```

Kodni bosish chatga tayyor savol qo'yadi (yubormaydi — foydalanuvchi
qiymat va davlatni qo'shishi kerak).

```
GET /api/hscode/browse                 → bo'limlar
GET /api/hscode/browse?section=XVI     → guruhlar
GET /api/hscode/browse?group=84        → tovar pozitsiyalari
GET /api/hscode/browse?heading=8418    → kodlar
```

### Kalkulyator

Tuzilishi GTD ga mos: tepada uch davlat tanlagichi, keyin uch tab
(Kalkulyator · Hujjatlar · Ierarxiya), kirish maydonlari ikki ustunda,
natija esa **to'lov kodlari ro'yxati** ko'rinishida.

**Davlat tanlagichlari** (`GET /api/countries`, 254 davlat):

| Grafa | Hisobga ta'siri |
|---|---|
| Yuk jo'natuvchi | Yo'q — GTD uchun. Tanlansa qolgan ikkitasini avto-to'ldiradi |
| **Kelib chiqish** | **Boj koeffitsienti** (BK 300-modda): erkin savdo ×0, EQR ×1, rejimsiz/aniqlanmagan ×2 |
| Savdo qiluvchi | Yo'q — offshor bo'lsa ogohlantirish |

Koeffitsientni frontend emas, **backend aniqlaydi** — kalkulyator faqat
davlat kodini yuboradi (`origin_country`). Ro'yxat va qoida bitta joyda.
Tanlagich ostida rejim darrov ko'rinadi («Erkin savdo — boj olinmaydi»).

Tanlagich — **yozib qidiriladigan kombo** (`CountryCombo.tsx`), oddiy
`<select>` emas: 254 davlat kod tartibida turadi va Angolani topish uchun
024 gacha aylantirish kerak edi. Qidiruv kod, o'zbekcha/ruscha nom, ISO
va sinonimlar bo'ylab («156», «ang», «герман», «de», «XXR» — hammasi
ishlaydi). Aniq mosliklar oldinda: «de» → Germaniya (ISO), keyingina
Bangla**de**sh. Har band ostida boj rejimi ko'rinadi, offshor belgi bilan.
Klaviatura: ↑/↓, Enter, Esc.

Ogohlantirishlar: offshor tomon; kelib chiqish jo'natuvchidan farq
qilsa — ST-1 talab qilinishi. Jonli sinov (`9405 42 003 9`, 1 000 kg):
Xitoy → 6 046 675 so'm, Rossiya (erkin savdo) → 0 va QQS ham qayta
hisoblandi, «aniqlanmagan» → 12 093 350 (ikki baravar, qat'iy qism ham).

**Tab'lar:** Hujjatlar — kod bo'yicha talablar (Risk bilan bitta manba,
qisqa ko'rinish); Ierarxiya — daraxt tanlangan kod pozitsiyasida
ochiladi, undan boshqa kod tanlansa kalkulyator o'sha kodga qayta
yuklanadi (to'liq stavkalar qidiruvdan olinadi — browse bargida qat'iy
qism yo'q).

**Kod maydoni — YOZGANDA qidiradi** (`CodeCombo.tsx`). Tab qatorining
o'ng tomonida turadi va ikki xil yozuvni qabul qiladi:

| Yozilgan | Nima bo'ladi |
|---|---|
| `8703 23` | Shu pozitsiyadagi kodlar ro'yxati (bo'shliqlar ahamiyatsiz) |
| `noutbuk` | Nom bo'yicha qidiradi — kodni bilmasa ham |

Har taklifda kod, **boj stavkasi**, kombinatsiyalangan qat'iy qism
(`$0.5/kg`) va o'lchov birligi ko'rinadi — tanlashdan oldin farqi
bilinsin. Tanlangach stavkalar darrov tortiladi va hisoblash mumkin.
Klaviatura: ↑/↓, Enter, Esc; ✕ tozalaydi.

Qidiruv SERVERDA — 13 142 kod brauzerga yuklanmaydi. Har harfda so'rov
ketmasligi uchun **200 ms kechikish**: sinovda 10 ta ketma-ket
o'zgarishga atigi **1 ta so'rov** ketdi. Sekin kelgan eski javob
yangisini bosib ketmasligi uchun so'rovlar raqamlanadi.

`/api/hscode/search` ga `limit` qo'shildi (sukut 5, taklif ro'yxatiga 12,
yuqori chegara 20) — butun bazani tortib bo'lmasin.

⚠️ Ierarxiyadan kelgan **10 xonali kod AYNAN topilishi shart** — yo'q
bo'lsa xato chiqadi, "yaqinini" jimgina yuklamaydi: boshqa kod boshqa
stavka degani. Sinovda shu qoida foydali chiqdi: `8703231940` (eski
nomenklatura kodi) TIF TN 2025 da YO'Q — haqiqiysi `8703231941`/…49.

Backend `matches: null` qaytarganda (hech narsa topilmasa) frontend
yiqilardi — `?? []` bilan tuzatildi (CalcPage va SearchPage).

To'liq sinov (8703 23 194 1, 1 500 sm³, $20 000, kurs 12 093,35):
yig'im 618 000 (1,5×BRV) · boj 36 280 050 (15%) · QQS 33 377 646 ·
utilizatsiya 49 440 000 (120×BRV) → **jami 119 715 696 so'm**.

```
10. Rasmiylashtiruv yig'imi   1–25 BRV              412 000 so'm  ☑
12. Bojxona nazorati                                103 000 so'm  ☑
20. Bojxona boji              [10]% $0.5 | kg     6 046 675 so'm  ☑
21. Qo'shimcha bojxona boji   [ 0]%                          –
27. Aksiz solig'i             [  ]% ●                        –
29. QQS                       [12]%               3 845 685 so'm  ☑
79. Utilizatsiya yig'imi                                     –
    Bojxona qiymati                              26 000 703 so'm
    Jami to'lovlar                               10 407 360 so'm
```

**Uchta qoida:**

1. **Barcha kodlar HAR DOIM ro'yxatda.** Qo'llanmaydigani xiralashadi va
   `–` bo'ladi. Kodni ro'yxatdan olib tashlash "biz uni hisobga olmadik"
   bilan "u bu tovarga tegishli emas" ni farqlab bo'lmaydigan holga
   keltirardi — deklarant esa GTD da har bir kod bo'yicha javob beradi.

2. **Stavka qatorda tahrirlanadi.** Aksiz stavkasi bazada yo'q, shuning
   uchun uni faqat foydalanuvchi kirita oladi. Bo'sh stavka yonida
   **yashil nuqta** — "noma'lum". `0%` esa noma'lum EMAS: bu "to'lov
   yo'q" degan javob.

3. **Qatorni o'chirish — SERVERDA qayta hisoblanadi**, jadvaldan
   ayirish bilan emas. QQS bazasi bojxona qiymati + boj + aksizdan
   iborat (SK 254-modda), shuning uchun bojni ayirib qo'yish QQS ni
   eski, katta bazada qoldirardi. Sinaldi: bojni o'chirganda QQS
   3 845 685 → 3 120 084 ga tushdi.

   > To'lovni chiqarish uchun **huquqiy asos** kerak (imtiyoz, rejim).
   > Shuning uchun biror qator o'chirilsa, ro'yxat ostida ogohlantirish
   > chiqadi.

**Faktura avtomatik:** birlik narxi × og'irlik. Foydalanuvchi fakturani
o'zi tergan bo'lsa, ustidan yozilmaydi.

**Miqdor maydoni** kodda qo'shimcha o'lchov birligi bo'lmasa o'chiq
turadi (`—`) — u faqat kg da o'lchanadigan tovar ekanini bildiradi.

Backend allaqachon bor edi (`duty/calculate`, `utilfee/calculate`), lekin
frontend uni **umuman chaqirmasdi** — `api.ts` dagi `calculateDuty` o'lik
kod bo'lib turgan edi.

Sinaldi: manba etalon holati interfeys orqali **aynan** mos keldi —
yig'im 10 300 000, QQS 1 821 258 510, jami 1 831 558 510 so'm.

### Kombinatsiyalangan boj stavkasi

Stavka «10%, lekin 1 kg uchun 0,5 dollardan kam emas» ko'rinishida
bo'ladi. **13 142 koddan 1 555 tasida** shunday.

⚠️ **Bu ma'lumot ilgari yo'qolardi.** `tools/extract-hscodes.mjs` manba
blokidagi qat'iy qismni o'qir, lekin **chiqarmasdi** — ya'ni kalkulyator
o'sha kodlarda bojni kam ko'rsatardi. Farq kichik emas: 1 000 000 so'mlik
1 000 kg tovarda **100 000** o'rniga **6 046 675 so'm**.

Manba blokining tuzilishi (manba `good.tp`):

```
[qonun | boshlanish | tugash | foiz | qat'iy | ? | birlik_kodi | ...]
   292   01.06.2025             10     0,5          (bo'sh = kg)
```

Qat'iy qism birligi: kg (796 kod), dona (365), litr (270), juft (91),
m² (29), 1000 dona (4).

> ⚠️ **Birlashtirish qoidasi huquqiy jihatdan tasdiqlanmagan.** Manbada
> faqat ikki raqam bor, ular qanday qo'shilishi yozilmagan. **Kattasi**
> olinadi («…dan kam emas» shakli) — bu EOII va O'zbekiston amaliyotida
> odatiy. Har holda bu hozirgidan yomon emas: ilgari qat'iy qism umuman
> hisobga olinmasdi, ya'ni natija **har doim** kam chiqardi.
>
> Natijada qaysi qism qo'llangani va ikkinchisi qancha bo'lishi **ochiq
> yoziladi** — deklarant o'zi tekshira oladi.

Kalkulyator qat'iy qismli kodda **miqdor** so'raydi. Miqdor yoki USD
kursi berilmasa, faqat foizli qism hisoblanadi va qatorga ⚠️ izoh
qo'yiladi — jim qolmaydi.

⚠️ Ikki ogohlantirish sahifada **doimiy ko'rinadi**:

- **Aksiz faqat foizda.** Aroq, sigaret, benzinda stavka qat'iy summa
  (so'm/litr) — bunday tovarni kalkulyator hisoblay olmaydi.
- **Yig'im shkalasi dollarda.** USD kursisiz yig'im 0 bo'lib qolardi va
  jami summa kam chiqardi. Kurs bo'sh qoldirilsa Markaziy bankdan
  olinadi; xohlasa qo'lda ham kiritish mumkin.

**Bojxona ko'rigi (12-kod)** ham qo'shildi: ish vaqtida 0,25×BRV/soat,
ish vaqtidan tashqari 2×BRV/soat. Backend buni allaqachon hisoblardi,
interfeysda esa maydon yo'q edi — ya'ni qator hech qachon chiqmasdi.

### Qonunlar bo'limi

Qonun korpusi bo'ylab ko'rish:

```
Hujjat (89)  →  Modda (1405)  →  To'liq matn
```

Hujjatlar **parchalar soni bo'yicha** tartiblangan, alifbo bo'yicha emas:
deklarantga avvalo yirik hujjatlar kerak, shuning uchun Bojxona kodeksi
(509 parcha) ro'yxat boshida turadi.

Moddaning to'liq matni panelda ko'rsatiladi, yonida ikkita amal:
**Chatda tushuntir** (chatga savol qo'yadi) va **lex.uz** (rasmiy manba).

```
GET /api/laws/browse            → hujjatlar
GET /api/laws/browse?doc=39534  → moddalar
GET /api/laws/browse?doc=39534&i=0 → to'liq matn
```

> Ro'yxatlarda faqat matn **boshi** yuboriladi (160 belgi). Bitta kodeksda
> 500 dan ortiq modda bor — to'liq matn yuzlab kilobayt bo'lardi va
> panelga baribir sig'masdi (`TestLawsBrowseDrillsDown` buni tekshiradi).

**Ma'lumot manbai.** Uchta darajaning ma'lumoti allaqachon bor edi:
`hscodes.json` da har kodda `section` va `group` maydonlari, tovar
pozitsiyasi sarlavhasi esa `path` ning bosh bo'g'ini. Yetishmagani —
bo'lim va guruh **nomlari**; ular `tools/extract-taxonomy.mjs` orqali
manba bazadan (`good` jadvali, `parent` daraxti) ajratilib
`data/taxonomy.json` ga yoziladi.

> Taksonomiya **ixtiyoriy**: fayl bo'lmasa brauzer raqamlarni ko'rsatadi,
> ishlashdan to'xtamaydi (`TestBrowseWorksWithoutTaxonomy`).

Extraktordagi ikkita nozik joy — ikkalasi ham manbadagi nomuvofiqlik:
kirill **homoglif** (`ХХI` dagi Х — U+0425, lotin X emas) va sarlavha
ichidagi **yangi qator** (regexga `s` bayrog'i kerak). Ularsiz XXI bo'lim
va bir necha guruh tanilmay qolgan edi.

## Ma'lumotlar bazasi (TIF TN)

`backend/data/hscodes.json` — **13 142 ta** to'liq 10 xonali kod, o'zbekcha va
ruscha ierarxik tavsif, stavkalar bilan. Bu fayl **generatsiya qilinadi**:

```bash
node tools/extract-hscodes.mjs --date=2026-07-19
```

Manba: `backend/data/manba/{help,info}.sqlite` (manba baza,
~295 MB, `.gitignore` da — git'ga tushmaydi). Ekstraktor:

- `good` daraxtini rekursiv aylanib to'liq tavsif yig'adi
- kodlangan stavka maydonlarini (`[qonun|dan|gacha|foiz|…]`) ochadi va
  `--date` sanasiga amal qiluvchi versiyani tanlaydi
- kirill o'zbekchani lotinga o'giradi (`tools/translit.mjs`)
- bazadagi ikkita bo'lim raqami xatosini tuzatadi (VII→VIII, XII→XIII)

**Huquqiy asos:** TIF TN 2025 — ПКМ № 181 (14.05.2025), 01.06.2025 dan amalda.
O'tish jadvali — ПКМ № 349 (04.06.2025). Xalqaro asos — Garmonizatsiyalangan
tizim konventsiyasi (Bryussel, 14.06.1983).

> ⚠️ Kodlar va stavkalar muntazam o'zgaradi. `meta.rates_as_of` va
> `meta.extracted_at` maydonlari ma'lumot qaysi holatga tegishli ekanini
> ko'rsatadi; `GET /api/health` ularni qaytaradi.

## Qonun korpusi (RAG)

`backend/data/laws.json` — **1 405 parcha, 89 hujjatdan** (3,8 MB). Generatsiya:

```bash
node tools/extract-laws.mjs [--dry]
```

To'liq korpus ~590 MB, shuning uchun tanlab olinadi:

| Nima | Qanday |
|------|--------|
| Bojxona kodeksi, ПКМ 55 (yig'im), ПКМ 347/358 (utilizatsiya) | **to'liq** |
| Soliq, Ma'muriy, Jinoyat kodekslari + 82 ta hujjat | faqat **bojxonaga oid moddalar** |
| **Bekor qilingan hujjatlar** (`DateFinish` o'tgan) | **chiqariladi** — 388 ta |
| Kalit so'z aldagan hujjatlar (`EXCLUDE`) | **chiqariladi** — 1 ta |

`EXCLUDE` nima uchun kerak: nomida "божхона" bo'lgani uchun korpusga
"Bojxona instituti bakalavriatiga qabul qilish tartibi" tushib qolgan edi —
35 parcha, korpusning 3%i. Mazmuni o'qishga kirish kvotalari haqida, deklarantga
foydasi yo'q, ustiga "bojxona ... foiz" kabi so'rovlarni chalg'itardi.

> ⚠️ TIF TN ni tasdiqlagan **ПКМ 181** va o'tish jadvali **ПКМ 349** matnlari
> manba bazada yo'q (`IsLoad=0`), shuning uchun korpusga kirmagan.

Bekor qilinganlarni chiqarish muhim: yig'im stavkalari bo'yicha **bir xil nomli
ikkita qaror** bor — ПКМ 700 (2020, 2025-05-04 da bekor qilingan) va ПКМ 55
(2025). Ikkalasi ham korpusda qolsa, RAG eskirgan stavkani qaytarishi mumkin.

Parchalash — moddalar bo'yicha (`N-modda`), 4000 belgidan uzunlari bo'linadi.

⚠️ **Ustki indeksli moddalar** (`289¹`) alohida e'tibor talab qiladi. Manba
HTML da ular `289<sup>1</sup>-модда` ko'rinishida; teglar shunchaki olib
tashlansa raqamlar birikib ketadi va korpusda **mavjud bo'lmagan
"2891-modda"** paydo bo'ladi. AI o'sha sarlavhani iqtibos qilsa, yo'q moddaga
havola bergan bo'lardi — aksiz stavkalari aynan 289¹–289³ da bo'lgani uchun
xato eng muhim joyga tushgan edi. Endi `<sup>` Unicode ustki indeksga
o'giriladi; `TestSuperscriptArticles` buni qo'riqlaydi.

⚠️ **Mundarija qatorlari** korpusga tushmasligi kerak, lekin ularni
"200 belgidan qisqa" deb filtrlash haqiqiy qisqa moddalarni ham yo'q qilardi
(Bojxona kodeksidan 7, 103, 110, 257-modda). Aniqroq belgi: mundarijada modda
bir qator, faqat sarlavhadan iborat. Shundan keyin Bojxona kodeksi 443 →
507 parchaga chiqdi, moddalar 1–412 (6 ta bo'shliq — bekor qilingan moddalar).
Matn **rasmiy o'zbekcha** versiyadan olinadi (ruschasi ayrim hujjatlarda mashina
tarjimasi bo'lib, yuridik kuchga ega emas) va lotinga transliteratsiya qilinadi.

Har bir parchaga imkon qadar **lex.uz havolasi** biriktiriladi, shunda AI javobda
rasmiy manbani ko'rsatadi va foydalanuvchi o'zi tekshira oladi. Moslik
`tools/lex-links.mjs` da **qo'lda** yuritiladi — manba bazasida lex.uz
identifikatorlari yo'q (Bojxona kodeksi: bazada `39534`, lex.uz da `2876352`).
Hozirgi qamrov — **77%** (1 079/1 405 parcha, 16 ta hujjat).

Nega qo'lda: lex.uz da ochiq API ham, `sitemap.xml` ham yo'q, qidiruvi esa
faqat JavaScriptda ishlaydi (`?query=` parametri e'tiborga olinmaydi),
`robots.txt` 20 soniyalik kechikish so'raydi. Shuning uchun har bir hujjat
alohida qidirib topiladi. Ekstraktor havolasiz eng ko'p uchraydiganlarini
ko'rsatib turadi — jadvalni shular bo'yicha kengaytirish mumkin.

⚠️ Moslashtirishda nom bilan birga **sana** ham tekshiriladi. Sababi: bir xil
nomli, turli yildagi hujjatlar bor va eskisi bekor qilingan bo'lishi mumkin —
masalan "О порядке применения акцизных марок…" 1999 (kuchini yo'qotgan) va
2024 (amaldagi). Faqat nom bo'yicha mos qo'ysak, amaldagi hujjatga bekor
qilinganining havolasi biriktirilib qolardi. Hozircha 2024 yilgisiga havola
topilmadi, shuning uchun u **havolasiz** qoldirilgan — noto'g'ri havoladan
ko'ra havolasiz yaxshiroq.

Chat har bir savolga bazadan **top-8 TIF TN kod** va **top-3 qonun parchasi**
topib qo'shadi — butun baza promptga tashlanmaydi, shu sababli prompt keshi
buzilmaydi. Qidiruv o'zbek tilining qo'shimchalarini hisobga oladi
(`vaqtinchalik` → `vaqtincha`) va kam uchraydigan so'zga ko'proq vazn beradi (IDF).

## Hujjat talablari

`backend/data/docs.json` — **15 112 qoida, 106 tur** (4,0 MB). Generatsiya:

```bash
node tools/extract-docs.mjs [--date=2026-07-19]
```

Manba: `help.sqlite` ning `OnlineTnvedInfo` (talab turi va tavsifi) va
`OnlineTnvedInfoItem` (kod oraliqlari, qonun, amal muddati) jadvallari.
Talab **kod oralig'i** bo'yicha beriladi (`3001000000`–`3001999999`), shuning
uchun qidiruv — oraliqqa tegishlilikni tekshirish.

| Bo'lim | Nima | Soni |
|--------|------|------|
| `litsenziya` | Litsenziya talab qilinadigan tovarlar | 219 |
| `sertifikat` | Muvofiqlik, sanitariya, ekologiya sertifikatlari | 4 765 |
| `imtiyoz` | To'lovdan ozod qilish (qaysi to'lovdan — `free` maydonida) | 2 792 |
| `boshqa` | Ro'yxatdan o'tish, ruxsatnoma va h.k. | 7 292 |
| `tavsif` | **Hujjat emas** — GTD 31-grafada ko'rsatilishi shart bo'lgan ma'lumot | 44 |

`tavsif` alohida ajratilgan: manbadagi `Specs_*` yozuvlari (dori nomi, dozasi,
ishlab chiqaruvchi) hujjat emas, tovar tavsifiga oid. Ularni "boshqa talab"
deb ko'rsatsak, foydalanuvchi mavjud bo'lmagan hujjatni izlab yurardi.

⚠️ Bir xil talab turli **sana oraliqlari** bilan takrorlanadi (sertifikat:
`2021-11-17..2024-11-24` va `2024-11-25..∞`). Ekstraktor `--date` ga amal
qiluvchisini oladi, aks holda eskirgan talab ko'rsatilardi — 6 792 ta
bekor qilingan yozuv chiqarildi.

⚠️ Bu ro'yxatda **kodga aniq bog'lanmagan** hujjatlar bo'lmaydi. Chat buni
ochiq aytadi: bo'lim ko'rsatilmagan bo'lsa, bu "talab qilinmaydi" degani emas.

## Ishlab chiqarish — deklarantpro.uz

Joylashtirilgan: **https://deklarantpro.uz** (DigitalOcean, Ubuntu 24.04,
1 vCPU / 512 MB / 10 GB, NYC1).

```
Caddy :443  ──┬── /api/*, /admin*  →  127.0.0.1:8080  (deklarant.service)
              └── qolgani          →  /var/www/deklarant  (React)
```

| Fayl | Vazifasi |
|---|---|
| `deploy/setup.sh` | Serverni BIR MARTA sozlaydi (serverda) |
| `deploy/release.sh` | Yangi versiyani chiqaradi (mahalliy) |
| `deploy/Caddyfile` | HTTPS + proksi + SPA |
| `deploy/deklarant.service` | systemd, cheklangan huquqlar |

Yangi versiya chiqarish — bitta buyruq:

```bash
bash deploy/release.sh
```

Backend **kross-kompilyatsiya** qilinadi (`CGO_ENABLED=0 GOOS=linux`), ya'ni
serverga Go o'rnatish shart emas va natija 7 MB statik fayl.

⚠️ **512 MB xotira — tor joy.** O'lchangan: Ubuntu ~110 MB, backend
80 MB, Caddy ~30 MB → 243/458 MB. Ikkita himoya qo'yilgan:
`GOMEMLIMIT=250MiB` (GC qattiqroq ishlaydi, OOM o'rniga sekinlashish)
va **1 GB swap** (DigitalOcean da sukut bo'yicha yo'q).

⚠️ **`users.json` hech qachon yuborilmaydi** — u serverda yasaladi va
parol xeshlari bor. Mahalliy nusxa bilan almashtirilsa hamma
foydalanuvchi yo'qolardi.

⚠️ **294 MB manba sqlite serverga chiqmaydi** — u faqat JSON
yasash uchun kerak.

Jurnal:

```bash
ssh -i ~/.ssh/deklarant_deploy root@deklarantpro.uz journalctl -u deklarant -f
```

## Ishga tushirish

### 1. Backend (Go)

```bash
cd backend
# AI (chat) uchun kalitni o'rnating — Windows PowerShell:
#   $env:ANTHROPIC_API_KEY="sk-ant-..."
# Linux/macOS:
#   export ANTHROPIC_API_KEY="sk-ant-..."
go run .
```

Server `http://localhost:8080` da ishga tushadi. Kalit o'rnatilmasa, HS qidiruv
va kalkulyator ishlaydi, faqat chat o'chiq bo'ladi.

> ⚠️ `backend/.env.example` — **qo'llanma, konfiguratsiya emas**. Loyihada
> tashqi bog'liqlik yo'q va hech qanday `.env` yuklovchi ishlatilmaydi
> (`os.Getenv` xolos), ya'ni faylni `.env` deb nusxalash HECH NARSA
> qilmaydi. Qiymatlarni haqiqiy muhitga bering (yuqoridagi buyruqlar
> yoki systemd `EnvironmentFile`). To'liq ro'yxat o'sha faylda va
> `/admin` → «Sozlamalar» da; `TestEnvExampleCoversRegistry` ikkovi
> ajralib ketmasligini qo'riqlaydi.

| Muhit o'zgaruvchisi | Majburiy | Tavsif |
|---|---|---|
| `ANTHROPIC_API_KEY` | chat uchun | Bo'lmasa chat o'chiq, qolgani ishlaydi |
| `ANTHROPIC_MODEL` | yo'q | Asosiy daraja, sukut `claude-opus-4-8` |
| `ANTHROPIC_MID_MODEL` | yo'q | O'rta daraja, sukut `claude-sonnet-5` |
| `ANTHROPIC_API_URL` | yo'q | Korporativ shlyuz yoki testdagi soxta server |
| `GLM_API_KEY` | yo'q | Arzon provayder (Z.ai). Bo'lmasa hammasi Claude da — "Xarajat nazorati" ga qarang |
| `PORT` | yo'q | Sukut bo'yicha `8080` |
| `HSCODE_DATA`, `LAWS_DATA`, `DOCS_DATA`, `COUNTRIES_DATA` | yo'q | Ma'lumot fayllari yo'li |
| `TAXONOMY_DATA` | yo'q | Bo'lim/guruh sarlavhalari (sidebar). Bo'lmasa brauzer raqamlarni ko'rsatadi |

## Kelib chiqish davlati va boj

`backend/data/countries.json` — **254 davlat**. Generatsiya:

```bash
node tools/extract-countries.mjs
```

**Boj faqat tovar kodiga emas, kelib chiqish davlatiga ham bog'liq** —
Bojxona kodeksi 300-modda:

| Rejim | Koeffitsient | Davlatlar | Qonun matni |
|---|---|---|---|
| Erkin savdo | **×0** | 10 ta (MDH) | "bojxona bojlari qoʻllanilmaydi" |
| Eng qulaylik | **×1** | 49 ta | "boj tarifi bilan belgilangan stavkalar" |
| Rejim yo'q yoki kelib chiqishi **aniqlanmagan** | **×2** | 195 ta | "stavkalari ikki baravar oshiriladi" |

Erkin savdo ro'yxati: Rossiya, Qozog'iston, Belarus, Qirg'iziston,
Tojikiston, Ozarbayjon, Armaniston, Gruziya, Moldova, Ukraina.

⚠️ **Erkin savdo imtiyozi avtomatik emas.** BK 300-moddasiga ko'ra tovar
shartnoma ishtirokchisi davlat rezidenti tomonidan eksport qilingan va
bevosita olib kirilgan bo'lishi, hamda **ST-1 sertifikati** taqdim
etilishi shart. Chat buni har javobda eslatadi.

API da davlat nomini yozish yetarli — koeffitsient server tomonida
aniqlanadi:

```bash
curl -X POST http://localhost:8080/api/duty/calculate \
  -d '{"customs_value":100000000,"usd_rate":12000,"import_duty":10,
       "vat":12,"origin_country":"Rossiya"}'
```

Nom, kod (`643`), ISO (`RU`), ruscha nom (`РОССИЯ`) va sinonim (`XXR`,
`Amerika Qo'shma Shtatlari`) — hammasi ishlaydi. Bazadagi nomlar ruscha
bo'lgani uchun asosiy savdo sheriklariga o'zbekcha nom qo'lda yozilgan:
transliteratsiya "КИТАЙ" → "KITAY" berardi, foydalanuvchi esa "Xitoy" deb
yozadi.

⚠️ Kelib chiqish **ko'rsatilmasa** ×1 olinadi (eng ko'p uchraydigan holat),
lekin natijada bu ochiq yoziladi. Qonun bo'yicha noma'lum kelib chiqishga
×2 qo'llanadi, ammo API chaqiruvchisi shunchaki maydonni to'ldirmagan
bo'lishi mumkin — jim ravishda bojni ikkilantirish noto'g'ri javob bo'lardi.

## Imtiyozlar (bojdan/QQS dan ozod tovarlar)

Bazada **29 ta imtiyoz dasturi** bor — `docs.json` ning `imtiyoz` bo'limidan
yig'iladi. Eng yiriklari:

| Ozod qiladi | Oraliq | Asos | Nima |
|---|---|---|---|
| boj + QQS | 1 236 | ПКМ 352 (04.06.2021) | O'xshashi ishlab chiqarilmaydigan texnologik uskunalar |
| boj | 498 | УП 55 / ПКМ 519 | Ayrim korxonalar uchun texnologik uskunalar |
| boj + aksiz | 222 | РП Р-5350 | «O'zcharmsanoat» korxonalari uchun xomashyo |
| boj | 154 | ПП 5262 (20.10.2021) | Nol stavka, 01.01.2027 ga qadar |
| QQS | 129 | МЮ 2502 | Veterinariya dori vositalari xomashyosi |

```bash
curl http://localhost:8080/api/exemptions              # barcha dasturlar
curl "http://localhost:8080/api/exemptions?code=8401100000"  # kod bo'yicha
```

Chatda ro'yxat **faqat savol imtiyoz haqida bo'lganda** qo'shiladi
("imtiyoz", "ozod", "льгот", "nol stavka"…) — u uzun va har savolga
qo'shilsa kontekstni behuda to'ldirardi. Aniq kod ma'lum bo'lsa, o'sha
kodga tegishlisi `<HUJJAT_TALABLARI>` blokida keladi.

⚠️ **Imtiyozlar SHARTLI va bu har joyda takrorlanadi.** Shart odatda:
"yuridik shaxs tomonidan", "respublikada o'xshashi ishlab chiqarilmaydi"
(Sanoat vazirligi ro'yxati), "ishlab chiqarish ehtiyoji uchun", yoki
muddat ("01.01.2027 ga qadar"). Chat imtiyozni **va'da qilmaydi** —
shartini aytadi va foydalanuvchining holatini so'raydi.
`TestProgramsBlockWarnsConditional` buni qo'riqlaydi.

## Ikki rejim: deklarant / tadbirkor

Chatda ikkita javob uslubi bor. So'rovga `mode` maydoni qo'shiladi
(`"deklarant"` — sukut, yoki `"tadbirkor"`).

| | Deklarant | Tadbirkor |
|---|---|---|
| Foydalanuvchi | TIF TN va GTD ni biladi | Atamalarni bilmaydi |
| Javob | Qisqa, jadval, GTD kodlari (10, 20, 29, 79) | Jami summa oldinda, tushuntirish bilan |
| Atamalar | Tushuntirilmaydi | Darrov ochib beriladi ("ST-1 — kelib chiqish sertifikati") |
| Ma'lumot yetishmasa | Variantlar beriladi | Oddiy tilda so'raladi |
| Oxirida | — | "Keyingi qadamlar" + murojaat kanali |

**Murojaat kanali.** Tadbirkor rejimida javob oxirida murojaat havolasi
beriladi (`CONTACT_TELEGRAM`, sukut bo'yicha `t.me/declarant_pro`).
Deklarant rejimida u **yo'q** — professional foydalanuvchiga kerak emas.

⚠️ Havola **oddiy tavsiya** sifatida beriladi, rasmiy davlat manbasi
sifatida emas, va `customs.uz` yoki bojxona brokeridan tasdiqlash
maslahatini **almashtirmaydi** — ikkalasi ham qoladi.
`TestContactDoesNotReplaceSafetyAdvice` buni qo'riqlaydi.

⚠️ **Eng muhim qoida: rejim faqat USLUBNI o'zgartiradi.** Faktlar,
stavkalar va ogohlantirishlar ikkalasida **bir xil**.

Bu shunchaki kelishuv emas, tuzilma bilan ta'minlangan: prompt ikki
qismdan iborat — `promptIntro[mode]` (uslub) va `sharedRules` (faktlar va
ogohlantirishlar). Ikkinchisi **bitta konstanta**, ikkala rejim uchun
umumiy. `TestSharedRulesIdentical` uning bayt-bayt bir xil ekanini
tekshiradi, `TestBothModesKeepWarnings` esa har bir ogohlantirish
(aksiz 289¹, ST-1, imtiyoz shartliligi, kursni taxmin qilmaslik,
utilizatsiya yig'imi) ikkala rejimda borligini qo'riqlaydi.

**Nega bunday qattiq:** bu loyihada tuzatilgan xatolarning aksariyati bir
turdan edi — xavfli narsani jim tushirib qoldirish (`excise: 0`, imtiyoz
e'tiborsiz, kelib chiqish hisobga olinmagan, kurs taxmin qilingan).
"Soddalashtirilgan rejim" aynan o'sha xatoning qaytish yo'li: soddalashtirganda
birinchi navbatda ogohlantirishlar qisqaradi.

## Utilizatsiya yig'imi (79)

Huquqiy asos: **ПКМ № 347** (02.06.2020), 1-ilova. Joriy tahrir —
ПКМ 52 (31.01.2025), 2025 yil 1 maydan. Jadval `internal/duty/utilfee.go`
da, ПКМ 55 shkalasi bilan bir uslubda.

Stavka **BRV karrasida** va ikki omilga bog'liq:

1. **O'lchov** — toifaga qarab: dvigatel hajmi (sm³), quvvat (kVt yoki
   ot kuchi) yoki to'la vazn (tonna)
2. **Yosh** — ishlab chiqarilganiga 3 yildan ortiqmi

| Toifa | Kod | Misol |
|---|---|---|
| Yengil avtomobillar (M1) | 8703 | 2 000 sm³: yangi 120 BRV, eski 210 BRV |
| Avtobuslar (M2, M3) | 8702 | 120…1 080 BRV |
| Yuk avtomobillari | 8704 | vazn bo'yicha, 100…1 410 BRV |
| Traktorlar | 8701 | quvvat bo'yicha, **yangisiga stavka yo'q** |
| Maxsus, tirkama, o'ziyurar mashinalar | 8705, 8716, 8426–8436 | |
| Shinalar (2-ilova) | 4011, 4012 | 1 kg uchun BRV ning 0,3% i |

```bash
curl -X POST http://localhost:8080/api/utilfee/calculate \
  -d '{"code":"8703 23 194 0","measure":2000,"age_years":5}'
# -> 210 × BRV = 86 520 000 so'm
```

⚠️ **Bu yig'im ko'pincha boj va QQS dan katta.** 2 litrli avtomobil uchun
86 mln so'm — shuning uchun chat kod utilizatsiya ro'yxatida ekanini
stavka yonida ogohlantiradi va kerakli o'lchovni so'raydi.

⚠️ **Ro'yxatda yo'q kod uchun 0 emas, XATO qaytariladi.** Aksiz bilan bir
xil mantiq: "yig'im yo'q" deb javob berish, aslida bor bo'lsa, jim va
xavfli xato bo'lardi.

⚠️ Ko'p toifada **yangi texnikaga stavka belgilanmagan** (qonun jadvalida
"—"). Bu holda yig'im olinmaydi, lekin natijada nega nol ekani yoziladi.

⚠️ Ishlatilgan shinalar (4012) qatorining foizi manba hujjatda **bo'sh
qolgan** — noma'lum deb qaytariladi, 0 deb emas.

**Imtiyozlar** (ПКМ 347, 2-band «б») — hozircha kalkulyatorda yo'q, chat
ularni qonun matnidan aytadi: diplomatik vakolatxonalar; 30 yildan oshgan
L va M1, 50 yildan oshgan M2/M3/N toifali retro transport; xayriya va
grant yordami; "vaqtinchalik olib kirish" rejimi; davlat tashqi qarzi
hisobidan moliyalanadigan loyihalar.

## Valyuta kursi

Manba: **cbu.uz** ochiq API (`internal/rates`). Kurs bazaga yozilmaydi —
har kuni o'zgargani uchun to'g'ridan-to'g'ri olinadi va keshlanadi.

**Nega kerak:** ilgari foydalanuvchi kursni qo'lda kiritardi, kiritmasa AI
uni **taxmin qilardi** — sinovda *"kurs ~12 600 deb hisobladim"* degan edi.
Bu jim va xavfli xato: noto'g'ri kurs butun hisobni buzadi va buni
foydalanuvchi sezmaydi.

Chatga joriy kurslar (USD, EUR, RUB, CNY, KZT, TRY) kontekst bloki sifatida
beriladi. API da valyuta kodini yozish yetarli:

```bash
curl -X POST http://localhost:8080/api/duty/calculate \
  -d '{"invoice":10000,"currency":"USD","import_duty":5,"vat":12,
       "origin_country":"Xitoy"}'
```

⚠️ **Kurs sanaga bog'liq.** Bojxona qiymati deklaratsiya **ro'yxatga olingan
kundagi** kurs bo'yicha hisoblanadi. `date` berilsa, o'sha kundagi kurs
olinadi (`/json/all/2026-07-10/`), aks holda oxirgi kurs.

⚠️ **`Nominal` maydoni.** cbu.uz ba'zi valyutalarni 10 birlik uchun
kotirovka qiladi (IDR, IRR, VND). Bo'lmasak, ular **o'n barobar xato**
chiqadi — shuning uchun `Rate / Nominal` hisoblanadi.

⚠️ **Xizmat ishlamasa — xato qaytadi, taxminiy kurs emas.** API 502 va
"kursni qo'lda kiriting" deb aytadi; chat esa kurs blokini qo'shmaydi va
ko'rsatmaga ko'ra foydalanuvchidan so'raydi.

Kesh: tarixiy kurs muddatsiz (u o'zgarmaydi), bugungisi kun oxirigacha.

| O'zgaruvchi | Sukut | Tavsif |
|---|---|---|
| `CBU_API_URL` | cbu.uz | Boshqa manba yoki testdagi soxta server |
| `CONTACT_TELEGRAM` | t.me/declarant_pro | Tadbirkor rejimida beriladigan murojaat kanali |

## Admin paneli

`http://<host>/admin` — parol bilan himoyalangan panel. Bitta o'z-o'zicha
HTML sahifa, Go ichiga `embed` bilan qo'yilgan — alohida build yo'q.
Uch tab:

```bash
ADMIN_PASSWORD="kuchli-parol" go run .   # panelni yoqadi
```

**1. Foydalanish** — model bo'yicha token (kirish/chiqish/kesh), so'nggi
so'rovlar, API so'rovlar soni, ishlash vaqti. Claude va GLM ulushini shu
yerdan ko'rasiz.

**2. Ma'lumot** — bazalar holati: kodlar, qonun korpusi (lex.uz havola
qamrovi bilan), hujjat talablari, davlatlar; har birining `rates_as_of` sanasi.

**3. Sozlamalar** — **21 ta muhit o'zgaruvchisi** bir joyda
(`internal/api/settings.go`), guruhlangan: AI provayderlari, cheklovlar,
server, ma'lumot manbalari, tashqi xizmatlar. Har biri uchun:

| Ustun | Ma'nosi |
|---|---|
| Amaldagi qiymat | Server AYNAN nima bilan ishlayapti |
| Manba | `muhit` (o'rnatilgan) yoki `sukut` (kodda yozilgan) |
| O'zgarishi | `darrov` yoki `qayta ishga tushirish` kerak |

Tepada ogohlantirishlar: kalit sozlanmagani, `TRUST_PROXY` yoqilgani,
parol qisqaligi, ikkala limit ham o'chirilgani, HTTPS bo'lmagan endpoint.

> ⚠️ **Maxfiy qiymatlar hech qachon qaytarilmaydi** — na to'liq, na qisman
> (oxirgi 4 belgi ham emas). Faqat "sozlangan / sozlanmagan" holati. Panel
> parol bilan himoyalangan bo'lsa ham, kalitni JSON ga chiqarish uni
> tarqatish uchun yana bir kanal ochib berardi: brauzer tarixi, proksi
> jurnali, ekran surati. `TestAdminSettingsNeverLeaksSecrets` shuni
> qo'riqlaydi.

> Sozlamalar **faqat o'qish uchun**. Panelga yozish qo'shilmadi: qiymatlarning
> ko'pi ishga tushishda bir marta o'qiladi (klient o'shanda yasaladi), ya'ni
> "saqlash" tugmasi ba'zi sozlamada ishlab, ba'zisida jimgina ishlamasdi.
> Bundan tashqari model tanlashni veb orqali o'zgartirish audit izisiz
> qolardi.

⚠️ **FAIL CLOSED — eng muhim xususiyat.** `ADMIN_PASSWORD` o'rnatilmagan
bo'lsa, `/admin` yo'llari **umuman mavjud emas** (404, 401 ham emas). Bu
tasodifan himoyasiz panel ochib qo'yishning oldini oladi.

Xavfsizlik qatlamlari (`internal/api/admin.go`):

| Nima | Qanday |
|---|---|
| Parol solishtirish | `crypto/subtle` — doimiy vaqt, timing hujumiga qarshi |
| Sessiya tokeni | `crypto/rand`, 32 bayt — taxmin qilib bo'lmaydi |
| Cookie | `HttpOnly` + `SameSite=Strict` + `Path=/admin`; HTTPS da `Secure` |
| Login urinishi | IP bo'yicha daqiqasiga 5 ta — brute-force ga qarshi |
| Sessiya muddati | 8 soat, keyin bekor |
| Clickjacking | `X-Frame-Options: DENY` + CSP `frame-ancestors 'none'` |

⚠️ Bu **bir kishilik** admin uchun. Sessiyalar xotirada — bir nechta
server nusxasi ishlaganda umumiy do'kon (Redis) kerak. Parol bitta,
foydalanuvchi hisoblari yo'q.

| O'zgaruvchi | Tavsif |
|---|---|
| `ADMIN_PASSWORD` | Panel paroli (bo'sh = panel o'chiq) |

## Foydalanuvchilar va rollar

`internal/users`. Ro'yxatdan o'tish, kirish, chiqish va to'rt rol.

```
POST /api/auth/register  {login, password, name?, role?}  → token + user
POST /api/auth/login     {login, password}                → token + user
POST /api/auth/logout                                     → tokenlarni bekor qiladi
GET  /api/auth/me                                         → joriy foydalanuvchi
GET  /api/auth/roles                                      → rollar, kvota va uslub
```

**Chat KIRISH TALAB QILADI.** Anonim tokenni istalgan skript bir chaqiruv
bilan oladi — u xarajatni hech kimga bog'lamaydi. Qidiruv, kalkulyator,
risk va qonunlar LLM ga bormaydi, ya'ni **kirishsiz ham ochiq**.

Interfeysda kirish **modal oyna** — alohida sahifa emas. Chat joyida
qoladi va kiritish maydonida ishora turadi: *"Savol berish uchun kiring —
shu yerga bosing"*. Maydonga (yoki yuborish tugmasiga, rasm biriktirishga,
tayyor savolga) bosilsa oyna ochiladi.

> Ilova nima qila olishini ko'rsatmasdan kirishga majburlash — savolga
> javob bermasdan pul so'ragandek. Shuning uchun chat ko'rinib turadi,
> faqat yozish yopiladi.

Maydon `disabled` emas, **`readOnly`**: o'chirilgan maydon bosishni
umuman qabul qilmaydi, ya'ni foydalanuvchi bosib ko'radi-yu, hech narsa
bo'lmaydi.

### Rollar

Rol UCH narsani belgilaydi. Ro'yxat ataylab qisqa: ishlamaydigan
"ruxsatlar" xavfsizlik tuyg'usini beradi-yu, hech narsani himoya qilmaydi.

| Rol | Chat uslubi | Kuniga | Izoh |
|---|---|---|---|
| `DECLARANT` | deklarant | 200 | Sukut rol |
| `BUSINESS` | tadbirkor | 30 | Sodda javob |
| `INSPECTOR` | deklarant | 300 | Bugun imkoniyatlari deklarantniki bilan bir xil |
| `ADMIN` | deklarant | 1000 | `/admin` paneliga kirish |

⚠️ **ADMIN ro'yxatdan o'tishda berilmaydi** — aks holda kim xohlasa admin
bo'lardi. Istisno faqat **birinchi** foydalanuvchi: yangi o'rnatmada
birorta admin bo'lmasa, rol tayinlash imkoni ham bo'lmasdi.

### Saqlash va parollar

Ombor — `data/users.json` (`USERS_DATA` bilan sozlanadi). Baza emas: barcha
ma'lumot allaqachon JSON fayllarda va loyihada birorta tashqi bog'liqlik
yo'q. `Store` interfeys ortida, ya'ni bazaga o'tilganda handler'lar
o'zgarmaydi.

- Parol — **PBKDF2-SHA256, 210 000 iteratsiya**, har foydalanuvchiga
  alohida tuz (`crypto/pbkdf2`, standart kutubxona)
- Solishtirish doimiy vaqtda; yo'q foydalanuvchi uchun ham xesh
  hisoblanadi — javob vaqti qaysi loginlar borligini oshkor qilmasin
- "Login topilmadi" va "parol noto'g'ri" — **bir xil javob**
- Telefon raqami normallashtiriladi: `+998 90 123-45-67` va
  `+998901234567` bir xil akkaunt

⚠️ `data/users.json` **git ga tushmaydi** (`.gitignore`) — unda parol
xeshlari va telefon raqamlari bor.

### Token

Token **serverda saqlanmaydi**, imzolangan: `u1.<id>.<versiya>.<muddat>.<imzo>`.
Shu tufayli server qayta ishga tushganda seanslar yo'qolmaydi.

"Chiqish" shundan qiyinlashadi — imzolangan tokenni o'chirib bo'lmaydi.
Yechim: foydalanuvchidagi **versiya raqami** token ichida ham bor va
chiqishda oshiriladi, ya'ni barcha qurilmadagi eski tokenlar darrov
kuchini yo'qotadi. Sinaldi: chiqishdan keyin o'sha token `401`.

### Kvota endi foydalanuvchiga bog'langan

Ilgari limit IP bo'yicha edi: bitta ofisdagi yigirma kishi bir-birining
kvotasini yeb qo'yardi, mobil tarmoqdagi odam esa IP almashtirib
chegarani aylanib o'tardi. Endi kirgan foydalanuvchining kvotasi
**rolidan** olinadi.

Parol qabul qiladigan yo'llar (`login`, `register`, `session`) IP bo'yicha
cheklangan. `me` va `logout` **ataylab cheklanmagan**: ular amaldagi
tokenni talab qiladi, ya'ni taxmin qilinadigan narsa yo'q — cheklansa,
sahifani bir necha marta yangilagan odam `429` olardi.

## Mijozni tanish (API yopiq)

`internal/api/auth.go`. Chat endpointlari (`/api/chat`, `/api/chat/stream`)
**tanilgan mijozni talab qiladi**; belgisiz so'rovga `401` qaytadi va LLM
ga umuman bormaydi.

Belgi `X-API-Key` sarlavhasida keladi va ikki turdagi bo'lishi mumkin:

| Tur | Manba | Kim uchun |
|---|---|---|
| **API kaliti** | `API_KEYS` muhit o'zgaruvchisi (`nom:sir` ro'yxati) | Mobil ilova, hamkorlar, server-server |
| **Anonim token** | `POST /api/session` — HMAC bilan imzolangan, muddati bor | Brauzer |

```bash
TOK=$(curl -s -X POST localhost:8080/api/session | jq -r .token)
curl -X POST localhost:8080/api/chat -H "X-API-Key: $TOK" -d '{"messages":[...]}'
```

⚠️ **Bu autentifikatsiya EMAS.** Anonim tokenni istalgan odam bir chaqiruv
bilan olishi mumkin. Bu qat'iy to'siq emas, **tezlik cheklovi**:

- boshqa saytlar API ni to'g'ridan-to'g'ri o'z sahifasiga ulay olmaydi
- har mijozga barqaror belgi beriladi (kelajakda kvota shunga bog'lanadi)
- akkauntlar kelganda o'sha joyga foydalanuvchi seansi qo'yiladi —
  endpointlar va mijozlar o'zgarmaydi

Brauzerdagi belgi ham **sir emas** — uni sahifadan olib qo'yish mumkin.
Haqiqiy himoya akkauntlar bilan keladi.

**Bepul qism ochiq qoladi:** qidiruv, boj kalkulyatori, TIF TN brauzeri va
qonunlar LLM ga umuman bormaydi, ya'ni xarajat yasamaydi. Qidiruvdagi AI
izohi (`use_ai`) esa tanilmagan mijozda **rad etilmaydi, o'chiriladi** —
kodlar baribir topiladi.

| O'zgaruvchi | Sukut | Tavsif |
|---|---|---|
| `API_KEYS` | — | `nom:sir` ro'yxati, vergul bilan |
| `CLIENT_TOKEN_SECRET` | tasodifiy | Token imzosi. Bir nechta nusxada **bir xil** bo'lishi shart |
| `CLIENT_TOKEN_TTL` | `24h` | Anonim token muddati |
| `ALLOWED_ORIGINS` | — | CORS ro'yxati. Bo'sh = hammasi (`*`) |

⚠️ `CLIENT_TOKEN_SECRET` berilmasa har ishga tushishda tasodifiy yasaladi
— tokenlar qayta ishga tushganda kuchini yo'qotadi. Mijozlar buni sezmaydi
(401 da bir marta avtomatik yangilaydi), lekin bir nechta server nusxasi
bo'lsa tokenlar umuman ishlamaydi.

## Xarajat nazorati

Har bir chat so'rovi pul turadi, shuning uchun besh qatlam qo'yilgan.
Sozlash — muhit o'zgaruvchilari bilan.

**1. Prompt keshi.** Tizim ko'rsatmasi ~7 900 belgi (o'lchangan:
`TestSystemPromptStaysCacheable`) va har so'rovda bir xil. Unga
`cache_control: ephemeral` biriktiriladi va keshdan o'qish narxi asl
narxning ~10%i. O'lchangan: **5 035 token** keshga tushadi.

Retrieval bloklari ataylab **oxirgi foydalanuvchi xabariga** qo'shiladi —
ular o'zgaruvchan, shuning uchun keshga tushmaydi va keshni buzmaydi ham.

⚠️ Sukut kesh muddati — **5 daqiqa**. Kesh barcha foydalanuvchilar uchun
umumiy, lekin 5 daqiqada bironta so'rov kelmasa u yo'qoladi va keyingi
so'rov keshni qayta yozadi — bu oddiy kirishdan 1,25 barobar QIMMAT.
Ya'ni siyrak trafikda kesh foyda emas, zarar keltiradi.
`ANTHROPIC_CACHE_TTL=1h` buni tuzatadi (yozish 2 barobar, o'qish o'sha-o'sha
0,1 barobar) — soatiga bitta so'rov bo'lsa ham keshni tirik saqlaydi.

Bu faraz emas, **jonli o'lchov** (server jurnalidan):

| So'rov | Oralik | Keshga yozildi | Keshdan o'qildi |
|---|---|---|---|
| 1 | — | 5 035 | 0 |
| 2 | 9 daq | 5 035 | 0 |
| 3 | 1 daq | 0 | **5 035** |

Ya'ni mexanizm ishlaydi, muddat esa yetmaydi: 9 daqiqa oraliqda kesh
har safar QAYTA yozildi va bir marta ham o'qilmadi — sof zarar.
Bir daqiqa oraliqda esa to'liq o'qildi. Trafik siyraklashguncha
`ANTHROPIC_CACHE_TTL=1h` yoqilgani ma'qul.

> Ko'rsatma qisqartirilsa kesh **xato bermasdan** o'chadi — so'rovlar
> shunchaki qimmatlashadi. `TestSystemPromptStaysCacheable` shu chegarani
> qo'riqlaydi.

**Xabarlar keshlanmaydi va bu tuzatib bo'lmaydigan holat emas, lekin
hozircha shunday:** retrieval faqat oxirgi xabarga qo'shiladi, keyingi
navbatda esa o'sha xabar mijozdan xom holda qaytib keladi — prefiks
o'zgaradi va kesh yaroqsiz bo'ladi. To'liq yechim — suhbatni **serverda**
saqlash (akkauntlar bilan birga keladi).

**2. Kontekst byudjeti** (`maxContext = 24 000` belgi). O'lchov shuni
ko'rsatdi: "salom" so'zi **57 KB** kontekst yasagan edi — sabab, qonun
parchalari 94 KB gacha bo'lgan (`splitLong` uzun abzatsni bo'lmasdi).
Parchalash tuzatilgach 13,5 KB ga tushdi, byudjet esa yuqori chegarani
kafolatlaydi. Sig'magan parchalar borligi kontekstda **ochiq yoziladi**.

> Ball chegarasi qo'yilmadi: o'lchovda "salom" 13,2 ball, haqiqiy savol
> 19,2 ball olgan — farq juda kichik va chegara foydali natijalarni ham
> kesib yuborardi. Byudjet sifat haqida qaror qabul qilmaydi.

**3. Model darajasi — UCHTA.** Tanlov savol QIYINLIGIGA emas, **xato
narxiga** qarab qilinadi: qiyinlikni javobdan oldin o'lchab bo'lmaydi,
xavfni esa bo'ladi.

**SUKUT HOLATDA HAMMA SO'ROV `ANTHROPIC_MODEL` GA KETADI** — bo'sh gap
ham. Uchala daraja mavjud, lekin ikkitasi o'chiq va asosiy modelga
qaytadi.

| Daraja | Sukut model | Qachon (yoqilgan bo'lsa) |
|---|---|---|
| `Full` | `claude-opus-4-8` | hamma narsa |
| `Mid` | — (**o'chiq**) | faqat sof ma'lumot o'qish: «bu kod nimani anglatadi» |
| `Cheap` | — (**o'chiq**) | faqat bo'sh gap: «salom», «rahmat» |

Ishga tushishda amaldagi holat jurnalga yoziladi va uni yashirib
bo'lmaydi:

```
Modellar — hamma so'rov: claude-opus-4-8
```

Biror daraja yoqilgan bo'lsa, o'sha qatorda ⚠️ bilan ko'rsatiladi —
GLM ham (`TestAllTiersUseMainModelByDefault`, `TestTiersReportsOverrides`,
`TestTiersReportsGLM`).

⚠️ **Yagona JIM yo'l — `ANTHROPIC_API_URL`.** Manzil boshqa shlyuzga
qo'yilsa, so'rov tanasida `claude-opus-4-8` yozilgan bo'lsa ham,
javobni haqiqatda qaysi model berganini kod tekshira olmaydi. Sarf
jurnali SO'RALGAN nomni yozadi, javob berganini emas. Shuning uchun
bu o'zgaruvchi ishlab chiqarishda rasmiy manzilda qolishi kerak.

Arzonlashtirish — OCHIQ TANLOV:

```bash
ANTHROPIC_MID_MODEL=claude-sonnet-5              # ma'lumot savollari
ANTHROPIC_FAST_MODEL=claude-haiku-4-5-20251001   # bo'sh gap
```

> Hisob-kitob va rasm arzonlashtiriladigan joy emas: noto'g'ri boj
> deklarantga jarima turadi. Shubha bo'lganda daraja **yuqoriga**
> qarab hal qilinadi.

⚠️ **QOIDA TESKARI QILINDI — bir marta xato qilingan joy.**
Avval qoida shunday edi: *«bazadan biror narsa topildimi — demak model
faqat faktni qayta ifodalaydi, o'rta daraja yetadi»*. Bu joriy etilgach
foydalanuvchi javob sifati pasayganini sezdi va o'lchov sababni
tasdiqladi: **25 ta haqiqiy savoldan 19 tasi (76%) Opus dan Sonnet ga
tushib ketgan.**

Xato faraz: «kontekst bor» ≠ «o'ylash kerak emas». Aksincha, eng ko'p
mulohaza talab qiladigan savollar aynan kontekstli:

| Toifa | Nega HUKM |
|---|---|
| tasnif | «elektr skuter qaysi kodga kiradi» — kontekstdan o'qish emas |
| kelib chiqish | 300-moddani tovar, davlat va sertifikat bilan bog'lash |
| imtiyoz | shart bajarilgan-bajarilmaganini tekshirish |
| bilvosita hisob | «olib kirsam nima bo'ladi» — aslida summa so'ralyapti |

Endi sukut — `Full`, arzon darajaga tushish uchun **asos** kerak.
Undan keyin `Mid` ham, `Cheap` ham butunlay o'chirildi: bu ilova pul
va jarima haqida javob beradi, shuning uchun arzonlashtirish ochiq
tanlov bo'lishi kerak, jimgina sukut emas.

⚠️ **Bo'sh gapga kontekst izlanmaydi.** Qidiruv «salom» ga ham parcha
topadi (so'z ichidan: «**salom**atlik»), shuning uchun salomlashishga
13 878 belgi qonun matni qo'shilardi — o'lchangan narxi $0,039, ya'ni
haqiqiy huquqiy savoldan qimmat. Endi «salom» so'rovi **6 token**
kirish bilan ketadi (`TestSmallTalkGetsNoContext`). Ya'ni hammasi
Opus da bo'lsa ham, bo'sh gap deyarli tekin.

⚠️ **Hisob-kitob suhbati oxirigacha `Full` da qoladi.** `modelOf` butun
tarixni ko'radi, faqat oxirgi xabarni emas. Sabab: davomi qisqa keladi —
*«unda 500 kg bo'lsa-chi?»* — unda na hisob so'zi, na kod bor va u
jimgina arzon darajaga tushardi (`TestRoutingRemembersCalcIntent`).

⚠️ **Valyuta kursi bloki `retrieved` ni YOQMAYDI.** U deyarli har
so'rovga qo'shiladi; sanalganda `retrieved` hech qachon `false`
bo'lmasdi va arzon daraja **o'lik kod** edi — o'lchandi, «salom» ham
Opus ga ketardi (`TestRatesBlockDoesNotCountAsRetrieval`).

**Muqobil provayder (GLM / Z.ai) — SUKUT BO'YICHA O'CHIQ.**
`internal/llm/glm.go` da OpenAI-mos adapter bor, lekin u **ikkita** shart
bilan yoqiladi: `GLM_ENABLED=1` va `GLM_API_KEY`. Faqat kalit yetarli
emas — muhitda tasodifan qolib ketgan kalit jimgina provayderni
almashtirib yuborardi va javob boshqa modeldan kelayotgani sezilmay
qolardi. Yoqilgani jurnalda va admin panelida OCHIQ ko'rsatiladi.

⚠️ GLM-5.2 **rasmni qabul qilmaydi** (Z.ai: "Input Modalities: Text").
Yoqilgan bo'lsa ham rasmli xabar Claude ga ketadi — buni
`llm.pickModel` qat'iy ta'minlaydi (`TestImageNeverReachesGLM`).
**4. So'rov cheklovlari** (`internal/api/limit.go`). IP bo'yicha daqiqasiga
va kuniga; hajm chegarasi hamma POST so'rovga. Faqat `/api/chat*` yo'llari
cheklanadi — health va kalkulyator arzon.

| O'zgaruvchi | Sukut | Tavsif |
|---|---|---|
| `RATE_PER_MIN` | 10 | Bitta IP dan daqiqasiga (0 = o'chiq) |
| `DAILY_QUOTA` | 100 | Bitta IP dan kuniga (0 = o'chiq) |
| `MAX_BODY_BYTES` | 8 MB | So'rov hajmi (rasm base64 bilan keladi) |
| `TRUST_PROXY` | — | `1` bo'lsa `X-Forwarded-For` ga ishoniladi |
| `ANTHROPIC_MAX_TOKENS` | 2048 | Javob uzunligi chegarasi |
| `ANTHROPIC_FAST_MODEL` | haiku-4.5 | GLM yo'q bo'lsa ishlatiladigan zaxira arzon model |
| `GLM_API_KEY` | — | Bo'lmasa arzon provayder o'chiq (hammasi Claude da) |
| `GLM_MODEL` | `glm-5.2` | Z.ai modeli |
| `GLM_API_URL` | Z.ai v4 | OpenAI-mos endpoint; testda soxta server |
| `GLM_TIMEOUT_SECONDS` | 180 | 1M kontekstda birinchi belgi kechikishi uzun |
| `ANTHROPIC_CACHE_TTL` | — | `1h` — siyrak trafikda arzonroq kesh |

Hisoblagich xotirada, ya'ni bitta server nusxasi uchun; bir nechta server
ishlaganda umumiy hisoblagich (Redis) kerak bo'ladi.

⚠️ `TRUST_PROXY` faqat ishonchli proksi orqasida yoqilsin — aks holda
mijoz `X-Forwarded-For` ni qalbakilashtirib chegarani aylanib o'tadi.

**5. Tarixni kesish** (`internal/chat/history.go`). Mijoz har so'rovda
BUTUN suhbatni yuboradi. Ya'ni 10-savolda avvalgi to'qqizta savol-javob
yana bir marta to'lanadi, eng og'iri esa **suratlar**: bitta 1600px surat
~300 KB base64 va u suhbat oxirigacha har safar qayta ketardi.

| Chegara | Qiymat | Nima uchun |
|---|---|---|
| `maxHistoryMessages` | 20 | 10 savol-javob; undan oldingisi odatda javoblarda takrorlangan |
| `maxHistoryChars` | 40 000 | Faqat MATN. Retrieval alohida 24 000 gacha qo'shadi |
| `imageMessages` | 2 | Suratlari saqlanadigan oxirgi xabarlar |

Suratlar matn byudjetiga **kirmaydi** va bu ataylab: bitta surat har
qanday matn byudjetidan katta, ikkalasini bitta hisobga qo'shsak qoidalar
bir-birini yeb qo'yardi (buni `TestTrimDropsOldImages` topgan). Suratlar
SONI bilan, matn HAJMI bilan chegaralanadi.

Olib tashlangan surat o'rniga kontekstga **eslatma yoziladi** — model
suratni "yo'q edi" deb emas, "olib tashlangan" deb bilsin.

O'lchangan: uchta suratli suhbatda kontekst **900 KB → 600 KB** (33%).

**Sarfni ko'rish.** Har so'rovdan keyin jurnalga yoziladi:

```
sarf: model=claude-opus-4-8 kirish=7524 chiqish=1141 kesh(yozildi=0 o'qildi=3543)
```

`o'qildi` doim 0 bo'lib qolsa — kesh ishlamayapti va ko'rsatma har safar
to'liq narxda to'lanmoqda.

### Testlar

```bash
cd backend && go test ./... -cover
```

Testlar tarmoqqa chiqmaydi: LLM so'rovlari `ANTHROPIC_API_URL` orqali
`httptest` serveriga yo'naltiriladi, shuning uchun API kaliti kerak emas.

### 2. Frontend (React)

```bash
cd frontend
npm install
npm run dev
```

Ilova `http://localhost:5173` da ochiladi. `/api` so'rovlari avtomatik backendga
yo'naltiriladi (Vite proxy).

## API endpointlari

| Metod | Yo'l                    | Tavsif                              |
|-------|-------------------------|-------------------------------------|
| GET   | `/api/health`           | Server holati, AI mavjudligi        |
| POST  | `/api/hscode/search`    | `{query, use_ai}` → mos kodlar      |
| POST  | `/api/duty/calculate`   | `{customs_value, import_duty, excise, vat, quantity}` → hisob-kitob |
| GET   | `/api/exemptions`       | `?code=` → imtiyoz dasturlari |
| POST  | `/api/utilfee/calculate` | `{code, measure, age_years, weight_kg}` → utilizatsiya yig'imi (79) |
| POST  | `/api/chat`             | `{messages, mode?}` → AI javobi (rasm: `images:[{media_type, data(base64)}]`) |
| POST  | `/api/chat/stream`      | Xuddi shu, lekin javob **oqim** (SSE) bo'lib keladi |

### Oqim (SSE)

Frontend `/api/chat/stream` ni ishlatadi. Sabab: to'liq javob **23–49 soniya**
oladi, birinchi bo'lak esa **~1,5 soniyada** keladi. Foydalanuvchi bo'sh
ekranga qarab turmaydi.

```
data: {"text":"bo'lak"}    — javob bo'lagi
data: {"error":"sabab"}    — xato
data: {"done":true}        — tugadi
```

⚠️ Xato oqim **boshlangandan keyin** ham chiqishi mumkin — o'shanda HTTP
status allaqachon 200 bo'lgan va uni o'zgartirib bo'lmaydi. Shuning uchun
xato hodisa sifatida yuboriladi; mijoz uni albatta ko'rsatishi kerak, aks
holda foydalanuvchi yarim javob olib, nima bo'lganini bilmay qoladi.

`X-Accel-Buffering: no` sarlavhasi qo'yiladi — busiz nginx kabi proksilar
oqimni buferlab qo'yadi va butun foyda yo'qoladi.

### Namuna: boj hisoblash

```bash
curl -X POST http://localhost:8080/api/duty/calculate \
  -H "Content-Type: application/json" \
  -d '{"customs_value":50000000,"import_duty":10,"excise":0,"vat":12,"quantity":1}'
```

## Hisoblash metodikasi

```
Bojxona qiymati (TQ) = (faktura qiymati + transport) × valyuta kursi

Yig'im (10) = BRV(sana) × karra(TQ ning dollardagi ekvivalenti)   -> PKM 55
Boj    (20) = TQ × boj%
Aksiz  (27) = TQ × aksiz%                    -> Soliq kodeksi 285-modda
QQS    (29) = (TQ + boj + qo'shimcha boj + aksiz) × QQS%  -> SK 254-modda
Jami        = Yig'im + Boj + Aksiz + QQS
```

Ikkita nozik joy — ikkalasi ham qonun matnidan olingan:

- **Aksiz bazasiga boj QO'SHILMAYDI** (SK 285-modda: advalor stavkada baza —
  bojxona qiymati). Ko'p manbalarda `(TQ + boj) × aksiz%` deb yozilgan, bu xato.
- **QQS bazasiga bojxona yig'imi KIRMAYDI** (SK 254-modda).

Yig'im qat'iy summa emas — bojxona qiymatining dollardagi ekvivalentiga qarab
BRV ning 1 dan 25 karragacha oralig'ida (ПКМ 55, 31.01.2025).

## Ma'lum kamchiliklar

Bular bilib turib qoldirilgan — ishlatishdan oldin hisobga olish kerak.

**Hisoblashda:**

- [x] **Bojning kombinatsiyalangan stavkasi** qo'shildi — 1 555 kodda
      («10%, lekin kg uchun $0,5 dan kam emas»). Ilgari qat'iy qism
      ekstraktorda o'qilar, lekin chiqarilmasdi va boj kam hisoblanardi.
- [ ] **Birlashtirish qoidasini huquqiy tasdiqlash.** Kattasi olinadi
      («…dan kam emas»), lekin bu manbada yozilmagan — ПП-3818 1-ilova
      matni bilan solishtirish kerak.
- [ ] **Aksizning qat'iy va kombinatsiyalangan stavkalari.** `Calculate` aksizni
      faqat foizda oladi. Aroq, sigaret, benzin kabi tovarlarda stavka qat'iy
      summa (so'm/litr, so'm/1000 dona) — bunday tovarni kalkulyator hisoblay
      olmaydi. Chat AI ni bundan ogohlantiradi, lekin API ogohlantirmaydi.
- [x] Utilizatsiya yig'imi (79) qo'shildi — ПКМ 347, 1 va 2-ilova.
- [ ] **Qo'shimcha boj (21)** — qaysi hollarda qo'llanishi tekshirilmagan.
- [ ] Utilizatsiya yig'imi IMTIYOZLARI kalkulyatorda yo'q (diplomatik,
      retro transport, grant, vaqtinchalik olib kirish) — chat ularni
      qonun matnidan aytadi.
- [x] Kalkulyator manba ning o'z natijasi bilan solishtirildi va **aynan
      mos keldi** (`TestReferenceReferenceCase`): kod 3001209000, faktura
      1 230 000 + transport 25 000 USD, kurs 12 093,35 → qiymat
      15 177 154 250, yig'im 10 300 000 (25×BRV), QQS 1 821 258 510,
      jami 1 831 558 510. Bu QQS bazasidan yig'im chiqarilishini ham
      tasdiqladi — aks holda QQS 1 236 000 so'mga ortiq chiqardi.

**Ma'lumotda:**

- [ ] **QQS kodga xos emas.** Manba bazada `nds` maydoni 20 929 yozuvning
      hammasida bir xil (`[|01.01.2023 ||12]`), ya'ni 12% — umumiy stavka,
      kodga qarab aniqlangan qiymat emas. Import qilinganda ozod qilish
      holatlari bor (Soliq kodeksi 246-modda, ПКМ 352 va h.k.).
      **3 856 kod (29%)** imtiyoz qoidasiga tushadi: 1 287 tasi QQS dan,
      2 520 tasi bojdan ozod bo'lishi mumkin. Imtiyoz SHARTLI ("yuridik
      shaxslar tomonidan", "ro'yxatga kiritilgan bo'lsa"), shuning uchun
      stavka avtomatik 0 qilinmaydi — chat stavka yonida imtiyoz borligini
      ogohlantiradi va shartini so'raydi.
- [ ] **Aksiz stavkalari TIF TN kodiga bog'lanmagan** va bog'lab bo'lmaydi:
      Soliq kodeksi 289¹–289³-moddalari stavkalarni TOVAR NOMI bo'yicha
      beradi. 13 142 kodning birontasida aksiz yo'q (`excise` maydoni umuman
      yozilmaydi — "0%" deb yozib qo'yish yolg'on bo'lardi).
- [ ] Qonun korpusining 22% i lex.uz havolasisiz (74 hujjat).
- [ ] `DateFinish` hujjat amaldaligi uchun ishonchli belgi emas — ayrim eski
      hujjatlarda ham bo'sh turadi.

**Qidiruvda:**

- [ ] Kalit so'z qidiruvi sinonimni tushunmaydi: `aroq` so'rovi alkogol
      moddasini 14-o'ringa tushiradi. Yechimi — embedding (vektor) qidiruvi.
      (`noutbuk` endi topiladi — sinonim jadvali orqali.)
- [x] **Qaratqich «-i» qo'shimchasi** kesiladigan bo'ldi (`stripIzafat`).
      O'zbekchada tovar deyarli doim shu shaklda ataladi — «havo
      konditsioneri», «telefon apparati». Unlidan keyingi «-si» ro'yxatda
      bor edi, undoshdan keyingi yalang «-i» esa yo'q — natijada «havo
      konditsioneri» so'rovi 8415 ni UMUMAN qaytarmay, yog'-moy guruhini
      (1518) birinchi qilardi. Eval to'plamida **top-1 16→17, top-5
      21→22**, regressiyasiz.
- [ ] ⚠️ **TIF TN qidiruvida ahamiyatlilik chegarasi MUMKIN EMAS** —
      sinaldi va o'lchov bilan rad etildi (`hscode.go` da batafsil).
      Xom ball hadlar yig'indisi, ya'ni uzun matn avtomatik yuqori
      chiqadi (ma'nosiz jumla 109, aniq «noutbuk» 73). Ballni so'rovning
      nazariy maksimumiga bo'lib normallashtirish **qisqa** so'rovlarda
      chiroyli ajratdi, lekin butun eval to'plamida to'plamlar ustma-ust
      tushdi:

      | nisbat | tur | so'rov |
      |---|---|---|
      | 0,066 | haqiqiy | Koreyadan 2019-yilgi Hyundai Sonata… |
      | 0,124 | haqiqiy | Dongfeng musor tashuvchi mashina… |
      | 0,153 | shovqin | Bu invoysdagi tovar… |
      | 0,272 | shovqin | kim yaratgan seni |
      | 0,333 | haqiqiy | naushnik |

      Sabab: «Hyundai», «Dongfeng» kabi noyob nomlar maxrajni shishiradi,
      lekin nomenklaturada uchramaydi — ya'ni brend nomi yozadigan
      foydalanuvchi eng ko'p jazolanardi. 0,35 chegarasi bilan eval
      top-5 **22/22 → 18/22**. Yechim faqat embedding qidiruvida.
- [ ] ⚠️ **Qonun qidiruvida ham chegara MUMKIN EMAS** — qo'yildi va
      keng o'lchovdan keyin OLIB TASHLANDI. Kichik namunada ajratish
      chiroyli ko'ringan edi (shovqin 13–14,5; haqiqiy 18,5–23,7),
      lekin 78 ta haqiqiy so'rovda to'plamlar ustma-ust tushdi:

      | ball | tur | so'rov |
      |---|---|---|
      | 11,46 | **haqiqiy** | bojxona jarimasi |
      | 12,79 | **haqiqiy** | jarima |
      | 13,22 | shovqin | salom |
      | 14,50 | shovqin | nima qila olasan |

      Ya'ni deklarantning eng muhim savollaridan biri — **jarima** —
      shovqindan ham past ball oladi va chegara uni qonun kontekstisiz
      qoldirardi (jami 78 dan 4 tasi butunlay, 8 tasi qisman).
      `TestRealQuestionsKeepLawContext` buni qo'riqlaydi.
- [x] **Shovqin boshqa yo'l bilan hal qilindi.** Ball chegarasi o'rniga
      savolning O'ZI tekshiriladi: bo'sh gap («salom», «rahmat») bo'lsa
      qidiruv UMUMAN ishlamaydi. O'lchangan natija chegaradan ham
      yaxshi — «salom» so'rovi 7 710 token o'rniga **8 token**
      (`TestSmallTalkGetsNoContext`).
- [ ] **Nomenklatura lug'at bo'shlig'i.** Ba'zi tovar bazada o'z nomi bilan
      YO'Q. 8705 (maxsus avtotransport) tavsifida misollar sanalgan
      ("аварийные, автокраны, пожарные, автобетономешалки, для уборки
      дорог"), lekin **мусоровоз yo'q** — shuning uchun "musor tashuvchi
      mashina" so'rovi 8433 (qishloq xo'jaligi mashinalari) ni birinchi
      qilgan edi, ya'ni 30% o'rniga 0–5% boj ko'rsatilardi. Bu real
      fotosuratda aniqlandi.
      Vaqtinchalik yechim — `phraseSynonyms` (ibora → nomenklatura atamasi,
      `hscode.go`). Har bir bo'shliq qo'lda topilishi kerak, ya'ni bu
      yechim MIQYOSLANMAYDI; chinakam yechim — embedding qidiruvi.
      `TestSpecialVehiclePhrases` va `TestSpecialVehicleNoRegression`
      shu holatni qo'riqlaydi.

**Sinovda:**

- [x] Chat haqiqiy `ANTHROPIC_API_KEY` bilan sinaldi (2026-07-19): kod
      qidirish, boj/QQS/yig'im hisobi va lex.uz havolasi ishlaydi.
- [x] **Ikkinchi jonli sinov (2026-08-09)** — to'rt so'rov, uchta nuqson
      topildi va uchalasi ham test bilan yopildi:
      1. Model kombinatsiyalangan stavkani e'tiborsiz qoldirdi (bojni
         2,56 mln dedi, aslida 6,05 mln) — `formatMatches` faqat foizni
         yozardi. `TestContextIncludesSpecificDuty`.
      2. Model kodni ALMASHTIRDI: `9405 42 003 9` so'raldi, javob
         `…003 2` bo'yicha keldi — jumla ichidagi aniq kod top-8 ga
         chiqmagan. `promoteExplicit` + `TestExplicitCodeGoesFirst`.
      3. Rasm biriktirilganda qidiruv begona kodlarni beradi (pastga
         qarang). `TestRetrievalWarnsAboutImages`.
      Rasm va SSE oqimi ishladi: 1,6 MB invoys yuborildi, 49 ta oqim
      hodisasi keldi, model tovar, model raqami, miqdor va $15 000 ni
      to'g'ri o'qidi.
- [ ] ⚠️ **Qidiruv SURATNI KO'RMAYDI.** Retrieval faqat savol MATNIDAN
      ishlaydi. Invoys surati yuborilib «tovarni o'qi va bojni hisobla»
      deyilsa, matnda tovar nomi bo'lmaydi va bazadan butunlay begona
      kodlar keladi — jonli sinovda konditsioner invoysiga yog'-moy
      guruhi (1515–2306) keldi. Hozircha kontekst boshiga OGOHLANTIRISH
      qo'yiladi va model o'sha stavkalarni ishlatmasligi aytiladi
      (sinovda model buni o'zi ham sezdi va aytdi). To'liq yechim —
      modelga `hscode_search` ASBOBINI berish, ya'ni u suratni o'qib
      BO'LGACH o'zi qidirsin. Bu chatni tool-use ga o'tkazishni talab
      qiladi (oqim va GLM provayderiga ham tegadi), shuning uchun
      alohida ish sifatida qoldirildi.
- [x] **«Misol bilan hisoblash» taqiqlandi.** Model raqam yetishmaganda
      «Hozircha misol bilan tushuntiraman (aniq raqamlar bergach, qayta
      hisoblab beraman)» deb butun hisobni SOXTA summalar bilan
      chiqarardi. Javobda bunday summa haqiqiysidan farq qilmaydi —
      foydalanuvchi uni ko'chirib olib deklaratsiyada ishlatishi mumkin.
      Endi yetishmayotgan raqam so'raladi; formula raqamsiz ko'rsatiladi.
      Istisno — foydalanuvchining o'zi misol so'rasa.
      `TestSystemPromptForbidsInventedNumbers`.
- [x] ⚠️ **Kesilgan javob endi BELGILANADI.** `max_tokens` ga urilgan
      javob API xato bermasdan gap o'rtasida tugaydi. Android sinovida
      (2026-08-09) boj hisobi aynan shunday kesildi va oxiridagi
      "kelib chiqish sertifikati bo'lmasa boj IKKI BAROBAR"
      ogohlantirishi yo'qoldi — foydalanuvchi javobni to'liq deb o'qib,
      kam to'lov hisoblab qolardi. Endi `stop_reason` o'qiladi va
      javob oxiriga ochiq ogohlantirish qo'shiladi (oqimda ham).
      `TestCompleteMarksTruncation`, `TestStreamMarksTruncation`.
- [ ] **Model arifmetikasi kalkulyatordan ~1 000 so'm chetlashdi**
      (10 304 240 vs 10 304 360). Kalkulyator asosiy manba;
      hisob-kitobda uning natijasini ko'rsatish kerak, modelnikini emas.
- [x] `chat`, `api` va `llm` paketlari test bilan qoplandi (71 test).
      LLM so'rovlari `ANTHROPIC_API_URL` orqali soxta serverga yo'naltiriladi,
      shuning uchun testlar tarmoqqa chiqmaydi va kalit talab qilmaydi.
- [ ] Frontendda test yo'q (Markdown.tsx va oqim mantig'i ayniqsa muhim).
- [ ] **Javob har doim bir xil emas.** `temperature` ni pasaytirib bo'lmaydi
      (bu modelda qo'llab-quvvatlanmaydi), shuning uchun barqarorlik faqat
      kontekstni aniq yozish bilan ta'minlanadi. Sinovda uch urinishning
      birida model aksizni "yo'q" deb aytdi — endi 289¹–289³-moddalar
      qamroviga asoslanib, ammo baribir bazadan tasdiqlamay.
```
"# deklarantai" 
