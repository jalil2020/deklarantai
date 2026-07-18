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
- Import/eksport tartiblari va qonunchilik bo'yicha maslahat.

Agar foydalanuvchi rasm yuborsa — undagi tovar, miqdor, narx kabi ma'lumotlarni
diqqat bilan o'qib chiq va shularga asoslanib javob ber.

BOJXONA TO'LOVLARI (GTD kodlari bilan):
  10. Bojxona yig'imi — bojxona qiymatining dollardagi ekvivalentiga qarab,
      BRV ning 1 dan 25 karragacha (ПКМ 55, 31.01.2025).
  12. Ko'rik — ish vaqtida 25% BRV/soat, tashqarida 2×BRV/soat (to'liq soatga yaxlitlanadi).
  20. Bojxona boji     = bojxona qiymati × boj%
  21. Qo'shimcha boj   = bojxona qiymati × qo'shimcha%
  27. Aksiz            = bojxona qiymati × aksiz%   (Soliq kodeksi 285-modda:
      advalor stavkada baza — bojxona qiymati, bojsiz)
  29. QQS = (bojxona qiymati + boj + qo'shimcha boj + aksiz) × QQS%
      (Soliq kodeksi 254-modda. DIQQAT: bojxona yig'imi QQS bazasiga KIRMAYDI.)
  79. Utilizatsiya yig'imi — avtotransport uchun, netto vazn bo'yicha (alohida qoida).

  Bojxona qiymati = (faktura qiymati + transport xarajati) × valyuta kursi.

QOIDALAR:
- O'zbek tilida, aniq va lo'nda javob ber (foydalanuvchi boshqa tilda so'rasa, o'sha tilda).
- Savolga qo'shib "TIF TN BAZASIDAN" va "QONUNCHILIKDAN" bloklari berilishi mumkin.
  Ular — bizning bazamizdan topilgan haqiqiy ma'lumot. Javobingni AVVALO o'shalarga
  asoslantir: ishlatgan kodingni va qonun moddasini ko'rsat
  (masalan: "Bojxona kodeksi, 346-modda").
- Bloklarda javob bo'lmasa — buni ochiq ayt ("bazada bu haqda ma'lumot topilmadi")
  va o'z bilimingga tayanayotganingni eslat. Blokdagi ma'lumotni o'ylab topilgan
  modda raqami bilan to'ldirma.
- Stavkalar baza olingan sanaga tegishli va o'zgarib turadi — muhim qarorlar uchun
  customs.uz yoki bojxona brokeridan tasdiqlashni tavsiya et.
- Bilmagan narsangni to'qib chiqarma.`

// Service — chat xizmati.
type Service struct {
	client *llm.Client
	codes  *hscode.Store
	laws   *laws.Store // ixtiyoriy; nil bo'lsa qonun konteksti qo'shilmaydi
}

// New — LLM klienti, TIF TN bazasi va qonun korpusi asosida xizmat yaratadi.
func New(client *llm.Client, codes *hscode.Store, lawStore *laws.Store) *Service {
	return &Service{client: client, codes: codes, laws: lawStore}
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
			blocks = append(blocks, formatMatches(s.codes.Meta(), m))
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
		fmt.Fprintf(&b, "\n  %s\n%s\n", c.Title, c.Text)
	}
	b.WriteString("</QONUNCHILIKDAN>")
	return b.String()
}

// formatMatches — topilgan kodlarni kontekst bloki sifatida shakllantiradi.
func formatMatches(m hscode.Meta, matches []hscode.Match) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<TIF_TN_BAZASIDAN>\nSavolga mos kodlar (%s, stavkalar %s holatiga):\n\n",
		m.Nomenclature, m.RatesAsOf)
	for _, mt := range matches {
		c := mt.Code
		fmt.Fprintf(&b, "%s — %s\n", formatCode(c.Code), c.PathUZ)
		fmt.Fprintf(&b, "   boj %g%% | QQS %g%%", c.ImportDuty, c.VAT)
		if c.Excise > 0 {
			fmt.Fprintf(&b, " | aksiz %g%%", c.Excise)
		}
		if c.ExportDuty > 0 {
			fmt.Fprintf(&b, " | eksport boji %g%%", c.ExportDuty)
		}
		if c.Unit != "" {
			fmt.Fprintf(&b, " | qo'shimcha o'lchov: %s", c.Unit)
		}
		b.WriteString("\n")
	}
	b.WriteString("</TIF_TN_BAZASIDAN>")
	return b.String()
}

// formatCode — "8701211019" → "8701 21 101 9".
func formatCode(c string) string {
	if len(c) != 10 {
		return c
	}
	return c[0:4] + " " + c[4:6] + " " + c[6:9] + " " + c[9:]
}
