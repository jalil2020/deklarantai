import { useEffect, useRef, useState } from 'react'
import { api, type ChatMessage, type ChatImage, type ChatMode } from '../api'
import Markdown from './Markdown'
import { LogoMark, MiniLogo } from './Logo'
import { saveChat } from '../store'

interface Props {
  aiAvailable: boolean
  /**
   * Tashqaridan (sidebar dan) kiritiladigan matn.
   *
   * Qiymat emas, HODISA sifatida ishlaydi: har safar yangi obyekt
   * yuboriladi, shunda bir xil kod ketma-ket ikki marta tanlansa ham
   * effekt qayta ishga tushadi.
   */
  inject?: { text: string; at: number }
  /** Foydalanuvchi kirganmi — kirmagan bo'lsa chat yozishga yopiq. */
  signedIn: boolean
  /** Kirish oynasini ochadi. */
  onNeedLogin: () => void
  /**
   * Tarixchadan ochilgan suhbat.
   *
   * `inject` kabi HODISA: bir xil suhbat ikki marta ochilsa ham
   * qaytadan yuklanishi kerak.
   */
  restore?: { id: string; messages: ChatMessage[]; at: number }
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

export default function Chat({ aiAvailable, inject, restore, signedIn, onNeedLogin }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [pending, setPending] = useState<Pending[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [listening, setListening] = useState(false)
  // Ketayotgan oqimni to'xtatish uchun.
  const abortRef = useRef<AbortController | null>(null)

  // Javob uslubi. Tanlov saqlanadi — foydalanuvchi har safar qayta
  // tanlashi shart emas. Rejim faqat uslubni o'zgartiradi: stavkalar va
  // ogohlantirishlar ikkalasida bir xil.
  const [mode, setMode] = useState<ChatMode>(
    () => (localStorage.getItem('mode') as ChatMode) || 'deklarant',
  )
  useEffect(() => {
    localStorage.setItem('mode', mode)
  }, [mode])

  // Sidebar dan kelgan matnni maydonga qo'yamiz va fokusni beramiz.
  //
  // ATAYLAB YUBORMAYMIZ: foydalanuvchi savolni to'ldirishi kerak bo'lishi
  // mumkin (qiymat, davlat, yil). Avtomatik yuborish uni yarim savol
  // bilan javob olishga majbur qilardi.
  useEffect(() => {
    if (!inject) return
    setInput(inject.text)
    taRef.current?.focus()
  }, [inject])
  const endRef = useRef<HTMLDivElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const taRef = useRef<HTMLTextAreaElement>(null)
  const recogRef = useRef<any>(null)
  // Joriy suhbat IDsi. Yangi suhbat boshlanganda yasaladi, tarixchadan
  // ochilganda esa o'shanikisi olinadi.
  const chatId = useRef<string | null>(null)

  const empty = messages.length === 0

  // Tarixchadan ochilgan suhbatni tiklaymiz va uning IDsini olamiz —
  // davom ettirilsa yangi yozuv emas, o'shanisi yangilansin.
  useEffect(() => {
    if (!restore) return
    chatId.current = restore.id
    setMessages(restore.messages)
  }, [restore])

  // Suhbatni saqlash.
  //
  // Javob TUGAGANDA saqlaymiz: oqim davomida har bo'lakda yozish
  // localStorage ga soniyasiga o'nlab marta murojaat qilardi.
  useEffect(() => {
    if (loading || messages.length === 0) return
    chatId.current ??= String(Date.now())
    saveChat(chatId.current, messages)
  }, [messages, loading])

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
    // Kirish tekshiruvi ENG BOSHIDA: hero'dagi taklif tugmalari ham
    // shu yerdan o'tadi, ya'ni bitta joyda ushlanadi (DRY).
    if (!signedIn) { onNeedLogin(); return }
    if ((!content && imgs.length === 0) || loading || !aiAvailable) return
    setError('')
    const next: ChatMessage[] = [...messages, { role: 'user', content, images: imgs.length ? imgs : undefined }]
    setMessages(next)
    setInput('')
    setPending([])
    setLoading(true)

    // Javob bo'lak-bo'lak keladi. Bo'sh assistant xabarini darrov qo'shamiz
    // va uni to'ldirib boramiz — shunda foydalanuvchi 40 soniya bo'sh
    // ekranga qarab turmaydi.
    setMessages([...next, { role: 'assistant', content: '' }])

    const ctrl = new AbortController()
    abortRef.current = ctrl

    // Har bo'lakda setMessages chaqirsak, React juda ko'p qayta chizadi.
    // Matnni to'plab, kadr chastotasida yangilaymiz.
    let acc = ''
    let queued = false
    const flush = () => {
      queued = false
      setMessages([...next, { role: 'assistant', content: acc }])
    }

    try {
      await api.chatStream(next, (chunk) => {
        acc += chunk
        if (!queued) {
          queued = true
          requestAnimationFrame(flush)
        }
      }, ctrl.signal, mode)
      flush()
    } catch (err) {
      if (ctrl.signal.aborted) {
        // Foydalanuvchi to'xtatdi — kelgan qismi qoladi, xato ko'rsatilmaydi.
        flush()
      } else {
        setError(err instanceof Error ? err.message : 'Xatolik')
        // Yarim javob qolib ketmasin: hech narsa kelmagan bo'lsa,
        // bo'sh pufakni olib tashlaymiz.
        setMessages(acc ? [...next, { role: 'assistant', content: acc }] : next)
      }
    } finally {
      abortRef.current = null
      setLoading(false)
    }
  }

  // To'xtatish — uzun javob kerak bo'lmay qolsa.
  function stop() {
    abortRef.current?.abort()
  }

  function submit(e?: React.FormEvent) {
    e?.preventDefault()
    send(input, pending.map((p) => p.img))
  }

  function applyStarter(starter: string) {
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

  // Kirmagan foydalanuvchi uchun maydon YOPIQ, lekin ko'rinib turadi:
  // ilova nima qila olishini ko'rmasdan kirishga majburlash — savolga
  // javob bermasdan pul so'ragandek.
  const locked = !signedIn
  const canSend = !locked && !loading && aiAvailable &&
    (input.trim().length > 0 || pending.length > 0)

  /** Yopiq holatda har qanday urinish kirish oynasini ochadi. */
  function guard(action: () => void) {
    return () => (locked ? onNeedLogin() : action())
  }

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
            <h1 className="wordmark">Deklarant <span className="wm-a">AI</span></h1>
            <p className="tagline">Bojxona rasmiylashtiruvida aqlli yordamchingiz</p>
          </div>
        </div>
      ) : (
        <div className="chat-window">
          {messages.map((m, i) => (
            // Bo'sh javob pufagi chizilmasin: oqim boshlanishini kutayotgan
            // paytda o'rniga "yozilmoqda" nuqtalari ko'rinadi.
            !m.content && !m.images?.length ? null : (
            // Oxirgi javob hali oqib kelayotgan bo'lsa — kursor ko'rsatamiz.
            <div key={i} className={`bubble ${m.role}${
              loading && i === messages.length - 1 && m.role === 'assistant' ? ' streaming' : ''
            }`}>
              {m.images?.map((img, j) => (
                <img key={j} className="bubble-img" src={`data:${img.media_type};base64,${img.data}`} alt="rasm" />
              ))}
              {/* Foydalanuvchi xabari — xom matn: u markdown yozmaydi va
                  yozgan belgisi o'zgarmasdan ko'rinishi kerak. AI javobi esa
                  markdown (jadval, sarlavha, qalin) bo'lib keladi. */}
              {m.content && (
                <div className="bubble-text">
                  {m.role === 'assistant' ? <Markdown text={m.content} /> : m.content}
                </div>
              )}
            </div>
            )
          ))}
          {/* Nuqtalar faqat birinchi bo'lak kelgunicha — undan keyin
              matnning o'zi ko'rinib turadi. */}
          {loading && messages[messages.length - 1]?.content === '' && (
            <div className="bubble assistant typing"><span></span><span></span><span></span></div>
          )}
          <div ref={endRef} />
        </div>
      )}

