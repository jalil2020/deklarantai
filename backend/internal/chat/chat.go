// Package chat bojxona bo'yicha savol-javob yordamchisini ta'minlaydi.
//
// Retrieval: 13 000+ kodni promptga sig'dirib bo'lmaydi, shuning uchun
// foydalanuvchining oxirgi savoliga mos kodlar bazadan qidirib topiladi va
// faqat o'shalar kontekstga qo'shiladi.
package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"deklarant-ai/backend/internal/docs"
	"deklarant-ai/backend/internal/duty"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
	"deklarant-ai/backend/internal/llm"
)

const (
	// topCodes — kontekstga qo'shiladigan TIF TN kodlari soni.
	topCodes = 8
	// topLaws — qonun parchalari soni. Ular yirik (~4 KB), shuning uchun kam.
	topLaws = 3
)

const basePrompt = `Sen "Deklarant AI" — O'zbekiston Respublikasi bojxona ishlari bo'yicha yordamchisan.

Vazifang: foydalanuvchilarga bojxona rasmiylashtiruvi bo'yicha yordam berish:
- Tovar tavsifi yoki RASMI (invoys, tovar surati, hujjat) asosida TIF TN kodini aniqlash.
- Bojxona to'lovlarini hisoblash.
- Tovarga qanday HUJJAT kerakligini aytish (litsenziya, sertifikat, imtiyoz).
- Import/eksport tartiblari va qonunchilik bo'yicha maslahat.

Agar foydalanuvchi rasm yuborsa — undagi tovar, miqdor, narx kabi ma'lumotlarni
diqqat bilan o'qib chiq va shularga asoslanib javob ber.

BOJXONA TO'LOVLARI (GTD kodlari bilan):
  10. Bojxona yig'imi — bojxona qiymatining dollardagi ekvivalentiga qarab,
      BRV ning karrasi (ПКМ 55, 31.01.2025). Shkala quyida berilgan —
      yig'imni DOIM hisoblab ber, "alohida belgilanadi" deb qoldirma.
  12. Ko'rik — ish vaqtida 25% BRV/soat, tashqarida 2×BRV/soat (to'liq soatga yaxlitlanadi).
  20. Bojxona boji     = bojxona qiymati × boj%
  21. Qo'shimcha boj   = bojxona qiymati × qo'shimcha%
  27. Aksiz            = bojxona qiymati × aksiz%   (Soliq kodeksi 285-modda:
      advalor stavkada baza — bojxona qiymati, bojsiz; qat'iy stavkada esa
      natural miqdor — masalan "340 000 so'm / 1000 dona")
  29. QQS = (bojxona qiymati + boj + qo'shimcha boj + aksiz) × QQS%
      (Soliq kodeksi 254-modda. DIQQAT: bojxona yig'imi QQS bazasiga KIRMAYDI.)
  79. Utilizatsiya yig'imi — avtotransport uchun, netto vazn bo'yicha (alohida qoida).

  Bojxona qiymati = (faktura qiymati + transport xarajati) × valyuta kursi.

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
- Bilmagan narsangni to'qib chiqarma.`

// Service — chat xizmati.
type Service struct {
	client *llm.Client
	codes  *hscode.Store
	laws   *laws.Store // ixtiyoriy; nil bo'lsa qonun konteksti qo'shilmaydi
	docs   *docs.Store // ixtiyoriy; hujjat talablari
}

// New — LLM klienti, TIF TN bazasi, qonun korpusi va hujjat talablari
// asosida xizmat yaratadi. lawStore va docStore nil bo'lishi mumkin.
func New(client *llm.Client, codes *hscode.Store, lawStore *laws.Store, docStore *docs.Store) *Service {
	return &Service{client: client, codes: codes, laws: lawStore, docs: docStore}
}

// Available — AI mavjudligini bildiradi.
func (s *Service) Available() bool { return s.client.Available() }

