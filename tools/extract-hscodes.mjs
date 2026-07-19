// manba bazasidan TIF TN kodlarini chiqarib, backend uchun hscodes.json yasaydi.
//
// Ishlatish:
//   node tools/extract-hscodes.mjs [--date=2026-07-19] [--out=backend/data/hscodes.json]
//
// Manba: backend/data/manba/{help,info}.sqlite  (git'ga kirmaydi)
// Natija: backend/data/hscodes.json                 (git'ga kiradi)

import { DatabaseSync } from 'node:sqlite'
import fs from 'node:fs'
import path from 'node:path'
import { toLatin } from './translit.mjs'
import { UNITS_UZ } from './units-uz.mjs'

const args = Object.fromEntries(
  process.argv.slice(2).map((a) => {
    const [k, v] = a.replace(/^--/, '').split('=')
    return [k, v ?? true]
  }),
)

const SRC = 'backend/data/manba'
const OUT = args.out ?? 'backend/data/hscodes.json'
const ON = args.date ? new Date(args.date) : new Date()

// ---------------------------------------------------------------- yordamchilar

/** "dd.mm.yyyy" → Date (yaroqsiz bo'lsa null). */
function parseDate(s) {
  if (!s) return null
  const m = String(s).trim().match(/^(\d{2})\.(\d{2})\.(\d{4})$/)
  return m ? new Date(+m[3], +m[2] - 1, +m[1]) : null
}

/**
 * Stavka maydonini o'qiydi: "[qonun|boshlanish|tugash|foiz|...][...]"
 *
 * Bir nechta blok ON sanasiga BIR VAQTDA amal qilishi mumkin — masalan
 * bosqichma-bosqich oshirilayotgan tarifda:
 *   [326|01.07.2025||10|…][326|01.01.2026||20|…][326|01.01.2028||30|…]
 * Bu yerda 2026-07-19 sanasida birinchi ikkitasi ham "amalda" (tugash
 * sanasi yo'q), lekin ustun bo'lgani — KEYINGI qabul qilingani, ya'ni 20%.
 * Shuning uchun eng kech boshlangan mos blok tanlanadi.
 *
 * Qaytaradi: { percent, law, extra[], since } yoki null.
 */
// Bir vaqtda bir nechta blok amal qilgan holatlar soni — diagnostika uchun.
// Bu holat jimgina noto'g'ri stavka berishi mumkin, shuning uchun ko'rinib tursin.
let overlapping = 0

function parseRate(raw) {
  if (!raw) return null
  let best = null
  let bestStart = null
  let applicable = 0
  for (const m of String(raw).matchAll(/\[([^\]]*)\]/g)) {
    const f = m[1].split('|')
    const start = parseDate(f[1])
    const finish = parseDate(f[2])
    if (start && ON < start) continue
    if (finish && ON > finish) continue
    applicable++
    if (applicable === 2) overlapping++
    // Sanasi yo'q blok eng quyi ustuvorlikda (bestStart = null).
    if (best && bestStart && (!start || start <= bestStart)) continue
    const pct = (f[3] ?? '').trim().replace(',', '.')
    best = {
      percent: pct === '' ? 0 : Number(pct),
      law: (f[0] ?? '').trim() || null,
      extra: f.slice(4).map((x) => x.trim()).filter(Boolean),
      since: f[1]?.trim() || null,
    }
    bestStart = start
  }
  return best
}

/** Rim raqami (1..21). Bo'lim nomlarini tuzatish uchun. */
const ROMAN = ['', 'I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII', 'IX', 'X',
  'XI', 'XII', 'XIII', 'XIV', 'XV', 'XVI', 'XVII', 'XVIII', 'XIX', 'XX', 'XXI']

// ---------------------------------------------------------------- baza

const help = new DatabaseSync(path.join(SRC, 'help.sqlite'), { readOnly: true })
const info = new DatabaseSync(path.join(SRC, 'info.sqlite'), { readOnly: true })

// Qo'shimcha o'lchov birliklari: kod → { ru, uz }.
// O'zbekchasi qo'lda jadvaldan (units-uz.mjs); bazada faqat ruschasi bor.
const units = new Map()
const missingUz = []
for (const u of info.prepare('SELECT code, name, title FROM unit').all()) {
  const code = String(u.code)
  const uz = UNITS_UZ[code]
  if (!uz) missingUz.push(`${code} (${u.name})`)
  units.set(code, { ru: u.name, uz: uz ?? u.name })
}

