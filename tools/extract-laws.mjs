// manba bazasidan qonun korpusini chiqarib, RAG uchun parchalarga bo'ladi.
//
// Ishlatish:
//   node tools/extract-laws.mjs [--out=backend/data/laws.json] [--dry]
//
// To'liq korpus ~590 MB — uni butunlay olib bo'lmaydi. Shuning uchun:
//   1) hujjatlar tanlanadi (bojxonaga aloqadorlari),
//   2) matn moddalarga bo'linadi ("N-модда."),
//   3) parchalar mavzu bo'yicha filtrlanadi,
//   4) o'zbekcha kirill matn lotinga o'giriladi.

import { DatabaseSync } from 'node:sqlite'
import zlib from 'node:zlib'
import fs from 'node:fs'
import path from 'node:path'
import { toLatin } from './translit.mjs'

const args = Object.fromEntries(process.argv.slice(2).map((a) => {
  const [k, v] = a.replace(/^--/, '').split('=')
  return [k, v ?? true]
}))
const SRC = 'backend/data/manba/help.sqlite'
const OUT = args.out ?? 'backend/data/laws.json'

// Bojxona uchun kalit atamalar (kirill o'zbek + rus).
//
// Aniqlik muhim: "утилизац" yoki yolg'iz "бож" kabi keng atamalar mavzuga
// aloqasiz parchalarni tortib keladi (masalan chiqindilarni utilizatsiya
// qilish). Shuning uchun faqat bojxonaga xos, uzunroq atamalar olingan.
// Utilizatsiya yig'imi haqidagi qaror CORE ro'yxatida to'liq olinadi.
const TOPIC = new RegExp([
  'божхона', 'тамож', 'пошлин', 'акциз', 'ққс', 'ндс',
  'декларац', 'декларант', 'тиф тн', 'тн вэд', 'импорт', 'экспорт',
  'контрабанда', 'божхона брокер', 'ташқи иқтисод', 'внешнеэконом',
  'божхона тўлов', 'таможенн\\w* сбор', 'таможенн\\w* тариф',
].join('|'), 'i')

// Butunlay olinadigan hujjatlar (nomi bo'yicha) — bular asosiy manba.
const CORE = [
  /Таможенный кодекс/i,
  /ставок таможенных сборов/i,          // ПКМ 55
  /Товарной номенклатуры внешнеэконом/i, // ПКМ 181
  /классификационные коды которых измен/i, // ПКМ 349
  /утилизационн/i,                       // ПКМ 358
]

// Faqat tegishli parchalari olinadigan hujjatlar.
const PARTIAL = [/Налоговый кодекс/i, /административной ответственности/i, /Уголовный кодекс/i]

// ---------------------------------------------------------------- yordamchilar

const unzip = (blob) => {
  if (!blob) return ''
  let b = Buffer.from(blob)
  try { if (b[0] === 0x1f && b[1] === 0x8b) b = zlib.gunzipSync(b) } catch { return '' }
  return b.toString('utf8')
}

