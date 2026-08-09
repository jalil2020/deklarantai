// Davlatlar ma'lumotnomasi va ularning boj rejimi.
//
//   node tools/extract-countries.mjs [--out=backend/data/countries.json]
//
// Manba: info.sqlite / countries (kod, nom, rejim, offshor belgisi).
//
// HUQUQIY ASOS — Bojxona kodeksi 300-modda:
//   • erkin savdo davlatlari         → "bojxona bojlari qoʻllanilmaydi"
//   • eng qulaylik rejimi (EQR)      → "boj tarifi bilan belgilangan stavkalar"
//   • rejim yoʻq yoki kelib chiqishi
//     aniqlanmagan                   → "stavkalar ikki baravar oshiriladi"
//
// Bazadagi "rate" maydoni aynan shu uch darajani beradi: 0, 1, 2.

import { DatabaseSync } from 'node:sqlite'
import fs from 'node:fs'
import path from 'node:path'
import { toLatin } from './translit.mjs'

const args = Object.fromEntries(
  process.argv.slice(2).map((a) => {
    const [k, v] = a.replace(/^--/, '').split('=')
    return [k, v ?? true]
  }),
)
const OUT = args.out || 'backend/data/countries.json'
const SRC = 'backend/data/manba/info.sqlite'

const db = new DatabaseSync(SRC, { readOnly: true })

// rate → boj koeffitsienti va rejim nomi.
const REGIME = {
  0: { multiplier: 0, name: 'erkin savdo', note: 'Bojxona boji qo\'llanilmaydi (BK 300-modda)' },
  1: { multiplier: 1, name: 'eng qulaylik rejimi', note: 'Boj tarifidagi odatdagi stavka' },
  2: { multiplier: 2, name: 'rejim yo\'q', note: 'Stavka ikki baravar oshiriladi (BK 300-modda)' },
}

// Haqiqiy o'zbekcha nomlar va sinonimlar.
//
// NEGA KERAK: bazada nomlar ruscha, transliteratsiya esa "КИТАЙ" → "KITAY",
// "США" → "SSHA" beradi. Foydalanuvchi "Xitoy", "AQSh" deb yozadi va
// topilmaydi. Bu yerda O'zbekistonning asosiy savdo sheriklari qo'lda
// yozilgan — qolganlari transliteratsiya bilan qoladi.
const UZ_NAMES = {
  '643': ['Rossiya', 'Rossiya Federatsiyasi'],
  '398': ['Qozogʻiston', "Qozog'iston"],
  '156': ['Xitoy', 'Xitoy Xalq Respublikasi', 'XXR'],
  '840': ['AQSh', 'Amerika Qoʻshma Shtatlari', "Amerika Qo'shma Shtatlari"],
  '792': ['Turkiya'],
  '417': ['Qirgʻiziston', "Qirg'iziston"],
  '762': ['Tojikiston'],
  '795': ['Turkmaniston'],
  '112': ['Belarus', 'Belorussiya'],
  '804': ['Ukraina'],
  '276': ['Germaniya'],
  '380': ['Italiya'],
  '250': ['Fransiya'],
  '826': ['Buyuk Britaniya', 'Angliya'],
  '392': ['Yaponiya'],
  '410': ['Janubiy Koreya', 'Koreya Respublikasi'],
  '356': ['Hindiston'],
  '364': ['Eron'],
  '004': ['Afgʻoniston', "Afg'oniston"],
  '784': ['BAA', 'Birlashgan Arab Amirliklari'],
  '702': ['Singapur'],
  '616': ['Polsha'],
  '860': ['Oʻzbekiston', "O'zbekiston"],
  '268': ['Gruziya'],
  '031': ['Ozarbayjon'],
  '051': ['Armaniston'],
  '498': ['Moldova'],
  '203': ['Chexiya'],
  '528': ['Niderlandiya', 'Gollandiya'],
  '724': ['Ispaniya'],
  '344': ['Gonkong'],
  '458': ['Malayziya'],
  '764': ['Tailand'],
  '704': ['Vetnam'],
  '682': ['Saudiya Arabistoni'],
  '818': ['Misr'],
  '586': ['Pokiston'],
}

