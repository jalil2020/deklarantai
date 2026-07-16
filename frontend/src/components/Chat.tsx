import { useEffect, useRef, useState } from 'react'
import { api, type ChatMessage, type ChatImage } from '../api'

interface Props {
  aiAvailable: boolean
}

const FLOAT_PROMPTS = [
  'Bu tovar qaysi TIF TN kodga kiradi?',
  '50 mln so\'mlik smartfon uchun bojni hisobla',
]

const QUICK = [
  { icon: '🧮', label: 'Boj hisoblash', starter: 'Bojni hisoblab ber. Bojxona qiymati: ___ so\'m, tovar: ___' },
  { icon: '🔎', label: 'HS kod topish', starter: 'Bu tovarning TIF TN kodini top: ' },
  { icon: '📋', label: 'Deklaratsiya', starter: 'Deklaratsiya (GTD) to\'ldirish bo\'yicha yordam ber: ' },
]

const MAX_EDGE = 1568

// Robot boshi (logoga mos) — halqa/hujjatsiz, header va katta mascot uchun umumiy.
function RobotHead() {
  return (
    <g>
      {/* antenna */}
      <line x1="130" y1="58" x2="130" y2="40" stroke="#2b3a55" strokeWidth="5" strokeLinecap="round" />
      <circle cx="130" cy="34" r="9" fill="url(#dk-ball)" />
      <circle cx="127" cy="31" r="3" fill="#bfe4ff" opacity="0.9" />
      {/* ears */}
      <rect x="60" y="96" width="22" height="42" rx="11" fill="url(#dk-body)" stroke="#e3e9f2" />
      <rect x="178" y="96" width="22" height="42" rx="11" fill="url(#dk-body)" stroke="#e3e9f2" />
      <circle cx="71" cy="117" r="6" fill="#2b3a55" opacity="0.85" />
      <circle cx="189" cy="117" r="6" fill="#2b3a55" opacity="0.85" />
      {/* head */}
      <rect x="70" y="58" width="120" height="104" rx="42" fill="url(#dk-body)" stroke="#e3e9f2" strokeWidth="2" />
      <ellipse cx="104" cy="86" rx="24" ry="14" fill="#ffffff" opacity="0.65" />
      {/* face */}
      <rect x="86" y="76" width="88" height="74" rx="30" fill="url(#dk-face)" />
      {/* smiling eyes */}
      <path d="M102 112 q10 -13 20 0" stroke="#2fd4f2" strokeWidth="6" strokeLinecap="round" fill="none" />
      <path d="M138 112 q10 -13 20 0" stroke="#2fd4f2" strokeWidth="6" strokeLinecap="round" fill="none" />
      {/* smile */}
      <path d="M114 128 q16 17 32 0" stroke="#2fd4f2" strokeWidth="6" strokeLinecap="round" fill="none" />
    </g>
  )
}

// To'liq logo belgisi: robot + moviy halqa + hujjat (checkmark bilan).
function LogoMark() {
  return (
    <svg className="logo-mark" viewBox="0 0 260 244" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <defs>
        <linearGradient id="dk-ring" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#37c0ff" />
          <stop offset="1" stopColor="#2563eb" />
        </linearGradient>
        <linearGradient id="dk-body" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#ffffff" />
          <stop offset="1" stopColor="#e0e8f3" />
        </linearGradient>
        <linearGradient id="dk-ball" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#66ccff" />
          <stop offset="1" stopColor="#2563eb" />
        </linearGradient>
        <radialGradient id="dk-face" cx="0.5" cy="0.4" r="0.85">
          <stop offset="0" stopColor="#15233d" />
          <stop offset="1" stopColor="#0a1220" />
        </radialGradient>
      </defs>
      {/* moviy halqa */}
      <circle cx="128" cy="118" r="96" stroke="url(#dk-ring)" strokeWidth="7" opacity="0.92" />
      {/* yelka */}
      <path d="M96 176 q34 -20 68 0 q10 6 10 20 h-88 q0 -14 10 -20 z" fill="url(#dk-body)" stroke="#e3e9f2" strokeWidth="2" />
      <RobotHead />
      {/* hujjat */}
      <g transform="rotate(5 176 168)">
        <rect x="150" y="120" width="74" height="90" rx="9" fill="#ffffff" stroke="#e3e9f2" strokeWidth="2" />
        <path d="M206 120 h18 v18 z" fill="#2f8bff" />
        <path d="M206 120 v18 h18" fill="none" stroke="#e3e9f2" strokeWidth="1.5" />
        <rect x="162" y="140" width="30" height="6" rx="3" fill="#29abff" />
        <rect x="162" y="154" width="48" height="4.5" rx="2.25" fill="#26324b" />
        <rect x="162" y="165" width="48" height="4.5" rx="2.25" fill="#26324b" />
        <rect x="162" y="176" width="34" height="4.5" rx="2.25" fill="#26324b" />
        <circle cx="196" cy="192" r="13" fill="url(#dk-ball)" />
        <path d="M190 192 l4 4 l8 -8" stroke="#ffffff" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
      </g>
    </svg>
  )
}

