import { useEffect, useRef, useState } from 'react'
import { api, type ChatMessage, type ChatImage } from '../api'

interface Props {
  aiAvailable: boolean
}

const SUGGESTIONS = [
  'Bu tovar qaysi TIF TN kodga to\'g\'ri keladi?',
  'Import qilishda qanday to\'lovlar bo\'ladi?',
  '50 mln so\'mlik smartfon uchun bojni hisoblab ber',
  'Aksiz solig\'i qaysi tovarlarga qo\'llaniladi?',
]

const MAX_EDGE = 1568 // Claude uchun tavsiya etilgan maksimal tomon (px)

// Faylni kichraytirib, base64 rasm va ko'rsatish uchun data-URL qaytaradi.
function fileToImage(file: File): Promise<{ img: ChatImage; url: string }> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(new Error('Faylni o\'qib bo\'lmadi'))
    reader.onload = () => {
      const image = new Image()
      image.onerror = () => reject(new Error('Rasmni ochib bo\'lmadi'))
      image.onload = () => {
        let { width, height } = image
        const scale = Math.min(1, MAX_EDGE / Math.max(width, height))
        width = Math.round(width * scale)
        height = Math.round(height * scale)
        const canvas = document.createElement('canvas')
        canvas.width = width
        canvas.height = height
        const ctx = canvas.getContext('2d')!
        ctx.drawImage(image, 0, 0, width, height)
        const url = canvas.toDataURL('image/jpeg', 0.85)
        const data = url.split(',')[1]
        resolve({ img: { media_type: 'image/jpeg', data }, url })
      }
      image.src = reader.result as string
    }
    reader.readAsDataURL(file)
  })
}

interface Pending {
  img: ChatImage
  url: string
}

export default function Chat({ aiAvailable }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [pending, setPending] = useState<Pending[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, loading, pending])

  async function addFiles(files: FileList | File[]) {
    setError('')
    const arr = Array.from(files).filter((f) => f.type.startsWith('image/'))
    for (const f of arr) {
      try {
        const p = await fileToImage(f)
        setPending((prev) => [...prev, p])
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Rasm xatosi')
      }
    }
  }

  async function send(text: string, imgs: ChatImage[]) {
    const content = text.trim()
    if ((!content && imgs.length === 0) || loading) return
    setError('')
    const userMsg: ChatMessage = { role: 'user', content, images: imgs.length ? imgs : undefined }
    const next = [...messages, userMsg]
    setMessages(next)
    setInput('')
    setPending([])
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

  function submit(e: React.FormEvent) {
    e.preventDefault()
    send(input, pending.map((p) => p.img))
  }

  function onPaste(e: React.ClipboardEvent) {
    const files = Array.from(e.clipboardData.files)
    if (files.length) {
      e.preventDefault()
      addFiles(files)
    }
  }

  if (!aiAvailable) {
    return (
      <div className="panel">
        <div className="error">
          AI xizmati sozlanmagan. Chat va rasm o'qish ishlashi uchun backendda{' '}
          <code>ANTHROPIC_API_KEY</code> muhit o'zgaruvchisini o'rnating.
        </div>
      </div>
    )
  }

  return (
    <div
      className={`chat-full ${dragOver ? 'dragover' : ''}`}
      onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
      onDragLeave={() => setDragOver(false)}
      onDrop={(e) => { e.preventDefault(); setDragOver(false); addFiles(e.dataTransfer.files) }}
    >
      <div className="chat-window">
        {messages.length === 0 && (
          <div className="welcome">
            <div className="welcome-icon">🛃</div>
            <h2>Deklarant AI</h2>
            <p>Savol yozing yoki tovar / invoys rasmini yuklang — kod, boj va qonunchilik bo'yicha yordam beraman.</p>
            <div className="suggestions">
              {SUGGESTIONS.map((s) => (
                <button key={s} onClick={() => send(s, [])}>{s}</button>
              ))}
            </div>
          </div>
        )}

        {messages.map((m, i) => (
          <div key={i} className={`bubble ${m.role}`}>
            {m.images?.map((img, j) => (
              <img
                key={j}
                className="bubble-img"
                src={`data:${img.media_type};base64,${img.data}`}
                alt="yuklangan rasm"
              />
            ))}
            {m.content && <div className="bubble-text">{m.content}</div>}
          </div>
        ))}
        {loading && <div className="bubble assistant loading">Yozmoqda...</div>}
        {error && <div className="error">{error}</div>}
        <div ref={endRef} />
      </div>

      {pending.length > 0 && (
        <div className="attachments">
          {pending.map((p, i) => (
            <div key={i} className="attachment">
              <img src={p.url} alt="biriktirilgan" />
              <button
                className="remove"
                onClick={() => setPending((prev) => prev.filter((_, k) => k !== i))}
                title="O'chirish"
              >×</button>
            </div>
          ))}
        </div>
      )}

      <form className="composer" onSubmit={submit}>
        <button
          type="button"
          className="attach-btn"
          title="Rasm biriktirish"
          onClick={() => fileRef.current?.click()}
        >📎</button>
        <input
          ref={fileRef}
          type="file"
          accept="image/*"
          multiple
          style={{ display: 'none' }}
          onChange={(e) => { if (e.target.files) addFiles(e.target.files); e.target.value = '' }}
        />
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onPaste={onPaste}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(e) }
          }}
          placeholder="Savol yozing yoki rasm tashlang (Enter — yuborish)..."
          rows={1}
          disabled={loading}
        />
        <button type="submit" className="send-btn" disabled={loading || (!input.trim() && pending.length === 0)}>
          ➤
        </button>
      </form>
    </div>
  )
}
