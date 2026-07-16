import { useEffect, useRef, useState } from 'react'
import { api, type ChatMessage } from '../api'

interface Props {
  aiAvailable: boolean
}

const SUGGESTIONS = [
  'Import qilishda qanday to\'lovlar mavjud?',
  'Bojxona qiymati qanday aniqlanadi?',
  'Aksiz solig\'i qaysi tovarlarga qo\'llaniladi?',
  'Fizik shaxs uchun import chegarasi qancha?',
]

export default function Chat({ aiAvailable }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, loading])

  async function send(text: string) {
    const content = text.trim()
    if (!content || loading) return
    setError('')
    const next: ChatMessage[] = [...messages, { role: 'user', content }]
    setMessages(next)
    setInput('')
    setLoading(true)
    try {
      const res = await api.chat(next)
      setMessages([...next, { role: 'assistant', content: res.reply }])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Xatolik')
    } finally {
      setLoading(false)
    }
  }

  if (!aiAvailable) {
    return (
      <div className="panel">
        <h2>Qonunchilik bo'yicha chat</h2>
        <div className="error">
          AI xizmati sozlanmagan. Chat ishlashi uchun backendда{' '}
          <code>ANTHROPIC_API_KEY</code> muhit o'zgaruvchisini o'rnating.
        </div>
      </div>
    )
  }

  return (
    <div className="panel chat-panel">
      <h2>Qonunchilik bo'yicha chat</h2>
      <p className="hint">Bojxona qonunchiligi, to'lovlar va rasmiylashtiruv bo'yicha savol bering.</p>

      <div className="chat-window">
        {messages.length === 0 && (
          <div className="suggestions">
            {SUGGESTIONS.map((s) => (
              <button key={s} onClick={() => send(s)}>{s}</button>
            ))}
          </div>
        )}
        {messages.map((m, i) => (
          <div key={i} className={`bubble ${m.role}`}>
            {m.content}
          </div>
        ))}
        {loading && <div className="bubble assistant loading">Yozmoqda...</div>}
        {error && <div className="error">{error}</div>}
        <div ref={endRef} />
      </div>

      <form className="chat-input" onSubmit={(e) => { e.preventDefault(); send(input) }}>
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Savolingizni yozing..."
          disabled={loading}
        />
        <button type="submit" disabled={loading || !input.trim()}>Yuborish</button>
      </form>
    </div>
  )
}
