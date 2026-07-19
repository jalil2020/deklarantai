// Hujjatlarning lex.uz dagi rasmiy havolalari.
//
// NEGA QO'LDA: manba bazasida lex.uz identifikatorlari YO'Q — u o'z ichki
// ID sini ishlatadi (masalan Bojxona kodeksi = 39534, lex.uz da esa 2876352).
// Moslikni faqat tashqaridan topish mumkin.
//
// NEGA HAMMASI EMAS: korpusdagi 90 ta hujjatdan eng yiriklari ustunlik qiladi —
// 4 tasi parchalarning 63%ini, 12 tasi 77%ini tashkil qiladi. Shuning uchun
// asosiylariga havola berilgan; qolganlari havolasiz ko'rsatiladi (nomi va
// sanasi baribir yozilади, foydalanuvchi lex.uz da qidira oladi).
//
// MOSLASHTIRISH QOIDASI: kodekslar uchun faqat NOM bo'yicha, sana bo'yicha
// emas — bazadagi sana lex.uz nikidan farq qilishi mumkin (masalan Ma'muriy
// javobgarlik kodeksi: bazada 01.04.1995, lex.uz da 22.09.1994 — biri kuchga
// kirish, ikkinchisi qabul qilish sanasi).
//
// TEKSHIRILGAN: 2026-07 da lex.uz qidiruvi orqali, har biri nomi bo'yicha
// solishtirib. Havola buzilsa — lex.uz da hujjat raqami va sanasi bo'yicha
// qayta topish kerak.

export const LEX_LINKS = [
  // --- Kodekslar (nom bo'yicha) ---
  { name: /^Таможенный кодекс/i, url: 'https://lex.uz/docs/2876352' },
  { name: /^Налоговый кодекс/i, url: 'https://lex.uz/docs/4674893' },
  { name: /об административной ответственности/i, url: 'https://lex.uz/docs/97661' },
  { name: /^Уголовный кодекс/i, url: 'https://lex.uz/docs/111457' },

  // --- Qarorlar (raqam + sana bo'yicha) ---
  { docId: '55', date: '31.01.2025', url: 'https://lex.uz/docs/7358261' },  // yig'im stavkalari
  { docId: '347', date: '02.06.2020', url: 'https://lex.uz/docs/4848953' }, // utilizatsiya yig'imi

  // --- Yo'riqnomalar ---
  // GTD to'ldirish tartibi (МЮ 2773) — deklarantlar uchun eng ko'p kerak bo'ladigan.
  { name: /Инструкции о порядке заполнения грузовой таможенной декларации/i,
    url: 'https://lex.uz/docs/2924953' },
]

/** Hujjatga mos lex.uz havolasini qaytaradi (topilmasa null). */
export function lexLink({ name, docId, date }) {
  for (const l of LEX_LINKS) {
    if (l.name && l.name.test(name || '')) return l.url
    if (l.docId && String(docId) === l.docId && l.date === date) return l.url
  }
  return null
}
