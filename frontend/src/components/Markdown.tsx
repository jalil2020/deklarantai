import type { ReactNode } from 'react'

/**
 * Kichik markdown renderer.
 *
 * NEGA O'ZIMIZNIKI: loyihada atigi ikkita bog'liqlik bor (react, react-dom).
 * react-markdown butun ekotizimni (remark/unified, ~20 paket) tortib keladi,
 * bizga esa modelning ishlatadigan bir nechta elementi yetarli.
 *
 * NEGA HTML EMAS: dangerouslySetInnerHTML ishlatilmaydi. Matn modeldan
 * keladi, unda foydalanuvchi yuborgan (masalan rasmdagi) matn ham bo'lishi
 * mumkin — HTML ga aylantirsak XSS yo'li ochilardi. Shuning uchun faqat
 * React elementlari yasaladi, teglar hech qachon matndan kelmaydi.
 *
 * Qo'llab-quvvatlanadi: sarlavha, jadval, ro'yxat (belgili/raqamli),
 * **qalin**, *kursiv*, `kod`, havola, --- ajratgich, iqtibos.
 */
export default function Markdown({ text }: { text: string }) {
  return <>{blocks(text.replace(/\r\n/g, '\n'))}</>
}

// ------------------------------------------------------------ blok darajasi

