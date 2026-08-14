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



## ⚠️ Ma'lumot fayllari repozitoriyada YO'Q

`backend/data/*.json` gitga chiqmaydi. Klon qilgandan keyin ularni
QAYTA YASASH kerak — aks holda server ishga tushmaydi.

| Fayl | Yasash |
|---|---|
| `hscodes.json` | `node tools/extract-hscodes.mjs --date=2026-07-19` |
| `laws.json` | `node tools/extract-laws.mjs` |
| `docs.json` | `node tools/extract-docs.mjs` |
| `countries.json` | `node tools/extract-countries.mjs` |
| `taxonomy.json` | `node tools/extract-taxonomy.mjs` |

Manba — `backend/data/manba/*.sqlite` (laws.json uchun lex.uz).
Ular ham gitda yo'q va **bo'lmasligi kerak**: ichida real
deklaratsiyalar, 4 670 ta noyob STIR raqami va 42 ta telefon raqami
bor — bu uchinchi tomonlarning ma'lumoti.

## Joylashtirish

Ishlab chiqarish: **https://deklarantpro.uz** (nginx + systemd).

```bash
bash deploy/setup.sh      # serverda, bir marta
bash deploy/release.sh    # mahalliy, har chiqarishda
```

Batafsil izohlar `deploy/` papkasidagi fayllarda.

## GTD grafalarini to'ldirish

Chatда deklaratsiya/GTD/grafalarni to'ldirish so'ralса, model **GTD
grafalari jadvalини** to'ldirib beradi (Yuk bojxona deklaratsiyasi,
ИМ 40). Ma'lumotnoма — `internal/gtd/fields.go`.

Har graf uch turдан biri bilan to'ldirilади:

| Tur | Kim | Misol |
|---|---|---|
| **avto** | mavjud hisob | 33 (TIF TN kod), 47 (to'lovlar), 12/45 (bojxona qiymati), 34 (kelib chiqish) |
| **foydalanuvchi** | rekvizit | 2 (jo'natuvchi), 8 (oluvchi STIR), 35 (vazn), 54 (imzo) |
| **ma'lumotnoma** | standart kod | 1 (ИМ 40), 37 (protsedura), 43 (baholash usuli) |

**NEGA MANTIQLI:** GTD ning «aql» talab qiladigan qismi — kod tanlash
va to'lov hisobi — loyihaning mavjud qismlari (`hscode`, `duty`,
`countries`) bilan allaqачон yechilган. 34 grafдан **14 tasi avtomatik**.

⚠️ **Skelet faqat GTD so'ralса qo'shiladi** (`hasGTDIntent`) — har
so'rovда emas, chunki u ~1,5 KB va aksariyat savol GTD haqида emas.

⚠️ **O'ylаб topilган qiymat yo'q.** «foydalanuvchi» grafalari
`[foydalanuvchi]` deb qoldiriladi, so'raladi — to'qilmaydi. To'ldirish
qoidаsи (grafalar formatи) qonun korpusидаgi rasmiy yo'riqnomага
(06.04.2016) tayanади. Javob oxirида «rasmiy yo'riqnoma va broker bilan
tekshiring» ogohlantirishi bilan — bu yordam, tayyor deklaratsiya emas.

Jonli sinov (2026-08-14): kod 9405 42 003 9, Xitoydan 1000 kg → kod,
bojxona qiymati (26 015 000), to'lovlar (kombinatsiyalangan stavka
bilan), kelib chiqish avtomatik to'ldirildi; jo'natuvchi/oluvchi/vazn
`[foydalanuvchi]` deb qoldirildi.
