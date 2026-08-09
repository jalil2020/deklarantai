import { useCallback, useEffect, useState } from 'react'
import {
  api,
  formatCode,
  formatSom,
  type Country,
  type DutyLineItem,
  type DutyResult,
  type HSMatch,
  type RequirementsResponse,
  type UtilFeeResult,
} from '../api'
import BrowsePanel from './BrowsePanel'
import CodeCombo from './CodeCombo'
import CountryCombo, { UNKNOWN } from './CountryCombo'

/** Qidiruvdan uzatiladigan boshlang'ich qiymatlar. */
export interface CalcSeed {
  code: string
  title: string
  importDuty: number
  exportDuty?: number
  vat: number
  /** Kombinatsiyalangan stavkaning qat'iy qismi (dollarda, birlik uchun). */
  specific?: number
  /** Qat'iy qism birligi: kg, dona, l, juft, m². */
  specificUnit?: string
  /** Qo'shimcha o'lchov birligi (dona, litr…). Bo'sh — faqat kg. */
  unit?: string
  at: number
}

interface Props {
  seed?: CalcSeed
}

/**
 * GTD to'lov kodlari — HAMMASI ro'yxatda turadi.
 *
 * NEGA HAMMASI: qo'llanmaydigan to'lovni ro'yxatdan olib tashlash
 * "biz uni hisobga olmadik" bilan "u bu tovarga tegishli emas" ni
 * farqlab bo'lmaydigan holga keltiradi. Deklarant esa aynan shuni
 * bilishi kerak — GTD da har bir kod bo'yicha javob beradi.
 */
const PAYMENTS: { code: string; name: string; rate?: RateKey }[] = [
  { code: '10', name: "Rasmiylashtiruv yig'imi" },
  { code: '12', name: 'Bojxona nazorati' },
  { code: '20', name: 'Bojxona boji', rate: 'duty' },
  { code: '21', name: "Qo'shimcha bojxona boji", rate: 'extra' },
  { code: '27', name: "Aksiz solig'i", rate: 'excise' },
  { code: '29', name: 'QQS', rate: 'vat' },
  { code: '79', name: "Utilizatsiya yig'imi" },
]

type RateKey = 'duty' | 'extra' | 'excise' | 'vat'
type Rates = Record<RateKey, string>
type Tab = 'calc' | 'docs' | 'tree'