// Butun daraxtni xotiraga olamiz (20 929 tugun — arzon).
const rows = help.prepare(`
  SELECT id, parent, code, title, title_uz, unit_code, tp, tpEx, nds, an, "end"
  FROM good ORDER BY id`).all()
const byId = new Map(rows.map((r) => [r.id, r]))

// --- Bo'lim nomlaridagi rim raqamlarini tuzatish -----------------------------
// manba bazasida takroriy raqamlar bor (VII ikki marta, XII ikki marta).
// Bo'limlar tartibda kelgani uchun pozitsiya bo'yicha to'g'rilaymiz.
const root = rows.find((r) => r.parent === -1)
const sections = rows.filter((r) => r.parent === root.id && /^Раздел /.test(r.title || ''))
const fixes = []
sections.forEach((s, i) => {
  const want = ROMAN[i + 1]
  const got = (s.title.match(/^Раздел\s+([IVXL]+)\./) || [])[1]
  if (got && got !== want) {
    fixes.push(`${got} → ${want}: ${s.title.slice(0, 52)}…`)
    s.title = s.title.replace(/^Раздел\s+[IVXL]+\./, `Раздел ${want}.`)
  }
  s._section = want
})

// Har bir tugun uchun bo'limni belgilaymiz (pastga tarqatish uchun kesh).
function sectionOf(row) {
  let r = row
  while (r && r.parent !== root.id) r = byId.get(r.parent)
  return r?._section ?? null
}

/** Ota tugunlar bo'ylab yuqoriga: [ildizdan pastga] zanjir. */
function chainOf(row) {
  const out = []
  for (let r = row; r && r.parent !== -1; r = byId.get(r.parent)) out.unshift(r)
  return out
}

/** Zanjirdan to'liq tavsif yig'adi (bo'sh va bo'lim/guruh sarlavhalarisiz). */
function pathOf(chain, field) {
  return chain
    .filter((r) => !/^(Раздел|Группа)\s/.test(r.title || ''))
    .map((r) => (r[field] || '').trim().replace(/[;:]\s*$/, ''))
    .filter(Boolean)
    .join('; ')
}

// ---------------------------------------------------------------- chiqarish

const codes = []
const stats = { noUnit: 0, noDuty: 0, noVat: 0, exciseKnown: 0 }

for (const r of rows) {
  const bare = String(r.code || '').replace(/\s/g, '')
  if (bare.length !== 10) continue // faqat to'liq 10 xonali kodlar

  const chain = chainOf(r)
  const tp = parseRate(r.tp)
  const nds = parseRate(r.nds)
  const tpEx = parseRate(r.tpEx)
  const an = parseRate(r.an)
  const unit = units.get(String(r.unit_code)) ?? null

  if (!unit) stats.noUnit++
  if (!tp) stats.noDuty++
  if (!nds) stats.noVat++
  if (an) stats.exciseKnown++

  codes.push({
    code: bare,
    name_ru: (r.title || '').trim(),
    name_uz: toLatin((r.title_uz || '').trim()),
    path_ru: pathOf(chain, 'title'),
    path_uz: toLatin(pathOf(chain, 'title_uz')),
    section: sectionOf(r),
    group: bare.slice(0, 2),
    // Qo'shimcha o'lchov birligi. null = faqat kg (asosiy birlik).
    unit_code: r.unit_code || null,
    unit: unit?.uz ?? null,
    unit_ru: unit?.ru ?? null,
    import_duty: tp?.percent ?? 0,
    export_duty: tpEx?.percent ?? 0,
    // Aksiz: manba bazada "an" maydoni bo'sh (joriy TIF TN da hech bir kodda
    // to'ldirilmagan). Uni 0 deb yozish YOLG'ON bo'lardi — "aksiz yo'q" degan
    // ma'noni beradi, holbuki aroq, sigaret, benzin aksizli. Shuning uchun
    // ma'lumot bo'lmasa, maydon UMUMAN yozilmaydi (noma'lum).
    // Haqiqiy stavkalar Soliq kodeksining 289¹–289³ moddalarida, tovar nomi
    // bo'yicha — ular qonun korpusida bor.
    ...(an ? { excise: an.percent } : {}),
    vat: nds?.percent ?? 0,
    duty_law: tp?.law ?? null,
  })
}

