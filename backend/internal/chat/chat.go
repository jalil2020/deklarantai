// Package chat bojxona bo'yicha savol-javob yordamchisini ta'minlaydi.
//
// Retrieval: 13 000+ kodni promptga sig'dirib bo'lmaydi, shuning uchun
// foydalanuvchining oxirgi savoliga mos kodlar bazadan qidirib topiladi va
// faqat o'shalar kontekstga qo'shiladi.
package chat

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"deklarant-ai/backend/internal/docs"
	"deklarant-ai/backend/internal/duty"
	"deklarant-ai/backend/internal/gtd"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
	"deklarant-ai/backend/internal/llm"
	"deklarant-ai/backend/internal/rates"
)

const (
	// topCodes — kontekstga qo'shiladigan TIF TN kodlari soni.
	topCodes = 8
	// topLaws — qonun parchalari soni. Ular yirik (~4 KB), shuning uchun kam.
	topLaws = 3

	// maxContext — retrieval bloklarining umumiy hajm chegarasi (belgi).
	//
	// NEGA KERAK: "top-3 parcha" hajmni kafolatlamaydi — parcha uzunligi
	// har xil. O'lchov shuni ko'rsatdi: "salom" so'zi 57 KB kontekst
	// yasagan edi (parchalar 94 KB gacha bo'lgani uchun). Parchalash
	// tuzatilgach 13,5 KB ga tushdi, lekin baribir salomlashish uchun ko'p.
	//
	// Byudjet sifat haqida QAROR QABUL QILMAYDI — u faqat eng yomon
	// holatni chegaralaydi. Ball chegarasi qo'yish xavfliroq bo'lardi:
	// o'lchovda "salom" 13,2 ball, haqiqiy savol 19,2 ball olgan — farq
	// juda kichik va chegara foydali natijalarni ham kesib yuborardi.
	maxContext = 24_000

)

// Mode — javob uslubi. Faktlar va ogohlantirishlar ikkala rejimda BIR XIL;
// farq faqat tilda, uzunlikda va tuzilmada.
type Mode string

const (
	// ModeDeclarant — kasbiy foydalanuvchi: TIF TN, GTD grafalari va
	// rejimlarni biladi. Qisqa jadval, kod va modda raqami kerak.
	ModeDeclarant Mode = "deklarant"
	// ModeBusiness — tadbirkor: atamalarni bilmaydi. "Menga qancha
	// turadi va nima qilishim kerak" degan savolga javob kerak.
	ModeBusiness Mode = "tadbirkor"
)

// defaultContact — tadbirkorga beriladigan murojaat kanali.
// CONTACT_TELEGRAM muhit o'zgaruvchisi bilan almashtiriladi.
const defaultContact = "https://t.me/declarant_pro"

// contact — murojaat kanali. Prompt ichida qo'lda yozilmaydi: kanal
// o'zgarsa, ko'rsatmaga tegmasdan sozlash mumkin bo'lsin.
func contact() string {
	if v := strings.TrimSpace(os.Getenv("CONTACT_TELEGRAM")); v != "" {
		return v
	}
	return defaultContact
}

// ParseMode — so'rovdagi qiymatni rejimga aylantiradi.
// Noma'lum yoki bo'sh bo'lsa — deklarant (asosiy auditoriya).
func ParseMode(s string) Mode {
	if Mode(strings.ToLower(strings.TrimSpace(s))) == ModeBusiness {
		return ModeBusiness
	}
	return ModeDeclarant
}

// promptIntro — rejimga qarab persona va uslub.
//
// DIQQAT: bu yerda FAQAT uslub bo'ladi. Har qanday fakt, stavka yoki
// ogohlantirish — sharedRules ichida, ikkala rejim uchun bitta manba.
// Aks holda ular vaqt o'tib bir-biridan ajralib ketardi.
var promptIntro = map[Mode]string{
	ModeDeclarant: `Sen "Deklarant AI" — O'zbekiston bojxona rasmiylashtiruvi bo'yicha yordamchisan.
Foydalanuvchi — TAJRIBALI DEKLARANT. U TIF TN, GTD grafalari va bojxona
rejimlarini biladi.

USLUB:
- Qisqa va zich. Jadval afzal, uzun izoh emas.
- To'lovlarni GTD kodlari bilan ber: 10, 12, 20, 21, 27, 29, 79.
- Kod va modda raqamini qator ichida ko'rsat, alohida abzats qilma.
- Atamalarni tushuntirma — u biladi.
- Ma'lumot yetishmasa, savol berish o'rniga VARIANTLARNI ber
  ("yangi bo'lsa 120 BRV, 3 yildan oshgan bo'lsa 210 BRV").`,

	ModeBusiness: `Sen "Deklarant AI" — O'zbekiston bojxona rasmiylashtiruvi bo'yicha yordamchisan.
Foydalanuvchi — TADBIRKOR. U tovar olib kelmoqchi, lekin bojxona
atamalarini bilmaydi va nimani so'rashni ham bilmaydi.

USLUB:
- Birinchi qatorda JAMI SUMMA. Tafsilot keyin.
- Atamani ishlatsang, darrov tushuntir: "ST-1 (kelib chiqish sertifikati —
  tovar qaysi davlatda ishlab chiqarilganini tasdiqlovchi hujjat)".
- U SO'RAMAGAN, lekin kerak bo'ladigan narsani ham ayt: litsenziya,
  sertifikat, ro'yxatdan o'tish.
- Javob oxirida "Keyingi qadamlar" ro'yxati ber. Oxirgi band sifatida
  murojaat kanalini qo'sh: {{CONTACT}}
  Uni oddiy tavsiya sifatida ber ("savol qolsa — shu yerda yozing"),
  rasmiy davlat manbasi sifatida EMAS. U customs.uz yoki bojxona
  brokeridan tasdiqlash maslahatini ALMASHTIRMAYDI — ikkalasi ham qolsin.
- GTD grafalari va to'lov kodlarini ko'rsatma — ular unga hech narsa
  demaydi. "Bojxona yig'imi" deb yoz, "10-kod" deb emas.
- Ma'lumot yetishmasa — ODDIY TIL bilan so'ra ("mashina necha yilligini
  bilasizmi?"), texnik atama bilan emas.`,
}

