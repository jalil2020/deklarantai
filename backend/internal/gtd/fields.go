// Package gtd — Yuk bojxona deklaratsiyasi (GTD/ГТД) grafalari ma'lumotnomasi.
//
// GTD — 54 grafli xalqaro SAD (Yagona ma'muriy hujjat) formatida.
// To'ldirish tartibi O'zbekiston yo'riqnomasi bilan belgilangan
// (Инструкция о порядке заполнения ГТД, 06.04.2016) — u qonun korpusida
// (laws.json) bor va chat javob berganda RAG orqali kontekstga tushadi.
//
// ⚠️ NEGA BU YERDA FAQAT SKELET: har grafning ANIQ qiymati (kod, format)
// yo'riqnomaga tayanadi va uni model korpusdan o'qiydi. Bu fayl faqat
// "qaysi graf bor va uni KIM to'ldiradi" ni belgilaydi — o'ylab topilgan
// qiymat yo'q. Bu loyihaning "har javob bazaga tayanadi" tamoyiliga mos.
package gtd

import "strings"

// Fill — grafani kim to'ldiradi.
type Fill int

const (
	// Auto — Deklarant AI ning MAVJUD qismi hisoblab beradi.
	Auto Fill = iota
	// User — foydalanuvchi kiritadi (rekvizit: jo'natuvchi, hujjat raqami).
	User
	// Ref — standart kod yoki yo'riqnomadan olinadigan qiymat.
	Ref
)

func (f Fill) String() string {
	switch f {
	case Auto:
		return "avto"
	case User:
		return "foydalanuvchi"
	default:
		return "ma'lumotnoma"
	}
}

// Field — bitta graf.
type Field struct {
	No   string // graf raqami ("31", "33", "47")
	Name string // o'zbekcha nom
	Fill Fill
	// Src — Auto uchun: bizning qaysi qism beradi. User/Ref uchun: nima kerak.
	Src string
}

