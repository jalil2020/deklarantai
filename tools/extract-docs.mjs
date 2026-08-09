// TIF TN kodiga bog'liq HUJJAT TALABLARINI ajratib oladi.
//
//   node tools/extract-docs.mjs [--date=2026-07-19] [--out=backend/data/docs.json]
//
// Manba: help.sqlite ning ikkita jadvali.
//   OnlineTnvedInfo     — talab TURI: tavsifi, qaysi rejimga tegishli, imtiyoz bayroqlari
//   OnlineTnvedInfoItem — KOD ORALIQLARI: qaysi kodlarga qaysi talab, qonun va amal muddati
//
// Bo'limlar manbadagi kabi (DocType):
//   Licence     → litsenziya
//   Certificate → sertifikat
//   Facility    → imtiyoz (IsFree* bayroqlari qaysi to'lovdan ozod qilishni ko'rsatadi)
//   Other       → boshqa talablar

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
const ON = args.date ? new Date(args.date) : new Date()
const OUT = args.out || 'backend/data/docs.json'
const SRC = 'backend/data/manba/help.sqlite'

const db = new DatabaseSync(SRC, { readOnly: true })

// ------------------------------------------------------------------ turlar

const CATEGORY = {
  Licence: 'litsenziya',
  Certificate: 'sertifikat',
  Facility: 'imtiyoz',
  Other: 'boshqa',
}

// "Specs_*" turlari HUJJAT EMAS — ular tovar tavsifida (GTD 31-grafa)
// ko'rsatilishi shart bo'lgan ma'lumotlar: dori nomi, dozasi, ishlab
// chiqaruvchi, brend va h.k. Ularni "boshqa talab" deb ko'rsatsak,
// foydalanuvchi hujjat izlab yurardi. Shuning uchun alohida bo'lim.
const isSpec = (type) => /^Specs_/.test(String(type))
const categoryOf = (type, docType) => (isSpec(type) ? 'tavsif' : CATEGORY[docType] || 'boshqa')

// Imtiyoz bayroqlari — qaysi to'lovdan ozod qilinishini ko'rsatadi.
// Nomlar GTD to'lov kodlariga mos: tp=boj(20), an=aksiz(27), nds=QQS(29), ts=yig'im(10).
const FREE = { IsFreeTp: 'boj', IsFreeAn: 'aksiz', IsFreeNds: 'qqs', IsFreeTs: 'yigim' }

/** Kirill o'zbekcha bo'lsa lotinga o'giradi; ruscha matn o'z holicha qoladi. */
const uzText = (uz, ru) => {
  const t = String(uz || '').trim()
  return t ? toLatin(t) : String(ru || '').trim()
}

const types = {}
for (const r of db.prepare('SELECT * FROM OnlineTnvedInfo').all()) {
  const free = Object.entries(FREE).filter(([col]) => Number(r[col]) === 1).map(([, name]) => name)
  types[r.Type] = {
    category: categoryOf(r.Type, r.DocType),
    text: uzText(r.TitleUz, r.Title),
    // Qaysi rejimda talab qilinadi. Hech biri belgilanmagan bo'lsa —
    // cheklov yo'q deb qaraymiz (hammasiga tegishli).
    im: Number(r.IsIm) === 1,
    ex: Number(r.IsEx) === 1,
    tr: Number(r.IsTr) === 1,
    ...(free.length ? { free } : {}),
  }
}

// ------------------------------------------------------------------ oraliqlar

const stats = { total: 0, expired: 0, future: 0, noType: 0, noText: 0, kept: 0 }
const rules = []