// sharedRules — FAKTLAR VA OGOHLANTIRISHLAR. Ikkala rejim uchun bitta
// manba: soddalashtirilgan rejimda ogohlantirish tushib qolmasligi kerak.
//
// Bu seansda tuzatilgan xatolarning aksariyati bir turdan edi — xavfli
// narsani jim tushirib qoldirish (aksiz "0%", imtiyoz e'tiborsiz, kelib
// chiqish hisobga olinmagan). "Soddalashtirilgan rejim" — o'sha xatoning
// qaytish yo'li. Shuning uchun bu blok bo'linmaydi.
const sharedRules = `
Agar foydalanuvchi rasm yuborsa — undagi tovar, miqdor, narx kabi ma'lumotlarni
diqqat bilan o'qib chiq va shularga asoslanib javob ber.

BOJXONA TO'LOVLARI (GTD kodlari bilan):
  10. Bojxona yig'imi — bojxona qiymatining dollardagi ekvivalentiga qarab,
      BRV ning karrasi (ПКМ 55, 31.01.2025). Shkala quyida berilgan —
      yig'imni DOIM hisoblab ber, "alohida belgilanadi" deb qoldirma.
  12. Ko'rik — ish vaqtida 25% BRV/soat, tashqarida 2×BRV/soat (to'liq soatga yaxlitlanadi).
  20. Bojxona boji     = bojxona qiymati × boj%
      ⚠️ KOMBINATSIYALANGAN STAVKA. Kod yonida "QAT'IY QISM: 1 kg uchun $0,5"
      yozilgan bo'lsa, stavka "10%, LEKIN 1 kg uchun 0,5 dollardan kam emas"
      degani. Ikkalasini ham hisobla va KATTASINI ol:
          foizli  = bojxona qiymati × boj%
          qat'iy  = miqdor × $qat'iy × USD kursi
          boj     = max(foizli, qat'iy)
      Miqdor (kg, dona, litr…) berilmagan bo'lsa — uni SO'RA, faqat foizli
      qismni hisoblab qo'yma: qat'iy qism ko'pincha kattaroq chiqadi.
  21. Qo'shimcha boj   = bojxona qiymati × qo'shimcha%
  27. Aksiz            = bojxona qiymati × aksiz%   (Soliq kodeksi 285-modda:
      advalor stavkada baza — bojxona qiymati, bojsiz; qat'iy stavkada esa
      natural miqdor — masalan "340 000 so'm / 1000 dona")
  29. QQS = (bojxona qiymati + boj + qo'shimcha boj + aksiz) × QQS%
      (Soliq kodeksi 254-modda. DIQQAT: bojxona yig'imi QQS bazasiga KIRMAYDI.)
  79. Utilizatsiya yig'imi — avtotransport va o'ziyurar mashinalar uchun
      (ПКМ 347, 02.06.2020). BRV karrasida, IKKI omilga bog'liq:
        • o'lchov — dvigatel hajmi (sm³), quvvat (kVt yoki ot kuchi)
          yoki to'la vazn (tonna); qaysi biri kerakligi toifaga qarab
        • texnikaning yoshi — 3 yildan ortiqmi yoki yo'q
      Ko'p toifada YANGI texnikaga stavka belgilanmagan (yig'im yo'q),
      eskisiga esa bor. Bu yig'im ko'pincha boj va QQS dan KATTA:
      masalan 2 litrli avtomobil uchun 120 BRV ≈ 49 mln so'm.
      Kod yonida "⚠️ UTILIZATSIYA YIG'IMI" yozilgan bo'lsa — kerakli
      o'lchov va yoshni SO'RA, keyin hisobla. Uni tushirib qoldirma.

  Bojxona qiymati = (faktura qiymati + transport xarajati) × valyuta kursi.

⚠️ VALYUTA KURSINI HECH QACHON TAXMIN QILMA.
  "<VALYUTA_KURSI>" bloki berilgan bo'lsa — faqat undagi raqamni ishlat.
  Blok bo'lmasa yoki kerakli valyuta yo'q bo'lsa — foydalanuvchidan SO'RA.
  "kurs taxminan 12 600 deb hisobladim" kabi javob BERMA: noto'g'ri kurs
  butun hisobni buzadi va buni foydalanuvchi sezmay qoladi.
  Kurs sanaga bog'liq: bojxona qiymati deklaratsiya ro'yxatga olingan
  kundagi Markaziy bank kursi bo'yicha hisoblanadi.

⚠️ O'YLAB TOPILGAN RAQAM BILAN HISOBLAMA.
  Faktura qiymati, miqdor, og'irlik, dvigatel hajmi, yoshi — birontasi
  aytilmagan bo'lsa, o'rniga "misol uchun", "faraz qilaylik", "taxminan"
  deb O'ZING raqam qo'yma.
  "Hozircha misol bilan tushuntiraman, aniq raqamlar bergach qayta
  hisoblab beraman" kabi javob BERMA.
  O'rniga: qaysi raqam yetishmayotganini AYT va SO'RA — qisqa, bitta
  jumlada. Formulani ko'rsatish mumkin, lekin RAQAMSIZ.
  NEGA: misoldagi summa javobda haqiqiy hisob bilan bir xil ko'rinadi.
  Foydalanuvchi uni ko'chirib olib, shartnoma yoki deklaratsiyada
  ishlatib yuborishi mumkin — u "misol" ekanini keyin eslamaydi.
  YAGONA ISTISNO: foydalanuvchining O'ZI misol so'rasa ("misol keltir",
  "taxminan qancha bo'ladi") — o'shanda hisobla, lekin har bir o'ylab
  topilgan raqamni javobda ochiq belgila.

QOIDALAR:
- O'zbek tilida, aniq va lo'nda javob ber (foydalanuvchi boshqa tilda so'rasa, o'sha tilda).
- Savolga qo'shib "TIF TN BAZASIDAN", "QONUNCHILIKDAN" va "HUJJAT TALABLARI"
  bloklari berilishi mumkin.
  Ular — bizning bazamizdan topilgan haqiqiy ma'lumot. Javobingni AVVALO o'shalarga
  asoslantir: ishlatgan kodingni va qonun moddasini ko'rsat
  (masalan: "Bojxona kodeksi, 346-modda").
- Parchada "Rasmiy manba:" havolasi berilgan bo'lsa, uni javob oxirida keltir —
  foydalanuvchi rasmiy matnni ochib tekshira olsin. Havolani O'ZING to'qib
  chiqarma: faqat blokda berilganini yoz, berilmagan bo'lsa havola bermay,
  hujjat nomi va sanasini ko'rsatish bilan cheklan.
- Bloklarda javob bo'lmasa — buni ochiq ayt ("bazada bu haqda ma'lumot topilmadi")
  va o'z bilimingga tayanayotganingni eslat. Blokdagi ma'lumotni o'ylab topilgan
  modda raqami bilan to'ldirma.

⚠️ KELIB CHIQISH DAVLATI BOJGA TA'SIR QILADI (Bojxona kodeksi 300-modda):
  Bazadagi boj stavkasi — TARIF stavkasi. Haqiqiy boj kelib chiqish
  davlatiga qarab o'zgaradi:
    • Erkin savdo davlatlari (MDH: Rossiya, Qozog'iston, Belarus,
      Qirg'iziston, Tojikiston, Ozarbayjon, Armaniston, Gruziya,
      Moldova, Ukraina) — boj UMUMAN QO'LLANILMAYDI.
      Lekin bu avtomatik emas: kelib chiqish sertifikati (ST-1) va
      bevosita yetkazib berish shart.
    • Eng qulaylik rejimi (Xitoy, AQSh, Turkiya, Yevropa va b.) —
      tarifdagi odatdagi stavka.
    • Rejim yo'q yoki kelib chiqishi ANIQLANMAGAN — stavka IKKI BARAVAR.
  Shuning uchun boj so'ralganda kelib chiqish davlatini SO'RA. Aytilmagan
  bo'lsa, odatdagi stavka bo'yicha hisobla, lekin javobda "kelib chiqish
  davlatiga qarab boj 0 ham, ikki barobar ham bo'lishi mumkin" deb ogohlantir.

⚠️ QQS VA IMTIYOZLAR HAQIDA:
  TIF TN bazasida QQS hamma kodda 12% deb turadi — bu UMUMIY stavka,
  kodga xos aniqlangan qiymat EMAS (manba bazada QQS maydoni hamma
  yozuvda bir xil). Ayrim tovarlar import qilinganda QQS dan ozod:
  Soliq kodeksi 246-modda, hamda alohida qarorlar (masalan ПКМ 352 —
  o'xshashi ishlab chiqarilmaydigan texnologik uskunalar: bojdan ham,
  QQS dan ham ozod). Kod yonida "⚠️ IMTIYOZ qoidasi bor" yozilgan
  bo'lsa — shartini tekshirmasdan 12% ni qo'llab hisoblama, avval
  imtiyoz shartini ayt va foydalanuvchidan holatini so'ra.
  Imtiyozlar SHARTLI: "yuridik shaxslar tomonidan", "ro'yxatga
  kiritilgan bo'lsa", "ishlab chiqarish uchun" kabi. Shartga tushmasa,
  odatdagi stavka qo'llanadi.

⚠️ HUJJAT TALABLARI HAQIDA:
  Blokda faqat TIF TN kodiga ANIQ bog'langan talablar bo'ladi. Bo'lim
  ko'rsatilmagan bo'lsa — "bazada yozuv yo'q", bu "kerak emas" DEGANI EMAS.
  Masalan "Litsenziya" bo'limi yo'q bo'lsa, "litsenziya talab qilinmaydi"
  deb qat'iy aytma; "bazada bu kod uchun litsenziya talabi yozilmagan,
  lekin tovar tavsifiga qarab boshqa hujjat kerak bo'lishi mumkin" de.
  Har bir talab yonidagi [asos: ...] qonunini javobda ko'rsat.

⚠️ AKSIZ HAQIDA ALOHIDA OGOHLANTIRISH:
  TIF TN bazasida aksiz stavkalari YO'Q. Kod yonida "aksiz: bu bazada yo'q"
  deb yozilgan bo'lsa, bu "aksiz to'lanmaydi" DEGANI EMAS — shunchaki bizda
  ma'lumot yo'q. Aroq, sigaret, benzin, avtomobil kabi tovarlar aksizli.
  Aksiz stavkalari Soliq kodeksining quyidagi moddalarida, TOVAR NOMI bo'yicha:
     289¹-modda — tamaki mahsulotlari
     289²-modda — alkogol mahsulotlari (import uchun alohida stavka!)
     289³-modda — neft mahsulotlari va boshqa aksizli tovarlar
  Aksiz haqida so'ralsa — shu moddalarga qara. Ular qonun korpusida bor.
  Hech qachon "bu tovarda aksiz yo'q" deb aytma, agar buni qonundan
  tasdiqlamagan bo'lsang.

  ⚠️ Jadvalda IKKI USTUN bo'lishi mumkin: "import qilinganda" va "ishlab
  chiqariladigan". Biz bojxona yordamchisimiz — DOIM "import qilinganda"
  ustunini ol. Masalan aroq uchun import stavkasi 60 000 so'm, mahalliy
  ishlab chiqarish uchun 48 000 so'm — bular boshqa-boshqa.

  ⚠️ Aksiz stavkalari ko'pincha QAT'IY summa (so'm/litr, so'm/1000 dona,
  so'm/tonna), foiz emas. Bunday holda jami summa = miqdor × stavka, va
  QQS bazasiga shu summa qo'shiladi. Miqdor berilmagan bo'lsa — so'ra.
- Stavkalar baza olingan sanaga tegishli va o'zgarib turadi — muhim qarorlar uchun
  customs.uz yoki bojxona brokeridan tasdiqlashni tavsiya et.
- Bilmagan narsangni to'qib chiqarma.

⚠️ GTD (YUK BOJXONA DEKLARATSIYASI) GRAFALARINI TO'LDIRISH:
  Foydalanuvchi deklaratsiya/GTD/grafalarni to'ldirishni so'rasa, savolga
  "GTD GRAFALARI" bloki qo'shiladi — u har grafni va uni KIM to'ldirishini
  ko'rsatadi. Qoidalar:
    • «avto» grafalar — YUQORIDAGI hisobdan to'ldir: 33-graf TIF TN kodini
      topilgan koddan, 47-grafni to'lovlar hisobidan (har to'lov kodi 10/20/29…
      alohida qator: turi, asosi, stavkasi, summasi), 45-grafni bojxona
      qiymatidan, 34-grafni kelib chiqish davlatidan.
    • «foydalanuvchi» grafalar — bu REKVIZIT (jo'natuvchi, oluvchi STIR,
      hujjat raqami, vazn, imzo). Ularni O'YLAB TOPMA — foydalanuvchidan
      SO'RA yoki "[foydalanuvchi kiritadi]" deb qoldir.
    • «ma'lumotnoma» grafalar — standart kod. Aniq kodni bilmasang, qonun
      korpusidagi to'ldirish yo'riqnomasiga (06.04.2016) tayan, taxmin qilma.
    • Natijani JADVAL qilib ber: | Graf | Nomi | Qiymat |. Miqdor yoki narx
      yetishmasa — o'sha grafni bo'sh qoldirib, nima yetishmayotganini ayt.
    • ⚠️ Bu YORDAM, tayyor deklaratsiya emas: oxirida "grafalarni rasmiy
      yo'riqnoma va bojxona brokeri bilan tekshiring" deb ogohlantir.`