// Kichik logo (header uchun): robot boshi + wordmark.
function MiniLogo() {
  return (
    <div className="mini-logo">
      <svg viewBox="46 24 168 148" className="mini-robot" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <defs>
          <linearGradient id="dk-body-s" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#ffffff" /><stop offset="1" stopColor="#e0e8f3" />
          </linearGradient>
          <linearGradient id="dk-ball-s" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" stopColor="#66ccff" /><stop offset="1" stopColor="#2563eb" />
          </linearGradient>
          <radialGradient id="dk-face-s" cx="0.5" cy="0.4" r="0.85">
            <stop offset="0" stopColor="#15233d" /><stop offset="1" stopColor="#0a1220" />
          </radialGradient>
        </defs>
        <g>
          <line x1="130" y1="58" x2="130" y2="40" stroke="#2b3a55" strokeWidth="5" strokeLinecap="round" />
          <circle cx="130" cy="34" r="9" fill="url(#dk-ball-s)" />
          <rect x="60" y="96" width="22" height="42" rx="11" fill="url(#dk-body-s)" />
          <rect x="178" y="96" width="22" height="42" rx="11" fill="url(#dk-body-s)" />
          <rect x="70" y="58" width="120" height="104" rx="42" fill="url(#dk-body-s)" stroke="#e3e9f2" strokeWidth="2" />
          <rect x="86" y="76" width="88" height="74" rx="30" fill="url(#dk-face-s)" />
          <path d="M102 112 q10 -13 20 0" stroke="#2fd4f2" strokeWidth="6" strokeLinecap="round" fill="none" />
          <path d="M138 112 q10 -13 20 0" stroke="#2fd4f2" strokeWidth="6" strokeLinecap="round" fill="none" />
          <path d="M114 128 q16 17 32 0" stroke="#2fd4f2" strokeWidth="6" strokeLinecap="round" fill="none" />
        </g>
      </svg>
      <span className="wordmark sm">Deklarant <span className="wm-a">Ai</span></span>
    </div>
  )
}

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

  function toggleVoice() {
    const SR = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition
    if (!SR) { setError('Brauzer ovozli kiritishni qo\'llab-quvvatlamaydi'); return }
    if (listening) { recogRef.current?.stop(); return }
    const r = new SR()
    r.lang = 'uz-UZ'
    r.interimResults = false
    r.onresult = (ev: any) => setInput((prev) => (prev ? prev + ' ' : '') + ev.results[0][0].transcript)
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
      {!empty && (
        <header className="topbar">
          <MiniLogo />
        </header>
      )}

      {empty ? (
        <div className="hero">
          <div className="mascot-stage">
            <button className="float-bubble b-left" onClick={() => send(FLOAT_PROMPTS[0], [])}>
              <span className="fb-dot" /> {FLOAT_PROMPTS[0]}
            </button>
            <LogoMark />
            <button className="float-bubble b-right" onClick={() => send(FLOAT_PROMPTS[1], [])}>
              <span className="fb-dot" /> {FLOAT_PROMPTS[1]}
            </button>
          </div>
          <div className="hero-brand">
            <h1 className="wordmark">Deklarant <span className="wm-a">Ai</span></h1>
            <p className="tagline">Bojxona rasmiylashtiruvida aqlli yordamchingiz</p>
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
            <div className="bubble assistant typing"><span></span><span></span><span></span></div>
          )}
          <div ref={endRef} />
        </div>
      )}

      <div className="composer-card">
        <div className={`composer-banner ${aiAvailable ? '' : 'warn'}`}>
          <span className="cb-left">
            {aiAvailable
              ? <>✨ Kod, boj va qonunchilik bo'yicha yordam beraman</>
              : <>⚠️ AI o'chirilgan — <code>ANTHROPIC_API_KEY</code> ni sozlang</>}
          </span>
          <a className="banner-btn" href="https://customs.uz" target="_blank" rel="noreferrer">customs.uz →</a>
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
            <button className={`round-btn mic ${listening ? 'on' : ''}`} onClick={toggleVoice} title="Ovozli kiritish" type="button">🎙️</button>
            <button className="round-btn send" onClick={() => submit()} disabled={!canSend} title="Yuborish" type="button">➤</button>
          </div>
        </div>
      </div>

      <input ref={fileRef} type="file" accept="image/*" multiple style={{ display: 'none' }}
        onChange={(e) => { if (e.target.files) addFiles(e.target.files); e.target.value = '' }} />
    </div>
  )
}
