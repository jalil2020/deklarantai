// Package chat bojxona qonunchiligi bo'yicha savol-javob yordamchisini ta'minlaydi.
// Suhbatga TIF TN bazasi va boj hisoblash metodikasi kiritiladi, shu bilan
// yordamchi kod topish va to'lov hisoblashni ham suhbat orqali bajaradi.
package chat

import (
	"context"
	"fmt"
	"strings"

	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/llm"
)

const basePrompt = `Sen "Deklarant AI" — O'zbekiston Respublikasi bojxona ishlari bo'yicha yordamchisan.

Vazifang: foydalanuvchilarga bojxona rasmiylashtiruvi bo'yicha yordam berish:
- Tovar tavsifi yoki RASMI (invoys, tovar surati, hujjat) asosida TIF TN kodini aniqlash.
- Bojxona to'lovlarini (import boji, aksiz, QQS, bojxona yig'imi) hisoblash.
- Import/eksport tartiblari va O'zbekiston bojxona qonunchiligi bo'yicha maslahat.

Agar foydalanuvchi rasm yuborsa — undagi tovar, miqdor, narx (invoys) kabi ma'lumotlarni
diqqat bilan o'qib chiq va shularga asoslanib javob ber.

Bojxona to'lovlarini hisoblash metodikasi (ketma-ket):
  Import boji = Bojxona qiymati × boj%
  Aksiz      = (Bojxona qiymati + Import boji) × aksiz%
  QQS        = (Bojxona qiymati + Import boji + Aksiz) × QQS%
  Bojxona yig'imi ≈ 490 000 so'm (shartli qat'iy summa)
  Jami       = Bojxona yig'imi + Import boji + Aksiz + QQS

Qoidalar:
- O'zbek tilida, aniq va lo'nda javob ber (foydalanuvchi boshqa tilda so'rasa, o'sha tilda).
- Boj hisoblaganda bosqichlarni ko'rsat va yakuniy summani ajratib yoz.
- Stavkalar va summalar TAXMINIY ekanligini eslat; rasmiy manba: customs.uz, Soliq kodeksi.
- Bilmagan narsangni to'qib chiqarma. Muhim qarorlar uchun bojxona brokeriga murojaatni tavsiya et.`

// Service — chat xizmati.
type Service struct {
	client *llm.Client
	codes  *hscode.Store
}

// New — LLM klienti va TIF TN bazasi asosida xizmat yaratadi.
func New(client *llm.Client, codes *hscode.Store) *Service {
	return &Service{client: client, codes: codes}
}

// Available — AI mavjudligini bildiradi.
func (s *Service) Available() bool {
	return s.client.Available()
}

// systemPrompt — baza ma'lumotlari bilan to'ldirilgan tizim ko'rsatmasi.
func (s *Service) systemPrompt() string {
	if s.codes == nil || len(s.codes.All()) == 0 {
		return basePrompt
	}
	var b strings.Builder
	b.WriteString(basePrompt)
	b.WriteString("\n\nMa'lumot uchun mavjud TIF TN kodlari (namunaviy stavkalar bilan):\n")
	for _, c := range s.codes.All() {
		b.WriteString(fmt.Sprintf("- %s | %s | boj %.0f%%, aksiz %.0f%%, QQS %.0f%% | %s\n",
			c.Code, c.Name, c.ImportDuty, c.Excise, c.VAT, c.Unit))
	}
	b.WriteString("\nRo'yxatda mos kod bo'lmasa, umumiy bilimingga tayanib TIF TN guruhini (birinchi 4-6 raqam) taklif qil.")
	return b.String()
}

// Reply — suhbat tarixiga javob qaytaradi.
func (s *Service) Reply(ctx context.Context, history []llm.Message) (string, error) {
	return s.client.Complete(ctx, s.systemPrompt(), history)
}