// Service — chat xizmati.
type Service struct {
	client *llm.Client
	codes  *hscode.Store
	laws   *laws.Store   // ixtiyoriy; nil bo'lsa qonun konteksti qo'shilmaydi
	docs   *docs.Store   // ixtiyoriy; hujjat talablari
	rates  *rates.Client // ixtiyoriy; valyuta kurslari
}

// New — LLM klienti va ma'lumot bazalari asosida xizmat yaratadi.
// lawStore, docStore va rateClient nil bo'lishi mumkin.
func New(client *llm.Client, codes *hscode.Store, lawStore *laws.Store,
	docStore *docs.Store, rateClient *rates.Client) *Service {
	return &Service{client: client, codes: codes, laws: lawStore, docs: docStore, rates: rateClient}
}

// ratesBlock — joriy valyuta kurslari.
//
// NEGA KERAK: kurssiz model uni TAXMIN qilardi ("kurs ~12 600 deb
// hisobladim") — bu jim va xavfli xato. Endi haqiqiy kurs beriladi.
//
// Xizmat ishlamasa blok qo'shilmaydi va model ko'rsatmaga ko'ra
// foydalanuvchidan kursni SO'RAYDI — taxmin qilmaydi.
func (s *Service) ratesBlock(ctx context.Context) string {
	if s.rates == nil {
		return ""
	}
	all, err := s.rates.On(ctx, time.Time{})
	if err != nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("<VALYUTA_KURSI>\n")
	first := true
	for _, ccy := range []string{"USD", "EUR", "RUB", "CNY", "KZT", "TRY"} {
		r, ok := all[ccy]
		if !ok {
			continue
		}
		if first {
			fmt.Fprintf(&b, "Markaziy bank kursi, %s holatiga:\n", r.Date)
			first = false
		}
		fmt.Fprintf(&b, "  1 %s = %.2f so'm\n", r.Ccy, r.Value)
	}
	if first {
		return "" // hech qaysi valyuta topilmadi
	}
	b.WriteString("Bojxona qiymati DEKLARATSIYA RO'YXATGA OLINGAN kundagi kurs\n")
	b.WriteString("bo'yicha hisoblanadi. Boshqa sanadagi kurs kerak bo'lsa — ayt.\n")
	b.WriteString("</VALYUTA_KURSI>")
	return b.String()
}

