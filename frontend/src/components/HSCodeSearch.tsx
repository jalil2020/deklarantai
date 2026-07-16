import { useState } from 'react'
import { api, type HSMatch } from '../api'

interface Props {
  aiAvailable: boolean
  onUseCode?: (m: HSMatch) => void
}

export default function HSCodeSearch({ aiAvailable, onUseCode }: Props) {
  const [query, setQuery] = useState('')
  const [useAI, setUseAI] = useState(false)
  const [matches, setMatches] = useState<HSMatch[]>([])
  const [aiComment, setAiComment] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [searched, setSearched] = useState(false)

  async function search(e: React.FormEvent) {
    e.preventDefault()
    if (!query.trim()) return
    setLoading(true)
    setError('')
    setAiComment('')
    try {
      const res = await api.searchHS(query, useAI && aiAvailable)
      setMatches(res.matches)
      setAiComment(res.ai_comment || '')
      setSearched(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Xatolik')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="panel">
      <h2>TIF TN / HS kod topish</h2>
      <p className="hint">Tovar nomini yozing — mos tovar nomenklatura kodini topamiz.</p>

      <form onSubmit={search} className="search-form">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Masalan: smartfon, charm poyabzal, muzlatgich..."
        />
        <button type="submit" disabled={loading}>
          {loading ? 'Qidirilmoqda...' : 'Qidirish'}
        </button>
      </form>

      <label className={`ai-toggle ${!aiAvailable ? 'disabled' : ''}`}>
        <input
          type="checkbox"
          checked={useAI && aiAvailable}
          disabled={!aiAvailable}
          onChange={(e) => setUseAI(e.target.checked)}
        />
        AI izohi bilan {!aiAvailable && '(AI sozlanmagan)'}
      </label>

      {error && <div className="error">{error}</div>}

      {aiComment && (
        <div className="ai-comment">
          <strong>🤖 AI tavsiyasi:</strong>
          <p>{aiComment}</p>
        </div>
      )}

      {searched && matches.length === 0 && !error && (
        <div className="empty">Mos kod topilmadi. Boshqacha so'z bilan urinib ko'ring.</div>
      )}

      <div className="results">
        {matches.map((m) => (
          <div key={m.code.code} className="hs-card">
            <div className="hs-card-head">
              <span className="hs-code">{m.code.code}</span>
              <span className="hs-name">{m.code.name}</span>
            </div>
            <p className="hs-desc">{m.code.description}</p>
            <div className="hs-rates">
              <span>Import boji: <b>{m.code.import_duty}%</b></span>
              <span>Aksiz: <b>{m.code.excise}%</b></span>
              <span>QQS: <b>{m.code.vat}%</b></span>
              <span>O'lchov: <b>{m.code.unit}</b></span>
            </div>
            {onUseCode && (
              <button className="link-btn" onClick={() => onUseCode(m)}>
                → Kalkulyatorda ishlatish
              </button>
            )}
          </div>
        ))}
      </div>

      <p className="disclaimer">
        ⚠️ Stavkalar taxminiy (demo). Rasmiy TIF TN jadvali:{' '}
        <a href="https://customs.uz" target="_blank" rel="noreferrer">customs.uz</a>
      </p>
    </div>
  )
}