const stats = { total: 0, byRate: {}, offshore: 0, skipped: 0, named: 0 }
const out = []

for (const r of db.prepare('SELECT * FROM countries ORDER BY code').all()) {
  stats.total++
  const rate = Number(r.rate)
  const reg = REGIME[rate]
  if (!reg) { stats.skipped++; continue }

  stats.byRate[rate] = (stats.byRate[rate] || 0) + 1
  if (Number(r.offshore) === 1) stats.offshore++

  const code = String(r.code)
  const uz = UZ_NAMES[code]
  if (uz && uz.length) stats.named++

  out.push({
    code,
    name_ru: String(r.name || '').trim(),
    // Qo'lda yozilgani bo'lsa — o'shani, aks holda transliteratsiya.
    name_uz: uz && uz.length ? uz[0] : toLatin(String(r.name || '').trim()),
    // Qidiruv uchun qo'shimcha nomlar ("Amerika Qo'shma Shtatlari", "XXR"…).
    ...(uz && uz.length > 1 ? { aliases: uz.slice(1) } : {}),
    // ISO alfa-2 (bazada "smallName"), ba'zilarida bo'sh.
    iso: String(r.smallName || '').trim() || undefined,
    regime: reg.name,
    // Boj shu songa ko'paytiriladi.
    duty_multiplier: reg.multiplier,
    // Offshor zonalar — alohida nazorat va hujjat talablari bo'lishi mumkin.
    ...(Number(r.offshore) === 1 ? { offshore: true } : {}),
  })
}

const file = {
  meta: {
    source: 'ichki manba baza — davlatlar',
    script: 'tools/extract-countries.mjs',
    legal_basis: 'Bojxona kodeksi 300-modda (tarif preferensiyalari)',
    legal_basis_lex: 'https://lex.uz/docs/2876352',
    note: 'duty_multiplier — boj stavkasi shu songa ko\'paytiriladi: ' +
      '0 = erkin savdo (boj yo\'q), 1 = eng qulaylik rejimi, ' +
      '2 = rejim yo\'q yoki kelib chiqishi aniqlanmagan.',
    warning: 'Erkin savdo imtiyozi AVTOMATIK EMAS: BK 300-moddasiga ko\'ra ' +
      'tovar shartnoma ishtirokchisi davlat rezidenti tomonidan eksport ' +
      'qilingan va o\'sha davlatdan bevosita olib kirilgan bo\'lishi kerak, ' +
      'hamda kelib chiqish sertifikati (ST-1) taqdim etilishi shart.',
    extracted_at: new Date().toISOString().slice(0, 10),
    total: out.length,
  },
  countries: out,
}

fs.mkdirSync(path.dirname(OUT), { recursive: true })
fs.writeFileSync(OUT, JSON.stringify(file, null, 1))

console.log(`Davlatlar : ${stats.total} dan ${out.length} ta olindi` +
  (stats.skipped ? ` (noma'lum rejim: ${stats.skipped})` : ''))
for (const [rate, n] of Object.entries(stats.byRate)) {
  const reg = REGIME[rate]
  console.log(`  rate=${rate} → boj ×${reg.multiplier}  ${String(n).padStart(3)} ta  (${reg.name})`)
}
console.log(`Offshor   : ${stats.offshore} ta`)
console.log(`O'zbekcha nomi qo'lda yozilgan: ${stats.named} ta`)

// Muhim davlatlarni ko'rsatib qo'yamiz — natijani ko'z bilan tekshirish uchun.
console.log('\nTekshiruv uchun:')
for (const code of ['643', '398', '156', '840', '792', '000']) {
  const c = out.find((x) => x.code === code)
  if (c) console.log(`  ${c.code}  ${c.name_uz.slice(0, 18).padEnd(20)} boj ×${c.duty_multiplier}  (${c.regime})`)
}
console.log(`\n✅ ${OUT}`)
