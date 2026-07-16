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

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
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
}

// Summa formatlash (so'mda).
export function formatSom(v: number): string {
  return new Intl.NumberFormat('uz-UZ', { maximumFractionDigits: 0 }).format(v) + " so'm"
}
