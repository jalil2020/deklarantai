// TIF TN qo'shimcha o'lchov birliklarining o'zbekcha nomlari.
//
// manba bazasidagi unit jadvalida faqat ruscha nom bor ("шт", "пар"),
// avtomatik transliteratsiya esa noto'g'ri natija beradi ("sht", "par").
// Shuning uchun qo'lda jadval.
//
// DIQQAT: TIF TN da asosiy o'lchov doim kg (netto). Bu yerdagilar —
// QO'SHIMCHA birliklar; ko'p kodlarda ular umuman yo'q (null).

export const UNITS_UZ = {
  '006': 'm',
  '055': 'm²',
  '112': 'l',
  '113': 'm³',
  '114': "1000 m³",
  '130': '1000 l',
  '162': 'karat',
  '163': 'g',
  '166': 'kg',
  '181': 'BRT', // brutto-registr tonna
  '185': 'yuk koʻtarish, t',
  '246': '1000 kVt·s',
  '305': 'Kyuri',
  '306': "g boʻlinuvchi izotop",
  '555': 'pachka',
  '556': 'sm³',
  '557': '15 g',
  '715': 'juft',
  '796': 'dona',
  '797': '100 dona',
  '798': '1000 dona',
  '831': 'l 100% spirt',
  '841': 'kg H₂O₂',
  '845': 'kg 90% quruq modda',
  '852': 'kg K₂O',
  '859': 'kg KOH',
  '861': 'kg N',
  '863': 'kg NaOH',
  '865': 'kg P₂O₅',
  '867': 'kg U',
}