const toText = (html) => html
  .replace(/<style[\s\S]*?<\/style>/gi, '').replace(/<script[\s\S]*?<\/script>/gi, '')
  .replace(/<\/t[dh]>/gi, ' | ').replace(/<\/tr>/gi, '\n').replace(/<br\s*\/?>/gi, '\n')
  .replace(/<\/(p|div|li|h\d)>/gi, '\n').replace(/<[^>]+>/g, '')
  .replace(/&nbsp;/g, ' ').replace(/&laquo;/g, '«').replace(/&raquo;/g, '»')
  .replace(/&quot;/g, '"').replace(/&amp;/g, '&')
  .replace(/&#(\d+);/g, (_, n) => String.fromCharCode(n))
  .replace(/[ \t]{2,}/g, ' ').replace(/ *\n */g, '\n').replace(/\n{3,}/g, '\n\n').trim()

const MAX_CHUNK = 4000 // belgi; uzun moddalar bo'linadi

/**
 * Matnni moddalarga bo'ladi. O'zbekchada "N-модда.", ruschada "Статья N.".
 * Mundarija (moddalar ro'yxati) tashlab yuboriladi — undagi bo'laklar juda qisqa.
 */
function chunkByArticle(text) {
  const re = /(?:^|\n)\s*(\d+(?:\s*\d+)?\s*-\s*модда\.|Статья\s+\d+(?:\.\d+)?\.)/g
  const marks = [...text.matchAll(re)]
  if (marks.length < 2) return splitLong({ title: '', text })

  const out = []
  for (let i = 0; i < marks.length; i++) {
    const start = marks[i].index
    const end = i + 1 < marks.length ? marks[i + 1].index : text.length
    const body = text.slice(start, end).trim()
    if (body.length < 200) continue // mundarija qatori — tashlaymiz
    const title = body.split('\n')[0].trim().slice(0, 160)
    out.push(...splitLong({ title, text: body }))
  }
  return out
}

/** Sarlavha bo'lmasa — matnning birinchi mazmunli qatoridan olamiz. */
function firstLine(text) {
  for (const line of text.split('\n')) {
    const s = line.trim()
    if (s.length >= 12) return s.slice(0, 160)
  }
  return text.trim().slice(0, 160)
}

/** Juda uzun parchani bo'laklarga bo'ladi (abzats chegarasida). */
function splitLong({ title, text }) {
  if (!title) title = firstLine(text)
  if (text.length <= MAX_CHUNK) return [{ title, text }]
  const parts = []
  let buf = ''
  for (const para of text.split(/\n\n+/)) {
    if (buf && buf.length + para.length > MAX_CHUNK) {
      parts.push(buf.trim())
      buf = ''
    }
    buf += (buf ? '\n\n' : '') + para
  }
  if (buf.trim()) parts.push(buf.trim())
  return parts.map((p, i) => ({
    title: parts.length > 1 ? `${title} (${i + 1}/${parts.length})` : title,
    text: p,
  }))
}

// ---------------------------------------------------------------- asosiy

const db = new DatabaseSync(SRC, { readOnly: true })
const rows = db.prepare(`
  SELECT id, doc_id, doc_date, doc_name, doc_text, doc_text_uzb, DateStart, DateFinish
  FROM laws WHERE length(doc_text) > 500`).all()

const chunks = []
const stats = { core: 0, partial: 0, keyword: 0, skipped: 0, expired: 0, uz: 0, ru: 0 }
const coreHits = new Map(CORE.map((re) => [String(re), 0]))
const NOW = new Date()

for (const r of rows) {
  const name = r.doc_name || ''

  // Bekor qilingan hujjatlarni tashlaymiz. Bu MUHIM: masalan yig'im
  // stavkalari bo'yicha ikkita bir xil nomli qaror bor — ПКМ 700 (2020,
  // 2025-05-04 da bekor qilingan) va ПКМ 55 (2025). Ikkalasini ham
  // korpusga qo'ysak, RAG eskirgan stavkalarni qaytarishi mumkin.
  const finish = r.DateFinish ? new Date(r.DateFinish) : null
  if (finish && !isNaN(finish) && finish < NOW) { stats.expired++; continue }

  const matchedCore = CORE.filter((re) => re.test(name))
  for (const re of matchedCore) coreHits.set(String(re), coreHits.get(String(re)) + 1)
  const isCore = matchedCore.length > 0
  const isPartial = PARTIAL.some((re) => re.test(name))
  const isKeyword = TOPIC.test(name)
  if (!isCore && !isPartial && !isKeyword) { stats.skipped++; continue }

  // O'zbekchasi bo'lsa — o'shani olamiz (rasmiy matn), aks holda ruscha.
  const uzHtml = unzip(r.doc_text_uzb)
  const lang = uzHtml.length > 1000 ? 'uz' : 'ru'
  const text = toText(lang === 'uz' ? uzHtml : unzip(r.doc_text))
  if (text.length < 400) continue
  stats[lang]++

  const docChunks = chunkByArticle(text)
  let kept = 0
  for (const c of docChunks) {
    // Core hujjatlardan hammasi, qolganlaridan faqat mavzuga tegishlisi.
    if (!isCore && !TOPIC.test(c.text)) continue
    chunks.push({
      doc: r.id,
      // Amal qilish boshlanishi — AI javobda "qaysi paytdan" deyishi uchun.
      since: r.DateStart ? String(r.DateStart).slice(0, 10) : null,
      // Hujjat nomi bazada FAQAT ruscha saqlanadi — uni lotinga o'girish
      // ma'nosiz natija beradi ("Kodeks Respubliki..."), shuning uchun
      // asl holida qoldiriladi. Matn esa o'zbekchadan o'giriladi.
      name: name,
      date: r.doc_date || null,
      title: lang === 'uz' ? toLatin(c.title) : c.title,
      text: lang === 'uz' ? toLatin(c.text) : c.text,
      lang,
    })
    kept++
  }
  if (kept) stats[isCore ? 'core' : isPartial ? 'partial' : 'keyword']++
}

const bytes = Buffer.byteLength(JSON.stringify(chunks))
const mb = (b) => (b / 1024 / 1024).toFixed(1)

console.log(`Hujjatlar : core ${stats.core}, qisman ${stats.partial}, kalit so'z ${stats.keyword}`)
console.log(`            (mavzuga kirmagan: ${stats.skipped}, BEKOR QILINGAN: ${stats.expired},`)
console.log(`             o'zbekcha ${stats.uz} / ruscha ${stats.ru})`)

// CORE naqshlari jim qolib ketmasligi kerak — mos kelmasa ogohlantiramiz.
const dead = [...coreHits].filter(([, n]) => n === 0).map(([re]) => re)
if (dead.length) {
  console.log(`\n⚠️  CORE naqshlari hech narsa topmadi (matni bazada yo'q):`)
  for (const re of dead) console.log(`     ${re}`)
}
console.log(`Parchalar : ${chunks.length.toLocaleString('ru-RU')}`)
console.log(`Hajm      : ${mb(bytes)} MB`)
console.log(`O'rtacha  : ${Math.round(bytes / chunks.length)} bayt/parcha`)

if (args.dry) { db.close(); process.exit(0) }

const out = {
  meta: {
    source: 'ichki manba baza',
    script: "lotin (rasmiy kirill matndan transliteratsiya qilingan)",
    selection: 'Bojxona kodeksi, ПКМ 55 (yig\'im stavkalari), ПКМ 347/358 '
      + '(utilizatsiya yig\'imi) to\'liq; Soliq/Ma\'muriy/Jinoyat kodekslari va '
      + 'boshqa hujjatlardan faqat bojxonaga oid moddalar. '
      + 'DIQQAT: TIF TN ni tasdiqlagan ПКМ 181 va o\'tish jadvali ПКМ 349 '
      + 'matnlari manba bazada yo\'q, shuning uchun korpusga kirmagan.',
    expired_excluded: stats.expired,
    chunking: 'Moddalar bo\'yicha ("N-modda"), 4000 belgidan uzunlari bo\'lingan.',
    docs: stats.core + stats.partial + stats.keyword,
    chunks: chunks.length,
    extracted_at: new Date().toISOString().slice(0, 10),
    note: 'Rasmiy matn o\'zbek tilida. Ruscha versiyalar ayrim hujjatlarda mashina '
      + 'tarjimasi bo\'lib, yuridik kuchga ega emas.',
  },
  chunks,
}

fs.mkdirSync(path.dirname(OUT), { recursive: true })
fs.writeFileSync(OUT, JSON.stringify(out))
console.log(`\n✅ ${OUT}`)

const sample = chunks.find((c) => /Bojxona kodeksi/i.test(c.name)) ?? chunks[0]
console.log('\nNamuna:')
console.log(`  [${sample.lang}] ${sample.name}`)
console.log(`  ${sample.title}`)
console.log('  ' + sample.text.slice(0, 260).replace(/\n/g, '\n  ') + '…')

db.close()