// Baza versiyasi (yangilanish raqami)
const dbVersion = help.prepare('SELECT MAX(num) v FROM version').get()?.v ?? null

const out = {
  meta: {
    nomenclature: 'TIF TN 2025',
    legal_basis: 'ПКМ № 181 (14.05.2025) — TIF TN tasdiqlangan, 01.06.2025 dan amalda',
    // Matni manba bazada yo'q (IsLoad=0), shuning uchun qonun korpusiga
    // kirmagan — lekin havolasi bo'lsa, foydalanuvchi asosni tekshira oladi.
    legal_basis_lex: 'https://lex.uz/docs/7533469',
    // Boj stavkalarining asosi: ПП-3818, 1-ilova (import boji stavkalari).
    duty_rates_basis: 'ПП-3818 (29.06.2018), 1-ilova',
    duty_rates_lex: 'https://lex.uz/docs/3802366',
    transition_list: 'ПКМ № 349 (04.06.2025) — kodlar o\'zgarishi ro\'yxati',
    international_basis: 'Garmonizatsiyalangan tizim konventsiyasi (Bryussel, 14.06.1983)',
    rates_as_of: ON.toISOString().slice(0, 10),
    source: 'ichki manba baza',
    source_db_version: dbVersion,
    extracted_at: new Date().toISOString().slice(0, 10),
    total_codes: codes.length,
    note: 'Stavkalar rates_as_of sanasiga olingan va vaqt o\'tishi bilan o\'zgaradi. '
      + 'Rasmiy manba: customs.uz, lex.uz.',
    unit_note: 'TIF TN da asosiy o\'lchov doim kg (netto). "unit" — QO\'SHIMCHA '
      + 'birlik; null bo\'lsa, faqat kg qo\'llaniladi.',
    excise_note: 'DIQQAT: bu bazada aksiz stavkalari YO\'Q — manba bazaning "an" '
      + 'maydoni bo\'sh. "excise" maydoni yozilmagan bo\'lsa, bu "aksiz yo\'q" '
      + 'DEGANI EMAS, balki "bu bazada ma\'lumot yo\'q" degani. Haqiqiy stavkalar '
      + 'Soliq kodeksining 289¹ (tamaki), 289² (alkogol), 289³ (neft mahsulotlari '
      + 'va boshqalar) moddalarida, tovar nomi bo\'yicha berilgan — ular qonun '
      + 'korpusida (laws.json) mavjud.',
    excise_known_codes: stats.exciseKnown,
  },
  codes,
}

fs.mkdirSync(path.dirname(OUT), { recursive: true })
// Generatsiya qilingan fayl — o'qish uchun emas, hajmni tejaymiz.
fs.writeFileSync(OUT, JSON.stringify(out))

// ---------------------------------------------------------------- hisobot

const mb = (fs.statSync(OUT).size / 1024 / 1024).toFixed(2)
console.log(`✅ ${OUT} — ${codes.length.toLocaleString('ru-RU')} kod, ${mb} MB`)
console.log(`   stavkalar sanasi: ${out.meta.rates_as_of}   baza versiyasi: ${dbVersion}`)
if (fixes.length) {
  console.log(`\n⚠️  Bo'lim raqamlari tuzatildi (${fixes.length} ta):`)
  for (const f of fixes) console.log('     ' + f)
}
if (missingUz.length)
  console.log(`\n⚠️  O'zbekcha nomi yo'q birliklar: ${missingUz.join(', ')}`)
console.log(`\n   qo'shimcha o'lchovsiz (faqat kg): ${stats.noUnit}`
  + `   bojsiz: ${stats.noDuty}   QQSsiz: ${stats.noVat}`)
console.log(`   bir vaqtda >1 stavka bloki amal qilgan holatlar: ${overlapping}`
  + ` (eng kech boshlangani olindi)`)
console.log(`   aksiz stavkasi ma'lum bo'lgan kodlar: ${stats.exciseKnown} / ${codes.length}`
  + (stats.exciseKnown === 0 ? "  ← maydon yozilmadi (noma'lum, 0 EMAS)" : ''))

const sample = codes.find((c) => c.code === '8701211019') ?? codes[0]
console.log('\n   Namuna:')
console.log('  ' + JSON.stringify(sample, null, 2).split('\n').join('\n  '))

help.close()
info.close()
