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
// Qolganlarida esa nom bilan BIRGA sana ham talab qilinadi. Sabab: bir xil
// nomli, turli yildagi hujjatlar bor va eskisi bekor qilingan bo'lishi mumkin.
// Masalan "О порядке применения акцизных марок…" — 1999 (kuchini yo'qotgan)
// va 2024 (amaldagi). Faqat nom bo'yicha mos qo'ysak, amaldagi hujjatga
// bekor qilinganining havolasi biriktirilib qolardi.
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

  // --- Deklaratsiya va rasmiylashtiruv tartibi ---
  // ДТС (bojxona qiymati deklaratsiyasi) to'ldirish — МЮ 2868.
  { name: /заполнения декларации таможенной стоимости/i, date: '14.03.2017',
    url: 'https://lex.uz/docs/3133239' },
  // Dastlabki, davriy va to'liqsiz YuBD — МЮ 3296.
  { name: /первичных, периодических и неполных таможенных грузовых деклараций/i,
    date: '04.05.2021', url: 'https://lex.uz/docs/5408763' },

  // --- Bojxona ma'muriyatchiligi ---
  { name: /О дополнительных мерах по организации деятельности органов государственной таможенной службы/i,
    date: '25.03.2025', url: 'https://lex.uz/docs/7452091' },  // ПП-122
  { name: /О дальнейшем совершенствовании некоторых процедур в таможенной сфере/i,
    date: '06.11.2025', url: 'https://lex.uz/docs/7830538' },  // ПКМ 700
  { name: /упрощению и повышению эффективности таможенного администрирования/i,
    date: '17.12.2025', url: 'https://lex.uz/docs/7934918' },  // УП-250
  { name: /грузовых операций в отношении товаров, находящихся под таможенным контролем/i,
    date: '20.08.2021', url: 'https://lex.uz/docs/5592823' },  // ПКМ 531

  // --- Boj stavkalari ---
  // ПП-3818 — import boji stavkalari (1-ilova). manba kalkulyatori ham
  // "20. Там. пошлина" yonida aynan shu hujjatga havola qiladi.
  { name: /упорядочению внешнеэкономической деятельности и совершенствованию системы таможенно/i,
    date: '29.06.2018', url: 'https://lex.uz/docs/3802366' },

  // --- Imtiyozlar va taqiqlar ---
  { name: /льгот по таможенной пошлине и налогу на добавленную стоимость/i,
    date: '27.11.2020', url: 'https://lex.uz/docs/5131865' },  // ПКМ 750
  { name: /озоноразрушающих веществ/i, date: '09.01.2018',
    url: 'https://lex.uz/docs/3500042' },                      // ПКМ 17
]

/**
 * Hujjatga mos lex.uz havolasini qaytaradi (topilmasa null).
 *
 * `date` yozuvda ko'rsatilgan bo'lsa — nom bilan birga u ham mos kelishi shart.
 * Ko'rsatilmagan bo'lsa (kodekslar) faqat nom bo'yicha tekshiriladi.
 */
export function lexLink({ name, docId, date }) {
  for (const l of LEX_LINKS) {
    if (l.name) {
      if (l.name.test(name || '') && (!l.date || l.date === date)) return l.url
      continue
    }
    if (l.docId && String(docId) === l.docId && l.date === date) return l.url
  }
  return null
}
