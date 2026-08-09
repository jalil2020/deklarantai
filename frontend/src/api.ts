// Backend bilan aloqa qiluvchi API klient.

/**
 * API manzilining asosi.
 *
 * NEGA KERAK: brauzerda frontend va backend bir xil manzildan keladi,
 * shuning uchun `/api/...` yetarli. Android ilovasida esa UI APK
 * ICHIDAN yuklanadi (capacitor://localhost) — u yerda `/api` hech
 * qayerga bormaydi. Shuning uchun manzil sozlanadigan bo'lishi kerak.
 *
 * Bo'sh bo'lsa — nisbiy manzil (brauzer va dev-proksi uchun).
 * Android build: VITE_API_BASE=https://api.example.uz
 */
const API_BASE = (import.meta.env.VITE_API_BASE ?? '').replace(/\/+$/, '')

/** url — API yo'lini to'liq manzilga aylantiradi. */
function url(path: string): string {
  return API_BASE + path
}

// ------------------------------------------------------------ foydalanuvchi

export type Role = 'DECLARANT' | 'BUSINESS' | 'INSPECTOR' | 'ADMIN'

export interface User {
  id: string
  login: string
  name: string
  role: Role
  created: string
  daily_quota: number
  chat_mode: ChatMode
}

export interface RoleInfo {
  role: Role
  chat_mode: ChatMode
  daily_quota: number
  /** ADMIN ni ro'yxatdan o'tishda tanlab bo'lmaydi. */
  self_signup: boolean
}

interface Session {
  token: string
  expires_at: string
  user: User
}

const USER_STORE = 'deklarant.user_token'

/**
 * Kirgan foydalanuvchi tokeni.
 *
 * Anonim tokendan ALOHIDA saqlanadi: chiqishda faqat shu o'chiriladi va
 * ilova anonim belgi bilan ishlashda davom etadi (qidiruv, kalkulyator,
 * qonunlar kirishsiz ham ochiq).
 */
function readUserToken(): string {
  try {
    return localStorage.getItem(USER_STORE) ?? ''
  } catch {
    return ''
  }
}

function writeUserToken(token: string) {
  try {
    if (token) localStorage.setItem(USER_STORE, token)
    else localStorage.removeItem(USER_STORE)
  } catch { /* localStorage yopiq — seans faqat shu sahifada qoladi */ }
}

export const auth = {
  token: readUserToken,
  register: async (body: {
    login: string; password: string; name?: string; role?: Role
  }): Promise<Session> => {
    const s = await post<Session>('/api/auth/register', body)
    writeUserToken(s.token)
    return s
  },
  login: async (body: { login: string; password: string }): Promise<Session> => {
    const s = await post<Session>('/api/auth/login', body)
    writeUserToken(s.token)
    return s
  },
  logout: async (): Promise<void> => {
    // Serverga ham aytamiz: token imzolangan, faqat brauzerdan
    // o'chirish uni bekor qilmaydi.
    await post('/api/auth/logout', {}).catch(() => { /* baribir chiqamiz */ })
    writeUserToken('')
  },
  me: async (): Promise<User | null> => {
    if (!readUserToken()) return null
    try {
      return await getJSON<User>('/api/auth/me')
    } catch {
      writeUserToken('') // muddati tugagan yoki bekor qilingan
      return null
    }
  },
  roles: () => getJSON<{ roles: RoleInfo[] }>('/api/auth/roles'),
}

// ------------------------------------------------------------ mijoz belgisi

/**
 * Chat endpointlari mijoz belgisini talab qiladi.
 *
 * Ikki manba:
 *   VITE_API_KEY — qurish vaqtida beriladigan doimiy kalit (hamkor build).
 *   /api/session — anonim token; brauzer uchun odatiy yo'l.
 *
 * DIQQAT: brauzerdagi belgi SIR EMAS — uni sahifadan olib qo'yish mumkin.
 * U API ni butunlay yopmaydi, faqat boshqa saytlar bizning API ni to'g'ridan
 * to'g'ri ulab olishini to'xtatadi va har mijozga barqaror belgi beradi.
 * Haqiqiy himoya akkauntlar bilan keladi.
 */
const BUILD_KEY: string = import.meta.env.VITE_API_KEY ?? ''
const CLIENT_HEADER = 'X-API-Key'
const TOKEN_STORE = 'deklarant.client_token'

interface StoredToken {
  token: string
  expires_at: string
}

