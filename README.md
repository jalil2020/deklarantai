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
└── frontend/                 # React SPA
    └── src/
        ├── api.ts            # backend klienti
        └── components/Chat.tsx   # yagona komponent (rasm yuklash bilan)
```

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

| Muhit o'zgaruvchisi | Majburiy | Tavsif |
|---|---|---|
| `ANTHROPIC_API_KEY` | chat uchun | Bo'lmasa chat o'chiq, qolgani ishlaydi |
| `ANTHROPIC_MODEL` | yo'q | Sukut bo'yicha `claude-opus-4-8` |
| `ANTHROPIC_API_URL` | yo'q | Korporativ shlyuz yoki testdagi soxta server |
| `PORT` | yo'q | Sukut bo'yicha `8080` |
| `HSCODE_DATA`, `LAWS_DATA`, `DOCS_DATA` | yo'q | Ma'lumot fayllari yo'li |

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
| Oxirida | — | "Keyingi qadamlar" ro'yxati |

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

## Xarajat nazorati

Har bir chat so'rovi pul turadi, shuning uchun to'rt qatlam qo'yilgan.
Sozlash — muhit o'zgaruvchilari bilan.

**1. Prompt keshi.** Tizim ko'rsatmasi ~6 000 belgi va har so'rovda bir xil.
Unga `cache_control: ephemeral` biriktiriladi va keshdan o'qish narxi asl
narxning ~10%i. O'lchangan: har so'rovda **3 543 token keshdan** o'qiladi.

Retrieval bloklari ataylab **oxirgi foydalanuvchi xabariga** qo'shiladi —
ular o'zgaruvchan, shuning uchun keshga tushmaydi va keshni buzmaydi ham.

**2. Kontekst byudjeti** (`maxContext = 24 000` belgi). O'lchov shuni
ko'rsatdi: "salom" so'zi **57 KB** kontekst yasagan edi — sabab, qonun
parchalari 94 KB gacha bo'lgan (`splitLong` uzun abzatsni bo'lmasdi).
Parchalash tuzatilgach 13,5 KB ga tushdi, byudjet esa yuqori chegarani
kafolatlaydi. Sig'magan parchalar borligi kontekstda **ochiq yoziladi**.

> Ball chegarasi qo'yilmadi: o'lchovda "salom" 13,2 ball, haqiqiy savol
> 19,2 ball olgan — farq juda kichik va chegara foydali natijalarni ham
> kesib yuborardi. Byudjet sifat haqida qaror qabul qilmaydi.

**3. Arzon model.** Bazadan hech narsa topilmagan va 120 belgidan qisqa
savol (salomlashish, "nima qila olasan") Haiku ga yo'naltiriladi.
O'lchangan: "rahmat" → Haiku, 2 879 token, 8 s (Opus da 21–28 s).

> ⚠️ Bazadan biror narsa topilgan bo'lsa — **doim** asosiy model. Stavka,
> modda raqami va hisob-kitob arzonlashtiriladigan joy emas.

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
| `ANTHROPIC_FAST_MODEL` | haiku-4.5 | Arzon model |

⚠️ Bu autentifikatsiya **o'rnini bosmaydi** — u kelgunicha eng kam himoya.
Hisoblagich xotirada, ya'ni bitta server nusxasi uchun; bir nechta server
ishlaganda umumiy hisoblagich (Redis) kerak bo'ladi.

⚠️ `TRUST_PROXY` faqat ishonchli proksi orqasida yoqilsin — aks holda
mijoz `X-Forwarded-For` ni qalbakilashtirib chegarani aylanib o'tadi.

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
      moddasini 14-o'ringa tushiradi, `noutbuk` esa umuman topilmaydi.
      Yechimi — embedding (vektor) qidiruvi.

**Sinovda:**

- [x] Chat haqiqiy `ANTHROPIC_API_KEY` bilan sinaldi (2026-07-19): kod
      qidirish, boj/QQS/yig'im hisobi va lex.uz havolasi ishlaydi.
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