// Available — AI mavjudligini bildiradi.
func (s *Service) Available() bool { return s.client.Available() }

// systemPrompt — barqaror tizim ko'rsatmasi.
// Kodlar bu yerga QO'SHILMAYDI — ular so'rovga qarab kontekstga qo'shiladi,
// shu tufayli prompt keshi buzilmaydi.
func (s *Service) systemPrompt(mode Mode) string {
	base := strings.ReplaceAll(promptIntro[mode], "{{CONTACT}}", contact()) + sharedRules
	if s.codes == nil {
		return base
	}
	m := s.codes.Meta()
	var b strings.Builder
	b.WriteString(base)

	// Yig'im shkalasi duty paketidan olinadi — bu yerda qo'lda yozilmaydi,
	// aks holda stavka o'zgarganda kalkulyator bilan prompt ajralib qolardi.
	fmt.Fprintf(&b, "\n\nBOJXONA YIG'IMI SHKALASI (ПКМ 55, %s holatiga):\n%s",
		m.RatesAsOf, duty.FeeScaleText(time.Now()))

	fmt.Fprintf(&b, `

QO'LINGDAGI BAZA:
  Nomenklatura : %s (%d ta kod)
  Huquqiy asos : %s
  Stavkalar    : %s holatiga`,
		m.Nomenclature, m.TotalCodes, m.LegalBasis, m.RatesAsOf)

	if s.laws != nil {
		lm := s.laws.Meta()
		fmt.Fprintf(&b, `
  Qonunchilik  : %d ta hujjatdan %d parcha (Bojxona kodeksi to'liq;
                 Soliq, Ma'muriy va Jinoyat kodekslaridan bojxonaga oid moddalar)`,
			lm.Docs, lm.Chunks)
	}
	if s.docs != nil {
		dm := s.docs.Meta()
		fmt.Fprintf(&b, `
  Hujjat talabi: %d ta qoida — kod oralig'i bo'yicha litsenziya, sertifikat,
                 imtiyoz va boshqa shartlar (%s holatiga)`,
			dm.Rules, dm.RulesAsOf)
	}
	b.WriteString("\nFoydalanuvchi \"nimalarni bilasan\" deb so'rasa, shu ma'lumotni ayt.")
	return b.String()
}

// Reply — suhbat tarixiga javob qaytaradi.
func (s *Service) Reply(ctx context.Context, mode Mode, history []llm.Message) (string, error) {
	msgs, retrieved := s.withRetrieval(ctx, trimHistory(history))
	return s.client.CompleteWith(ctx, s.modelOf(history, retrieved), s.systemPrompt(mode), msgs)
}

// ReplyStream — javobni bo'lak-bo'lak qaytaradi (SSE uchun).
//
// Retrieval Reply bilan bir xil — farqi faqat javobning yetkazilishida,
// shuning uchun ikkala yo'l ham bir xil kontekst, ko'rsatma VA MODEL bilan
// ishlaydi. Model tanlash ikkalasida bir joydan (modelOf) olinadi: ilgari
// Reply doim asosiy modelga ketardi, ya'ni oqimsiz javob qimmatroq edi.
func (s *Service) ReplyStream(ctx context.Context, mode Mode, history []llm.Message, onChunk llm.StreamFunc) error {
	msgs, retrieved := s.withRetrieval(ctx, trimHistory(history))
	return s.client.Stream(ctx, s.modelOf(history, retrieved), s.systemPrompt(mode), msgs, onChunk)
}

// modelOf — suhbat tarixiga qarab darajani tanlaydi.
//
// ⚠️ BUTUN TARIX qaraladi, faqat oxirgi xabar emas.
//
// NEGA: hisob-kitob suhbati qisqa savol bilan davom etadi —
//
//	"9405 42 003 9, Xitoydan 1000 kg, faktura $2000, bojni hisobla"
//	"unda 500 kg bo'lsa-chi?"
//
// Ikkinchi xabarda na hisob so'zi, na kod bor. Faqat o'shanga qarasak,
// u arzon darajaga tushardi va foydalanuvchi javob boshqa modeldan
// kelganini bilmasdi. Suhbatda bir marta hisob-kitob boshlangan bo'lsa,
// u OXIRIGACHA Full da qoladi.
func (s *Service) modelOf(history []llm.Message, retrieved bool) llm.Model {
	var last string
	calc := false
	for _, m := range history {
		if m.Role != "user" {
			continue
		}
		// Rasm — doim Full (llm.pickModel ham buni ta'minlaydi, lekin
		// qaror shu yerda ham ko'rinib tursin).
		if len(m.Images) > 0 {
			return llm.Full
		}
		if hasCalcIntent(m.Content) {
			calc = true
		}
		last = m.Content
	}
	if calc {
		return llm.Full
	}
	return modelFor(last, retrieved)
}

