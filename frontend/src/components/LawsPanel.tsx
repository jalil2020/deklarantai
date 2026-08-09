import { useCallback, useEffect, useState } from 'react'
import { browseLaws, type LawsBrowseResponse } from '../api'

interface Props {
  /** Modda haqida chatda so'rash. */
  onAsk: (question: string) => void
}

/**
 * Qonun korpusi bo'yicha ko'rish.
 *
 * NEGA KERAK: qidiruv "nima izlayotganingizni bilasiz" deb faraz qiladi.
 * Deklarant ko'pincha aksincha ish tutadi — "Bojxona kodeksida bu haqda
 * nima deyilgan?" deb hujjatni ochib, moddalarni ko'zdan kechiradi.
 *
 *	Hujjat (89)  →  Modda (1405)  →  To'liq matn
 */
export default function LawsPanel({ onAsk }: Props) {
  const [data, setData] = useState<LawsBrowseResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(async (params: { doc?: number; i?: number } = {}) => {
    setLoading(true)
    setError('')
    try {
      setData(await browseLaws(params))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Yuklab bo\'lmadi')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { if (!data) void load() }, [data, load])

  if (loading) return <div className="sb-note">Yuklanmoqda…</div>
  if (error) return <div className="sb-note err">{error}</div>

  // --- To'liq matn
  if (data?.level === 'article' && data.article) {
    const a = data.article
    return (
      <>
        <div className="sb-crumbs">
          <button onClick={() => void load()}>Hujjatlar</button>
          <span className="sb-sep">›</span>
          <button onClick={() => void load({ doc: a.doc })} title={a.name}>Moddalar</button>
        </div>
        <div className="sb-list">
          <div className="law-title">{a.title}</div>
          <div className="law-src">
            {a.name}{a.date ? ` · ${a.date}` : ''}
          </div>
          <div className="law-text">{a.text}</div>
          <div className="law-acts">
            <button onClick={() => onAsk(`"${a.title}" moddasini oddiy tilda tushuntir.`)}>
              Chatda tushuntir
            </button>
            {a.lex && (
              // rel: yangi oynada ochilgan sahifa bizning oynamizga
              // window.opener orqali tegmasligi uchun.
              <a href={a.lex} target="_blank" rel="noopener noreferrer">lex.uz</a>
            )}
          </div>
        </div>
      </>
    )
  }

  // --- Hujjatning moddalari
  if (data?.level === 'articles') {
    return (
      <>
        <div className="sb-crumbs">
          <button onClick={() => void load()}>Hujjatlar</button>
          <span className="sb-sep">›</span>
          <span className="sb-here" title={data.doc_name}>{data.articles?.length} modda</span>
        </div>
        <div className="sb-list">
          {(data.articles ?? []).map((a) => (
            <button
              key={a.index}
              className="sb-item law"
              onClick={() => void load({ doc: a.doc, i: a.index })}
            >
              <span className="sb-title">{a.title || '(sarlavhasiz)'}</span>
              <span className="law-prev">{a.preview}</span>
            </button>
          ))}
        </div>
      </>
    )
  }

  // --- Hujjatlar ro'yxati
  return (
    <>
      <div className="sb-crumbs"><span className="sb-here">Hujjatlar</span></div>
      <div className="sb-list">
        {(data?.docs ?? []).map((d) => (
          <button key={d.id} className="sb-item law" onClick={() => void load({ doc: d.id })}>
            <span className="sb-title">{d.name}</span>
            <span className="law-prev">
              {d.chunks} parcha{d.date ? ` · ${d.date}` : ''}{d.lex ? ' · lex.uz' : ''}
            </span>
          </button>
        ))}
      </div>
    </>
  )
}
