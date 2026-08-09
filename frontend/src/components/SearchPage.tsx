import { useState } from 'react'
import {
  api,
  formatCode,
  type ExemptionsResponse,
  type HSMatch,
} from '../api'
import { toggleFavorite, useFavorites } from '../store'
import BrowsePanel from './BrowsePanel'

interface Props {
  /** Kod bo'yicha boj hisoblash — kalkulyator sahifasiga o'tadi. */
  onCalc: (m: HSMatch) => void
  /** Kod haqida chatda so'rash. */
  onAsk: (code: string, title: string) => void
  /** Kod bo'yicha risk tekshiruvi. */
  onRisk: (code: string) => void
}

type Tab = 'search' | 'browse'

/**
 * TIF TN kodini topish.
 *
 * Ikki yo'l ATAYLAB yonma-yon:
 *   Qidiruv — tovar nomini bilganda ("наушниклар" → 8518 30)
 *   Ko'rish — bilmaganda; ierarxiya atama talab qilmaydi
 *
 * Maketdagi uch qadamli sehrgar ("Tovar turi", "Ishlatilish sohasi")
 * ATAYLAB olinmadi: u foydalanuvchidan nomenklatura tasnifini bilishni
 * talab qilardi — aynan shu muammodan qutulish uchun bitta erkin maydon
 * va ierarxiya qilingan.
 */
export default function SearchPage({ onCalc, onAsk, onRisk }: Props) {
  const favs = useFavorites()
  const [tab, setTab] = useState<Tab>('search')
  const [query, setQuery] = useState('')
  const [matches, setMatches] = useState<HSMatch[] | null>(null)
  const [exempt, setExempt] = useState<Record<string, ExemptionsResponse>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function run(e?: React.FormEvent) {
    e?.preventDefault()
    const q = query.trim()
    if (!q || loading) return

    setLoading(true)
    setError('')
    setExempt({})
    try {
      const res = await api.searchHS(q, false)
      // Hech narsa topilmasa backend `matches: null` qaytaradi.
      const list = res.matches ?? []
      setMatches(list)

      // Imtiyozlar alohida so'raladi: 13 142 koddan 3 856 tasi (29%)
      // imtiyoz qoidasiga tushadi va buni stavka yonida ko'rsatmaslik
      // foydalanuvchini ortiqcha to'lovga olib borardi.
      const pairs = await Promise.all(
        list.map(async (m) => {
          try {
            return [m.code.code, await api.exemptions(m.code.code)] as const
          } catch {
            return null // imtiyoz ma'lumoti yo'q — natijani buzmaydi
          }
        }),
      )
      setExempt(Object.fromEntries(pairs.filter((p) => p !== null)))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Qidiruv xatosi')
      setMatches(null)
    } finally {
      setLoading(false)
    }
  }

  // Ball MUTLAQ emas — u IDF asosidagi ochiq shkalada. Shuning uchun
  // eng yaxshi natijaga NISBATAN ko'rsatiladi va "foiz ishonch" deb
  // atalmaydi: bu ehtimollik emas, tartib ko'rsatkichi.
  const best = matches?.[0]?.score ?? 0

  // Nechta kod birinchi o'rinni BO'LISHIB olgan.
  //
  // NEGA MUHIM: "noutbuk" so'roviga to'rtta kod AYNAN bir xil ball oladi
  // (73,177). Bunday holatda birinchisiga "Eng mos" deb yozish — qidiruv
  // topmagan g'olibni o'ylab topish bo'lardi. Deklarant esa shu nishonga
  // qarab kod tanlaydi va noto'g'ri kod jarima degani.
  const tied = matches
    ? matches.filter((m) => best > 0 && m.score >= best * 0.999).length
    : 0
  const hasWinner = tied === 1

  return (
    <div className="page">
      <div className="page-head">
        <h1>TIF TN kodini topish</h1>
        <div className="seg" role="tablist">
          <button role="tab" aria-selected={tab === 'search'} onClick={() => setTab('search')}>Qidiruv</button>
          <button role="tab" aria-selected={tab === 'browse'} onClick={() => setTab('browse')}>Ko'rish</button>
        </div>
      </div>

      {tab === 'browse' ? (
        <BrowsePanel onPick={(code, title) => onAsk(code, title)} />
      ) : (
        <>
          <form className="search-bar" onSubmit={run}>
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Tovar nomi — masalan: noutbuk, naushniklar, elektr dvigatel"
              aria-label="Tovar nomi"
            />
            <button type="submit" disabled={loading || !query.trim()}>
              {loading ? '…' : 'Qidirish'}
            </button>
          </form>

          {error && <div className="sb-note err">{error}</div>}

          {matches?.length === 0 && (
            <div className="empty">
              Mos kod topilmadi. Boshqacha atash bilan urinib ko'ring yoki
              <button className="linklike" onClick={() => setTab('browse')}>ierarxiya bo'yicha ko'ring</button>.
            </div>
          )}

          {tied > 1 && (
            <div className="tie-note">
              <b>{tied} ta kod bir xil ball oldi</b> — ular orasidagi tartib
              tasodifiy. Tavsifni o'qib o'zingiz tanlang yoki chatda so'rang.
            </div>
          )}

          <div className="results">
            {matches?.map((m, i) => (
              <ResultCard
                key={m.code.code}
                match={m}
                best={hasWinner && i === 0}
                share={best > 0 ? m.score / best : 0}
                exempt={exempt[m.code.code]}
                saved={favs.some((f) => f.kind === 'code' && f.id === m.code.code)}
                onCalc={() => onCalc(m)}
                onRisk={() => onRisk(m.code.code)}
                onSave={() => toggleFavorite({
                  kind: 'code',
                  id: m.code.code,
                  title: m.code.name_uz || m.code.name_ru,
                  meta: `${m.code.import_duty}% boj · ${m.code.vat}% QQS`,
                })}
                onAsk={() => onAsk(m.code.code, m.code.path_uz || m.code.name_uz)}
              />
            ))}
          </div>
        </>
      )}
    </div>
  )
}