// withRetrieval — oxirgi foydalanuvchi savoliga mos TIF TN kodlari va qonun
// parchalarini topib, uning matniga qo'shib qo'yadi.
// Asl tarix o'zgartirilmaydi (nusxa qaytariladi).
//
// Ikkinchi qiymat — MA'NOLI topilma bo'ldimi (kod, qonun, hujjat, imtiyoz).
// Valyuta kursi bunga KIRMAYDI: u deyarli har so'rovga qo'shiladi va uni
// sanash model tanlashni buzardi (pastda batafsil).
func (s *Service) withRetrieval(ctx context.Context, history []llm.Message) ([]llm.Message, bool) {
	if len(history) == 0 {
		return history, false
	}
	last := history[len(history)-1]
	if last.Role != "user" || strings.TrimSpace(last.Content) == "" {
		return history, false // rasm-only xabar yoki bo'sh — qidiradigan narsa yo'q
	}
	// BO'SH GAPGA KONTEKST IZLANMAYDI.
	//
	// Qidiruv "salom" ga ham parcha topadi (so'z ichidan: "salomatlik"),
	// natijada salomlashishga 13 878 belgi qonun matni qo'shilardi —
	// o'lchangan narxi $0,039, ya'ni haqiqiy huquqiy savoldan qimmat.
	//
	// Bir marta bu ball chegarasi bilan hal qilishga urinilgan va u
	// haqiqiy savollarni ham kesib yuborgan (laws.go dagi izohga qarang).
	// To'g'ri joyi shu yer: savolning O'ZI bo'sh gapmi — buni bilish
	// uchun ball kerak emas.
	if isSmallTalk(last.Content) && len(last.Images) == 0 {
		return history, false
	}

	var blocks []string
	// Aniq kodga bog'langan imtiyoz topildimi — umumiy ro'yxat kerakmi,
	// shuni hal qiladi.
	codeExemptionFound := false
	if s.codes != nil {
		matches := s.codes.Search(last.Content, topCodes)
		// ANIQ KOD birinchi turishi shart.
		//
		// Jonli sinovda foydalanuvchi "9405 42 003 9 kodi bo'yicha…" deb
		// so'radi, lekin kod jumla ichida bo'lgani uchun qidiruv uni
		// top-8 ga ham chiqarmadi — model esa yaqin kodni (…003 2) olib,
		// javobni BOSHQA kod bo'yicha berdi. Stavkalar bu safar bir xil
		// edi, lekin boshqa kodda javob butunlay noto'g'ri bo'lardi.
		matches = promoteExplicit(s.codes, last.Content, matches)
		if m := matches; len(m) > 0 {
			// Imtiyoz belgisi aynan STAVKA yonida ko'rsatiladi. Uni faqat
			// alohida blokda bersak, model "QQS 12%" ni ko'rib hisoblab
			// yuborishi mumkin — imtiyoz esa pastda qolib ketardi.
			// Faqat "imtiyoz bor" emas, SHARTI bilan birga.
			//
			// Sinovda model "8701 24 uchun imtiyoz bor, u elektr
			// tyagachlarga tegishli, sizniki dizel — imtiyoz yo'q" dedi.
			// Aslida imtiyoz hamma kodga tegishli va sharti butunlay
			// boshqa ("Yevro-5 va undan yuqori ekologik toifa"). Sabab:
			// kod yonida faqat belgi turardi, shart esa boshqa blokda —
			// modelga ikkovini bog'lash kerak bo'ldi va u xato qildi.
			exempt := map[string][]docs.Requirement{}
			if s.docs != nil {
				for _, mt := range m {
					var ex []docs.Requirement
					for _, req := range s.docs.For(mt.Code.Code, docs.Import) {
						if req.Category == "imtiyoz" {
							ex = append(ex, req)
						}
					}
					if len(ex) > 0 {
						exempt[mt.Code.Code] = ex
						codeExemptionFound = true
					}
				}
			}
			blocks = append(blocks, formatMatches(s.codes.Meta(), m, exempt))
			// Hujjat talablari eng mos kodlar bo'yicha qo'shiladi. Faqat
			// bir nechtasi — talab matnlari uzun, sakkizta kodniki
			// kontekstni bosib ketardi.
			if s.docs != nil {
				if d := formatDocs(s.docs, m); d != "" {
					blocks = append(blocks, d)
				}
			}
		}
	}
	if s.laws != nil {
		if m := s.laws.Search(last.Content, topLaws); len(m) > 0 {
			// Qonun parchalari eng yirik qism — qolgan byudjet ularga
			// beriladi. Kod va hujjat bloklari aniqroq bo'lgani uchun
			// ular birinchi navbatda joy oladi.
			used := 0
			for _, b := range blocks {
				used += len(b)
			}
			if b := formatLaws(m, maxContext-used); b != "" {
				blocks = append(blocks, b)
			}
		}
	}
	// Imtiyoz dasturlarining UMUMIY ro'yxati — faqat so'rov shu haqda
	// bo'lsa VA aniq kodga bog'langan imtiyoz topilmagan bo'lsa.
	//
	// NEGA SHART: sinovda "Volvo FH egarli tyagach, imtiyoz bormi?"
	// so'roviga model ПКМ 352 (texnologik uskunalar) ni keltirib
	// "tushmaydi" dedi — holbuki shu kod uchun aniq imtiyoz bor edi
	// (ПП 251, yuk tashish texnikasi). Sabab: umumiy ro'yxat qamrov
	// bo'yicha tartiblangan, ПКМ 352 birinchi turadi va aniq javobni
	// bosib qo'yadi. Aniq kod topilganda umumiy ro'yxat shovqin.
	if !codeExemptionFound {
		if pb := s.programsBlock(last.Content); pb != "" {
			blocks = append(blocks, pb)
		}
	}

	// GTD GRAFALAR SKELETI — faqat foydalanuvchi deklaratsiya to'ldirishni
	// so'raganda. Har so'rovda emas: skelet ~1,5 KB va aksariyat savol
	// GTD haqida emas.
	//
	// Skelet yuqoridagi kod/boj bloklari BILAN birga ishlaydi: model
	// 33/47/45-grafalarni o'sha hisobdan, qolganini yo'riqnoma (qonun
	// bloki) va foydalanuvchi ma'lumoti bilan to'ldiradi.
	if hasGTDIntent(last.Content) {
		blocks = append(blocks, gtd.PromptBlock())
	}

	// MODEL TANLASH uchun belgi — kurs blokidan OLDIN olinadi.
	//
	// ⚠️ NEGA AYNAN SHU YERDA: kurs bloki deyarli DOIM qo'shiladi (u
	// mustaqil ravishda foydali). Uni ham "bazadan topildi" deb hisoblasak,
	// `retrieved` hech qachon false bo'lmaydi — o'lchandi: "salom",
	// "rahmat", "nima qila olasan" ham Opus ga ketardi, ya'ni arzon
	// daraja ishlab chiqarishda O'LIK KOD edi.
	//
	// Bu yerda faqat MA'NOLI topilma sanaladi: kod, qonun moddasi,
	// hujjat talabi yoki imtiyoz dasturi.
	substantive := len(blocks) > 0

	// Valyuta kursi — bloklar bo'lmasa ham foydali (masalan "bugun kurs qancha").
	if rb := s.ratesBlock(ctx); rb != "" {
		blocks = append(blocks, rb)
	}
	if len(blocks) == 0 {
		return history, false
	}

	// SURAT QIDIRUVGA KO'RINMAYDI — buni modelga AYTAMIZ.
	//
	// Qidiruv faqat MATNDAN ishlaydi. Foydalanuvchi invoys surati yuborib
	// "tovarni o'qi va bojni hisobla" desa, matnda tovar nomi umuman
	// bo'lmaydi va qidiruv butunlay begona kodlarni qaytaradi.
	//
	// Jonli sinovda aynan shunday bo'ldi: konditsioner invoysiga bazadan
	// yog'-moy guruhi (1515–2306) keldi. Model o'sha safar o'zi sezib
	// "bu tovarga aloqasi yo'q" dedi, lekin bunga tayanib bo'lmaydi —
	// kontekstdagi stavkani ishonchli deb olsa, javob butunlay xato
	// bo'lardi. Ogohlantirish ro'yxatning BOSHIDA turadi.
	if len(last.Images) > 0 {
		blocks = append([]string{
			"⚠️ DIQQAT: quyidagi baza ma'lumotlari FAQAT SAVOL MATNI bo'yicha " +
				"topilgan — QIDIRUV SURATNI KO'RMAYDI. Suratdagi tovar bilan " +
				"ular mos kelmasligi mumkin. Suratdan o'qigan tovaring quyidagi " +
				"kodlarga to'g'ri kelmasa, ULARNING STAVKASINI ISHLATMA: " +
				"kodni o'zing ayt va foydalanuvchidan uni alohida so'rashini " +
				"yoki kodni yozib qayta so'rashini iltimos qil.",
		}, blocks...)
	}

	out := make([]llm.Message, len(history))
	copy(out, history)
	out[len(out)-1].Content = last.Content + "\n\n" + strings.Join(blocks, "\n\n")
	return out, substantive
}

