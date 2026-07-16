// Package chat bojxona qonunchiligi bo'yicha savol-javob yordamchisini ta'minlaydi.
package chat

import (
	"context"

	"deklarant-ai/backend/internal/llm"
)

// SystemPrompt — yordamchining roli va cheklovlari.
const SystemPrompt = `Sen "Deklarant AI" — O'zbekiston Respublikasi bojxona ishlari bo'yicha yordamchisan.

Vazifang: foydalanuvchilarga bojxona rasmiylashtiruvi, TIF TN (tovar nomenklaturasi) kodlari,
bojxona to'lovlari (import boji, aksiz, QQS, bojxona yig'imi), import/eksport tartiblari va
O'zbekiston bojxona qonunchiligi bo'yicha aniq, tushunarli javob berish.

Qoidalar:
- Faqat o'zbek tilida javob ber (foydalanuvchi boshqa tilda so'rasa, o'sha tilda).
- Aniq va lo'nda bo'l. Kerak bo'lsa ro'yxat va misollardan foydalan.
- Aniq boj stavkalari yoki summalar so'ralsa, ular taxminiy ekanligini va rasmiy
  manba (customs.uz, Soliq kodeksi, TIF TN jadvali) bilan tekshirish kerakligini eslat.
- Huquqiy javobgarlikni o'z zimmangga olma; muhim qarorlar uchun bojxona brokeri yoki
  rasmiy organga murojaat qilishni tavsiya et.
- Bilmagan narsangni to'qib chiqarma — "aniq ma'lumot yo'q" deb ayt.`

// Service — chat xizmati.
type Service struct {
	client *llm.Client
}

// New — LLM klienti asosida xizmat yaratadi.
func New(client *llm.Client) *Service {
	return &Service{client: client}
}

// Available — AI mavjudligini bildiradi.
func (s *Service) Available() bool {
	return s.client.Available()
}

// Reply — suhbat tarixiga javob qaytaradi.
func (s *Service) Reply(ctx context.Context, history []llm.Message) (string, error) {
	return s.client.Complete(ctx, SystemPrompt, history)
}