// systemPrompt — barqaror tizim ko'rsatmasi.
// Kodlar bu yerga QO'SHILMAYDI — ular so'rovga qarab kontekstga qo'shiladi,
// shu tufayli prompt keshi buzilmaydi.
func (s *Service) systemPrompt() string {
	if s.codes == nil {
		return basePrompt
	}
	m := s.codes.Meta()
	var b strings.Builder
	b.WriteString(basePrompt)

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
func (s *Service) Reply(ctx context.Context, history []llm.Message) (string, error) {
	return s.client.Complete(ctx, s.systemPrompt(), s.withRetrieval(history))
}

// withRetrieval — oxirgi foydalanuvchi savoliga mos TIF TN kodlari va qonun
// parchalarini topib, uning matniga qo'shib qo'yadi.
// Asl tarix o'zgartirilmaydi (nusxa qaytariladi).
func (s *Service) withRetrieval(history []llm.Message) []llm.Message {
	if len(history) == 0 {
		return history
	}
	last := history[len(history)-1]
	if last.Role != "user" || strings.TrimSpace(last.Content) == "" {
		return history // rasm-only xabar yoki bo'sh — qidiradigan narsa yo'q
	}

	var blocks []string
	if s.codes != nil {
		if m := s.codes.Search(last.Content, topCodes); len(m) > 0 {
			// Imtiyoz belgisi aynan STAVKA yonida ko'rsatiladi. Uni faqat
			// alohida blokda bersak, model "QQS 12%" ni ko'rib hisoblab
			// yuborishi mumkin — imtiyoz esa pastda qolib ketardi.
			exempt := map[string][]string{}
			if s.docs != nil {
				for _, mt := range m {
					if e := s.docs.Exemptions(mt.Code.Code, docs.Import); len(e) > 0 {
						exempt[mt.Code.Code] = e
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
			blocks = append(blocks, formatLaws(m))
		}
	}
	if len(blocks) == 0 {
		return history
	}

	out := make([]llm.Message, len(history))
	copy(out, history)
	out[len(out)-1].Content = last.Content + "\n\n" + strings.Join(blocks, "\n\n")
	return out
}

// formatLaws — topilgan qonun parchalarini kontekst bloki sifatida shakllantiradi.
func formatLaws(matches []laws.Match) string {
	var b strings.Builder
	b.WriteString("<QONUNCHILIKDAN>\n")
	for _, m := range matches {
		c := m.Chunk
		fmt.Fprintf(&b, "\n— Manba: %s", c.Name)
		if c.Date != "" {
			fmt.Fprintf(&b, " (%s)", c.Date)
		}
		if c.Since != "" {
			fmt.Fprintf(&b, ", %s dan amalda", c.Since)
		}
		// lex.uz havolasi — foydalanuvchi rasmiy matnni ochib tekshira olsin.
		if c.Lex != "" {
			fmt.Fprintf(&b, "\n  Rasmiy manba: %s", c.Lex)
		}
		fmt.Fprintf(&b, "\n  %s\n%s\n", c.Title, c.Text)
	}
	b.WriteString("</QONUNCHILIKDAN>")
	return b.String()
}

// formatMatches — topilgan kodlarni kontekst bloki sifatida shakllantiradi.
func formatMatches(m hscode.Meta, matches []hscode.Match, exempt map[string][]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<TIF_TN_BAZASIDAN>\nSavolga mos kodlar (%s, stavkalar %s holatiga):\n\n",
		m.Nomenclature, m.RatesAsOf)
	for _, mt := range matches {
		c := mt.Code
		fmt.Fprintf(&b, "%s — %s\n", formatCode(c.Code), c.PathUZ)
		fmt.Fprintf(&b, "   boj %g%% | QQS %g%%", c.ImportDuty, c.VAT)
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
		if e := exempt[c.Code]; len(e) > 0 {
			fmt.Fprintf(&b, "   ⚠️ bu kodga IMTIYOZ qoidasi bor (%s) — stavka yuqorida\n"+
				"      ko'rsatilgandek bo'lmasligi mumkin. Shart <HUJJAT_TALABLARI>\n"+
				"      blokidagi \"Imtiyozlar\" bo'limida; uni tekshirmasdan hisoblama.\n",
				strings.Join(e, ", "))
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