// calcWords — HISOB-KITOB niyatini bildiruvchi so'zlar.
//
// Bu so'zlardan biri uchrasa, savol Claude ga ketadi. Sabab iqtisodiy
// emas, xavf bilan bog'liq: noto'g'ri hisoblangan boj deklarantga real
// pul va jarima turadi, "bu kod nimani anglatadi" degan savoldagi
// noaniqlik esa shunchaki qayta so'rashga olib keladi.
//
// Ro'yxat ATAYLAB keng: shubha bo'lsa qimmatroq modelga yuborish
// arzonroq modelda xato qilishdan xavfsizroq.
var calcWords = map[string]bool{
	"boj": true, "bojni": true, "bojini": true, "bojxona": true,
	"qqs": true, "nds": true, "aksiz": true, "aktsiz": true,
	"utilizatsiya": true, "utilizatsion": true, "yig'im": true, "yigim": true,
	"to'lov": true, "tolov": true, "to'lovlar": true, "to'lovni": true,
	"hisobla": true, "hisoblab": true, "hisob": true, "hisoblang": true,
	"qancha": true, "narx": true, "narxi": true, "summa": true,
	"stavka": true, "stavkasi": true, "kurs": true, "soliq": true,
	"chiqadi": true, "tushadi": true, "brv": true,
}

// modelFor — savolga qaysi daraja mos.
//
// SIYOSAT: xato narxi yuqori bo'lgan joy Claude da qoladi, qolgani arzon
// provayderga (GLM-5.2) ketadi.
//
//	rasm         → Full  (GLM-5.2 rasmni umuman qabul qilmaydi)
//	hisob-kitob  → Full  (boj/QQS/aksiz/utilizatsiya — jarima xavfi bor)
//	qolgani      → Cheap (hujjat ro'yxati, kod izohi, salomlashish)
//
// Rasm tekshiruvi llm.pickModel da TAKRORLANADI — bu ataylab: bu yerda
// yo'naltirish siyosati, u yerda esa provayder imkoniyati tekshiriladi.
// Shu qatlam xato qilsa ham rasm matn-only modelga tushmaydi.
// ⚠️ SUKUT — Full. Arzonlashtirish faqat AYTIB o'tilgan ikki holatda.
//
// Ilgari qoida teskari edi: "bazadan biror narsa topildimi — demak
// model faqat faktni qayta ifodalaydi, o'rta daraja yetadi". O'lchov
// buni rad etdi: 25 ta haqiqiy deklarant savolidan 19 tasi (76%)
// Opus dan Sonnet ga tushib ketgan va foydalanuvchi javob sifati
// pasayganini sezgan.
//
// Xato faraz shunda edi: "kontekst bor" ≠ "o'ylash kerak emas".
// Aksincha, eng ko'p mulohaza talab qiladigan savollar aynan
// kontekstli bo'ladi:
//
//	tasnif        — "elektr skuter qaysi kodga kiradi" (bu HUKM,
//	                kontekstdan o'qish emas)
//	kelib chiqish — 300-moddani tovar, davlat va sertifikat bilan
//	                bog'lash kerak
//	imtiyoz       — shart bajarilgan-bajarilmaganini tekshirish
//
// Shuning uchun endi teskari: hamma narsa Full, arzon darajaga
// tushish uchun ASOS kerak.
func modelFor(text string, retrieved bool) llm.Model {
	// Bo'sh gap: "salom", "rahmat". Faktik da'vo yo'q, xato narxi nol.
	//
	// ⚠️ `retrieved` SHART EMAS. Ilgari "kontekst topilmagan bo'lsa"
	// deb qo'yilgan edi va u ishlamadi: qidiruv "salom" ga ham parcha
	// topadi ("salom" ⊂ "salomatlik"), ya'ni retrieved doim true bo'lib,
	// salomlashish eng qimmat modelga ketardi. Bo'sh gapga topilgan
	// moslik — ta'rifi bo'yicha shovqin.
	if isSmallTalk(text) {
		return llm.Cheap
	}
	// Sof MA'LUMOT O'QISH: "bu kod nimani anglatadi", "qaysi modda".
	// Javob to'liq kontekstdagi matnga tayanadi.
	//
	// Ro'yxat ATAYLAB tor va oq ro'yxat: shubha bo'lsa Full qoladi.
	// Kengaytirishdan oldin o'lchash kerak.
	if retrieved && isLookup(text) && !hasCalcIntent(text) {
		return llm.Mid
	}
	return llm.Full
}

// smallTalkWords — bazaga ham, bojxonaga ham aloqasi yo'q so'zlar.
var smallTalkWords = map[string]bool{
	"salom": true, "assalom": true, "assalomu": true, "alaykum": true,
	"rahmat": true, "tashakkur": true, "xayr": true, "qalaysan": true,
	"qalaysiz": true, "yaxshimisiz": true, "salomatmisiz": true,
	"hello": true, "hi": true, "privet": true, "spasibo": true,
	"ok": true, "okay": true, "zo'r": true, "zur": true, "bo'ldi": true,
}

// isSmallTalk — savol butunlay bo'sh gapdanmi.
//
// HAMMA so'z ro'yxatda bo'lishi shart: "salom, muzlatgich bojini
// hisobla" bo'sh gap emas.
func isSmallTalk(text string) bool {
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) == 0 || len(fields) > 4 {
		return false
	}
	for _, f := range fields {
		if !smallTalkWords[trimWord(f)] {
			return false
		}
	}
	return true
}

// lookupPhrases — sof ma'lumot o'qish belgilari.
//
// ⚠️ Bu yerga TASNIF, IMTIYOZ va KELIB CHIQISH iboralari qo'shilmasin
// — ular hukm talab qiladi va Full da qolishi shart (yuqoridagi izoh).
var lookupPhrases = []string{
	"nimani anglatadi", "nima degani", "nimani bildiradi",
	"tavsifi", "izohla", "tushuntir",
	"qaysi modda", "nechanchi modda", "qaysi qonun",
}

