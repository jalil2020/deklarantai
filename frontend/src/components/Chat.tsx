import { useEffect, useRef, useState } from 'react'
import { api, type ChatMessage, type ChatImage } from '../api'

interface Props {
  aiAvailable: boolean
}

// Xush kelibsiz ekranidagi suzuvchi bubble'lar (bosilsa — savol yuboriladi).
const FLOAT_PROMPTS = [
  'Bu tovar qaysi TIF TN kodga kiradi?',
  '50 mln so\'mlik smartfon uchun bojni hisobla',
]

// Pastdagi tezkor amal chiplari (bosilsa — matn maydoniga boshlang'ich yoziladi).
const QUICK = [
  { icon: '🧮', label: 'Boj hisoblash', starter: 'Bojni hisoblab ber. Bojxona qiymati: ___ so\'m, tovar: ___' },
  { icon: '🔎', label: 'HS kod topish', starter: 'Bu tovarning TIF TN kodini top: ' },
  { icon: '📋', label: 'Deklaratsiya', starter: 'Deklaratsiya (GTD) to\'ldirish bo\'yicha yordam ber: ' },
]

const MAX_EDGE = 1568 // Claude uchun tavsiya etilgan maksimal tomon (px)

// Do'stona robot-mascot.
function Mascot() {
  return (
    <svg className="mascot" viewBox="0 0 170 165" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <ellipse cx="85" cy="150" rx="42" ry="8" fill="#6d5cf0" opacity="0.15" />
      <line x1="85" y1="20" x2="85" y2="40" stroke="#c7d0f7" strokeWidth="3" strokeLinecap="round" />
      <circle cx="85" cy="15" r="6.5" fill="#6d5cf0" />
      <rect x="6" y="66" width="27" height="50" rx="13.5" fill="#6d5cf0" />
      <rect x="137" y="66" width="27" height="50" rx="13.5" fill="#8a7bf5" />
      <rect x="30" y="40" width="110" height="98" rx="28" fill="#20263a" />
      <rect x="44" y="56" width="82" height="66" rx="20" fill="#11151f" />
      <circle cx="70" cy="86" r="7.5" fill="#ffffff" />
      <circle cx="100" cy="86" r="7.5" fill="#ffffff" />
      <circle cx="72" cy="86" r="2.6" fill="#20263a" />
      <circle cx="102" cy="86" r="2.6" fill="#20263a" />
      <g fill="#9aa4c4">
        <circle cx="78" cy="106" r="2.6" />
        <circle cx="85" cy="106" r="2.6" />
        <circle cx="92" cy="106" r="2.6" />
      </g>
    </svg>
  )
}

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
        resolve({ img: { media_type: 'image/jpeg', data: url.split(',')[1] }, url })
      }
      image.src = reader.result as string
    }
    reader.readAsDataURL(file)
  })
}

interface Pending { img: ChatImage; url: string }