// Bir vaqtda bir nechta so'rov ketsa, token bir marta olinsin.
let pending: Promise<string> | null = null

function readToken(): string {
  try {
    const raw = localStorage.getItem(TOKEN_STORE)
    if (!raw) return ''
    const t = JSON.parse(raw) as StoredToken
    // Muddati tugashiga bir daqiqa qolganda ham yangisini olamiz —
    // so'rov ketayotganda tugab qolmasin.
    if (Date.parse(t.expires_at) - Date.now() < 60_000) return ''
    return t.token
  } catch {
    return '' // localStorage yopiq yoki yozuv buzuq — yangisini olamiz
  }
}

async function clientToken(refresh = false): Promise<string> {
  // Kirgan foydalanuvchi tokeni ENG USTUN: chat faqat shu bilan
  // ishlaydi, va kvota ham unga bog'langan.
  const user = readUserToken()
  if (user) return user
  if (BUILD_KEY) return BUILD_KEY
  if (!refresh) {
    const cached = readToken()
    if (cached) return cached
  }
  pending ??= (async () => {
    try {
      const res = await fetch(url('/api/session'), { method: 'POST' })
      if (!res.ok) throw new Error(`Belgi olinmadi: ${res.status}`)
      const t = (await res.json()) as StoredToken
      try {
        localStorage.setItem(TOKEN_STORE, JSON.stringify(t))
      } catch {
        // Saqlab bo'lmadi — har so'rovda qaytadan olinadi, ishlaydi.
      }
      return t.token
    } finally {
      pending = null
    }
  })()
  return pending
}

/**
 * Belgili so'rov.
 *
 * 401 kelsa BIR MARTA yangilab qayta urinadi: server qayta ishga tushganda
 * eski tokenlar kuchini yo'qotadi va foydalanuvchi buni sezmasligi kerak.
 */
async function authedFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const send = async (token: string) =>
    fetch(url(path), { ...init, headers: { ...init.headers, [CLIENT_HEADER]: token } })

  const res = await send(await clientToken())
  // Foydalanuvchi tokeni bilan 401 kelsa yangilashning ma'nosi yo'q —
  // qayta kirish kerak.
  if (res.status !== 401 || BUILD_KEY || readUserToken()) return res
  return send(await clientToken(true))
}

/** Belgili GET. */
async function getJSON<T>(path: string): Promise<T> {
  const res = await authedFetch(path)
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || `Xato: ${res.status}`)
  return data as T
}

