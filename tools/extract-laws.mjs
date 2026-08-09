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
import { lexLink } from './lex-links.mjs'

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

// TOPIC ga tushib qolgan, lekin deklarantga keraksiz hujjatlar.
//
// Kalit so'z hujjat NOMIDA uchraydi, ammo hujjat bojxona MUOMALASI haqida
// emas — bojxona ORGANINING ichki-tashkiliy ishi haqida. Eng yirigi:
// "Bojxona instituti bakalavriatiga qabul qilish tartibi" — nomida
// "божхона" bor, mazmuni esa o'qishga kirish kvotalari (35 parcha,
// korpusning 3%i). Bunday matn RAG ni chalg'itadi: "bojxona ... foiz"
// so'rovi qabul kvotasi moddasini tortib kelishi mumkin.
const EXCLUDE = [/приема на учебу/i, /бакалавриат/i]

// ---------------------------------------------------------------- yordamchilar

const unzip = (blob) => {
  if (!blob) return ''
  let b = Buffer.from(blob)
  try { if (b[0] === 0x1f && b[1] === 0x8b) b = zlib.gunzipSync(b) } catch { return '' }
  return b.toString('utf8')
}

// Ustki indeks raqamlari. Qonunlarda "289¹-modda" kabi moddalar bor —
// ular asosiy moddaga keyinchalik qo'shilgan. HTML da <sup>1</sup> bilan
// beriladi.
//
// NEGA MUHIM: teglarni shunchaki olib tashlasak, raqamlar BIRIKIB ketadi —
// "289" + "1" = "2891". Natijada korpusda mavjud bo'lmagan "2891-modda"
// paydo bo'ladi va AI o'sha sarlavhani iqtibos qilib, YO'Q moddaga havola
// beradi. Aksiz stavkalari aynan 289¹–289³ moddalarida, ya'ni bu xato eng
// muhim joyga tushgan edi.
const SUPERSCRIPT = { 0: '⁰', 1: '¹', 2: '²', 3: '³', 4: '⁴', 5: '⁵', 6: '⁶', 7: '⁷', 8: '⁸', 9: '⁹' }
const supDigits = (s) => s.replace(/\d/g, (d) => SUPERSCRIPT[d] ?? d)

const toText = (html) => html
  .replace(/<style[\s\S]*?<\/style>/gi, '').replace(/<script[\s\S]*?<\/script>/gi, '')
  .replace(/<sup[^>]*>\s*([\d\s]+?)\s*<\/sup>/gi, (_, d) => supDigits(d.replace(/\s+/g, '')))
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
const SUP = '⁰¹²³⁴⁵⁶⁷⁸⁹'