      <div className="composer-card">
        <div className={`composer-banner ${aiAvailable && !locked ? '' : 'warn'}`}>
          <span className="cb-left">
            {locked
              ? <>🔑 Chat uchun kirish kerak</>
              : aiAvailable
                ? <>✨ Kod, boj va qonunchilik bo'yicha yordam beraman</>
                : <>⚠️ AI o'chirilgan — <code>ANTHROPIC_API_KEY</code> ni sozlang</>}
          </span>
          {locked && (
            <button className="banner-btn" type="button" onClick={onNeedLogin}>Kirish</button>
          )}
          {/* Rejim tanlovi. Faqat javob uslubini o'zgartiradi —
              stavkalar va ogohlantirishlar ikkalasida bir xil. */}
          <div className="mode-switch" role="group" aria-label="Javob uslubi">
            {([
              ['deklarant', 'Deklarant', 'Qisqa, GTD kodlari bilan'],
              ['tadbirkor', 'Tadbirkor', 'Tushuntirish bilan, oddiy tilda'],
            ] as const).map(([id, label, hint]) => (
              <button
                key={id}
                type="button"
                className={`mode-btn ${mode === id ? 'on' : ''}`}
                aria-pressed={mode === id}
                title={hint}
                onClick={() => setMode(id)}
              >
                {label}
              </button>
            ))}
          </div>
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
          placeholder={locked
            ? 'Savol berish uchun kiring — shu yerga bosing'
            : "So'rov yozing yoki tovar / invoys rasmini yuklang..."}
          rows={1}
          // `readOnly`, `disabled` EMAS: o'chirilgan maydon bosishni
          // umuman qabul qilmaydi, ya'ni foydalanuvchi bosib ko'radi-yu
          // hech narsa bo'lmaydi.
          readOnly={locked}
          onClick={locked ? onNeedLogin : undefined}
          onFocus={locked ? onNeedLogin : undefined}
          disabled={loading}
        />

        <div className="composer-actions">
          <div className="chips">
            <button className="chip" onClick={guard(() => fileRef.current?.click())}>
              <span className="chip-ic">📎</span> Rasm biriktirish
            </button>
            {QUICK.map((q) => (
              <button key={q.label} className="chip" onClick={guard(() => applyStarter(q.starter))}>
                <span className="chip-ic">{q.icon}</span> {q.label}
              </button>
            ))}
          </div>
          <div className="send-group">
            <button className={`round-btn mic ${listening ? 'on' : ''}`} onClick={guard(toggleVoice)} title="Ovozli kiritish" type="button">🎙️</button>
            {/* Javob oqayotganda yuborish o'rniga to'xtatish — uzun javob
                kerak bo'lmay qolsa, foydalanuvchi kutib turmasin. */}
            {loading ? (
              <button className="round-btn stop" onClick={stop} title="To'xtatish" type="button">■</button>
            ) : (
              <button
                className="round-btn send"
                onClick={guard(() => submit())}
                // Yopiq holatda tugma FAOL qoladi: bosilganda kirish
                // oynasi ochiladi. O'chirilgan tugma esa hech nima
                // demasdan jim turardi.
                disabled={locked ? false : !canSend}
                title={locked ? 'Kirish' : 'Yuborish'}
                type="button"
              >➤</button>
            )}
          </div>
        </div>
      </div>

      <input ref={fileRef} type="file" accept="image/*" multiple style={{ display: 'none' }}
        onChange={(e) => { if (e.target.files) addFiles(e.target.files); e.target.value = '' }} />
    </div>
  )
}
