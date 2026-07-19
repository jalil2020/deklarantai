# 🛃 Deklarant AI

O'zbekiston bojxona rasmiylashtiruvi uchun AI yordamchi. Asosiy interfeys — **chat**,
qo'shimcha strukturaviy vositalar bilan:

1. **💬 Chat (asosiy)** — bojxona kod, boj va qonunchilik bo'yicha suhbat.
   **Rasm o'qiydi**: tovar surati yoki invoysni yuklang — AI (Claude vision) undagi tovar,
   miqdor va narxni o'qib, TIF TN kodini taklif qiladi va bojni hisoblab beradi.
   Rasm tanlash, nusxa-joylash (paste) yoki sudrab tashlash (drag & drop) mumkin.
2. **🧮 Kalkulyator** — import boji, aksiz, QQS va bojxona yig'imini aniq hisoblaydi.
3. **🔎 HS kod** — tovar nomiga qarab TIF TN kodini qidiradi (ixtiyoriy AI izohi bilan).

> ⚠️ **Demo:** Kodlar bazasi va stavkalar namunaviy. Ishlab chiqarishda rasmiy TIF TN
> jadvali va joriy stavkalar bilan almashtirilishi kerak (manba: [customs.uz](https://customs.uz)).

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
│   ├── data/hscodes.json     # namunaviy TIF TN bazasi
│   └── internal/
│       ├── api/              # HTTP handlerlar + CORS
│       ├── hscode/           # kod qidiruv
│       ├── duty/             # boj hisoblash
│       ├── chat/             # qonunchilik chatti
│       └── llm/              # Claude API klienti
└── frontend/                 # React SPA
    └── src/
        ├── api.ts            # backend klienti
        └── components/       # Chat (rasm yuklash), DutyCalculator, HSCodeSearch
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

`backend/data/laws.json` — **1 045 parcha, 89 hujjatdan** (3,9 MB). Generatsiya:

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
Matn **rasmiy o'zbekcha** versiyadan olinadi (ruschasi ayrim hujjatlarda mashina
tarjimasi bo'lib, yuridik kuchga ega emas) va lotinga transliteratsiya qilinadi.

Har bir parchaga imkon qadar **lex.uz havolasi** biriktiriladi, shunda AI javobda
rasmiy manbani ko'rsatadi va foydalanuvchi o'zi tekshira oladi. Moslik
`tools/lex-links.mjs` da **qo'lda** yuritiladi — manba bazasida lex.uz
identifikatorlari yo'q (Bojxona kodeksi: bazada `39534`, lex.uz da `2876352`).
Hozirgi qamrov — **78%** (815/1 045 parcha, 15 ta hujjat).

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
| POST  | `/api/chat`             | `{messages: [{role, content, images?}]}` → AI javobi (rasm: `images:[{media_type, data(base64)}]`) |

### Namuna: boj hisoblash

```bash
curl -X POST http://localhost:8080/api/duty/calculate \
  -H "Content-Type: application/json" \
  -d '{"customs_value":50000000,"import_duty":10,"excise":0,"vat":12,"quantity":1}'
```

## Hisoblash metodikasi

```
Import boj = Bojxona qiymati × boj%
Aksiz      = (Bojxona qiymati + Import boj) × aksiz%
QQS        = (Bojxona qiymati + Import boj + Aksiz) × QQS%
Jami       = Bojxona yig'imi + Import boj + Aksiz + QQS
```

## Keyingi qadamlar (rivojlantirish g'oyalari)

- [ ] Rasmiy TIF TN bazasini to'liq import qilish (10 raqamli barcha kodlar)
- [ ] Aksiz uchun qat'iy stavkalar (so'm/dona) va valyuta konvertatsiyasi
- [ ] AI qidiruvni RAG bilan kuchaytirish (bojxona kodeksi hujjatlari asosida)
- [ ] Foydalanuvchi hisobi va hisob-kitoblar tarixi
- [ ] GTD (yuk bojxona deklaratsiyasi) grafalarini to'ldirishga yordam
```