for (const it of db.prepare('SELECT * FROM OnlineTnvedInfoItem').all()) {
  stats.total++

  // Amal muddati. Bir xil talab turli sana oraliqlari bilan TAKRORLANADI
  // (masalan sertifikat: 2021-11-17..2024-11-24 va 2024-11-25..∞).
  // Hisob sanasiga amal qilmaydiganini olsak, eskirgan talab ko'rsatilardi.
  const start = it.Start ? new Date(it.Start) : null
  const finish = it.Finish ? new Date(it.Finish) : null
  if (finish && !isNaN(finish) && finish < ON) { stats.expired++; continue }
  if (start && !isNaN(start) && start > ON) { stats.future++; continue }

  const t = types[it.Type]
  // Turi tavsiflanmagan bo'lsa, oraliqning o'z matni bo'lsa ishlatamiz.
  const own = uzText(it.TitleUz, it.Title)
  if (!t && !own && !it.LawNum) { stats.noType++; continue }
  const text = own || (t ? t.text : '')
  if (!text && !it.LawNum) { stats.noText++; continue }

  rules.push({
    type: it.Type,
    // Tur tavsiflanmagan bo'lsa ham Specs_ qoidasi ishlashi kerak.
    category: t ? t.category : categoryOf(it.Type, null),
    min: String(it.MinCode),
    max: String(it.MaxCode),
    ...(it.LawNum ? { law: String(it.LawNum).trim() } : {}),
    ...(own ? { text: own } : {}),
    ...(t && !t.im && !t.ex && !t.tr ? {} : t ? { im: t.im, ex: t.ex, tr: t.tr } : {}),
  })
  stats.kept++
}

// Turlardan faqat ishlatilganlarini qoldiramiz — fayl kichik bo'lsin.
const used = new Set(rules.map((r) => r.type))
const outTypes = Object.fromEntries(Object.entries(types).filter(([k]) => used.has(k)))

rules.sort((a, b) => (a.min < b.min ? -1 : a.min > b.min ? 1 : 0))

const out = {
  meta: {
    source: 'ichki manba baza',
    script: 'tools/extract-docs.mjs',
    note: 'Kod oralig\'i bo\'yicha hujjat talablari. Bo\'limlar: litsenziya, ' +
      'sertifikat, imtiyoz, boshqa. Matn o\'zbekchasi bo\'lsa lotinlashtirilgan, ' +
      'aks holda ruscha asl matn.',
    warning: 'Bu ro\'yxatda kod bo\'yicha aniq ko\'rsatma bermaydigan hujjatlar ' +
      'BO\'LMASLIGI mumkin. Ular ham rasmiylashtiruvda hisobga olinishi kerak.',
    rules_as_of: ON.toISOString().slice(0, 10),
    extracted_at: new Date().toISOString().slice(0, 10),
    types: Object.keys(outTypes).length,
    rules: rules.length,
    expired_excluded: stats.expired,
  },
  types: outTypes,
  rules,
}

fs.mkdirSync(path.dirname(OUT), { recursive: true })
fs.writeFileSync(OUT, JSON.stringify(out, null, 1))

// ------------------------------------------------------------------ hisobot

const byCat = {}
for (const r of rules) byCat[r.category] = (byCat[r.category] || 0) + 1

console.log(`Oraliqlar : ${stats.total} dan ${stats.kept} ta olindi`)
console.log(`            (BEKOR QILINGAN: ${stats.expired}, hali kuchga kirmagan: ${stats.future},`)
console.log(`             tavsifsiz va qonunsiz: ${stats.noType + stats.noText})`)
console.log(`Turlar    : ${Object.keys(outTypes).length}`)
console.log('Bo\'limlar :', Object.entries(byCat).map(([k, v]) => `${k} ${v}`).join(', '))

const noDesc = [...used].filter((t) => !types[t])
if (noDesc.length) {
  console.log(`\n⚠️  ${noDesc.length} ta tur OnlineTnvedInfo da tavsiflanmagan —`)
  console.log(`    ular faqat qonun raqami bilan ko'rsatiladi: ${noDesc.slice(0, 8).join(', ')}`)
}

const size = fs.statSync(OUT).size
console.log(`\nHajm      : ${(size / 1048576).toFixed(1)} MB`)
console.log(`✅ ${OUT}`)