function ResultCard({
  match, best, share, exempt, saved, onCalc, onRisk, onSave, onAsk,
}: {
  match: HSMatch
  /** Faqat BIRINCHI o'rin yolg'iz bo'lganda true — teng ball bo'lsa g'olib yo'q. */
  best: boolean
  share: number
  exempt?: ExemptionsResponse
  saved: boolean
  onCalc: () => void
  onRisk: () => void
  onSave: () => void
  onAsk: () => void
}) {
  const c = match.code
  const free = exempt?.free ?? []

  return (
    <article className={'card' + (best ? ' top' : '')}>
      <header className="card-head">
        <span className="card-code">{formatCode(c.code)}</span>
        {best && <span className="badge best">Eng mos</span>}
        <span
          className="match"
          title="Eng yaxshi natijaga nisbatan. Bu ehtimollik emas — natijalarni tartiblash uchun."
        >
          <span className="match-bar"><i style={{ width: `${Math.round(share * 100)}%` }} /></span>
        </span>
        <button
          className={'star' + (saved ? ' on' : '')}
          onClick={onSave}
          aria-pressed={saved}
          aria-label={saved ? 'Sevimlilardan olib tashlash' : 'Sevimlilarga saqlash'}
          title={saved ? 'Sevimlilardan olib tashlash' : 'Sevimlilarga saqlash'}
        >{saved ? '★' : '☆'}</button>
      </header>

      <p className="card-name">{c.name_uz || c.name_ru || '—'}</p>
      {parentPath(c.path_uz) && <p className="card-path">{parentPath(c.path_uz)}</p>}

      <div className="rates">
        <span><b>{c.import_duty}%</b> boj</span>
        {/* Kombinatsiyalangan stavka — 1 555 kodda. Ko'rsatilmasa,
            deklarant faqat foizni ko'rib, bojni kam hisoblardi. */}
        {c.import_duty_specific != null && (
          <span className="combo" title="Kombinatsiyalangan stavka: foizli va qat'iy qismdan kattasi olinadi">
            + <b>${c.import_duty_specific}</b>/{c.import_duty_specific_unit}
          </span>
        )}
        <span><b>{c.vat}%</b> QQS</span>
        {c.unit && <span className="unit">{c.unit}</span>}
        {/* Aksiz kodga bog'lanmagan — "0%" deb yozish yolg'on bo'lardi. */}
        <span className="unknown" title="Aksiz TIF TN kodiga bog'lanmagan: Soliq kodeksi 289¹–289³ stavkalarni tovar NOMI bo'yicha beradi">
          aksiz: noma'lum
        </span>
      </div>

      {free.length > 0 && (
        <div className="exempt">
          <span className="badge warn">imtiyoz bo'lishi mumkin</span>
          <span>
            {free.map(freeLabel).join(', ')} ozod qilish qoidasi bor —{' '}
            <b>sharti tekshirilsin</b>
          </span>
        </div>
      )}

      <footer className="card-foot">
        <button onClick={onCalc}>Bojni hisoblash</button>
        <button className="ghost" onClick={onRisk}>Risk tekshiruvi</button>
        <button className="ghost" onClick={onAsk}>Chatda so'rash</button>
      </footer>
    </article>
  )
}

/**
 * Yo'lning oxirgi ikki bo'g'ini.
 *
 * To'liq `path_uz` 500 belgigacha bo'ladi va oxirgi bo'g'ini kod nomining
 * o'zi — ya'ni kartochkada u ikki marta takrorlanardi va butun ro'yxatni
 * o'qib bo'lmas holga keltirardi.
 */
function parentPath(path: string): string {
  const parts = path.split(';').map((s) => s.trim()).filter(Boolean)
  return parts.slice(Math.max(0, parts.length - 3), parts.length - 1).join(' › ')
}

/** "boj" → "bojdan": imtiyoz matni o'zbekcha o'qilsin. */
function freeLabel(kind: string): string {
  const map: Record<string, string> = {
    boj: 'bojdan',
    aksiz: 'aksizdan',
    qqs: 'QQS dan',
    yigim: 'yig\'imdan',
  }
  return map[kind] ?? kind + 'dan'
}
