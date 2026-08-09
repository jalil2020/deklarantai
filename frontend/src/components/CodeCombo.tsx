import { useEffect, useRef, useState } from 'react'
import { api, formatCode, type HSMatch } from '../api'

interface Props {
  /** Tanlangan kod (10 raqam) yoki ''. */
  value: string
  /** Kod tanlanganda — to'liq yozuv bilan (stavkalar shu yerda). */
  onPick: (m: HSMatch) => void
  /** Tanlovni bekor qilish. */
  onClear: () => void
}

/** Taklif ro'yxatida nechta variant. */
const SUGGEST_LIMIT = 12

/** Qidiruv boshlanadigan eng qisqa so'rov. */
const MIN_QUERY = 2

/**
 * TIF TN kodi maydoni — YOZGANDA qidiradi.
 *
 * Ikki xil yozuvni ham qabul qiladi va bu ataylab:
 *
 *	raqam  — "8703", "8703 23" — kodni bilgan deklarant uchun
 *	so'z   — "noutbuk", "naushnik" — kodni bilmagani uchun
 *
 * Ya'ni alohida "Qidiruv" bo'limiga o'tish shart emas: kalkulyator
 * o'rnidan turmasdan kod topiladi va stavkalar darrov tortiladi.
 *
 * Qidiruv SERVERDA (13 142 kod brauzerga yuklanmaydi), shuning uchun
 * har harfda so'rov yubormaslik uchun kechiktiriladi (debounce).
 */
export default function CodeCombo({ value, onPick, onClear }: Props) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<HSMatch[]>([])
  const [active, setActive] = useState(0)
  const [busy, setBusy] = useState(false)
  const [empty, setEmpty] = useState(false)

  const inputRef = useRef<HTMLInputElement>(null)
  // Har so'rovga raqam beramiz: sekin kelgan eski javob yangisini
  // bosib ketmasin (poyga holati).
  const seq = useRef(0)

  // Tanlangan kod tashqaridan o'zgarsa (qidiruv yoki ierarxiya orqali),
  // maydon ham yangilanadi.
  useEffect(() => {
    if (!open) setQuery(value ? formatCode(value) : '')
  }, [value, open])

  useEffect(() => {
    if (!open) return
    const q = query.trim()
    if (q.length < MIN_QUERY) {
      setItems([])
      setEmpty(false)
      return
    }

    // 200 ms — teruvchi to'xtaganda so'rov ketadi. Undan qisqasi har
    // harfda so'rov yuborardi, uzunrog'i esa sekin his qilinardi.
    const id = setTimeout(async () => {
      const my = ++seq.current
      setBusy(true)
      try {
        const r = await api.searchHS(q, false, SUGGEST_LIMIT)
        if (my !== seq.current) return // eskirgan javob
        const list = r.matches ?? []
        setItems(list)
        setEmpty(list.length === 0)
        setActive(0)
      } catch {
        if (my === seq.current) { setItems([]); setEmpty(true) }
      } finally {
        if (my === seq.current) setBusy(false)
      }
    }, 200)
    return () => clearTimeout(id)
  }, [query, open])

  const pick = (m: HSMatch) => {
    onPick(m)
    setOpen(false)
    setQuery(formatCode(m.code.code))
    inputRef.current?.blur()
  }

  const onKey = (e: React.KeyboardEvent) => {
    if (!open) return
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setActive((a) => Math.min(a + 1, items.length - 1))
        break
      case 'ArrowUp':
        e.preventDefault()
        setActive((a) => Math.max(a - 1, 0))
        break
      case 'Enter':
        e.preventDefault()
        if (items[active]) pick(items[active])
        break
      case 'Escape':
        setOpen(false)
        setQuery(value ? formatCode(value) : '')
        inputRef.current?.blur()
        break
    }
  }

  return (
    <div className="combo code-combo">
      <input
        ref={inputRef}
        className="code-input"
        role="combobox"
        aria-expanded={open}
        aria-autocomplete="list"
        value={query}
        placeholder="Kod yoki tovar nomi…"
        onFocus={() => { setOpen(true); setActive(0) }}
        // Bandga bosish blur'dan OLDIN ishlashi uchun ro'yxatda
        // onMouseDown ishlatiladi; bu yerda oynani kechiktirib yopamiz.
        onBlur={() => setTimeout(() => setOpen(false), 120)}
        onChange={(e) => { setQuery(e.target.value); setOpen(true) }}
        onKeyDown={onKey}
        aria-label="TIF TN kodi yoki tovar nomi"
      />

      {value && !open && (
        <button
          className="head-x"
          onClick={onClear}
          aria-label="Kodni tozalash"
          title="Kodni tozalash"
        >✕</button>
      )}

      {open && (
        <ul className="combo-list code-list" role="listbox">
          {busy && items.length === 0 && <li className="combo-empty">Qidirilmoqda…</li>}
          {!busy && query.trim().length < MIN_QUERY && (
            <li className="combo-empty">
              Kod raqamini yoki tovar nomini yozing — masalan «8703» yoki «noutbuk»
            </li>
          )}
          {empty && !busy && <li className="combo-empty">Topilmadi</li>}

          {items.map((m, i) => (
            <li
              key={m.code.code}
              role="option"
              aria-selected={m.code.code === value}
              className={'combo-item' + (i === active ? ' active' : '')}
              onMouseDown={(e) => { e.preventDefault(); pick(m) }}
              onMouseEnter={() => setActive(i)}
            >
              <span className="combo-label">
                <b className="mono">{formatCode(m.code.code)}</b>
                <span className="code-rates">
                  {m.code.import_duty}% boj
                  {/* Kombinatsiyalangan stavka — 1 555 kodda. Taklifda
                      ham ko'rinsin: faqat foizni ko'rgan deklarant
                      bojni kam hisoblardi. */}
                  {m.code.import_duty_specific != null &&
                    ` · $${m.code.import_duty_specific}/${m.code.import_duty_specific_unit}`}
                  {m.code.unit ? ` · ${m.code.unit}` : ' · kg'}
                </span>
              </span>
              <span className="combo-sub">{m.code.name_uz || m.code.name_ru}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