function chunkByArticle(text) {
  const re = new RegExp(
    `(?:^|\\n)\\s*(\\d+[${SUP}]*\\s*-\\s*модда\\.|Статья\\s+\\d+[${SUP}]*(?:\\.\\d+)?\\.)`, 'g')
  const marks = [...text.matchAll(re)]
  if (marks.length < 2) return splitLong({ title: '', text })

  const out = []
  for (let i = 0; i < marks.length; i++) {
    const start = marks[i].index
    const end = i + 1 < marks.length ? marks[i + 1].index : text.length
    const body = text.slice(start, end).trim()
    // MUNDARIJA qatorini tashlaymiz. Avval "200 belgidan qisqa bo'lsa tashla"
    // deyilgan edi, lekin bu haqiqiy qisqa moddalarni ham yo'q qilardi —
    // Bojxona kodeksidan 7, 103, 110, 257-modda shu tariqa tushib qolgan edi.
    //
    // Aniqroq belgi: mundarijada modda BIR QATOR bo'lib, faqat sarlavhadan
    // iborat; haqiqiy moddada esa sarlavhadan keyin matn keladi, ya'ni
    // qator ajratgichi bo'ladi (masalan 257-modda — 85 belgi, 2 qator).
    if (!body.includes('\n') && body.length < 160) continue
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
/**
 * Chegaradan uzun abzatsni bo'laklarga ajratadi.
 *
 * Avval gap oxiridan (nuqta, ; yoki yangi qator) bo'lishga urinamiz —
 * shunda parcha o'qiladigan bo'lib qoladi. Gap chegarasi topilmasa
 * (masalan uzun jadval qatori), qat'iy kesamiz: yarim jumla bo'lsa ham,
 * 94 KB lik parchadan ko'ra yaxshiroq.
 */
function hardSplit(para) {
  const out = []
  let rest = para
  while (rest.length > MAX_CHUNK) {
    const window = rest.slice(0, MAX_CHUNK)
    // Oxirgi gap chegarasini qidiramiz, lekin juda oldinga qaytib
    // ketmaymiz — aks holda parchalar juda mayda bo'lib ketardi.
    let cut = Math.max(
      window.lastIndexOf('. '),
      window.lastIndexOf(';'),
      window.lastIndexOf('\n'),
    )
    if (cut < MAX_CHUNK * 0.5) cut = MAX_CHUNK
    else cut += 1
    out.push(rest.slice(0, cut).trim())
    rest = rest.slice(cut)
  }
  if (rest.trim()) out.push(rest.trim())
  return out
}

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
    // Abzatsning O'ZI chegaradan uzun bo'lishi mumkin — bunday holda
    // yuqoridagi shart hech qachon yordam bermaydi va parcha butunligicha
    // qolib ketadi. Bir vaqtlar shu sababli 94 KB lik parcha bor edi:
    // u promptga tushganda ~25 000 token yeb ketardi va kerakli jumla
    // uning ichida ko'milib qolardi. Shuning uchun uzun abzatsni
    // majburan bo'lamiz — avval gap chegarasidan, topilmasa qat'iy.
    if (para.length > MAX_CHUNK) {
      for (const piece of hardSplit(para)) parts.push(piece)
      continue
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
const stats = { core: 0, partial: 0, keyword: 0, skipped: 0, expired: 0, offtopic: 0, uz: 0, ru: 0 }
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
  // Faqat kalit so'z bilan tushganlarni filtrlaymiz — CORE/PARTIAL ataylab
  // tanlangani uchun ularga tegmaymiz.
  if (!isCore && !isPartial && EXCLUDE.some((re) => re.test(name))) {
    stats.offtopic++
    continue
  }

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
      lex: lexLink({ name, docId: r.doc_id, date: r.doc_date }),
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
const withLex = chunks.filter((c) => c.lex).length

console.log(`Hujjatlar : core ${stats.core}, qisman ${stats.partial}, kalit so'z ${stats.keyword}`)
console.log(`            (mavzuga kirmagan: ${stats.skipped}, BEKOR QILINGAN: ${stats.expired},`)
console.log(`             kalit so'z aldagan (EXCLUDE): ${stats.offtopic},`)
console.log(`             o'zbekcha ${stats.uz} / ruscha ${stats.ru})`)

// CORE naqshlari jim qolib ketmasligi kerak — mos kelmasa ogohlantiramiz.
const dead = [...coreHits].filter(([, n]) => n === 0).map(([re]) => re)
if (dead.length) {
  console.log(`\n⚠️  CORE naqshlari hech narsa topmadi (matni bazada yo'q):`)
  for (const re of dead) console.log(`     ${re}`)
}
console.log(`Parchalar : ${chunks.length.toLocaleString('ru-RU')}`)
console.log(`Hajm      : ${mb(bytes)} MB (o'rtacha ${Math.round(bytes / chunks.length)} bayt/parcha)`)
console.log(`lex.uz    : ${withLex} parchada havola`
  + ` (${Math.round((withLex / chunks.length) * 100)}%)`)

// Havolasiz eng ko'p uchraydigan hujjatlarni ko'rsatamiz — lex-links.mjs ni
// kengaytirish uchun aynan shular ustuvor.
const noLex = new Map()
for (const c of chunks) {
  if (c.lex) continue
  const k = `${c.date ?? '—'}  ${c.name}`
  noLex.set(k, (noLex.get(k) ?? 0) + 1)
}
const topNoLex = [...noLex].sort((a, b) => b[1] - a[1]).slice(0, 5)
if (topNoLex.length) {
  console.log(`\n   Havolasiz, eng ko'p uchraydiganlari (lex-links.mjs ga qo'shsa bo'ladi):`)
  for (const [k, n] of topNoLex)
    console.log(`     ${String(n).padStart(4)}  ${k.slice(0, 68)}`)
}

if (args.dry) { db.close(); process.exit(0) }

const out = {
  meta: {
    source: 'ichki manba baza — qonun matnlari',
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