// ImportFields — ИМ 40 (erkin muomalaga import) uchun asosiy grafalar.
//
// ⚠️ TO'LIQ 54 EMAS: bu yerda import GTD da AMALDA to'ldiriladigan
// grafalar. Kam ishlatiladigan yoki maxsus rejim grafalari (kvota,
// ombor, tranzit rekvizitlari) ataylab qoldirilgan — ular kerak
// bo'lganda yo'riqnomadan qo'shiladi.
//
// Auto grafalar — loyihaning kuchi: kod tanlash va to'lov hisobi
// "aql" talab qiladi va aynan shu qism allaqachon tayyor.
var ImportFields = []Field{
	{"1", "Deklaratsiya turi", Ref, "ИМ va rejim kodi (erkin muomala uchun 40)"},
	{"2", "Jo'natuvchi / Eksportchi", User, "nom, manzil, davlat"},
	{"5", "Tovarlar soni", Auto, "deklaratsiyadagi tovar pozitsiyalari soni"},
	{"6", "O'ram soni", User, "jami o'ram (joy) soni"},
	{"8", "Oluvchi / Importchi", User, "nom, STIR (INN), manzil"},
	{"9", "Moliyaviy tomon", User, "to'lovni amalga oshiruvchi (odatda oluvchi)"},
	{"11", "Savdo davlati", User, "shartnoma tuzilgan davlat kodi"},
	{"12", "Umumiy bojxona qiymati", Auto, "duty: bojxona qiymati (so'm)"},
	{"14", "Deklarant / vakil", User, "deklarant nomi va STIR"},
	{"15", "Jo'natuvchi davlat", User, "tovar jo'natilgan davlat"},
	{"16", "Kelib chiqish davlati", Auto, "countries: tovar kelib chiqqan davlat"},
	{"18", "Transport (jo'natishda)", User, "transport vositasi va davlat belgisi"},
	{"20", "Yetkazib berish sharti", User, "Incoterms (CIF, FOB, DAP...) va joy"},
	{"22", "Valyuta va umumiy summa", User, "shartnoma valyutasi va faktura summasi"},
	{"23", "Valyuta kursi", Auto, "rates: Markaziy bank kursi (deklaratsiya sanasiga)"},
	{"24", "Bitim tabiati", Ref, "bitim turi kodi (yo'riqnoma ilovasi)"},
	{"25", "Chegarada transport turi", User, "transport turi kodi"},
	{"31", "Tovar o'ramlari va tavsifi", Auto, "hscode: to'liq nomenklatura tavsifi + o'ram/miqdor"},
	{"32", "Tovar tartib raqami", Auto, "pozitsiya tartib raqami"},
	{"33", "TIF TN kod", Auto, "hscode: 10 xonali kod"},
	{"34", "Kelib chiqish davlati kodi", Auto, "countries: davlat harf kodi"},
	{"35", "Brutto vazn (kg)", User, "o'ram bilan umumiy vazn"},
	{"36", "Imtiyoz (preferentsiya)", Auto, "countries: kelib chiqishga qarab boj rejimi (0×/1×/2×)"},
	{"37", "Protsedura", Ref, "rejim kodi (masalan 40 00) — yo'riqnoma"},
	{"38", "Netto vazn (kg)", User, "o'ramsiz sof vazn"},
	{"41", "Qo'shimcha o'lchov birligi", Auto, "hscode: kodning qo'shimcha birligi (dona, litr, m²)"},
	{"42", "Tovar narxi", User, "faktura bo'yicha tovar narxi (valyutada)"},
	{"43", "Baholash usuli", Ref, "bojxona qiymati usuli (odatda 1 — bitim narxi)"},
	{"44", "Qo'shimcha ma'lumot / hujjatlar", Auto, "docs: kod bo'yicha kerakli sertifikat/litsenziya ro'yxati"},
	{"45", "Bojxona qiymati (tovar)", Auto, "duty: shu tovarning bojxona qiymati"},
	{"46", "Statistik qiymat", Auto, "duty: statistik qiymat (dollarda)"},
	{"47", "To'lovlar hisobi", Auto, "duty: har to'lov (10/12/20/21/27/29/79) — turi, asosi, stavka, summa"},
	{"48", "To'lovni kechiktirish", User, "kechiktirish bo'lsa (odatda bo'sh)"},
	{"54", "Joy, sana, imzo", User, "deklarant imzosi va sana"},
}

// AutoCount — biz avtomatik to'ldiradigan grafalar soni.
func AutoCount() int {
	n := 0
	for _, f := range ImportFields {
		if f.Fill == Auto {
			n++
		}
	}
	return n
}

// PromptBlock — grafalar skeletini chat tizim ko'rsatmasi uchun matn qiladi.
//
// Model shu skeletni oladi va Auto grafalarni bizning hisob natijasi bilan,
// qolganini yo'riqnoma (RAG) va foydalanuvchi ma'lumoti bilan to'ldiradi.
func PromptBlock() string {
	var b strings.Builder
	b.WriteString("GTD (Yuk bojxona deklaratsiyasi) GRAFALARI — import (ИМ 40):\n")
	b.WriteString("Har graf: [raqam] nom — kim to'ldiradi.\n")
	b.WriteString("  «avto» = quyidagi hisobdan olinadi (kod, boj, kelib chiqish);\n")
	b.WriteString("  «foydalanuvchi» = rekvizit, undan so'ra;\n")
	b.WriteString("  «ma'lumotnoma» = standart kod, yo'riqnomadan.\n\n")
	for _, f := range ImportFields {
		b.WriteString("  [")
		b.WriteString(f.No)
		b.WriteString("] ")
		b.WriteString(f.Name)
		b.WriteString(" — ")
		b.WriteString(f.Fill.String())
		if f.Fill == Auto {
			b.WriteString(" (")
			b.WriteString(f.Src)
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
	return b.String()
}