export default function CalcPage({ seed }: Props) {
  // Kod IKKI manbadan keladi: qidiruvdan (prop) va Ierarxiya tabidan
  // (mahalliy tanlov). Shuning uchun prop to'g'ridan-to'g'ri emas,
  // mahalliy holat orqali ishlatiladi.
  const [sd, setSd] = useState<CalcSeed | undefined>(seed)
  useEffect(() => setSd(seed), [seed])

  const [tab, setTab] = useState<Tab>('calc')

  // ---- davlatlar (GTD 2, 11, 16-grafalar) ----
  const [countries, setCountries] = useState<Country[]>([])
  const [dispatch, setDispatch] = useState('')  // yuk jo'natuvchi davlat
  const [originC, setOriginC] = useState('')    // kelib chiqish ('' | kod | 'unknown')
  const [trading, setTrading] = useState('')    // savdo qiluvchi davlat
  // Foydalanuvchi qo'lda o'zgartirganini eslab qolamiz — avto-to'ldirish
  // uning tanlovini bosib ketmasin.
  const [originTouched, setOriginTouched] = useState(false)
  const [tradingTouched, setTradingTouched] = useState(false)

  useEffect(() => {
    api.countries()
      .then((r) => setCountries(r.countries))
      .catch(() => setCountries([])) // ro'yxatsiz ham hisoblash ishlaydi
  }, [])

  /**
   * Jo'natuvchi tanlanganda qolgan ikkitasi ham shu davlat bo'ladi
   * (tegilmagan bo'lsa).
   *
   * NEGA: oddiy import holatida uchchala davlat bir xil — Xitoydan
   * Xitoy tovarini Xitoy sotuvchisidan olish. Uchta ro'yxatdan bir xil
   * davlatni uch marta qidirish ma'nosiz ish edi.
   */
  const pickDispatch = (code: string) => {
    setDispatch(code)
    if (!originTouched) setOriginC(code)
    if (!tradingTouched) setTrading(code)
  }

  const byCode = useCallback(
    (code: string) => countries.find((c) => c.code === code),
    [countries],
  )
  const originInfo = originC && originC !== UNKNOWN ? byCode(originC) : undefined

  // ---- kirish ----
  const [regime, setRegime] = useState<'import' | 'export'>('import')
  const [weight, setWeight] = useState('')
  const [qty, setQty] = useState('')
  const [unitPrice, setUnitPrice] = useState('')
  const [invoice, setInvoice] = useState('')
  const [invoiceTouched, setInvoiceTouched] = useState(false)
  const [transport, setTransport] = useState('')
  const [usd, setUsd] = useState('')
  const [inspectDay, setInspectDay] = useState('')
  const [inspectNight, setInspectNight] = useState('')
  const [preliminary, setPreliminary] = useState(false)

  // Stavkalar qatorlarda TAHRIRLANADI (maketdagi ✏️ kabi).
  const [rates, setRates] = useState<Rates>({ duty: '', extra: '0', excise: '', vat: '12' })
  // Qaysi to'lov hisobga kiritiladi.
  const [on, setOn] = useState<Record<string, boolean>>(
    Object.fromEntries(PAYMENTS.map((p) => [p.code, true])),
  )

  // ---- natija ----
  const [res, setRes] = useState<DutyResult | null>(null)
  const [util, setUtil] = useState<UtilFeeResult | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // Qidiruvdan kelgan kod stavkalarni to'ldiradi — ularni qo'lda
  // ko'chirish xato qilinadigan joy.
  useEffect(() => {
    if (!sd) return
    setRates((r) => ({
      ...r,
      duty: String(regime === 'export' ? (sd.exportDuty ?? 0) : sd.importDuty),
      vat: String(sd.vat),
    }))
    setRes(null)
    setUtil(null)
  }, [sd, regime])

  // Birlik narxi × og'irlik → faktura. Foydalanuvchi fakturani o'zi
  // tergan bo'lsa, ustidan yozmaymiz.
  useEffect(() => {
    if (invoiceTouched) return
    const p = num(unitPrice)
    const w = num(weight) ?? num(qty)
    if (p !== undefined && w !== undefined) setInvoice(String(round2(p * w)))
  }, [unitPrice, weight, qty, invoiceTouched])

  /** Qat'iy qism birligidagi miqdor: kg bo'lsa og'irlik, aks holda miqdor. */
  const specificQty = sd?.specificUnit === 'kg' ? num(weight) : num(qty)

  /**
   * Hisoblash.
   *
   * `inc` — qaysi to'lovlar hisobga kiritilgani. U SERVERGA uzatiladi
   * (stavkani nolga tushirish orqali), mahalliy ayirish bilan emas:
   * QQS bazasi bojxona qiymati + boj + aksizdan iborat (SK 254-modda),
   * bojni jadvaldan ayirib qo'yish QQS ni eski bazada qoldirardi.
   *
   * Kelib chiqish davlati KODI yuboriladi — koeffitsientni (0/1/2)
   * backend BK 300-modda bo'yicha o'zi aniqlaydi. Ro'yxat va qoida
   * bitta joyda tursin.
   */
  const calc = useCallback(async (inc: Record<string, boolean>) => {
    setBusy(true)
    setError('')
    try {
      const out = await api.calculateDuty({
        invoice: num(invoice),
        transport: num(transport),
        currency: 'USD',
        usd_rate: num(usd),
        currency_rate: num(usd),
        import_duty: inc['20'] ? (num(rates.duty) ?? 0) : 0,
        extra_duty: inc['21'] ? num(rates.extra) : 0,
        excise: inc['27'] ? (num(rates.excise) ?? 0) : 0,
        vat: inc['29'] ? (num(rates.vat) ?? 0) : 0,
        ...(originC === UNKNOWN
          ? { origin_multiplier: 2 }
          : originC
            ? { origin_country: originC }
            : {}),
        import_duty_specific: inc['20'] ? sd?.specific : undefined,
        specific_quantity: specificQty,
        inspect_day: inc['12'] ? num(inspectDay) : 0,
        inspect_night: inc['12'] ? num(inspectNight) : 0,
        fee_exempt: !inc['10'],
        preliminary,
      })
      setRes(out)

      // 79 — alohida endpoint. Ro'yxatda yo'q kod uchun xato qaytadi
      // va bu NORMAL: yig'im hamma tovarga tegishli emas.
      if (sd?.code) {
        try {
          setUtil(await api.utilFee({ code: sd.code, measure: num(qty) ?? num(weight) }))
        } catch { setUtil(null) }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Hisoblab bo\'lmadi')
      setRes(null)
    } finally {
      setBusy(false)
    }
  }, [invoice, transport, usd, rates, originC, sd, specificQty, weight, qty,
      inspectDay, inspectNight, preliminary])

  const submit = (e?: React.FormEvent) => {
    e?.preventDefault()
    void calc(on)
  }

  /** Qatorni yoqish/o'chirish — DARROV serverda qayta hisoblanadi. */
  const toggle = (code: string) => {
    const next = { ...on, [code]: !on[code] }
    setOn(next)
    if (res) void calc(next)
  }

  // Stavka qatorda o'zgarsa — blur da qayta hisoblanadi (maketdagi ✏️).
  function editRate(key: RateKey, value: string) {
    setRates((r) => ({ ...r, [key]: value }))
  }

  /** Topilgan kodni kalkulyatorga qo'yadi (stavkalar bilan birga). */
  const applyMatch = (m: HSMatch) => {
    setError('')
    setSd({
      code: m.code.code,
      title: m.code.name_uz || m.code.name_ru,
      importDuty: m.code.import_duty,
      exportDuty: m.code.export_duty,
      vat: m.code.vat,
      specific: m.code.import_duty_specific,
      specificUnit: m.code.import_duty_specific_unit,
      unit: m.code.unit,
      at: Date.now(),
    })
    setTab('calc')
  }

  /**
   * Ierarxiyadan kelgan KOD bo'yicha yuklash.
   *
   * Daraxt faqat kod raqamini beradi, stavkalarni esa yo'q (browse
   * bargida qat'iy qism va birlik saqlanmaydi) — shuning uchun to'liq
   * yozuv qidiruvdan olinadi.
   */
  const loadCode = async (raw: string) => {
    const code = raw.replace(/\D/g, '')
    if (code.length < 4) return
    setError('')
    try {
      const r = await api.searchHS(code, false, 20)
      // Hech narsa topilmasa backend `matches: null` qaytaradi — bo'sh
      // ro'yxat emas. Buni test topdi: 9999999999 uchun .find yiqilgan.
      const list = r.matches ?? []
      // 10 xonali kod uchun AYNAN o'sha topilishi shart — "yaqinini"
      // jimgina yuklash boshqa stavka bilan hisoblash degani.
      const m = list.find((x) => x.code.code === code) ?? (code.length < 10 ? list[0] : undefined)
      if (!m) {
        setError(`Kod topilmadi: ${formatCode(code)}. Raqamlarni tekshiring.`)
        return
      }
      applyMatch(m)
    } catch {
      setError('Kod ma\'lumotini olib bo\'lmadi')
    }
  }

  const byCodeItems = groupByCode(res?.items ?? [])
  // 79 alohida endpointdan keladi, shuning uchun jamiga QO'LDA qo'shiladi.
  const utilAmount = on['79'] && util && !util.needs_measure ? util.amount : 0
  const total = (res?.total ?? 0) + utilAmount
  const excluded = PAYMENTS.some((p) => on[p.code] === false)

  const dispatchInfo = byCode(dispatch)
  const tradingInfo = byCode(trading)
  const offshoreParty = [dispatchInfo, tradingInfo].find((c) => c?.offshore)

  return (
    <div className="page">
      <div className="page-head">
        <h1>Kalkulyator</h1>
      </div>

      {/* GTD davlat grafalari — maketdagi uch tanlagich. Hisobga FAQAT
          kelib chiqish ta'sir qiladi; qolgan ikkitasi GTD uchun va
          ogohlantirishlar uchun. */}
      <div className="gtd-row">
        <Field label="Yuk jo'natuvchi">
          <CountryCombo countries={countries} value={dispatch} onChange={pickDispatch} />
        </Field>
        <Field
          label="Kelib chiqish"
          hint={originHint(originC, originInfo)}
        >
          <CountryCombo
            countries={countries}
            value={originC}
            onChange={(v) => { setOriginC(v); setOriginTouched(true) }}
            unknownOption
          />
        </Field>
        <Field label="Savdo qiluvchi">
          <CountryCombo
            countries={countries}
            value={trading}
            onChange={(v) => { setTrading(v); setTradingTouched(true) }}
          />
        </Field>
      </div>

      {offshoreParty && (
        <div className="warn-box small">
          ⚠️ <b>{offshoreParty.name_uz}</b> — offshor zona. Boj rejimiga
          ta'sir qilmaydi, lekin qo'shimcha nazorat va hujjat talablari
          bo'lishi mumkin.
        </div>
      )}
      {originC && originC !== UNKNOWN && dispatch && originC !== dispatch && (
        <div className="warn-box small">
          ⚠️ Kelib chiqish jo'natuvchi davlatdan farq qiladi — kelib
          chiqish <b>sertifikat bilan tasdiqlanishi</b> kerak (ST-1 yoki
          shakl A), aks holda boj ×2 qo'llanadi.
        </div>
      )}

      {/* Panel tab'lari — maketdagi Kalkulyator | Hujjatlar | Ierarxiya.
          O'ngda tanlangan kod va uni tozalash. */}
      <div className="calc-tabs">
        <div className="seg" role="tablist">
          <button role="tab" aria-selected={tab === 'calc'} onClick={() => setTab('calc')}>🧮 Kalkulyator</button>
          <button
            role="tab"
            aria-selected={tab === 'docs'}
            onClick={() => setTab('docs')}
            disabled={!sd}
            title={sd ? undefined : 'Avval kod tanlang (Ierarxiya yoki Qidiruv)'}
          >📄 Hujjatlar</button>
          <button role="tab" aria-selected={tab === 'tree'} onClick={() => setTab('tree')}>🌳 Ierarxiya</button>
        </div>
        {/* Kod maydoni — YOZGANDA qidiradi. Qidiruv bo'limiga o'tish
            shart emas: kod raqami ham, tovar nomi ham ishlaydi. */}
        <span className="code-box">
          <CodeCombo
            value={sd?.code ?? ''}
            onPick={applyMatch}
            onClear={() => { setSd(undefined); setRes(null); setUtil(null) }}
          />
          {sd && <span className="head-unit">({sd.unit || 'kg'})</span>}
        </span>
      </div>

      {tab === 'tree' && (
        <BrowsePanel
          onPick={(code) => { void loadCode(code) }}
          initialHeading={sd?.code.slice(0, 4)}
        />
      )}

      {tab === 'docs' && sd && <DocsTab code={sd.code} />}

      {tab === 'calc' && (
        <>
          {sd && (
            <div className="seed">
              {sd.title}
              {sd.specific != null && (
                <div className="seed-rate">
                  Kombinatsiyalangan stavka: <b>{rates.duty || sd.importDuty}%</b>, lekin
                  1 {sd.specificUnit} uchun <b>${sd.specific}</b> dan kam emas
                </div>
              )}
            </div>
          )}

          <form className="calc-form" onSubmit={submit}>
            <Field label="Bojxona rejimi">
              <select value={regime} onChange={(e) => setRegime(e.target.value as 'import' | 'export')}>
                <option value="import">Import</option>
                <option value="export">Eksport</option>
              </select>
            </Field>

            <Field label="Mahsulot og'irligi, kg">
              <input type="number" min="0" step="any" value={weight}
                onChange={(e) => setWeight(e.target.value)} placeholder="0" />
            </Field>

            {/* Qo'shimcha birlik faqat kodda bo'lsa. Yo'q bo'lsa maydon
                O'CHIQ turadi — maketdagi kabi: "bu tovar faqat kg da". */}
            <Field
              label={sd?.unit ? `Mahsulot miqdori, ${sd.unit}` : 'Mahsulot miqdori'}
              hint={sd && !sd.unit ? 'Bu kodda qo\'shimcha o\'lchov birligi yo\'q' : undefined}
            >
              <input type="number" min="0" step="any" value={qty}
                onChange={(e) => setQty(e.target.value)}
                placeholder={sd && !sd.unit ? '—' : '0'}
                disabled={!!sd && !sd.unit} />
            </Field>

            <Field label={`Birlik narxi (${sd?.unit || 'kg'}), USD`}>
              <input type="number" min="0" step="any" value={unitPrice}
                onChange={(e) => setUnitPrice(e.target.value)} placeholder="0" />
            </Field>

            <Field label="Faktura qiymati, USD" required
              hint={!invoiceTouched && unitPrice ? 'Birlik narxidan hisoblandi' : undefined}>
              <input type="number" min="0" step="any" value={invoice}
                onChange={(e) => { setInvoice(e.target.value); setInvoiceTouched(true) }}
                placeholder="0" required />
            </Field>

            <Field label="Transport xarajati, USD">
              <input type="number" min="0" step="any" value={transport}
                onChange={(e) => setTransport(e.target.value)} placeholder="0" />
            </Field>

            <Field label="Yig'im (ish vaqti), soat" hint="0,25 × BRV / soat">
              <input type="number" min="0" step="any" value={inspectDay}
                onChange={(e) => setInspectDay(e.target.value)} placeholder="0" />
            </Field>

            <Field label="Yig'im (tashqari), soat" hint="2 × BRV / soat">
              <input type="number" min="0" step="any" value={inspectNight}
                onChange={(e) => setInspectNight(e.target.value)} placeholder="0" />
            </Field>

            <Field label="USD kursi, so'm" hint="Bo'sh qoldirilsa Markaziy bank kursi olinadi">
              <input type="number" min="0" step="any" value={usd}
                onChange={(e) => setUsd(e.target.value)} placeholder="avtomatik" />
            </Field>

            <label className="check-wide">
              <input type="checkbox" checked={preliminary}
                onChange={(e) => setPreliminary(e.target.checked)} />
              Oldindan deklaratsiya (yig'imga 20% chegirma)
            </label>

            <button type="submit" className="calc-go" disabled={busy}>
              {busy ? '…' : '🧮 Hisoblash'}
            </button>
          </form>

          {regime === 'export' && (
            <div className="warn-box">
              ⚠️ <b>Eksport rejimi cheklangan.</b> Boj stavkasi eksport
              tarifidan olinadi, lekin QQS nol stavkasi, aksiz va
              utilizatsiya yig'imi eksportda boshqa qoidalar bo'yicha
              ishlaydi — ularni chatda tekshiring.
            </div>
          )}

          <div className="warn-box">
            ⚠️ <b>Aksizli tovarlarga ehtiyot bo'ling.</b> Kalkulyator aksizni
            faqat <b>foizda</b> hisoblaydi. Aroq, sigaret, benzin kabi
            tovarlarda stavka qat'iy summa (so'm/litr) — bunday tovarni bu
            yerda hisoblab bo'lmaydi.
          </div>

          {error && <div className="sb-note err">{error}</div>}

          {res && (
            <div className="pay-list">
              {PAYMENTS.map((p) => (
                <PayRow
                  key={p.code}
                  code={p.code}
                  name={p.name}
                  items={byCodeItems[p.code]}
                  util={p.code === '79' ? util : null}
                  rateKey={p.rate}
                  rates={rates}
                  specific={p.code === '20' ? sd : undefined}
                  checked={on[p.code] ?? false}
                  onToggle={() => toggle(p.code)}
                  onRate={editRate}
                  onRecalc={submit}
                />
              ))}

              <div className="pay-row sum">
                <span className="pay-name">Bojxona qiymati</span>
                <span className="pay-amount">{formatSom(res.customs_value)}</span>
              </div>
              <div className="pay-row total">
                <span className="pay-name">Jami to'lovlar</span>
                <span className="pay-amount">{formatSom(total)}</span>
              </div>

              {excluded && (
                <div className="warn-box small">
                  ⚠️ Ba'zi to'lovlar hisobdan CHIQARILGAN. To'lovni chiqarish
                  uchun huquqiy asos kerak (imtiyoz, rejim) — shartini
                  tekshirmasdan qo'llab bo'lmaydi.
                </div>
              )}

              <p className="meta">BRV {formatSom(res.brv)}</p>
            </div>
          )}
        </>
      )}
    </div>
  )
}

