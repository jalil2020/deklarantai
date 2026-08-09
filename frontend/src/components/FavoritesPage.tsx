import { formatCode } from '../api'
import { formatWhen, removeFavorite, useFavorites } from '../store'
import { LocalNote } from './HistoryPage'

interface Props {
  /** Kod tanlanganda — kalkulyator yoki risk uchun. */
  onCode: (code: string, title: string) => void
  onAsk: (question: string) => void
}

export default function FavoritesPage({ onCode, onAsk }: Props) {
  const favs = useFavorites()
  const codes = favs.filter((f) => f.kind === 'code')
  const laws = favs.filter((f) => f.kind === 'law')

  return (
    <div className="page">
      <div className="page-head"><h1>Sevimlilar</h1></div>
      <LocalNote />

      {favs.length === 0 ? (
        <div className="empty">
          Bo'sh. Qidiruv natijasidagi ★ tugmasi bilan kod saqlang.
        </div>
      ) : (
        <>
          {codes.length > 0 && (
            <>
              <h2 className="sec-title">TIF TN kodlari</h2>
              <div className="results">
                {codes.map((f) => (
                  <article key={f.id} className="card row-card">
                    <button className="row-main" onClick={() => onCode(f.id, f.title)}>
                      <span className="row-title">
                        <b className="mono">{formatCode(f.id)}</b> {f.title}
                      </span>
                      <span className="row-meta">{f.meta} · {formatWhen(f.at)}</span>
                    </button>
                    <button
                      className="row-x"
                      onClick={() => removeFavorite('code', f.id)}
                      aria-label="Olib tashlash"
                    >✕</button>
                  </article>
                ))}
              </div>
            </>
          )}

          {laws.length > 0 && (
            <>
              <h2 className="sec-title">Qonun moddalari</h2>
              <div className="results">
                {laws.map((f) => (
                  <article key={f.id} className="card row-card">
                    <button
                      className="row-main"
                      onClick={() => onAsk(`"${f.title}" moddasini oddiy tilda tushuntir.`)}
                    >
                      <span className="row-title">{f.title}</span>
                      <span className="row-meta">{f.meta} · {formatWhen(f.at)}</span>
                    </button>
                    <button
                      className="row-x"
                      onClick={() => removeFavorite('law', f.id)}
                      aria-label="Olib tashlash"
                    >✕</button>
                  </article>
                ))}
              </div>
            </>
          )}
        </>
      )}
    </div>
  )
}