func isLookup(text string) bool {
	low := strings.ToLower(text)
	for _, p := range lookupPhrases {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// trimWord — so'zdan tinish belgilarini oladi.
func trimWord(s string) string {
	return strings.Trim(s, ".,!?()[]:;\"«»'`")
}

// hasCalcIntent — savolda hisob-kitob niyati bormi.
func hasCalcIntent(text string) bool {
	for _, f := range strings.Fields(strings.ToLower(text)) {
		f = strings.Trim(f, ".,!?()[]:;\"«»")
		f = strings.NewReplacer("ʻ", "'", "ʼ", "'", "'", "'", "'", "'").Replace(f)
		if calcWords[f] {
			return true
		}
	}
	return false
}

// hasGTDIntent — foydalanuvchi deklaratsiya (GTD) to'ldirishni so'rayaptimi.
//
// Ibora bo'yicha tekshiriladi, chunki "grafa" yoki "deklaratsiya" so'zi
// yolg'iz o'zi ham ("deklaratsiya nima") GTD to'ldirish niyati emas.
// Aniqroq iboralar: "gtd", "grafalarni to'ldir", "deklaratsiya to'ldir".
func hasGTDIntent(text string) bool {
	low := strings.ToLower(text)
	for _, p := range gtdPhrases {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// gtdPhrases — GTD to'ldirish niyatini bildiruvchi iboralar.
var gtdPhrases = []string{
	"gtd", "г тд", "гтд", "byud", "бюд",
	"grafa", "grafalar", "grafani", "grafalarni",
	"deklaratsiya to'ldir", "deklaratsiyani to'ldir", "deklaratsiya to'ldi",
	"yuk deklaratsiya", "bojxona deklaratsiya",
}

// formatLaws — topilgan qonun parchalarini kontekst bloki sifatida shakllantiradi.
// budget — blokka ajratilgan hajm (belgi). Parchalar shu hajm to'lgunicha
// qo'shiladi; sig'maganlari tushib qoladi. Qidiruv ularni mos tartibda
// bergani uchun avval eng mosi joy oladi.
func formatLaws(matches []laws.Match, budget int) string {
	var b strings.Builder
	b.WriteString("<QONUNCHILIKDAN>\n")

	added, skipped := 0, 0
	for _, m := range matches {
		c := m.Chunk
		var one strings.Builder
		fmt.Fprintf(&one, "\n— Manba: %s", c.Name)
		if c.Date != "" {
			fmt.Fprintf(&one, " (%s)", c.Date)
		}
		if c.Since != "" {
			fmt.Fprintf(&one, ", %s dan amalda", c.Since)
		}
		// lex.uz havolasi — foydalanuvchi rasmiy matnni ochib tekshira olsin.
		if c.Lex != "" {
			fmt.Fprintf(&one, "\n  Rasmiy manba: %s", c.Lex)
		}
		fmt.Fprintf(&one, "\n  %s\n%s\n", c.Title, c.Text)

		// Birinchi parcha byudjetdan oshsa ham qo'shamiz: bo'sh blok
		// qaytargandan ko'ra bitta parcha berilgani yaxshiroq.
		if added > 0 && b.Len()+one.Len() > budget {
			skipped++
			continue
		}
		b.WriteString(one.String())
		added++
	}
	// Tushib qolgani bo'lsa — buni yashirmaymiz, model "hammasi shu" deb
	// o'ylamasligi kerak.
	if skipped > 0 {
		fmt.Fprintf(&b, "\n(yana %d ta mos parcha bor, hajm chegarasi tufayli kiritilmadi)\n", skipped)
	}
	b.WriteString("</QONUNCHILIKDAN>")
	return b.String()
}

// formatMatches — topilgan kodlarni kontekst bloki sifatida shakllantiradi.
func formatMatches(m hscode.Meta, matches []hscode.Match, exempt map[string][]docs.Requirement) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<TIF_TN_BAZASIDAN>\nSavolga mos kodlar (%s, stavkalar %s holatiga):\n\n",
		m.Nomenclature, m.RatesAsOf)
	for _, mt := range matches {
		c := mt.Code
		fmt.Fprintf(&b, "%s — %s\n", formatCode(c.Code), c.PathUZ)
		fmt.Fprintf(&b, "   boj %g%% | QQS %g%%", c.ImportDuty, c.VAT)
		// KOMBINATSIYALANGAN STAVKA — 13 142 koddan 1 555 tasida.
		//
		// Buni tushirib qoldirish JIDDIY xato: jonli sinovda model
		// 9405 42 003 9 uchun faqat 10% ni hisoblab, bojni 2,56 mln
		// so'm dedi. Aslida 1 000 kg × $0,5 qat'iy qismi kattaroq —
		// 5,96 mln. Ya'ni javob ikki baravardan ko'proq kam edi.
		if c.ImportDutySpecific != nil {
			fmt.Fprintf(&b, " | ⚠️ QAT'IY QISM: 1 %s uchun $%g",
				c.ImportDutySpecificUnit, *c.ImportDutySpecific)
		}
		// Aksiz nil bo'lsa — bu "yo'q" emas, "noma'lum". Buni ochiq yozamiz,
		// aks holda model "aksiz yo'q" degan xulosaga kelishi mumkin.
		if c.Excise != nil {
			fmt.Fprintf(&b, " | aksiz %g%%", *c.Excise)
		} else {
			b.WriteString(" | aksiz: bu bazada yo'q")
		}
		if c.ExportDuty > 0 {
			fmt.Fprintf(&b, " | eksport boji %g%%", c.ExportDuty)
		}
		if c.Unit != "" {
			fmt.Fprintf(&b, " | qo'shimcha o'lchov: %s", c.Unit)
		}
		b.WriteString("\n")
		// Utilizatsiya yig'imi (79) — avtotransport va o'ziyurar
		// mashinalar uchun. Bu yig'im ko'pincha boj va QQS dan KATTA
		// bo'ladi (masalan 2 litrli avtomobil uchun 120 BRV = 49 mln),
		// shuning uchun uni tushirib qoldirish jiddiy xato bo'lardi.
		if m := duty.UtilFeeMeasure(c.Code); m != "" {
			fmt.Fprintf(&b, "   ⚠️ bu kodga UTILIZATSIYA YIG'IMI (79) qo'llanadi —\n"+
				"      hisoblash uchun %s va texnikaning yoshi kerak (ПКМ 347)\n", m)
		} else if duty.HasUtilFee(c.Code) {
			b.WriteString("   ⚠️ bu kodga UTILIZATSIYA YIG'IMI (79) qo'llanadi (ПКМ 347)\n")
		}
		// Imtiyoz SHARTI bilan birga — kod yonida. Faqat "imtiyoz bor"
		// deb qo'ysak, model uni boshqa blokdagi ro'yxat bilan bog'lashga
		// urinib xato qiladi.
		for _, e := range exempt[c.Code] {
			fmt.Fprintf(&b, "   ⚠️ IMTIYOZ (%s dan ozod): %s",
				strings.Join(e.Free, ", "), strings.Join(strings.Fields(e.Text), " "))
			if e.Law != "" {
				fmt.Fprintf(&b, " [asos: %s]", e.Law)
			}
			b.WriteString("\n      Shart bajarilsa stavka yuqoridagidek BO'LMAYDI.\n" +
				"      Shartni tekshirmasdan hisoblama va imtiyozni va'da qilma.\n")
		}
	}
	// Ro'yxat oxirida yana bir bor eslatamiz. Sinovda model har bir kod
	// yonidagi "aksiz: bu bazada yo'q" belgisini ko'ra turib ham "traktor
	// aksizli tovar emas" degan xulosaga kelgan — ya'ni yo'qlikni "0%" deb
	// o'qigan. Bu yerda bo'shliq aniq aytiladi.
	if !anyExcise(matches) {
		b.WriteString("\n⚠️ Yuqoridagi kodlarning HECH BIRIDA aksiz ma'lumoti yo'q.\n" +
			"Bu \"aksiz to'lanmaydi\" degani EMAS. Aksiz haqida xulosa faqat\n" +
			"<QONUNCHILIKDAN> blokidagi Soliq kodeksi 289¹–289³-moddalariga\n" +
			"tayanib chiqariladi. Ular blokda bo'lmasa — \"tasdiqlay olmayman,\n" +
			"289¹–289³-moddalarni tekshiring\" deb ayt, o'zingdan hukm qilma.\n")
	}
	b.WriteString("</TIF_TN_BAZASIDAN>")
	return b.String()
}

// docCodes — hujjat talablari qidiriladigan kodlar soni.
// Talab matnlari uzun, shuning uchun faqat eng mos kodlar olinadi.
const docCodes = 3

// docSections — bo'limlarning o'zbekcha sarlavhalari va tartibi.
var docSections = []struct{ key, title string }{
	{"litsenziya", "Litsenziya"},
	{"sertifikat", "Sertifikat va boshqa tasdiqnomalar"},
	{"imtiyoz", "Imtiyozlar"},
	{"boshqa", "Boshqa talablar"},
	{"tavsif", "Tovar tavsifida ko'rsatilishi shart"},
}

// formatDocs — topilgan kodlar bo'yicha hujjat talablarini blok qiladi.
func formatDocs(store *docs.Store, matches []hscode.Match) string {
	n := docCodes
	if len(matches) < n {
		n = len(matches)
	}

	// Bir nechta kod bir xil oraliqqa tushishi mumkin — takrorlanmasin.
	seen := map[string]bool{}
	byCat := map[string][]docs.Requirement{}
	for _, mt := range matches[:n] {
		for _, r := range store.For(mt.Code.Code, docs.Import) {
			key := r.Category + "|" + r.Law + "|" + r.Text
			if seen[key] {
				continue
			}
			seen[key] = true
			byCat[r.Category] = append(byCat[r.Category], r)
		}
	}
	if len(seen) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("<HUJJAT_TALABLARI>\n")
	fmt.Fprintf(&b, "Eng mos %d ta kod bo'yicha, import rejimi uchun:\n", n)
	for _, sec := range docSections {
		rs := byCat[sec.key]
		if len(rs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n%s:\n", sec.title)
		for _, r := range rs {
			b.WriteString("  — ")
			if r.Text != "" {
				b.WriteString(strings.Join(strings.Fields(r.Text), " "))
			}
			if r.Law != "" {
				fmt.Fprintf(&b, " [asos: %s]", r.Law)
			}
			if len(r.Free) > 0 {
				fmt.Fprintf(&b, " [ozod: %s]", strings.Join(r.Free, ", "))
			}
			b.WriteString("\n")
		}
	}
	// Bo'sh bo'limni "talab yo'q" deb aytish uchun ochiq ko'rsatma kerak:
	// aks holda model ma'lumot yo'qligini "talab yo'q" deb o'qishi mumkin.
	b.WriteString("\nDIQQAT: yuqorida ko'rsatilmagan bo'lim bo'yicha bazada yozuv yo'q.\n")
	b.WriteString("Bu \"talab qilinmaydi\" DEGANI EMAS — kodga aniq bog'lanmagan\n")
	b.WriteString("hujjatlar bu ro'yxatga tushmaydi, lekin rasmiylashtiruvda kerak bo'lishi mumkin.\n")
	b.WriteString("</HUJJAT_TALABLARI>")
	return b.String()
}

// anyExcise — topilgan kodlar ichida aksiz ma'lumoti bori bormi.
func anyExcise(matches []hscode.Match) bool {
	for _, m := range matches {
		if m.Code.Excise != nil {
			return true
		}
	}
	return false
}

// formatCode — "8701211019" → "8701 21 101 9".
func formatCode(c string) string {
	if len(c) != 10 {
		return c
	}
	return c[0:4] + " " + c[4:6] + " " + c[6:9] + " " + c[9:]
}

// exemptionWords — savol imtiyoz haqida ekanini bildiruvchi so'zlar.
var exemptionWords = []string{
	"imtiyoz", "ozod", "льгот", "освобожд", "nol stavka", "нулевая",
	"bojsiz", "qqssiz", "chegirma",
}

// programsBlock — imtiyoz DASTURLARI ro'yxati.
//
// NEGA KERAK: imtiyoz ma'lumoti faqat aniq kod topilganda ko'rinardi.
// "Qanday imtiyozlar bor?" degan umumiy savolga javob yo'q edi —
// holbuki tadbirkor uchun bu eng qimmatli savollardan biri: imtiyoz
// bojni ham, QQS ni ham butunlay olib tashlashi mumkin.
//
// Ro'yxat faqat SO'ROV imtiyoz haqida bo'lganda qo'shiladi: u uzun va
// har savolga qo'shilsa kontekstni behuda to'ldirardi.
func (s *Service) programsBlock(question string) string {
	if s.docs == nil {
		return ""
	}
	q := strings.ToLower(question)
	hit := false
	for _, w := range exemptionWords {
		if strings.Contains(q, w) {
			hit = true
			break
		}
	}
	if !hit {
		return ""
	}

	progs := s.docs.Programs()
	if len(progs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("<IMTIYOZ_DASTURLARI>\n")
	fmt.Fprintf(&b, "Bazada %d ta imtiyoz dasturi bor. Eng ko'p tovarni qamraganlari:\n\n", len(progs))

	const top = 12
	n := min(top, len(progs))
	for _, p := range progs[:n] {
		fmt.Fprintf(&b, "— %s dan ozod", strings.Join(p.Free, ", "))
		if len(p.Laws) > 0 {
			fmt.Fprintf(&b, " [asos: %s]", strings.Join(p.Laws, "; "))
		}
		fmt.Fprintf(&b, "\n  %s\n", strings.Join(strings.Fields(p.Text), " "))
	}
	if len(progs) > n {
		fmt.Fprintf(&b, "\n(yana %d ta dastur bor — aniq tovar kodini aytsa, o'shanga tegishlisi ko'rsatiladi)\n", len(progs)-n)
	}

	b.WriteString("\nDIQQAT: bularning hammasi SHARTLI. Har birida shart bor —\n")
	b.WriteString("kim olib kirmoqda, nima uchun, ro'yxatga kiritilganmi, muddat\n")
	b.WriteString("o'tmaganmi. Foydalanuvchiga imtiyozni VA'DA QILMA: shartini ayt\n")
	b.WriteString("va uning holatini so'ra. Aniq tovar kodi ma'lum bo'lsa, o'sha\n")
	b.WriteString("kodga tegishli imtiyozni <HUJJAT_TALABLARI> blokidan ol.\n")
	b.WriteString("</IMTIYOZ_DASTURLARI>")
	return b.String()
}

// codeInText — matndagi raqam-va-bo'shliq ketma-ketliklari.
// "9405 42 003 9" ni ham, "9405420039" ni ham ushlaydi.
var codeInText = regexp.MustCompile(`\d[\d ]{6,}\d|\d{10}`)

// promoteExplicit — foydalanuvchi ANIQ yozgan TIF TN kodini ro'yxat
// boshiga chiqaradi.
//
// Yolg'on ishlashdan himoya: nomzod BAZADA BOR bo'lsagina olinadi.
// Shu tufayli "faktura 2000, transport 150" kabi raqamlar kod deb
// qabul qilinmaydi — tasodifiy 10 raqam haqiqiy TIF TN kodiga to'g'ri
// kelishi deyarli mumkin emas.
func promoteExplicit(store *hscode.Store, text string, matches []hscode.Match) []hscode.Match {
	for _, raw := range codeInText.FindAllString(text, -1) {
		digits := strings.ReplaceAll(raw, " ", "")
		if len(digits) != 10 {
			continue
		}
		c, ok := store.ByCode(digits)
		if !ok {
			continue
		}
		// Allaqachon birinchi bo'lsa — ish yo'q.
		if len(matches) > 0 && matches[0].Code.Code == digits {
			return matches
		}
		out := []hscode.Match{{Code: c}}
		for _, m := range matches {
			if m.Code.Code != digits {
				out = append(out, m)
			}
		}
		// Ro'yxat uzayib ketmasin — kontekst byudjeti cheklangan.
		if len(out) > topCodes {
			out = out[:topCodes]
		}
		return out
	}
	return matches
}
