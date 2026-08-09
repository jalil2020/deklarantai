// TIF TN ierarxiyasining YUQORI DARAJALARINI ajratib oladi: bo'lim va guruh.
//
// NEGA ALOHIDA: extract-hscodes.mjs faqat BARG kodlarni (10 xonali) oladi va
// ota-tugunlarni "path" satriga yig'ib yuboradi. Natijada hscodes.json da
// har kodda "section": "XVI" va "group": "84" bor, lekin ularning NOMI yo'q —
// ya'ni "XVI bo'lim nima?" degan savolga javob yo'q.
//
// Brauzer (sidebar) uchun aynan shu nomlar kerak. Ular manba bazada bor:
// `good` jadvali `parent` orqali daraxt tuzadi, ildizga yaqin tugunlar esa
// bo'lim va guruh sarlavhalari:
//
//	id=0  parent=-1  "ТНВЭД 2025"                      ← ildiz
//	id=1  parent=0   "I БЎЛИМ. ТИРИК ҲАЙВОНЛАР…"        ← bo'lim
//	id=2  parent=1   "01-ГУРУҲ. ТИРИК ҲАЙВОНЛАР"        ← guruh
//
// Chiqish: backend/data/taxonomy.json
//
// Ishga tushirish:
//   node tools/extract-taxonomy.mjs

import { DatabaseSync } from 'node:sqlite'
import { writeFileSync } from 'node:fs'
import { toLatin } from './translit.mjs'

const DB = 'backend/data/manba/help.sqlite'
const OUT = 'backend/data/taxonomy.json'

const db = new DatabaseSync(DB, { readOnly: true })

// Ildiz — parent = -1.
const root = db.prepare('SELECT id, title, title_uz FROM good WHERE parent = -1').get()
if (!root) throw new Error('ildiz tugun topilmadi (parent = -1)')

const version = db.prepare('SELECT MAX(num) v FROM version').get()?.v ?? null

// Bo'limlar — ildizning bevosita bolalari.
const sections = db
  .prepare('SELECT id, title, title_uz FROM good WHERE parent = ? ORDER BY id')
  .all(root.id)

// HOMOGLIFLAR. Kirill matnda rim raqamlari ko'pincha KIRILL harflar bilan
// yoziladi: "ХХI БЎЛИМ" dagi Х — U+0425 (kirill), lotin X emas. Ko'zga
// bir xil ko'rinadi, lekin regex uchun butunlay boshqa belgi — shu sababli
// XXI bo'lim birinchi urinishda tanilmay qolgan edi.
const HOMOGLYPH = new Map([
  ['Х', 'X'], ['х', 'X'], // kirill Х (U+0425) → lotin X
  ['І', 'I'], ['і', 'I'], // kirill І (U+0406) → lotin I
])

// romanize — FAQAT rim raqami uchun. Butun satrga qo'llash mumkin emas:
// "БЎЛИМ" so'zining o'zida Л va М bor, ular ham homoglif deb almashtirilsa
// so'z tanilmay qoladi (birinchi urinishda aynan shunday bo'lgan edi).
function romanize(s) {
  return [...s].map((ch) => HOMOGLYPH.get(ch) ?? ch).join('').toUpperCase()
}

// "I БЎЛИМ. ТИРИК ҲАЙВОНЛАР…" → { num: "I", title: "TIRIK HAYVONLAR…" }
//
// Raqam sinfiga LOTIN va KIRILL homogliflari ham kiritilgan, keyin
// romanize faqat o'sha qismni tozalaydi.
function splitSection(raw) {
  // `s` bayrog'i SHART: uzun sarlavhalar ichida yangi qator bor va usiz
  // `.` uni ushlamaydi, natijada X va XXI bo'limlar tanilmay qolardi.
  const m = /^\s*([IVXLivxlХхІі]+)\s*БЎЛИМ\s*[.\-–—]?\s*(.*)$/su.exec(raw)
  if (!m) return null
  return { num: romanize(m[1]), title: m[2].trim() }
}

// "01-ГУРУҲ. ТИРИК ҲАЙВОНЛАР" → { num: "01", title: "TIRIK HAYVONLAR" }
//
// Ajratgich bir xil emas: "01-ГУРУҲ", "80 ГУРУҲ", "90–ГУРУҲ" (en tire).
// Shuning uchun tire TURLARI ham, oddiy probel ham qabul qilinadi.
function splitGroup(raw) {
  // ГУРУҲ ba'zan ГУРУХ deb yozilgan (Ҳ o'rniga kirill Х) — 72-guruhda
  // aynan shunday. Ikkalasi ham qabul qilinadi.
  const m = /^\s*(\d{1,2})\s*[-–—]?\s*ГУРУ[ҲХ]\s*[.\-–—]?\s*(.*)$/isu.exec(raw)
  if (!m) return null
  return { num: m[1].padStart(2, '0'), title: m[2].trim() }
}

// Sarlavhani ko'rsatishga tayyorlaymiz: manba TO'LIQ BOSH HARFDA yozilgan,
// bu esa uzun matnda o'qishni qiyinlashtiradi. Birinchi harf katta,
// qolgani kichik qilinadi.
function tidy(s) {
  const lat = toLatin(s.toLowerCase())
  return lat.charAt(0).toUpperCase() + lat.slice(1)
}

const out = []
let groupCount = 0

for (const s of sections) {
  const raw = s.title_uz || s.title || ''
  const parsed = splitSection(raw)
  if (!parsed) {
    console.warn(`  ⚠️  bo'lim tanilmadi, tashlab ketildi: ${raw.slice(0, 60)}`)
    continue
  }

  const groups = []
  const children = db
    .prepare('SELECT id, title, title_uz FROM good WHERE parent = ? ORDER BY id')
    .all(s.id)

  for (const g of children) {
    const rawG = g.title_uz || g.title || ''
    const pg = splitGroup(rawG)
    if (!pg) {
      console.warn(`  ⚠️  guruh tanilmadi: ${rawG.slice(0, 60)}`)
      continue
    }
    groups.push({ group: pg.num, title: tidy(pg.title) })
    groupCount++
  }

  out.push({ section: parsed.num, title: tidy(parsed.title), groups })
}

const doc = {
  meta: {
    source: 'ichki manba baza — bo'lim va guruh sarlavhalari',
    source_db_version: version,
    nomenclature: toLatin(String(root.title_uz || root.title || '')),
    extracted_at: new Date().toISOString().slice(0, 10),
    script: "lotin (rasmiy kirill matndan transliteratsiya qilingan)",
    note:
      "Bo'lim va guruh SARLAVHALARI. Kodlarning o'zi hscodes.json da; " +
      'bog'.concat("lanish 'section' va 'group' maydonlari orqali."),
    sections: out.length,
    groups: groupCount,
  },
  sections: out,
}

writeFileSync(OUT, JSON.stringify(doc, null, 1))
console.log(`✓ ${OUT}: ${out.length} bo'lim, ${groupCount} guruh`)
