import { useMemo, useRef, useState } from 'react'
import type { Country } from '../api'

interface Props {
  countries: Country[]
  /** '' — tanlanmagan, davlat kodi yoki 'unknown'. */
  value: string
  onChange: (code: string) => void
  /** Kelib chiqish uchun: "Aniqlanmagan — boj ×2" bandi. */
  unknownOption?: boolean
}

/** "Aniqlanmagan" bandining maxsus qiymati (BK 300-modda: boj ×2). */
export const UNKNOWN = 'unknown'

/**
 * Davlat tanlagichi — YOZIB QIDIRILADI.
 *
 * NEGA ODDIY <select> EMAS: ro'yxatda 254 davlat bor va ular kod
 * tartibida. "Angola"ni topish uchun 024 gacha aylantirish kerak edi.
 * Endi "ang" yoki "024" yoki "AO" deb yozish yetarli.
 *
 * Qidiruv kod, o'zbekcha/ruscha nom, ISO va sinonimlar bo'ylab —
 * deklarant GTD dagi raqamli kodni ham, kundalik nomni ham ishlatadi
 * ("Rossiya" ham, "643" ham ishlashi kerak).
 */
export default function CountryCombo({ countries, value, onChange, unknownOption }: Props) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const selectedLabel = useMemo(() => {
    if (value === UNKNOWN) return '❓ Aniqlanmagan'
    const c = countries.find((x) => x.code === value)
    return c ? `${c.code} ${c.name_uz}` : ''
  }, [value, countries])

  type Item = { key: string; label: string; sub?: string; offshore?: boolean }

  const items = useMemo<Item[]>(() => {
    const q = query.trim().toLowerCase()
    const out: Item[] = []

    // "Tanlanmagan" va "Aniqlanmagan" — faqat ro'yxat boshida yoki mos
    // kelganda. Ular ham qidiriladi: "aniq" deb yozsa chiqsin.
    if (!q || 'tanlanmagan'.includes(q)) {
      out.push({ key: '', label: '— tanlanmagan —' })
    }
    if (unknownOption && (!q || 'aniqlanmagan'.includes(q) || '×2'.includes(q))) {
      out.push({ key: UNKNOWN, label: '❓ Aniqlanmagan', sub: 'boj ×2 (BK 300-modda)' })
    }

    // Aniq mosliklar OLDINDA tursin: "de" yozilganda Germaniya (ISO: DE)
    // "Bangladesh"dan yuqorida chiqishi kerak — davlatlar ro'yxati kod
    // tartibida va saralashsiz tasodifiy substring g'olib bo'lardi.
    const scored: { c: Country; rank: number }[] = []
    for (const c of countries) {
      const rank = q ? matchRank(c, q) : 3
      if (rank < 0) continue
      scored.push({ c, rank })
    }
    scored.sort((a, b) => a.rank - b.rank)

    for (const { c } of scored) {
      out.push({
        key: c.code,
        label: `${c.code} ${c.name_uz}`,
        sub: regimeLabel(c),
        offshore: c.offshore,
      })
      // Ro'yxatni chegaralaymiz: "a" deb yozilganda 200 ta DOM tuguni
      // yasashning ma'nosi yo'q — foydalanuvchi baribir aniqlashtiradi.
      if (out.length >= 40) break
    }
    return out
  }, [countries, query, unknownOption])

  const pick = (key: string) => {
    onChange(key)
    setOpen(false)
    setQuery('')
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
        if (items[active]) pick(items[active].key)
        break
      case 'Escape':
        setOpen(false)
        setQuery('')
        inputRef.current?.blur()
        break
    }
  }

  return (
    <div className="combo">
      <input
        ref={inputRef}
        role="combobox"
        aria-expanded={open}
        aria-autocomplete="list"
        // Yopiq holatda tanlangan qiymat, ochiqda — qidiruv matni.
        value={open ? query : selectedLabel}
        placeholder={open ? 'Kod yoki nom yozing…' : '— tanlanmagan —'}
        onFocus={() => { setOpen(true); setQuery(''); setActive(0) }}
        // Bandga bosish blur'dan OLDIN ishlashi uchun ro'yxatda
        // onMouseDown ishlatiladi; bu yerda esa oynani kechiktirib
        // yopamiz — aks holda tanlov ulgurmay qolardi.
        onBlur={() => setTimeout(() => setOpen(false), 120)}
        onChange={(e) => { setQuery(e.target.value); setActive(0) }}
        onKeyDown={onKey}
      />
      {open && (
        <ul className="combo-list" role="listbox">
          {items.length === 0 && <li className="combo-empty">Topilmadi</li>}
          {items.map((it, i) => (
            <li
              key={it.key || 'none'}
              role="option"
              aria-selected={it.key === value}
              className={'combo-item' + (i === active ? ' active' : '')}
              onMouseDown={(e) => { e.preventDefault(); pick(it.key) }}
              onMouseEnter={() => setActive(i)}
            >
              <span className="combo-label">
                {it.label}
                {it.offshore && <span className="combo-off">offshor</span>}
              </span>
              {it.sub && <span className="combo-sub">{it.sub}</span>}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

/**
 * Moslik darajasi: 0 — eng aniq, -1 — mos emas.
 *
 *	0  kod prefiksi ("156") yoki ISO aynan ("de", "cn")
 *	1  nom boshlanishi ("ang" → Angola)
 *	2  nom ichida ("ang" → Bangladesh)
 */
function matchRank(c: Country, q: string): number {
  if (c.code.startsWith(q)) return 0
  if (c.iso && c.iso.toLowerCase() === q) return 0
  const uz = c.name_uz.toLowerCase()
  const ru = c.name_ru.toLowerCase()
  if (uz.startsWith(q) || ru.startsWith(q)) return 1
  if (uz.includes(q) || ru.includes(q)) return 2
  if ((c.aliases ?? []).some((a) => a.toLowerCase().includes(q))) return 2
  return -1
}

/** Boj rejimi — band ostida darrov ko'rinsin, tanlab bilib o'tirmasin. */
function regimeLabel(c: Country): string {
  switch (c.duty_multiplier) {
    case 0: return 'erkin savdo — boj yo\'q'
    case 2: return 'rejim yo\'q — boj ×2'
    default: return 'eng qulaylik rejimi'
  }
}