export default function Chat({ aiAvailable }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [pending, setPending] = useState<Pending[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [listening, setListening] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const taRef = useRef<HTMLTextAreaElement>(null)
  const recogRef = useRef<any>(null)

  const empty = messages.length === 0

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, loading, pending])

  // Matn maydoni balandligini tarkibga moslash.
  useEffect(() => {
    const ta = taRef.current
    if (ta) { ta.style.height = 'auto'; ta.style.height = Math.min(ta.scrollHeight, 140) + 'px' }
  }, [input])

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
    if ((!content && imgs.length === 0) || loading || !aiAvailable) return
    setError('')
    const next: ChatMessage[] = [...messages, { role: 'user', content, images: imgs.length ? imgs : undefined }]
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

  function submit(e?: React.FormEvent) {
    e?.preventDefault()
    send(input, pending.map((p) => p.img))
  }

  function useStarter(starter: string) {
    setInput(starter)
    taRef.current?.focus()
  }

  function onPaste(e: React.ClipboardEvent) {
    const files = Array.from(e.clipboardData.files)
    if (files.length) { e.preventDefault(); addFiles(files) }
  }

  // Ovozli kiritish (Web Speech API).
  function toggleVoice() {
    const SR = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition
    if (!SR) { setError('Brauzer ovozli kiritishni qo\'llab-quvvatlamaydi'); return }
    if (listening) { recogRef.current?.stop(); return }
    const r = new SR()
    r.lang = 'uz-UZ'
    r.interimResults = false
    r.onresult = (ev: any) => {
      const t = ev.results[0][0].transcript
      setInput((prev) => (prev ? prev + ' ' : '') + t)
    }
    r.onerror = () => setListening(false)
    r.onend = () => setListening(false)
    recogRef.current = r
    setListening(true)
    r.start()
  }

  const canSend = !loading && aiAvailable && (input.trim().length > 0 || pending.length > 0)

  return (
    <div
      className={`chat-root ${dragOver ? 'dragover' : ''}`}
      onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
      onDragLeave={() => setDragOver(false)}
      onDrop={(e) => { e.preventDefault(); setDragOver(false); addFiles(e.dataTransfer.files) }}
    >
      {empty ? (
        <div className="hero">
          <h1 className="hero-title">
            <span className="greet">Salom!</span> Bojxona ishlarini birga hal qilamiz
          </h1>
          <div className="mascot-stage">
            <button className="float-bubble b-left" onClick={() => send(FLOAT_PROMPTS[0], [])}>
              <span className="fb-dot" /> {FLOAT_PROMPTS[0]}
            </button>
            <Mascot />
            <button className="float-bubble b-right" onClick={() => send(FLOAT_PROMPTS[1], [])}>
              <span className="fb-dot" /> {FLOAT_PROMPTS[1]}
            </button>
          </div>
        </div>
      ) : (
        <div className="chat-window">
          {messages.map((m, i) => (
            <div key={i} className={`bubble ${m.role}`}>
              {m.images?.map((img, j) => (
                <img key={j} className="bubble-img" src={`data:${img.media_type};base64,${img.data}`} alt="rasm" />
              ))}
              {m.content && <div className="bubble-text">{m.content}</div>}
            </div>
          ))}
          {loading && (
            <div className="bubble assistant typing">
              <span></span><span></span><span></span>
            </div>
          )}
          <div ref={endRef} />
        </div>
      )}

      {/* Composer karta */}
      <div className="composer-card">
        <div className={`composer-banner ${aiAvailable ? '' : 'warn'}`}>
          <span className="cb-left">
            {aiAvailable
              ? <>✨ Deklarant AI — kod, boj va qonunchilik yordamchisi</>
              : <>⚠️ AI o'chirilgan — <code>ANTHROPIC_API_KEY</code> ni sozlang</>}
          </span>
          <a className="banner-btn" href="https://customs.uz" target="_blank" rel="noreferrer">
            customs.uz →
          </a>
        </div>

        {error && <div className="composer-error">{error}</div>}

        {pending.length > 0 && (
          <div className="attachments">
            {pending.map((p, i) => (
              <div key={i} className="attachment">
                <img src={p.url} alt="biriktirilgan" />
                <button className="remove" onClick={() => setPending((prev) => prev.filter((_, k) => k !== i))}>×</button>
              </div>
            ))}
          </div>
        )}

        <textarea
          ref={taRef}
          className="composer-input"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onPaste={onPaste}
          onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit() } }}
          placeholder="So'rov yozing yoki tovar / invoys rasmini yuklang..."
          rows={1}
          disabled={loading}
        />

        <div className="composer-actions">
          <div className="chips">
            <button className="chip" onClick={() => fileRef.current?.click()}>
              <span className="chip-ic">📎</span> Rasm biriktirish
            </button>
            {QUICK.map((q) => (
              <button key={q.label} className="chip" onClick={() => useStarter(q.starter)}>
                <span className="chip-ic">{q.icon}</span> {q.label}
              </button>
            ))}
          </div>
          <div className="send-group">
            <button
              className={`round-btn mic ${listening ? 'on' : ''}`}
              onClick={toggleVoice}
              title="Ovozli kiritish"
              type="button"
            >🎙️</button>
            <button
              className="round-btn send"
              onClick={() => submit()}
              disabled={!canSend}
              title="Yuborish"
              type="button"
            >➤</button>
          </div>
        </div>
      </div>

      <input
        ref={fileRef}
        type="file"
        accept="image/*"
        multiple
        style={{ display: 'none' }}
        onChange={(e) => { if (e.target.files) addFiles(e.target.files); e.target.value = '' }}
      />
    </div>
  )
}
