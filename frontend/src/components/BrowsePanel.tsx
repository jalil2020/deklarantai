import { useCallback, useEffect, useState } from 'react'
import { browse, formatCode, type BrowseCrumb, type BrowseNode, type BrowseResponse } from '../api'

interface Props {
  /** Kod tanlanganda chaqiriladi. */
  onPick: (code: string, title: string) => void
  /**
   * Boshlang'ich tovar pozitsiyasi (4 xonali).
   *
   * Kalkulyatorning "Ierarxiya" tabi shu bilan ochiladi — foydalanuvchi
   * tanlangan kod daraxtning QAYERIDA turganini darrov ko'rsin,
   * bo'limlardan qo'lda tushib yurmasin.
   */
  initialHeading?: string
}

/**
 * TIF TN ierarxiyasi bo'yicha ko'rish.
 *
 * NEGA KERAK: qidiruv foydalanuvchidan tovarni NOMENKLATURA TILIDA
 * atashni talab qiladi. Bilmasa — hech narsa topilmaydi. Ierarxiya esa
 * hech qanday atama bilishni talab qilmaydi: bo'limdan pastga tushiladi.
 *
 *	Bo'lim (21) → Guruh (96) → Tovar pozitsiyasi (4 xonali) → Kod
 *
 * Har darajada ichidagi kodlar SONI ko'rsatiladi — foydalanuvchi qayerga
 * kirish kerakligini shundan biladi.
 */
export default function BrowsePanel({ onPick, initialHeading }: Props) {
  const [data, setData] = useState<BrowseResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(async (params: Parameters<typeof browse>[0] = {}) => {
    setLoading(true)
    setError('')
    try {
      setData(await browse(params))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Yuklab bo\'lmadi')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!data) void load(initialHeading ? { heading: initialHeading } : {})
  }, [data, load, initialHeading])

  // Tugunga bosilganda pastga tushamiz; barg bo'lsa — tashqariga beramiz.
  const enter = (n: BrowseNode) => {
    if (n.leaf) {
      onPick(n.id, n.title)
      return
    }
    switch (data?.level) {
      case 'sections': void load({ section: n.id }); break
      case 'groups': void load({ group: n.id }); break
      case 'headings': void load({ heading: n.id }); break
    }
  }

  const goCrumb = (c: BrowseCrumb) => {
    if (c.level === 'section') void load({ section: c.id })
    else if (c.level === 'group') void load({ group: c.id })
    else void load({ heading: c.id })
  }

  return (
    <div className="browse">
      {/* Yo'l zanjiri — qayerdaligini ko'rsatadi va yuqoriga qaytaradi */}
      <div className="sb-crumbs">
        <button onClick={() => void load()}>Bo'limlar</button>
        {(data?.path ?? []).map((c) => (
          <span key={c.level + c.id}>
            <span className="sb-sep">›</span>
            <button onClick={() => goCrumb(c)} title={c.title}>{c.id}</button>
          </span>
        ))}
      </div>

      <div className="sb-list">
        {loading && <div className="sb-note">Yuklanmoqda…</div>}
        {error && <div className="sb-note err">{error}</div>}

        {!loading && !error && (data?.items ?? []).map((n) => (
          <button key={n.id} className={'sb-item' + (n.leaf ? ' leaf' : '')} onClick={() => enter(n)}>
            <span className="sb-id">{n.leaf ? formatCode(n.id) : n.id}</span>
            <span className="sb-title">{n.title || '—'}</span>
            {n.leaf
              ? <span className="sb-rate">{n.import_duty ?? 0}%</span>
              : <span className="sb-count">{n.count}</span>}
          </button>
        ))}

        {!loading && !error && (data?.items ?? []).length === 0 && (
          <div className="sb-note">Bo'sh</div>
        )}
      </div>
    </div>
  )
}
