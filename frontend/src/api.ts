// Backend bilan aloqa qiluvchi API klient.

export interface HSCode {
  code: string
  name: string
  description: string
  keywords: string[]
  unit: string
  import_duty: number
  excise: number
  vat: number
}

export interface HSMatch {
  code: HSCode
  score: number
}

export interface HSSearchResponse {
  matches: HSMatch[]
  ai_comment?: string
  source: string
}

export interface DutyRequest {
  customs_value: number
  import_duty: number
  excise: number
  vat: number
  quantity: number
}

export interface DutyLineItem {
  name: string
  rate: number
  base: number
  amount: number
}

export interface DutyResult {
  items: DutyLineItem[]
  total: number
}

export interface ChatImage {
  media_type: string // "image/jpeg", "image/png", ...
  data: string       // base64 (data: prefiksisiz)
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  images?: ChatImage[]
}

export interface Health {
  status: string
  ai_available: boolean
  codes: number
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error((data as { error?: string }).error || `Xato: ${res.status}`)
  }
  return data as T
}

export const api = {
  health: async (): Promise<Health> => {
    const res = await fetch('/api/health')
    return res.json()
  },
  searchHS: (query: string, useAI: boolean) =>
    post<HSSearchResponse>('/api/hscode/search', { query, use_ai: useAI }),
  calculateDuty: (req: DutyRequest) =>
    post<DutyResult>('/api/duty/calculate', req),
  chat: (messages: ChatMessage[]) =>
    post<{ reply: string }>('/api/chat', { messages }),
  chatStream,
}

/**
 * Javobni bo'lak-bo'lak oladi (Server-Sent Events).
 *
 * NEGA KERAK: to'liq javob 23–49 soniya oladi. Foydalanuvchi shuncha vaqt
 * bo'sh ekranga qarab turmasligi uchun matn yozilayotganda ko'rsatiladi.
 *
 * DIQQAT: xato oqim BOSHLANGANDAN keyin ham kelishi mumkin — o'shanda HTTP
 * status allaqachon 200 bo'lgan bo'ladi. Shuning uchun xato hodisa sifatida
 * keladi va uni tashlab yuborish kerak emas, aks holda foydalanuvchi yarim
 * javob olib, nima bo'lganini bilmay qolardi.
 *
 * `signal` — suhbatni to'xtatish uchun (foydalanuvchi bekor qilsa).
 */
async function chatStream(
  messages: ChatMessage[],
  onChunk: (text: string) => void,
  signal?: AbortSignal,
): Promise<void> {
  const res = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messages }),
    signal,
  })

  // Oqim boshlanmasdan xato bo'lsa — javob oddiy JSON.
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error((data as { error?: string }).error || `Xato: ${res.status}`)
  }
  if (!res.body) throw new Error('Brauzer oqimni qo\'llab-quvvatlamaydi')

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  // Bo'lak SSE hodisasi o'rtasida kelishi mumkin, shuning uchun to'liq
  // bo'lmagan qatorni keyingi bo'lakka qoldiramiz.
  let buf = ''

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })

    const lines = buf.split('\n')
    buf = lines.pop() ?? ''

    for (const line of lines) {
      if (!line.startsWith('data: ')) continue
      let ev: { text?: string; error?: string; done?: boolean }
      try {
        ev = JSON.parse(line.slice(6))
      } catch {
        continue
      }
      if (ev.error) throw new Error(ev.error)
      if (ev.done) return
      if (ev.text) onChunk(ev.text)
    }
  }
}

// Summa formatlash (so'mda).
export function formatSom(v: number): string {
  return new Intl.NumberFormat('uz-UZ', { maximumFractionDigits: 0 }).format(v) + " so'm"
}
