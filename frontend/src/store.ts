import { useSyncExternalStore } from 'react'
import type { ChatMessage } from './api'

/**
 * Tarixcha va sevimlilar — BRAUZERDA saqlanadi.
 *
 * NEGA SHUNDAY: server tarafda saqlash akkaunt va bazani talab qiladi,
 * ular esa keyingi bosqichda. Lekin bu bo'limlarni "keyin" deb qoldirish
 * ham noto'g'ri bo'lardi — brauzerda saqlash BUGUN ishlaydi va
 * foydalanuvchi uchun haqiqiy qiymat beradi.
 *
 * CHEGARASI OCHIQ AYTILADI (interfeysda ham): yozuvlar shu qurilmada,
 * shu brauzerda qoladi. Boshqa qurilmaga ko'chmaydi, brauzer ma'lumoti
 * tozalansa yo'qoladi. Akkauntlar kelganda shu ma'lumot serverga
 * ko'chiriladi.
 */

const CHATS = 'deklarant.chats'
const FAVS = 'deklarant.favorites'

/** Saqlanadigan suhbatlar soni. */
const MAX_CHATS = 50

export interface SavedChat {
  id: string
  title: string
  at: number
  messages: ChatMessage[]
}

export interface Favorite {
  kind: 'code' | 'law'
  id: string
  title: string
  /** Qo'shimcha satr: stavkalar yoki hujjat nomi. */
  meta?: string
  at: number
}

// ---------------------------------------------------------------- o'qish

function read<T>(key: string): T[] {
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T[]) : []
  } catch {
    return [] // yozuv buzuq yoki localStorage yopiq
  }
}

function write(key: string, value: unknown): boolean {
  try {
    localStorage.setItem(key, JSON.stringify(value))
    emit()
    return true
  } catch {
    // Joy tugadi yoki localStorage yopiq. Jimgina o'tamiz: saqlanmagani
    // ilovaning ishlashini to'xtatmasligi kerak.
    return false
  }
}

// ------------------------------------------------------- obuna (React uchun)

const listeners = new Set<() => void>()
function emit() {
  for (const l of listeners) l()
}
function subscribe(l: () => void) {
  listeners.add(l)
  // Boshqa tabda o'zgarsa ham xabar beramiz.
  const onStorage = () => l()
  window.addEventListener('storage', onStorage)
  return () => {
    listeners.delete(l)
    window.removeEventListener('storage', onStorage)
  }
}

// `useSyncExternalStore` har chaqiruvda BIR XIL havolani kutadi, aks holda
// cheksiz qayta chizish boshlanadi — shuning uchun keshlaymiz.
let chatsCache: SavedChat[] | null = null
let favsCache: Favorite[] | null = null
function invalidate() {
  chatsCache = null
  favsCache = null
}
listeners.add(invalidate)

export function useChats(): SavedChat[] {
  return useSyncExternalStore(subscribe, () => (chatsCache ??= read<SavedChat>(CHATS)), () => [])
}

export function useFavorites(): Favorite[] {
  return useSyncExternalStore(subscribe, () => (favsCache ??= read<Favorite>(FAVS)), () => [])
}

// ---------------------------------------------------------------- suhbatlar

/**
 * Suhbatni saqlaydi (bor bo'lsa yangilaydi).
 *
 * SURATLAR OLIB TASHLANADI: bitta 1600px surat base64 da ~300 KB, brauzer
 * xotirasi esa odatda 5 MB. Ikki-uchta suratli suhbat butun tarixchani
 * to'ldirib qo'yardi va saqlash umuman ishlamay qolardi.
 */
export function saveChat(id: string, messages: ChatMessage[]): void {
  if (messages.length === 0) return
  const all = read<SavedChat>(CHATS).filter((c) => c.id !== id)
  const first = messages.find((m) => m.role === 'user')?.content?.trim() ?? ''

  all.unshift({
    id,
    title: first.slice(0, 80) || 'Suratli savol',
    at: Date.now(),
    messages: messages.map((m) => ({ role: m.role, content: m.content })),
  })
  write(CHATS, all.slice(0, MAX_CHATS))
}

export function deleteChat(id: string): void {
  write(CHATS, read<SavedChat>(CHATS).filter((c) => c.id !== id))
}

export function clearChats(): void {
  write(CHATS, [])
}

// --------------------------------------------------------------- sevimlilar

export function toggleFavorite(fav: Omit<Favorite, 'at'>): void {
  const all = read<Favorite>(FAVS)
  const i = all.findIndex((f) => f.kind === fav.kind && f.id === fav.id)
  if (i >= 0) all.splice(i, 1)
  else all.unshift({ ...fav, at: Date.now() })
  write(FAVS, all)
}

export function removeFavorite(kind: Favorite['kind'], id: string): void {
  write(FAVS, read<Favorite>(FAVS).filter((f) => !(f.kind === kind && f.id === id)))
}

/** Sana — ro'yxatlarda bir xil ko'rinishda. */
export function formatWhen(at: number): string {
  return new Date(at).toLocaleString('uz-UZ', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}
