import { useCallback, useEffect, useState } from 'react'
import {
  api,
  formatCode,
  type ExemptionsResponse,
  type Requirement,
  type RequirementsResponse,
  type UtilFeeResult,
} from '../api'

interface Props {
  /** Qidiruvdan uzatilgan kod. */
  seed?: { code: string; at: number }
  onAsk: (question: string) => void
}

/**
 * Risk baholash — rasmiylashtiruvda NIMA XATO KETISHI mumkinligi.
 *
 * DIQQAT: bu yerda o'ylab topilgan "risk foizi" YO'Q. Har bir band
 * bazadagi aniq yozuvga yoki hisoblashdagi ma'lum bo'shliqqa tayanadi.
 * Raqamli ball qo'yish ishonchli ko'rinardi-yu, ortida hech narsa
 * turmasdi — deklarant esa unga qarab qaror qabul qilardi.
 *
 * Uch manba:
 *   /api/requirements — litsenziya, sertifikat va boshqa talablar
 *   /api/exemptions   — SHARTLI imtiyozlar
 *   /api/utilfee      — utilizatsiya yig'imi qo'llanadimi
 *
 * Ularga hisoblashdagi bilib turib qoldirilgan bo'shliqlar qo'shiladi.
 */
export default function RiskPage({ seed, onAsk }: Props) {
  const [code, setCode] = useState(seed?.code ?? '')
  const [reqs, setReqs] = useState<RequirementsResponse | null>(null)
  const [exempt, setExempt] = useState<ExemptionsResponse | null>(null)
  const [util, setUtil] = useState<UtilFeeResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const check = useCallback(async (raw: string) => {
    const c = raw.replace(/\s/g, '')
    if (!c) return
    setBusy(true)
    setError('')
    setReqs(null); setExempt(null); setUtil(null)
    try {
      const [r, e] = await Promise.all([api.requirements(c), api.exemptions(c)])
      setReqs(r)
      setExempt(e)
      // Utilizatsiya yig'imi — ro'yxatda bo'lmasa xato qaytadi, bu
      // NORMAL holat: yig'im barcha tovarga tegishli emas.
      try {
        setUtil(await api.utilFee({ code: c }))
      } catch {
        setUtil(null)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Tekshirib bo\'lmadi')
    } finally {
      setBusy(false)
    }
  }, [])

  useEffect(() => {
    if (seed?.code) {
      setCode(seed.code)
      void check(seed.code)
    }
  }, [seed, check])

  const byCategory = (cat: string): Requirement[] =>
    (reqs?.requirements ?? []).filter((r) => r.category === cat)

  const permits = [...byCategory('litsenziya'), ...byCategory('sertifikat')]
  const other = (reqs?.requirements ?? []).filter(
    (r) => !['litsenziya', 'sertifikat', 'imtiyoz'].includes(r.category),
  )

  return (
    <div className="page">
      <div className="page-head"><h1>Risk baholash</h1></div>

      <p className="lead">
        TIF TN kodi bo'yicha rasmiylashtiruvda nima to'xtatib qo'yishi
        mumkinligini ko'rsatadi: ruxsatnomalar, shartli imtiyozlar va
        hisoblashdagi ma'lum bo'shliqlar.
      </p>

      <form className="search-bar" onSubmit={(e) => { e.preventDefault(); void check(code) }}>
        <input
          value={code}
          onChange={(e) => setCode(e.target.value)}
          placeholder="TIF TN kodi — masalan 8703 23 194 0"
          aria-label="TIF TN kodi"
        />
        <button type="submit" disabled={busy || !code.trim()}>{busy ? '…' : 'Tekshirish'}</button>
      </form>

      {error && <div className="sb-note err">{error}</div>}

      {reqs && (
        <>
          <p className="meta">
            {formatCode(reqs.code)} · import · stavkalar {reqs.as_of} holatiga
          </p>

          <RiskGroup
            level="high"
            title="Ruxsatnoma va sertifikat"
            empty="Kod oralig'i bo'yicha ruxsatnoma topilmadi."
            items={permits}
          />

          <RiskGroup
            level="mid"
            title="Shartli imtiyoz"
            empty="Bu kodga imtiyoz qoidasi topilmadi."
            items={byCategory('imtiyoz')}
            hint={
              (exempt?.free?.length ?? 0) > 0
                ? `Imtiyoz SHARTLI: ${exempt!.free!.join(', ')} bo'yicha ozod qilish mumkin, ` +
                  'lekin shartini (kim, nima uchun, ro\'yxatda bormi, muddat o\'tmaganmi) tekshirmasdan qo\'llab bo\'lmaydi.'
                : undefined
            }
          />

          {other.length > 0 && (
            <RiskGroup level="low" title="Boshqa talablar" empty="" items={other} />
          )}

          {/* Bazada emas, HISOBLASHDA bilib turib qoldirilgan bo'shliqlar.
              Ular kod bilan bog'liq emas, lekin har rasmiylashtiruvda
              xatoga olib kelishi mumkin. */}
          <section className="risk-group low">
            <h2>Doimiy e'tibor</h2>
            <ul className="risk-list">
              <li>
                <b>Aksiz</b> — stavka TIF TN kodiga bog'lanmagan (Soliq kodeksi
                289¹–289³ uni tovar NOMI bo'yicha beradi). Aroq, sigaret,
                benzinda stavka qat'iy summa; kalkulyator bunday tovarni
                hisoblay olmaydi.
              </li>
              <li>
                <b>Kelib chiqish</b> — ST-1 yoki boshqa sertifikat bilan
                tasdiqlanmasa, boj <b>×2</b> qo'llanadi (Bojxona kodeksi
                300-modda).
              </li>
              {util && !util.needs_measure && (
                <li>
                  <b>Utilizatsiya yig'imi</b> qo'llanadi — {util.category}
                  {util.rate ? `, ${util.rate}× BRV` : ''} (ПКМ 347).
                </li>
              )}
              {util?.needs_measure && (
                <li>
                  <b>Utilizatsiya yig'imi</b> qo'llanadi. Summa uchun{' '}
                  {util.needs_measure} kerak — Kalkulyator bo'limida hisoblang.
                </li>
              )}
            </ul>
          </section>

          {/* ENG MUHIM: bo'sh ro'yxat "hech narsa kerak emas" degani emas. */}
          <div className="warn-box">⚠️ {reqs.note}</div>

          <button
            className="ghost wide"
            onClick={() => onAsk(
              `${formatCode(reqs.code)} kodini import qilishda qanday hujjatlar va ruxsatnomalar kerak? Xavflarni tushuntir.`,
            )}
          >
            Chatda batafsil so'rash
          </button>
        </>
      )}
    </div>
  )
}

function RiskGroup({ level, title, items, empty, hint }: {
  level: 'high' | 'mid' | 'low'
  title: string
  items: Requirement[]
  empty: string
  hint?: string
}) {
  if (items.length === 0 && !empty) return null
  return (
    <section className={'risk-group ' + level}>
      <h2>{title} {items.length > 0 && <span className="risk-count">{items.length}</span>}</h2>
      {hint && <p className="risk-hint">{hint}</p>}
      {items.length === 0 ? (
        <p className="risk-empty">{empty}</p>
      ) : (
        <ul className="risk-list">
          {items.map((r, i) => (
            <li key={r.category + i}>
              <span className="risk-cat">{r.category}</span>
              {r.text}
              {r.law && <span className="risk-law">{r.law}</span>}
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
