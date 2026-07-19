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
│   ├── data/laws.json        # qonun korpusi (1 111 parcha)
│   ├── data/docs.json        # hujjat talablari (15 112 qoida)
│   └── internal/
│       ├── api/              # HTTP handlerlar + CORS
│       ├── hscode/           # kod qidiruv
│       ├── duty/             # boj hisoblash
│       ├── docs/             # hujjat talablari (kod oralig'i bo'yicha)
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

`backend/data/laws.json` — **1 111 parcha, 89 hujjatdan** (3,9 MB). Generatsiya:

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
Hozirgi qamrov — **80%** (887/1 111 parcha, 16 ta hujjat).

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
- [ ] **Utilizatsiya yig'imi (79)** — avtotransport uchun, netto vazn bo'yicha.
      Umuman qo'llab-quvvatlanmaydi.
- [ ] **Qo'shimcha boj (21)** — qaysi hollarda qo'llanishi tekshirilmagan.
- [x] Kalkulyator manba ning o'z natijasi bilan solishtirildi va **aynan
      mos keldi** (`TestReferenceReferenceCase`): kod 3001209000, faktura
      1 230 000 + transport 25 000 USD, kurs 12 093,35 → qiymat
      15 177 154 250, yig'im 10 300 000 (25×BRV), QQS 1 821 258 510,
      jami 1 831 558 510. Bu QQS bazasidan yig'im chiqarilishini ham
      tasdiqladi — aks holda QQS 1 236 000 so'mga ortiq chiqardi.

**Ma'lumotda:**

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
- [ ] `chat` va `api` paketlarida avtomatik test yo'q — sinov qo'lda
      qilingan.
- [ ] **Javob har doim bir xil emas.** `temperature` ni pasaytirib bo'lmaydi
      (bu modelda qo'llab-quvvatlanmaydi), shuning uchun barqarorlik faqat
      kontekstni aniq yozish bilan ta'minlanadi. Sinovda uch urinishning
      birida model aksizni "yo'q" deb aytdi — endi 289¹–289³-moddalar
      qamroviga asoslanib, ammo baribir bazadan tasdiqlamay.
```
