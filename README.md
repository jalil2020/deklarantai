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
