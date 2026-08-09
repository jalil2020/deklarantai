# Deklarant AI — Android

Native ilova: **Kotlin + Jetpack Compose + Voyager**.

## Ishga tushirish

Sinaldi va **ishlaydi**: emulyator `Medium_Phone_API_36.0` (API 36).

1. **Backend'ni ishga tushiring** (loyiha ildizida):

   ```bash
   cd backend && ANTHROPIC_API_KEY=sk-ant-... go run .
   ```

2. **Ilovani o'rnating:**

   ```bash
   # JDK — Android Studio ichidagi (alohida o'rnatish shart emas)
   export JAVA_HOME="C:/Program Files/Android/Android Studio1/jbr"

   ./gradlew :app:installDebug
   ```

   Yoki Android Studio da `android/` papkasini oching.

3. **Manzil.** Sukut — emulyator uchun (`10.0.2.2` mezbon mashinaning
   localhost'i). Haqiqiy telefonda LAN manzilini bering:

   ```bash
   ./gradlew installDebug -PapiBaseUrl=http://192.168.1.5:8080
   ```

   Yoki `gradle.properties` dagi `deklarant.apiBaseUrl` ni o'zgartiring.

`local.properties` da SDK yo'li **oldinga qiya chiziq bilan** yozilishi kerak:

```properties
sdk.dir=C:/Users/user/AppData/Local/Android/Sdk
```

> Teskari chiziq ishlatilsa, Java `.properties` uni unicode escape deb
> o'qiydi va `Malformed \uxxxx encoding` xatosi chiqadi.

### Versiyalar

| Nima | Versiya | Izoh |
|---|---|---|
| JDK | 25 | Android Studio ichidagi JBR |
| Gradle | 9.6.1 | JDK 25 uchun 9.x kerak; 8.x qo'llab-quvvatlamaydi |
| AGP | 9.3.1 | |
| Kotlin | 2.3.10 | |
| Compose BOM | 2025.06.00 | |

> ⚠️ **AGP 9 dan boshlab `org.jetbrains.kotlin.android` plagini KERAK EMAS**
> — Kotlin qo'llab-quvvatlash AGP ichiga kirgan. U qoldirilsa qurish
> "plugin is no longer required" xatosi bilan to'xtaydi.

## Belgi (ikonka)

Launcher ikonkasi — dizayner bergan rasmning O'ZI, vektorga ko'chirilmagan.
Barcha o'lchamlar bitta buyruq bilan yasaladi (loyiha ildizida):

```bash
go run tools/logo/main.go asl-logo.png
```

Dastur matnni kesadi, oq fonni shaffofga o'giradi va `mipmap-*dpi/` ga
adaptive ikonka qatlamini, `drawable-nodpi/logo_mark.png` ga esa ilova
ichidagi belgini yozadi. Adaptive fon — oq (`ic_launcher_background`),
chunki belgining ichki oq rangi uning bir qismi.

## Tuzilishi

```
domain/   — modellar va repozitoriy INTERFEYSLARI (toza Kotlin)
data/     — DTO, HTTP klient, repozitoriy amalga oshirilishi
ui/       — Compose ekranlari va Voyager ScreenModel lari
di/       — AppContainer (bog'liqliklar yig'iladigan yagona joy)
```

Bog'liqlik yo'nalishi bir tomonlama: `ui → domain ← data`. Ekran
modellari interfeysga bog'lanadi, amalga oshirilishini esa `AppContainer`
ulaydi — shuning uchun testda soxta repozitoriy berish uchun ekran kodini
o'zgartirish kerak emas.

**DI freymvorki yo'q** (Hilt/Koin): uchta repozitoriy va bitta HTTP klient
uchun annotatsiya ishlovchisi va qo'shimcha qurish vaqti oqlanmaydi.
Konstruktor orqali uzatish shu miqyosda soddaroq.

## Bo'limlar

| Bo'lim | Nima qiladi | Kirish |
|---|---|---|
| **Chat** | Suhbat, javob oqim bilan keladi (SSE). Surat biriktirish | ✅ talab qilinadi |
| **TIF TN** | Qidiruv + ierarxiya (Bo'lim → Guruh → Pozitsiya → Kod) | kerak emas |
| **Qonunlar** | Hujjat → Modda → To'liq matn, lex.uz havolasi bilan | kerak emas |

### Navigatsiya — YON MENYU, pastki panel emas

Bo'limlar `ModalNavigationDrawer` da, yuqoridagi ☰ tugmasi ortida.
Ekranning **pasti butunlay chatga tegishli**.

**NEGA:** chat — asosiy ekran va u balandlikni talab qiladi. Pastda
doimiy `NavigationBar` turganda kiritish maydoni, klaviatura va panel
joy talashardi. Pastki panel bo'limlar TENG va tez-tez almashtirilganda
oqlanadi; bu yerda nisbat boshqacha — chat 90% vaqt, TIF TN va Qonunlar
esa kerak bo'lganda kiriladigan yordamchi bo'limlar.

Foydalanuvchi ma'lumoti (ism, rol, kunlik kvota, chiqish) ham menyuga
ko'chdi. Ilgari u chat ekranining yuqorisida rejim tugmalari bilan bir
qatorda turardi va telefon enida sig'masdi — matn qirqilardi.

⚠️ **Ikkita nozik joy, ikkalasi ham emulyatorda sinab topilgan:**

- **`BackHandler` QO'LDA qo'yilgan.** `ModalNavigationDrawer` orqaga
  tugmasini o'zi ushlamaydi: menyu ochiq turganda "orqaga" bosilsa
  ilova butunlay yopilib, bosh ekranga chiqib ketardi.
- **Surish ishorasi faqat menyu OCHIQ bo'lganda** (`gesturesEnabled =
  drawerState.isOpen`). Aks holda chatdagi va TIF TN yo'l zanjiridagi
  gorizontal surishlar menyuni tortib ochardi.

Kirish oynasi menyudan emas, **chat ekranidan** ochiladi: `LoginDialog`
ga `LoginScreenModel` kerak, u esa `rememberScreenModel` bilan olinadi
va bu Voyager `Screen` ning kengaytmasi — oddiy composable ichida
ishlamaydi. Shuning uchun menyu faqat SO'RAYDI (`LoginPrompt`), oynani
chat ochadi. Bu `ChatHandoff` bilan bir xil naqsh.

## TIF TN kodini topish

Ikki yo'l **bitta ekranda** — deklarant kodni ba'zan biladi, ba'zan yo'q:

| Yozilgan | Nima bo'ladi |
|---|---|
| `8703` | Kod raqami bo'yicha ro'yxat |
| `noutbuk` | Tovar nomi bo'yicha (12 natija chiqdi) |
| *(bo'sh)* | Ierarxiya qaytadi — Bo'lim → Guruh → Pozitsiya → Kod |

Har qatorda kod, nomi va **boj stavkasi**; kombinatsiyalangan stavkali
kodda foiz ostida qat'iy qism ham (`$0.5/kg`) — 1 555 kodda bor va uni
ko'rsatmaslik bojni kam hisoblashga olib borardi.

Qidiruv paytida yo'l zanjiri yashiriladi: natijalar ierarxiyaning bir
joyiga emas, butun baza bo'ylab tegishli.

⚠️ So'rov **250 ms kechikish** bilan ketadi va oldingi ish bekor
qilinadi (`searchJob.cancel()`) — aks holda «noutbuk» yetti so'rov
yuborardi va sekin kelgan eski javob yangisini bosib ketishi mumkin edi.

⚠️ `SearchResponse.matches` **nullable** qilindi: hech narsa topilmasa
backend bo'sh ro'yxat emas, `null` qaytaradi — nullable bo'lmasa
kotlinx serializatsiyasi xato tashlardi. Aynan shu nuqson webda ham
chiqqan edi.

## Kirish

Chat **pul turadi**, shuning uchun u kirishsiz ochilmaydi. Qolgan
bo'limlar AI ga bormaydi va kirishsiz ishlayveradi.

```
ui/auth/LoginDialog      — kirish va ro'yxatdan o'tish (MODAL)
data/auth/AuthStore      — token (SharedPreferences)
AuthRepositoryImpl       — StateFlow<User?>, ilova bo'ylab yagona haqiqat
```

Kirish **alohida ekran emas, modal oyna**: chat joyida qoladi va
foydalanuvchi ilova nima qila olishini ko'rib turadi. Kiritish maydonida
ishora bor — *"Savol berish uchun kiring — bosing"* — va bosilganda oyna
ochiladi.

⚠️ Yopiq maydon ustiga **shaffof qatlam** qo'yilgan. `Modifier.clickable`
ni maydonning o'ziga qo'yish ishlamaydi: `OutlinedTextField` bosishni
o'zi yutib, kursorni qo'yadi va tashqi bosish chaqirilmaydi (emulyatorda
shunday chiqdi). `enabled = false` esa bosishni umuman o'tkazmaydi va
maydon o'chgan ko'rinadi.

Oyna kirish muvaffaqiyatli bo'lganda **o'zi yopiladi** — natijani
`AuthRepository.user` oqimi beradi, ya'ni ikkita haqiqat manbai
bo'lmaydi.

Rollar va kvota **serverdan** (`GET /api/auth/roles`) olinadi — ikki
joyda yozilsa, vaqt o'tib ajralib ketardi.

⚠️ Token `SharedPreferences` da, **shifrlanmagan**. Qabul qilingan
murosa: token muddatli va serverdan bekor qilinadi (chiqish token
versiyasini oshiradi), parolning o'zi esa qurilmada umuman saqlanmaydi.

Sinaldi emulyatorda: kirish → chat ochildi → ilova qayta ishga tushdi,
seans saqlandi → chiqish → token fayli tozalandi va TIF TN bo'limi
kirishsiz ishlashda davom etdi.

**API kaliti** (`-PapiKey`) hamon ishlaydi — u server-server va hamkor
ulanishlari uchun, foydalanuvchi seansini talab qilmaydi.

Kod yoki modda tanlansa, chatga tayyor savol qo'yiladi va chat bo'limi
ochiladi. Savol **yuborilmaydi** — foydalanuvchi qiymat va davlatni
qo'shishi kerak.

## Rasm biriktirish

Tovar yoki invoys surati yuborilsa, AI undagi tovar, miqdor va narxni
o'qib TIF TN kodini taklif qiladi va bojni hisoblaydi.

Ikki yo'l, **ikkalasi ham ish vaqti ruxsatisiz**:

| Yo'l | Mexanizm | Nega ruxsat kerak emas |
|---|---|---|
| Galereya | Photo Picker (`PickMultipleVisualMedia`) | Tizim tanlagichi faqat tanlangan faylga kirish beradi |
| Kamera | `TakePicture` + FileProvider | Surat tizim kamera ilovasi orqali olinadi; biz CAMERA ni e'lon qilmaymiz |

> Agar manifestda CAMERA e'lon qilinsa, Android uni BIZDAN ham talab
> qilardi — foydasiz qo'shimcha dialog. Shuning uchun e'lon qilinmagan.

**Har rasm uch qadamdan o'tadi** (`AndroidImageRepository`):

1. **Kichraytirish** 1600px gacha — telefon surati 4000px atrofida, base64
   hajmni yana ~33% oshiradi va backend'ning 8 MB chegarasidan oshib ketardi
2. **EXIF burish** — kamera suratni burilgan saqlaydi, aks holda model
   yonboshlagan invoysni o'qishga urinardi
3. **JPEG siqish** (sifat 85) — PNG matnli suratda bir necha barobar kattaroq

Yuborilgan surat suhbatda **ko'rinib turadi**: foydalanuvchi nima
jo'natganini eslab qolsin va noto'g'ri surat ketgan bo'lsa darrov bilsin.

## Chat javobi — Markdown

Model javobi deyarli doim JADVAL bilan keladi (boj, QQS, yig'im GTD
kodlari bo'yicha). Ilgari u oddiy `Text` bilan chizilar va ekranda xom
`| Kod | To'lov |` hamda `**JAMI**` ko'rinardi.

`ui/common/Markdown.kt` — o'z chizuvchimiz. Qo'llab-quvvatlanadi:
sarlavha, **jadval**, ro'yxat, qalin/kursiv, `kod`, havola, ajratgich,
iqtibos. Webdagi `Markdown.tsx` bilan BIR XIL to'plam.

Tayyor kutubxona olinmadi: ko'pchiligi jadvalni umuman chizmaydi yoki
WebView orqali chizadi.

⚠️ **Jadval ustunlari mazmunga qarab kengayadi.** Teng ulush berilsa,
"GTD" yoki "10" kabi tor ustun keng summalar bilan bir xil joy olardi
va raqamlar har qatorda ikkiga bo'linib ketardi.

⚠️ **Markdown faqat AI javobiga** qo'llanadi. Foydalanuvchi xabari
o'zi yozgani holicha chiziladi — aks holda savoldagi yulduzcha yoki
tik chiziq matnni "bezab" yuborardi.

AI puchug'i kengroq (`fillMaxWidth(0.97f)`), foydalanuvchiniki esa
320dp da qoladi — jadval sig'sin, lekin qisqa savol o'ngga tekislangani
ko'rinib tursin.

**Nusxa olish** tugmasi har javob ostida: deklarant koddan, summadan
yoki modda raqamidan boshqa joyga ko'chiradi, `Text` esa tanlanmaydi.
Bufergga XOM Markdown yoziladi — jadval belgilari yo'qotilsa, boshqa
joyga qo'yilganda ustunlar aralashib ketardi.

Sinaldi emulyatorda: sarlavha, qalin matn, jadval va lex.uz havolasi
to'g'ri chizildi; "Nusxa olish" bosilganda tasdiq chiqdi va buferda
xom Markdown turdi.

## Suhbat tarixi

Suhbat **qurilmada saqlanadi** — ilova yopilsa yoki tizim uni fonda
o'ldirsa ham yo'qolmaydi. Ilgari tarix `ChatScreenModel` ichida, faqat
xotirada edi: bitta hisob-kitob uzun savol (kod, miqdor, faktura,
transport, davlat) va uzun javob, ularni qaytadan yozish og'ir.

```
data/chat/ChatHistoryStore  — filesDir/chat-history.json
```

**NEGA FAYL, SharedPreferences EMAS:** xabarlarda base64 rasm bo'ladi
(bitta invoys ~1,6 MB). SharedPreferences butun XML ni xotirada ushlaydi
va har yozishda qayta serializatsiya qiladi. `AuthStore` da
SharedPreferences qoladi — u yerda bitta qisqa token.

**NEGA Room EMAS:** so'rov ham, indeks ham kerak emas — tarix butunlay
o'qiladi va butunlay yoziladi. Bitta ro'yxat uchun baza, annotatsiya
ishlovchisi va migratsiyalar oqlanmaydi.

⚠️ **Yozish ATOMAR**: avval `.tmp` ga yoziladi, keyin ko'chiriladi.
Yozish o'rtasida ilova o'ldirilsa, yarim fayl qolib ketardi va keyingi
ochilishda tarix BUTUNLAY yo'qolardi.

⚠️ **Savol darrov saqlanadi**, javobni kutmasdan — javob 30–50 soniya
keladi va shu orada ilova o'ldirilsa, uzun savol yo'qolardi.

⚠️ **Rasm SONI bo'yicha cheklanadi** (oxirgi 2 ta), hajmi bo'yicha emas.
Hajm bo'yicha hisoblansa, bitta katta surat oldidagi butun yozishmani
siqib chiqarardi — aynan shu xatoga backend tarixini qisqartirishda ham
yo'l qo'yilgan va test uni ushlagan edi. Eski rasm o'chirilganda xabarga
"[surat saqlanmadi]" eslatmasi qo'shiladi.

⚠️ **Chiqishda tarix o'chiriladi** — bitta qurilmadan foydalanadigan
keyingi odam oldingisining yozishmasini ko'rmasin.

Tozalash tugmasi (🗑) faqat gap bo'lsa ko'rinadi va **tasdiq so'raydi**:
tarix endi diskda, ya'ni tasodifiy bosish bir necha kunlik yozishmani
o'chirib yuborishi mumkin.

Sinaldi emulyatorda: savol yuborildi → `chat-history.json` yaratildi →
ilova `force-stop` bilan butunlay yopildi → qayta ochilganda suhbat
joyida edi → "Tozalash" bosilganda fayl o'chdi.

## Ma'lum kamchiliklar (UI/UX)

- [x] Javob Markdown sifatida chiziladi (yuqoriga qarang).
- [x] Suhbat qurilmada saqlanadi (yuqoriga qarang).
- [x] Javobni nusxalash tugmasi qo'shildi.
- [x] **Oqim paytida ro'yxat endi majburan surilmaydi** — faqat
      foydalanuvchi PASTDA turganda. Ilgari har bo'lakda
      `animateScrollToItem` chaqirilar va yuqoriga qarab o'qiyotgan
      odamni orqaga tortardi. O'z xabaringga esa doim tushiladi.
- [x] **Xatodan keyin «Qayta urinish»** — ilgari tarmoq uzilsa, uzun
      savolni (kod, miqdor, narx, davlat) qaytadan terish kerak edi.
- [ ] **To'rt bo'lim yetishmaydi.** Webda bor, Androidda yo'q:
      Kalkulyator, Risk baholash, Tarixcha, Sevimlilar. Menyu ularni
      qo'shishga tayyor (`sections` ro'yxati).
- [ ] **Bitta suhbat saqlanadi**, webdagidek "Tarixcha" ro'yxati emas —
      eski yozishmalarga qaytib bo'lmaydi, faqat oxirgisi davom etadi.
- [ ] **Chiqish tasdiqsiz.** Menyudagi tugma darrov chiqaradi (va endi
      tarixni ham o'chiradi).
- [ ] Javob oqimi paytida "yozilmoqda" belgisi yo'q — faqat matn
      paydo bo'lishidan bilinadi.