/** Bitta TIF TN kodi. Maydonlar backenddagi `hscode.Code` bilan bir xil. */
export interface HSCode {
  code: string
  name_ru: string
  name_uz: string
  path_ru: string
  path_uz: string
  section: string
  group: string
  unit_code: string
  unit: string
  unit_ru: string
  import_duty: number
  export_duty: number
  vat: number
  duty_law?: string
  /**
   * Kombinatsiyalangan stavkaning QAT'IY qismi — dollarda, bitta birlik
   * uchun: «10%, lekin 1 kg uchun 0,5 dollardan kam emas». 1 555 kodda bor.
   */
  import_duty_specific?: number
  /** Qat'iy qism qaysi birlikka: kg, dona, l, juft, m². */
  import_duty_specific_unit?: string
  /**
   * Aksiz stavkasi, foizda.
   *
   * `undefined` = NOMA'LUM, 0% DEGANI EMAS. Bazada aksiz kodga
   * bog'lanmagan (Soliq kodeksi 289¹–289³ stavkalarni tovar NOMI bo'yicha
   * beradi), shuning uchun hozircha hamma kodda bo'sh.
   */
  excise?: number
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

/**
 * Boj hisoblash so'rovi.
 *
 * Bojxona qiymatini ikki usulda berish mumkin: to'g'ridan-to'g'ri
 * `customs_value` da yoki `invoice + transport` dan kurs bilan.
 */
export interface DutyRequest {
  date?: string
  customs_value?: number
  invoice?: number
  transport?: number
  currency?: string
  currency_rate?: number
  usd_rate?: number
  /** Bojxona kodeksi 300-modda: 0 — erkin savdo, 1 — odatiy, 2 — rejimsiz. */
  origin_multiplier?: number
  origin_country?: string
  import_duty: number
  extra_duty?: number
  /** Kombinatsiyalangan stavkaning qat'iy qismi (dollarda, birlik uchun). */
  import_duty_specific?: number
  /** Qat'iy qism birligidagi miqdor. */
  specific_quantity?: number
  excise: number
  vat: number
  fee_exempt?: boolean
  preliminary?: boolean
  inspect_day?: number
  inspect_night?: number
  quantity?: number
}

export interface DutyLineItem {
  /** GTD to'lov kodi: 10, 12, 20, 21, 27, 29. */
  code: string
  name: string
  rate?: number
  base?: number
  amount: number
  note?: string
}

export interface DutyResult {
  customs_value: number
  brv: number
  items: DutyLineItem[]
  total: number
}

/** Utilizatsiya yig'imi (79) — ПКМ 347. */
export interface UtilFeeRequest {
  code: string
  /** Toifaga mos o'lchov: dvigatel hajmi, quvvat yoki to'la vazn. */
  measure?: number
  age_years?: number
  weight_kg?: number
}

export interface UtilFeeResult {
  category: string
  brv: number
  rate?: number
  amount: number
  note?: string
  /** Bo'sh bo'lmasa — qaysi o'lchov kerakligi. */
  needs_measure?: string
}

/** Imtiyoz talabi (hujjat, litsenziya, ozod qilish sharti). */
export interface Requirement {
  category: string
  text: string
  law?: string
  /** Qaysi to'lovdan ozod: boj, aksiz, qqs, yigim. */
  free?: string[]
}

export interface RequirementsResponse {
  code: string
  regime: string
  requirements: Requirement[]
  counts: Record<string, number>
  as_of: string
  /** Ro'yxat to'liq emasligi haqidagi ogohlantirish — ko'rsatilishi shart. */
  note: string
}

/** Davlat — GTD tanlagichlari uchun (kod, nom, boj rejimi). */
export interface Country {
  code: string
  name_ru: string
  name_uz: string
  iso?: string
  regime: string
  /** Boj stavkasi shu songa ko'paytiriladi: 0, 1 yoki 2 (BK 300-modda). */
  duty_multiplier: number
  /** Offshor zona — qo'shimcha nazorat va hujjat talablari bo'lishi mumkin. */
  offshore?: boolean
  /** Muqobil nomlar (XXR, Rossiya Federatsiyasi…) — qidiruvda ishlatiladi. */
  aliases?: string[]
}

export interface ExemptionsResponse {
  note: string
  code?: string
  /** Shu kod bo'yicha qaysi to'lovlardan ozod bo'lish MUMKIN. */
  free?: string[]
  for_code?: Requirement[]
}

export interface ChatImage {
  media_type: string // "image/jpeg", "image/png", ...
  data: string       // base64 (data: prefiksisiz)
}

/**
 * Javob uslubi.
 *
 * DIQQAT: rejim faqat USLUBNI o'zgartiradi — faktlar, stavkalar va
 * ogohlantirishlar ikkalasida bir xil. Soddalashtirilgan javobda
 * ogohlantirish tushib qolmasligi backendda test bilan qo'riqlanadi
 * (TestBothModesKeepWarnings).
 */
export type ChatMode = 'deklarant' | 'tadbirkor'

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
  const res = await authedFetch(path, {
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
    const res = await fetch(url('/api/health'))
    return res.json()
  },
  /**
   * TIF TN qidiruvi.
   *
   * `limit` — taklif ro'yxati uchun. Sukut 5, backend 20 gacha beradi.
   * DIQQAT: `matches` hech narsa topilmasa `null` keladi, bo'sh ro'yxat
   * emas — chaqiruvchi `?? []` qilishi kerak.
   */
  searchHS: (query: string, useAI: boolean, limit?: number) =>
    post<HSSearchResponse>('/api/hscode/search', { query, use_ai: useAI, limit }),
  calculateDuty: (req: DutyRequest) =>
    post<DutyResult>('/api/duty/calculate', req),
  utilFee: (req: UtilFeeRequest) =>
    post<UtilFeeResult>('/api/utilfee/calculate', req),
  exemptions: async (code: string): Promise<ExemptionsResponse> => {
    const res = await fetch(url('/api/exemptions?code=' + encodeURIComponent(code)))
    if (!res.ok) throw new Error(`Xato: ${res.status}`)
    return res.json()
  },
  countries: async (): Promise<{ countries: Country[]; legal: string; warning: string }> => {
    const res = await fetch(url('/api/countries'))
    if (!res.ok) throw new Error(`Xato: `)
    return res.json()
  },
  requirements: async (code: string): Promise<RequirementsResponse> => {
    const res = await fetch(url('/api/requirements?code=' + encodeURIComponent(code)))
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      throw new Error((d as { error?: string }).error || `Xato: ${res.status}`)
    }
    return res.json()
  },
  chat: (messages: ChatMessage[], mode: ChatMode = 'deklarant') =>
    post<{ reply: string }>('/api/chat', { messages, mode }),
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
  mode: ChatMode = 'deklarant',
): Promise<void> {
  const res = await authedFetch('/api/chat/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messages, mode }),
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

/**
 * TIF TN kodini rasmiy ko'rinishda yozadi: 4-2-3-1 → `8418 10 800 2`.
 *
 * Bir bo'lak 10 raqam o'qishga qiyin va nusxa olishda xatoga olib keladi.
 */
export function formatCode(code: string): string {
  if (code.length !== 10) return code
  return `${code.slice(0, 4)} ${code.slice(4, 6)} ${code.slice(6, 9)} ${code.slice(9)}`
}

// ---------------------------------------------------------------- brauzer

/**
 * Ierarxiya tugunini bildiradi.
 *
 * Bitta tur to'rt daraja uchun ham ishlatiladi (bo'lim, guruh, tovar
 * pozitsiyasi, kod) — ularning ko'rinishi bir xil: nom + ichidagi kodlar
 * soni. Farqi faqat `leaf` bayrog'ida: barg bo'lsa stavkalar keladi.
 */
export interface BrowseNode {
  id: string
  title: string
  count?: number
  leaf?: boolean
  import_duty?: number
  vat?: number
  unit?: string
}

export interface BrowseCrumb {
  level: 'section' | 'group' | 'heading'
  id: string
  title: string
}

export interface BrowseResponse {
  level: 'sections' | 'groups' | 'headings' | 'codes'
  parent?: string
  items: BrowseNode[]
  path?: BrowseCrumb[]
}

/**
 * Ierarxiya bo'yicha ko'rish.
 *
 * NEGA KERAK: qidiruv tovarni NOMENKLATURA TILIDA atashni talab qiladi.
 * Foydalanuvchi "musor tashuvchi mashina" deydi, nomenklatura esa "maxsus
 * avtotransport". Ierarxiya hech qanday atama bilishni talab qilmaydi.
 */
export async function browse(params: {
  section?: string
  group?: string
  heading?: string
} = {}): Promise<BrowseResponse> {
  const qs = new URLSearchParams()
  if (params.heading) qs.set('heading', params.heading)
  else if (params.group) qs.set('group', params.group)
  else if (params.section) qs.set('section', params.section)
  const res = await fetch(url('/api/hscode/browse') + (qs.toString() ? '?' + qs : ''))
  if (!res.ok) throw new Error(`Xato: ${res.status}`)
  return res.json()
}

// ---------------------------------------------------------------- qonunlar

export interface LawDoc {
  id: number
  name: string
  date?: string
  since?: string
  lex?: string
  chunks: number
}

export interface LawArticle {
  doc: number
  index: number
  title: string
  preview: string
  lex?: string
}

export interface LawChunk {
  doc: number
  name: string
  date?: string
  since?: string
  lex?: string
  title: string
  text: string
  lang: string
}

export interface LawsBrowseResponse {
  level: 'docs' | 'articles' | 'article'
  docs?: LawDoc[]
  articles?: LawArticle[]
  article?: LawChunk
  doc_name?: string
  doc_lex?: string
}

/**
 * Qonun korpusi bo'yicha ko'rish.
 *
 * Uch daraja: hujjatlar → moddalar → to'liq matn. Ro'yxatlarda faqat
 * matn boshi keladi — bitta kodeksda 500 dan ortiq modda bor va
 * ularning to'liq matni yuzlab kilobayt bo'lardi.
 */
export async function browseLaws(params: { doc?: number; i?: number } = {}): Promise<LawsBrowseResponse> {
  const qs = new URLSearchParams()
  if (params.doc != null) qs.set('doc', String(params.doc))
  if (params.i != null) qs.set('i', String(params.i))
  const res = await fetch(url('/api/laws/browse') + (qs.toString() ? '?' + qs : ''))
  if (!res.ok) {
    const d = await res.json().catch(() => ({}))
    throw new Error((d as { error?: string }).error || `Xato: ${res.status}`)
  }
  return res.json()
}