/** Kelib chiqish tanlagichi ostidagi izoh — rejim darrov ko'rinsin. */
function originHint(value: string, info?: Country): string | undefined {
  if (value === UNKNOWN) return 'Boj stavkasi ikki baravar (BK 300-modda)'
  if (!info) return undefined
  switch (info.duty_multiplier) {
    case 0: return 'Erkin savdo — boj olinmaydi'
    case 2: return 'Rejim yo\'q — boj ×2'
    default: return 'Eng qulaylik rejimi — tarifdagi stavka'
  }
}

// ------------------------------------------------------------ hujjatlar tabi

/**
 * Kod bo'yicha hujjat talablari — maketdagi "Hujjatlar" tabi.
 *
 * Ma'lumot Risk bo'limidagi bilan bitta manbadan (/api/requirements),
 * lekin bu yerda QISQA ro'yxat: kalkulyator yonida turib tez qarash
 * uchun. To'liq tahlil Risk bo'limida.
 */
function DocsTab({ code }: { code: string }) {
  const [reqs, setReqs] = useState<RequirementsResponse | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    setReqs(null)
    setError('')
    api.requirements(code)
      .then(setReqs)
      .catch((e) => setError(e instanceof Error ? e.message : 'Yuklab bo\'lmadi'))
  }, [code])

  if (error) return <div className="sb-note err">{error}</div>
  if (!reqs) return <div className="sb-note">Yuklanmoqda…</div>

  return (
    <div className="docs-tab">
      {reqs.requirements.length === 0 ? (
        <div className="empty">Kod oralig'i bo'yicha yozuv topilmadi.</div>
      ) : (
        <ul className="risk-list">
          {reqs.requirements.map((r, i) => (
            <li key={r.category + i}>
              <span className="risk-cat">{r.category}</span>
              {r.text}
              {r.law && <span className="risk-law">{r.law}</span>}
            </li>
          ))}
        </ul>
      )}
      {/* Bo'sh ro'yxat "hech narsa kerak emas" degani EMAS. */}
      <div className="warn-box small">⚠️ {reqs.note}</div>
    </div>
  )
}

