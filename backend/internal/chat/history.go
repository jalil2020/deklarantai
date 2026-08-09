package chat

import (
	"strings"

	"deklarant-ai/backend/internal/llm"
)

// Suhbat tarixini chegaralash.
//
// MUAMMO: mijoz HAR SO'ROVDA butun suhbatni yuboradi va u to'liq holda
// modelga ketadi. Ya'ni 10-savolda birinchi to'qqizta savol-javob yana
// bir marta to'lanadi. Eng og'iri — SURATLAR: bitta 1600px surat ~1 500
// token, va u suhbat oxirigacha har safar qayta yuborilardi.
//
// Kesh bu yerda YORDAM BERMAYDI: retrieval bloklari faqat OXIRGI
// foydalanuvchi xabariga qo'shiladi, keyingi navbatda esa o'sha xabar
// mijozdan xom holda qaytib keladi. Shu sababli xabarlar prefiksi har
// navbatda o'zgaradi va kesh uchun yaroqsiz bo'ladi. Buni to'liq hal
// qilish uchun suhbatni SERVERDA saqlash kerak — u akkauntlar bilan
// birga keladi. Shu paytgacha yagona ishonchli richag — hajmni kesish.
const (
	// maxHistoryMessages — modelga yuboriladigan xabarlar soni.
	// 20 ta = 10 savol-javob; undan uzun suhbatda boshidagi gaplar
	// odatda allaqachon javoblarda takrorlangan bo'ladi.
	maxHistoryMessages = 20

	// maxHistoryChars — tarixning MATN hajmi chegarasi.
	//
	// Suratlar bu hisobga KIRMAYDI va bu ataylab: bitta surat 300 KB
	// atrofida, ya'ni har qanday matn byudjetidan katta. Ikkalasini
	// bitta hisobga qo'shsak, qoidalar bir-birini yeb qo'yardi —
	// imageMessages "oxirgi ikkita suratni saqla" deydi, byudjet esa
	// o'sha ikkitasini darrov chiqarib tashlardi. Suratlar SONI bilan
	// chegaralanadi (imageMessages), matn esa HAJMI bilan.
	//
	// Retrieval bloklari alohida 24 000 belgigacha qo'shiladi
	// (maxContext).
	maxHistoryChars = 40_000

	// imageMessages — suratlari SAQLANADIGAN oxirgi xabarlar soni.
	//
	// 1 emas, 2: foydalanuvchi ko'pincha suratdan keyin bir necha savol
	// beradi ("bu qaysi kod?", "endi bojini hisobla") va ikkinchi
	// savolda ham surat kerak bo'lishi mumkin. Undan keyin javobdagi
	// tavsif yetarli.
	imageMessages = 2
)

// imagesDropped — surat olib tashlanganini modelga bildiruvchi belgi.
// Jimgina o'chirish yomon bo'lardi: model suratni "ko'rmadim" deb emas,
// "yo'q edi" deb o'ylardi.
const imagesDropped = "\n[eslatma: bu xabardagi surat(lar) kontekstdan olib tashlandi — " +
	"kerak bo'lsa qaytadan yuborishni so'ra]"

// trimHistory — modelga yuboriladigan tarixni chegaralaydi.
//
// Asl ro'yxat o'zgartirilmaydi.
func trimHistory(history []llm.Message) []llm.Message {
	if len(history) == 0 {
		return history
	}
	out := dropOldImages(history)

	// Oxiridan boshlab byudjet sarflanadi: yangi xabarlar eskilaridan
	// muhimroq. Oxirgi xabar — joriy savol — HAR DOIM qoladi.
	start, used := 0, 0
	for i := len(out) - 1; i >= 0; i-- {
		used += len(out[i].Content)
		if i == len(out)-1 {
			continue
		}
		if used > maxHistoryChars || len(out)-i > maxHistoryMessages {
			start = i + 1
			break
		}
	}

	// Birinchi xabar "user" bo'lishi SHART (Anthropic talabi).
	for start < len(out)-1 && out[start].Role != "user" {
		start++
	}
	return out[start:]
}

// dropOldImages — eski xabarlardagi suratlarni olib tashlaydi.
func dropOldImages(history []llm.Message) []llm.Message {
	// Avval sanaymiz: nusxa faqat kerak bo'lgandagina yasalsin.
	withImages := 0
	for _, m := range history {
		if len(m.Images) > 0 {
			withImages++
		}
	}
	if withImages <= imageMessages {
		return history
	}

	out := make([]llm.Message, len(history))
	copy(out, history)

	seen := 0
	for i := len(out) - 1; i >= 0; i-- {
		if len(out[i].Images) == 0 {
			continue
		}
		seen++
		if seen <= imageMessages {
			continue
		}
		out[i].Images = nil
		out[i].Content = strings.TrimSpace(out[i].Content) + imagesDropped
	}
	return out
}

// size — xabarning to'liq hajmi, surat bilan birga (o'lchov uchun).
func size(m llm.Message) int {
	n := len(m.Content)
	for _, img := range m.Images {
		n += len(img.Data)
	}
	return n
}