function blocks(src: string): ReactNode[] {
  const lines = src.split('\n')
  const out: ReactNode[] = []
  let i = 0
  let key = 0

  // Paragrafga yig'ilayotgan qatorlar. Bo'sh qator yoki boshqa blok
  // boshlanganda chiqariladi.
  let para: string[] = []
  const flush = () => {
    if (!para.length) return
    out.push(<p key={key++} className="md-p">{inline(para.join('\n'))}</p>)
    para = []
  }

  while (i < lines.length) {
    const ln = lines[i]

    if (!ln.trim()) { flush(); i++; continue }

    // Ajratgich: --- yoki ***
    if (/^\s*([-*_])\s*\1\s*\1[\s-*_]*$/.test(ln)) {
      flush()
      out.push(<hr key={key++} className="md-hr" />)
      i++
      continue
    }

    // Sarlavha: # dan ###### gacha
    const h = ln.match(/^(#{1,6})\s+(.*)$/)
    if (h) {
      flush()
      const lvl = Math.min(h[1].length, 6)
      // h1/h2 ni ham h3 dan boshlab beramiz — suhbat ichida sahifa
      // sarlavhasidek ulkan matn kerak emas.
      const Tag = (`h${Math.min(lvl + 2, 6)}`) as 'h3' | 'h4' | 'h5' | 'h6'
      out.push(<Tag key={key++} className={`md-h md-h${lvl}`}>{inline(h[2])}</Tag>)
      i++
      continue
    }

    // Jadval: sarlavha qatori + ajratgich (|---|---|)
    if (ln.includes('|') && i + 1 < lines.length && isDivider(lines[i + 1])) {
      flush()
      const head = cells(ln)
      const align = cells(lines[i + 1]).map(alignOf)
      i += 2
      const rows: string[][] = []
      while (i < lines.length && lines[i].includes('|') && lines[i].trim()) {
        rows.push(cells(lines[i]))
        i++
      }
      out.push(
        <div key={key++} className="md-table-wrap">
          <table className="md-table">
            <thead>
              <tr>{head.map((c, j) => <th key={j} style={{ textAlign: align[j] }}>{inline(c)}</th>)}</tr>
            </thead>
            <tbody>
              {rows.map((r, j) => (
                <tr key={j}>
                  {/* Sarlavhada nechta ustun bo'lsa, shuncha katak chiqaramiz:
                      model ba'zan qatorda ustun tashlab ketadi. */}
                  {head.map((_, k) => <td key={k} style={{ textAlign: align[k] }}>{inline(r[k] ?? '')}</td>)}
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      )
      continue
    }

    // Iqtibos: > matn
    if (/^\s*>\s?/.test(ln)) {
      flush()
      const buf: string[] = []
      while (i < lines.length && /^\s*>\s?/.test(lines[i])) {
        buf.push(lines[i].replace(/^\s*>\s?/, ''))
        i++
      }
      out.push(<blockquote key={key++} className="md-quote">{blocks(buf.join('\n'))}</blockquote>)
      continue
    }

    // Ro'yxat: "- ", "* ", "1. "
    if (isItem(ln)) {
      flush()
      const ordered = /^\s*\d+[.)]\s+/.test(ln)
      const items: string[][] = []
      while (i < lines.length && (isItem(lines[i]) || (items.length > 0 && cont(lines[i])))) {
        if (isItem(lines[i])) {
          items.push([lines[i].replace(/^\s*(?:[-*•]|\d+[.)])\s+/, '')])
        } else {
          // Davomi: oldingi bandga qo'shiladi (model uzun bandni bo'ladi).
          items[items.length - 1].push(lines[i].trim())
        }
        i++
      }
      const List = ordered ? 'ol' : 'ul'
      out.push(
        <List key={key++} className="md-list">
          {items.map((it, j) => <li key={j}>{inline(it.join(' '))}</li>)}
        </List>,
      )
      continue
    }

    para.push(ln)
    i++
  }
  flush()
  return out
}

const isItem = (s: string) => /^\s*(?:[-*•]|\d+[.)])\s+/.test(s)
// Ro'yxat bandining davomi: bo'sh emas, yangi band emas, va chekinish bilan.
const cont = (s: string) => Boolean(s.trim()) && !isItem(s) && /^\s{2,}/.test(s)

/** "|---|:--:|" ko'rinishidagi jadval ajratgichimi. */
function isDivider(s: string): boolean {
  return s.includes('|') && /^[\s|:-]+$/.test(s) && s.includes('-')
}

/** "| a | b |" → ["a", "b"] */
function cells(s: string): string[] {
  return s.trim().replace(/^\|/, '').replace(/\|$/, '').split('|').map((c) => c.trim())
}

function alignOf(spec: string): 'left' | 'center' | 'right' {
  const l = spec.startsWith(':')
  const r = spec.endsWith(':')
  if (l && r) return 'center'
  if (r) return 'right'
  return 'left'
}

// ------------------------------------------------------------ satr ichi

// Tartib muhim: `kod` birinchi bo'lishi kerak, aks holda kod ichidagi
// ** belgilari qalin matn deb o'qilardi.
const INLINE = /(`[^`]+`)|(\*\*[^*]+\*\*)|(\*[^*\n]+\*)|(\[[^\]]+\]\([^)\s]+\))|(https?:\/\/[^\s<>()]+)/g

function inline(text: string): ReactNode[] {
  const out: ReactNode[] = []
  let last = 0
  let key = 0
  for (const m of text.matchAll(INLINE)) {
    const at = m.index ?? 0
    if (at > last) out.push(...breaks(text.slice(last, at), key++))
    const tok = m[0]
    if (tok.startsWith('`')) {
      out.push(<code key={key++} className="md-code">{tok.slice(1, -1)}</code>)
    } else if (tok.startsWith('**')) {
      out.push(<strong key={key++}>{tok.slice(2, -2)}</strong>)
    } else if (tok.startsWith('*')) {
      out.push(<em key={key++}>{tok.slice(1, -1)}</em>)
    } else if (tok.startsWith('[')) {
      const cut = tok.indexOf('](')
      out.push(link(tok.slice(1, cut), tok.slice(cut + 2, -1), key++))
    } else {
      out.push(link(tok, tok, key++))
    }
    last = at + tok.length
  }
  if (last < text.length) out.push(...breaks(text.slice(last), key++))
  return out
}

/**
 * Havola. Faqat http(s) ga ruxsat beramiz — matn modeldan kelgani uchun
 * javascript: kabi sxema o'tib ketmasligi kerak.
 */
function link(label: string, href: string, key: number): ReactNode {
  if (!/^https?:\/\//i.test(href)) return <span key={key}>{label}</span>
  return (
    <a key={key} className="md-link" href={href} target="_blank" rel="noopener noreferrer">
      {label}
    </a>
  )
}

/** Paragraf ichidagi yangi qatorlarni <br> ga aylantiradi. */
function breaks(s: string, key: number): ReactNode[] {
  const parts = s.split('\n')
  const out: ReactNode[] = []
  parts.forEach((p, i) => {
    if (i > 0) out.push(<br key={`${key}-b${i}`} />)
    if (p) out.push(<span key={`${key}-t${i}`}>{p}</span>)
  })
  return out
}