// -------------------------------------------------------------- to'lov qatori

function PayRow({
  code, name, items, util, rateKey, rates, specific, checked, onToggle, onRate, onRecalc,
}: {
  code: string
  name: string
  items?: DutyLineItem[]
  util: UtilFeeResult | null
  rateKey?: RateKey
  rates: Rates
  specific?: CalcSeed
  checked: boolean
  onToggle: () => void
  onRate: (k: RateKey, v: string) => void
  onRecalc: () => void
}) {
  // 79 alohida endpointdan keladi.
  const amount = code === '79'
    ? (util && !util.needs_measure ? util.amount : null)
    : (items ? items.reduce((s, i) => s + i.amount, 0) : null)

  const applies = amount !== null
  // Nuqta — stavka BO'SH, ya'ni noma'lum. Nol stavka bilan
  // aralashtirmaslik kerak: 0% — bu "to'lov yo'q" degan JAVOB,
  // bo'sh maydon esa "javob yo'q".
  const unknownRate = rateKey !== undefined && rates[rateKey].trim() === ''

  return (
    <div className={'pay-row' + (applies ? '' : ' na')}>
      <span className="pay-code">{code}.</span>
      <span className="pay-name">{name}</span>

      <span className="pay-rate">
        {code === '10' && applies && <span className="chip-rate">1–25 BRV</span>}

        {/* Stavka HAR DOIM tahrirlanadi — qator hozir qo'llanmasa ham.
            Aks holda aksiz stavkasini kiritishning iloji bo'lmasdi. */}
        {rateKey && (
          <>
            <input
              className="rate-input"
              type="number"
              min="0"
              step="any"
              value={rates[rateKey]}
              onChange={(e) => onRate(rateKey, e.target.value)}
              onBlur={onRecalc}
              aria-label={`${name} stavkasi, %`}
              title="Stavkani tahrirlash"
            />
            <span className="rate-pct">%</span>
          </>
        )}

        {code === '20' && specific?.specific != null && (
          <span className="chip-rate combo" title="Kombinatsiyalangan: kattasi olinadi">
            ${specific.specific} | {specific.specificUnit}
          </span>
        )}

        {code === '79' && util?.needs_measure && (
          <span className="chip-rate warn">{util.needs_measure} kerak</span>
        )}
        {unknownRate && <span className="dot" title="Stavka kiritilmagan — noma'lum" />}
      </span>

      <span className="pay-amount">{applies ? formatSom(amount) : '–'}</span>

      {applies ? (
        <input
          type="checkbox"
          className="pay-check"
          checked={checked}
          onChange={onToggle}
          aria-label={`${name} — hisobga kiritish`}
          title="Hisobga kiritish"
        />
      ) : <span className="pay-check" />}

      {/* Izoh — qaysi qism qo'llangani, kelib chiqish ta'siri va h.k. */}
      {applies && items?.some((i) => i.note) && (
        <p className="pay-note">{items.filter((i) => i.note).map((i) => i.note).join(' · ')}</p>
      )}
      {code === '79' && util?.note && <p className="pay-note">{util.note}</p>}
    </div>
  )
}

// ---------------------------------------------------------------- yordamchi

function Field({ label, hint, required, children }: {
  label: string
  hint?: string
  required?: boolean
  children: React.ReactNode
}) {
  return (
    <label className="field">
      <span className="field-label">{label}{required && <b aria-hidden="true"> *</b>}</span>
      {children}
      {hint && <span className="field-hint">{hint}</span>}
    </label>
  )
}

/** Bo'sh maydon 0 emas, "berilmagan" — backend ularni farqlaydi. */
function num(v: string): number | undefined {
  const t = v.trim()
  if (t === '') return undefined
  const n = Number(t)
  return Number.isFinite(n) ? n : undefined
}

function round2(v: number): number {
  return Math.round(v * 100) / 100
}

/** Bitta kodda bir necha qator bo'lishi mumkin (12 — kunduzgi va tungi ko'rik). */
function groupByCode(items: DutyLineItem[]): Record<string, DutyLineItem[]> {
  const out: Record<string, DutyLineItem[]> = {}
  for (const i of items) (out[i.code] ??= []).push(i)
  return out
}
